package reload_cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"ssl_reload/conf"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	config        string
	domain        string
	certSourceDir string
	keyMod        string
	StartCmd      = &cobra.Command{
		Use:          "reload",
		Example:      "ssl_renewal reload -c config.json -d you.com",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&config, "config", "c", "config.json", "provided configuration file")
	StartCmd.Flags().StringVarP(&domain, "domain", "d", "", "域名")
	StartCmd.Flags().StringVarP(&certSourceDir, "cert-dir", "", "", "SSL Certificate Directory")
	StartCmd.Flags().StringVarP(&keyMod, "key_mod", "", "", "证书类型")
	StartCmd.MarkFlagRequired("domain")
}

func run() error {
	var c conf.ConfList
	if err := conf.Load(config, &c); err != nil {
		log.Fatalf("❌ error: config file %s, %s", config, err.Error())
	}

	if len(c.List) == 0 {
		return fmt.Errorf("❌ conf_list is empty")
	}

	if domain == "" {
		return fmt.Errorf("❌ domain is required")
	}

	if certSourceDir == "" {
		if c.CertSourceDir == "" {
			exePath, err := os.Executable()
			if err != nil {
				fmt.Println("获取可执行文件路径失败:", err)
				certSourceDir = "/tmp/cert_zip"
			} else {
				// 获取可执行文件所在目录
				certSourceDir = filepath.Join(filepath.Dir(exePath), "cert_zip")
			}
		} else {
			certSourceDir = c.CertSourceDir
		}
	}

	certSourceDir = filepath.Join(certSourceDir, keyMod)
	err := os.MkdirAll(certSourceDir, 0755)
	if err != nil {
		fmt.Printf("❌ 创建目录失败: %v\n", err)
		return nil
	}

	// 打包
	var (
		files     = []string{filepath.Join(certSourceDir, domain+".key"), filepath.Join(certSourceDir, domain+".pem")}
		fileInfos []FileInfo
	)

	for _, filePath := range files {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			log.Fatalf("❌ 解析文件路径失败: %s: %v", filePath, err)
		}
		fileInfos = append(fileInfos, FileInfo{
			SourcePath: absPath,
			TarPath:    filepath.Base(filePath),
		})
	}

	var CERT_PACKAGE = filepath.Join(certSourceDir, domain+".tar.gz")

	fmt.Printf("📦 打包中 %s", CERT_PACKAGE)
	if err = TarSpecificFiles(CERT_PACKAGE, fileInfos, true); err != nil {
		log.Fatalf("❌ TarSpecificFiles : %v", err)
	}
	fmt.Printf("📦 打包完成 %s", CERT_PACKAGE)

	confList := getDomainConfs(c, domain)
	if err := DeployWithSSH(confList, CERT_PACKAGE, keyMod); err != nil {
		log.Fatalf("❌ 部署失败: %v", err)
	}

	return nil
}

// getDomainConfs 获取这个域名的 目标配置
func getDomainConfs(c conf.ConfList, domain string) []*conf.Target {
	var list []*conf.Target
	var set = map[string]bool{}
	for _, c2 := range c.List {
		if c2.Domain == domain && c2.Status == 1 {
			for _, target := range c2.Targets {
				if target.Status == 1 {
					key := domain + target.TargetIP + target.TargetPort + target.TargetUser + target.TargetDir
					set[key] = true
					list = append(list, target)
				}
			}
		}
	}

	return list
}

// FileInfo 文件信息结构
type FileInfo struct {
	SourcePath string // 源文件路径
	TarPath    string // 在tar包中的路径
}

// TarCzvf 打包整个目录
func TarCzvf(outputFile, sourceDir string, verbose bool) error {
	return AdvancedTarCzvf(outputFile, sourceDir, nil, nil, verbose)
}

// TarSpecificFiles 打包特定文件
func TarSpecificFiles(outputFile string, files []FileInfo, verbose bool) error {
	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("❌ 创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 创建 gzip 写入器
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	// 创建 tar 写入器
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, fileInfo := range files {
		// 检查文件是否存在
		if _, err := os.Stat(fileInfo.SourcePath); os.IsNotExist(err) {
			return fmt.Errorf("❌ 文件不存在: %s", fileInfo.SourcePath)
		}

		// 获取文件信息
		info, err := os.Stat(fileInfo.SourcePath)
		if err != nil {
			return fmt.Errorf("❌ 获取文件信息失败: %s: %v", fileInfo.SourcePath, err)
		}

		// 创建 tar 头部
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("❌ 创建tar头部失败: %s: %v", fileInfo.SourcePath, err)
		}

		// 设置tar包中的路径
		if fileInfo.TarPath != "" {
			header.Name = fileInfo.TarPath
		} else {
			header.Name = filepath.Base(fileInfo.SourcePath)
		}

		// 写入头部
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("❌ 写入tar头部失败: %s: %v", fileInfo.SourcePath, err)
		}

		// 如果是普通文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(fileInfo.SourcePath)
			if err != nil {
				return fmt.Errorf("❌ 打开文件失败: %s: %v", fileInfo.SourcePath, err)
			}

			if _, err := io.Copy(tarWriter, file); err != nil {
				file.Close()
				return fmt.Errorf("❌ 写入文件内容失败: %s: %v", fileInfo.SourcePath, err)
			}
			file.Close()

			if verbose {
				fmt.Printf("📝 添加文件: %s -> %s\n", fileInfo.SourcePath, header.Name)
			}
		} else if verbose {
			fmt.Printf("📝 添加目录: %s -> %s/\n", fileInfo.SourcePath, header.Name)
		}
	}

	return nil
}

// AdvancedTarCzvf 增强版的打包函数
func AdvancedTarCzvf(outputFile, sourceDir string, excludePatterns []string, specificFiles []string, verbose bool) error {
	// 如果有特定文件，优先处理特定文件
	if len(specificFiles) > 0 {
		var files []FileInfo
		for _, filePath := range specificFiles {
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return err
			}
			files = append(files, FileInfo{
				SourcePath: absPath,
				TarPath:    filepath.Base(filePath),
			})
		}
		return TarSpecificFiles(outputFile, files, verbose)
	}

	// 原始目录打包逻辑...
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("❌ 创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if isExcluded(relPath, excludePatterns) {
			if verbose {
				fmt.Printf("❌ 排除: %s\n", relPath)
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}

			if verbose {
				fmt.Printf("📝 添加: %s\n", relPath)
			}
		} else if verbose {
			fmt.Printf("📝 添加目录: %s/\n", relPath)
		}

		return nil
	})
}

// isExcluded 检查文件是否在排除列表中
func isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/") + "/*"
			matched, err = filepath.Match(dirPattern, path)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

type SSHConfig struct {
	User    string
	Host    string
	Port    string
	Mod     int
	KeyPath string
}

// DeployWithSSH 使用纯 Go SSH 实现部署
func DeployWithSSH(confList []*conf.Target, certPackage, keyMod string) error {
	certData, err := os.ReadFile(certPackage)
	if err != nil {
		return fmt.Errorf("❌ 读取证书文件失败: %v", err)
	}

	certFileName := filepath.Base(certPackage)

	for _, c := range confList {
		fmt.Printf("🚀 === 正在部署到服务器: %s === %s \n", c.TargetIP, filepath.Join(c.TargetDir, keyMod))

		if c.TargetIP == "" {
			fmt.Printf("❌ Name:%s TargetIP is empty", c.TargetName)
		}

		var user = c.TargetUser
		if user == "" {
			user = "root"
		}

		var port string
		if c.TargetPort == "" {
			port = "22"
		}

		sshConfig := &SSHConfig{
			User:    user,
			Host:    c.TargetIP,
			Port:    port,
			Mod:     c.TargetMod,
			KeyPath: c.TargetKey,
		}

		client, err := createSSHClient(sshConfig)
		if err != nil {
			log.Printf("❌ SSH连接失败 %s: %v", c.TargetIP, err)
			continue
		}
		defer client.Close()

		// 上传文件

		remotePath := filepath.Join("/tmp", keyMod, certFileName)
		log.Printf("remotePath:%s", remotePath)
		if err := uploadFile(client, certData, remotePath); err != nil {
			log.Printf("❌ 文件上传失败 %s: %v", c.TargetIP, err)
			continue
		}

		if c.TargetDir != "" {
			// 执行命令
			if err := executeCommands(client, filepath.Join(c.TargetDir, keyMod), certFileName, keyMod, c.ReloadCmd); err != nil {
				log.Printf("❌ 命令执行失败 %s: %v", c.TargetIP, err)
				continue
			}
		}

		fmt.Printf("✅ 服务器 %s 部署成功\n", c.TargetIP)
	}
	return nil
}

func createSSHClient(config *SSHConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// 使用 SSH 密钥认证

	if config.Mod == 1 {
		ssh.Password(config.KeyPath)
	} else {
		var keyPath = config.KeyPath
		if keyPath == "" {
			keyPath = "/root/.ssh/id_rsa"
		}

		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("❌ 读取SSH密钥失败: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("❌ 解析SSH密钥失败: %v", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 配置 SSH 客户端
	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	address := net.JoinHostPort(config.Host, config.Port)
	fmt.Printf("🛰️ 远程链接: %s@%s\n", sshConfig.User, address)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("❌ address:%s@%s SSH连接失败: %v", sshConfig.User, address, err)
	}

	return client, nil
}

func uploadFile(client *ssh.Client, data []byte, remotePath string) error {

	session, err := client.NewSession()
	session.Run(fmt.Sprintf("mkdir -p %s", filepath.Dir(remotePath)))
	session.Close()

	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		stdin, _ := session.StdinPipe()
		defer stdin.Close()

		fmt.Fprintf(stdin, "C0644 %d %s\n", len(data), filepath.Base(remotePath))
		stdin.Write(data)
		fmt.Fprint(stdin, "\x00")
		fmt.Printf("🔼 上传中:")
	}()

	cmd := fmt.Sprintf("/usr/bin/scp -t %s", filepath.Dir(remotePath))
	fmt.Printf("⬆️ 上传: %s\n", cmd)
	return session.Run(cmd)
}

func executeCommands(client *ssh.Client, remotePath, certFileName, keyMod, reloadCmd string) error {
	commands := []string{
		fmt.Sprintf("mkdir -p %s", remotePath),
		fmt.Sprintf("tar -xzvf %s -C %s", filepath.Join("/tmp", keyMod, certFileName), remotePath),
		fmt.Sprintf("find %s -name '*.key' -exec chmod 600 {} \\;", remotePath),
		fmt.Sprintf("find %s -name '*.pem' -exec chmod 644 {} \\;", remotePath),
		reloadCmd,
		"echo \"✅ 证书部署成功。\"",
	}

	for _, cmd := range commands {
		fmt.Printf("🔧 执行命令: %s\n", cmd)
		if cmd == "" {
			continue
		}

		session, err := client.NewSession()
		if err != nil {
			return fmt.Errorf("❌创建SSH会话失败: %v", err)
		}

		var stdout, stderr bytes.Buffer
		session.Stdout = &stdout
		session.Stderr = &stderr

		if err := session.Run(cmd); err != nil {
			session.Close()
			return fmt.Errorf("❌ 命令执行失败: %s\n错误: %v\n输出: %s", cmd, err, stderr.String())
		}

		if output := stdout.String(); output != "" {
			fmt.Printf("输出: %s\n", output)
		}

		session.Close()
	}

	return nil
}
