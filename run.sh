#!/bin/bash

set -e

# 默认值
DAYS=""
SERVER=""
RENEW=""
RENEW_ALL=""
FORCE=""
STAGING=""

# 用法提示
usage() {
    echo "Usage: $0 -d domain [--days DAYS] [--server SERVER] [--renew] [--renew-all] [--force]"
    exit 1
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        -d|--domain)
            DOMAIN="$2"
            shift 2
            ;;
        --days)
            DAYS="--days $2"
            shift 2
            ;;
        --server)
            SERVER="--server $2"
            shift 2
            ;;
        --renew)
            RENEW="--renew"
            shift
            ;;
        --renew-all)
            RENEW_ALL="--renew-all"
            shift
            ;;
        --force)
            FORCE="--force"
            shift
            ;;
        --staging)
            STAGING="--staging"
            shift
            ;;
        *)
            usage
            ;;
    esac
done

# 检查必选参数
if [[ -z "$DOMAIN" ]]; then
    echo "Error: domain (-d) is required"
    usage
fi

# 构建 acme.sh 命令
ACME_CMD="/root/.acme.sh/ssl_renewal acme -c /root/.acme.sh/config.json -d $DOMAIN $DAYS $SERVER $RENEW $RENEW_ALL $FORCE $STAGING"

echo "执行命令: $ACME_CMD"
eval "$ACME_CMD"

# 构建证书安装命令
KEY_FILE="/tmp/cert_distributions/${DOMAIN}.key"
FULLCHAIN_FILE="/tmp/cert_distributions/${DOMAIN}.pem"
RELOAD_CMD="/root/.acme.sh/ssl_renewal reload -c /root/.acme.sh/config.json -d $DOMAIN"

INSTALL_CMD="/root/.acme.sh/acme.sh --installcert -d $DOMAIN \
    --key-file $KEY_FILE \
    --fullchain-file $FULLCHAIN_FILE \
    --reloadcmd \"$RELOAD_CMD\""

echo "执行安装命令: $INSTALL_CMD"
eval "$INSTALL_CMD"

echo "证书安装完成: $DOMAIN"
