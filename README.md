# ssl_renewal
证书续签 

```shell
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/ssl_renewal
```

```shell
./run.sh -d  asleyu.com
```