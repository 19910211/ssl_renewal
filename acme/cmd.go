package acme

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"ssl_reload/conf"

	"github.com/spf13/cobra"
)

var (
	config, domain, server, days    string
	staging, force, renew, renewAll bool
	StartCmd                        = &cobra.Command{
		Use:          "acme",
		Example:      "ssl_renewal acme -c config.json -d you.com --server letsencrypt --days 60 --staging",
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&config, "config", "c", "config.json", "provided configuration file")
	StartCmd.PersistentFlags().StringVarP(&server, "server", "", "", "证书服务商")
	StartCmd.PersistentFlags().StringVarP(&days, "days", "", "", "有效天数")
	StartCmd.Flags().StringVarP(&domain, "domain", "d", "", "域名")
	StartCmd.Flags().BoolVarP(&staging, "staging", "", false, "测试")
	StartCmd.Flags().BoolVarP(&renew, "renew", "", false, "更新单个")
	StartCmd.Flags().BoolVarP(&renewAll, "renew-all", "", false, "全部更新")
	StartCmd.Flags().BoolVarP(&force, "force", "", false, "强制更新")
	StartCmd.MarkFlagRequired("domain")
}

func run() error {
	var c conf.ConfList
	if err := conf.Load(config, &c); err != nil {
		log.Fatalf("error: config file %s, %s", config, err.Error())
	}

	if len(c.List) == 0 {
		return fmt.Errorf("❌ conf_list is empty")
	}

	if domain == "" {
		return fmt.Errorf("❌ domain is required")
	}

	conf := getDomainConf(c, domain)
	if conf == nil {
		return fmt.Errorf(fmt.Sprintf("❌ conf:%s is nil", domain))
	}

	switch conf.DNS {
	case "dns_ali": // 阿里云
		os.Setenv("Ali_Key", conf.AccessKey)
		os.Setenv("Ali_Secret", conf.AccessSecret)
	case "dns_tencent": // 腾讯云
		os.Setenv("TENCENTCLOUD_SECRET_ID", conf.AccessKey)
		os.Setenv("TENCENTCLOUD_SECRET_KEY", conf.AccessSecret)
	case "dns_huaweicloud": // 华为云
		os.Setenv("HUAWEICLOUD_ACCESS_KEY", conf.AccessKey)
		os.Setenv("HUAWEICLOUD_SECRET_KEY", conf.AccessSecret)
	case "dns_volcengine": // 火山云
		os.Setenv("VOLC_ACCESSKEY", conf.AccessKey)
		os.Setenv("VOLC_SECRETKEY", conf.AccessSecret)
	default:
		fmt.Println("使用系统自带的")
	}
	const acmeShPath = "/root/.acme.sh/acme.sh"
	//--issue --dns dns_ali -d "asleyu.com" -d "*.asleyu.com" --days 70 --server letsencrypt --staging
	var args = []string{
		"--issue",
		"--dns",
		conf.DNS,
		"-d",
		domain,
		"-d",
		"*." + domain,
	}

	if days != "" {
		args = append(args, "--days", days)
	}

	if server != "" {
		args = append(args, "--server", server)
	}

	if staging {
		args = append(args, "--staging")
	}

	if force {
		args = append(args, "--force")
	}

	if renew {
		args = append(args, "--renew")
	}

	if renewAll {
		args = append(args, "--renew-all")
	}

	// 调用 acme.sh
	cmd := exec.Command(acmeShPath, args...)
	fmt.Println(cmd.String())
	fmt.Printf("🛠️ 构建中 %s\n", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "❌ 执行 acme.sh 出错:", cmd.String(), err)
		os.Exit(1)
	}

	return nil
}

// getDomainConf 获取这个域名的 目标配置
func getDomainConf(c conf.ConfList, domain string) *conf.Config {
	for _, c2 := range c.List {
		if c2.Domain == domain && c2.Status == 1 {
			return c2
		}
	}
	return nil
}
