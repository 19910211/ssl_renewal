# ssl_renewal

证书续签

```shell
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/ssl_renewal
```

config.json 配置文件
ssl_renewal 二进制执行文件
run.sh 运行脚本

```shell
./run.sh  --cert-dir /usr/local/nginx/conf/cert --domain you.com 
```