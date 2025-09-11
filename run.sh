#!/bin/bash

set -e

# 默认值
DAYS=""
SERVER=""
RENEW=""
RENEW_ALL=""
FORCE=""
STAGING=""
CERT_DIR=""
GENERATE_CERT=true
INSTALL_CERT=true

# 获取当前脚本所在目录
get_script_dir() {
    SOURCE="${BASH_SOURCE[0]}"
    while [ -h "$SOURCE" ]; do
        DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
        SOURCE="$(readlink "$SOURCE")"
        [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
    done
    DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
    echo "$DIR"
}

# 设置证书文件路径
set_cert_paths() {
    local default_dir="/tmp/cert_zip"

    # 如果指定了证书目录，使用指定的目录
    if [[ -n "$CERT_DIR" ]]; then
        mkdir -p "$CERT_DIR"
        KEY_FILE="$CERT_DIR/${DOMAIN}.key"
        FULLCHAIN_FILE="$CERT_DIR/${DOMAIN}.pem"
    else
        # 使用脚本所在目录或默认目录
        local script_dir=$(get_script_dir)
        if [[ -d "$script_dir" && -w "$script_dir" ]]; then
            CERT_DIR="$script_dir/cert_zip"
        else
            CERT_DIR="$default_dir"
        fi
        mkdir -p "$CERT_DIR"
        KEY_FILE="$CERT_DIR/${DOMAIN}.key"
        FULLCHAIN_FILE="$CERT_DIR/${DOMAIN}.pem"
    fi
}

# 用法提示
usage() {
    echo "Usage: $0 -d domain [--days DAYS] [--server SERVER] [--renew] [--renew-all] [--force] [--staging] [--cert-dir CERT_DIR] [--no-generate] [--no-install]"
        echo ""
        echo "Options:"
        echo "  -d, --domain DOMAIN          域名 (必需)"
        echo "      --days DAYS              证书有效期天数"
        echo "      --server SERVER          ACME服务器"
        echo "      --renew                  续订证书"
        echo "      --renew-all              续订所有证书"
        echo "      --force                  强制操作"
        echo "      --staging                使用测试环境"
        echo "      --cert-dir CERT_DIR      证书文件目录"
        echo "      --no-generate            不执行证书生成，只安装"
        echo "      --no-install             不执行证书安装，只生成"
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
        --cert-dir)
            CERT_DIR="$2"
            shift 2
            ;;
        --no-generate)
            GENERATE_CERT=false
            shift
            ;;
        --no-install)
            INSTALL_CERT=false
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "未知参数: $1"
            usage
            ;;
    esac
done

# 检查必选参数
if [[ -z "$DOMAIN" ]]; then
    echo "Error: domain (-d) is required"
    usage
fi

# 执行证书生成
if [[ "$GENERATE_CERT" == true ]]; then
    # 构建 acme.sh 命令
    ACME_CMD="/root/.acme.sh/ssl_renewal acme -c /root/.acme.sh/config.json -d $DOMAIN $DAYS $SERVER $RENEW $RENEW_ALL $FORCE $STAGING"

    echo "[执行SSL证书生成命令]: $ACME_CMD"
    eval "$ACME_CMD"
    echo "[证书生成完成]: $DOMAIN"
else
    echo "[跳过证书生成]"
fi



# 设置证书文件路径
set_cert_paths
# 执行证书安装
if [[ "$INSTALL_CERT" == true ]]; then
    # 构建证书安装命令
    RELOAD_CMD="/root/.acme.sh/ssl_renewal reload -c /root/.acme.sh/config.json -d $DOMAIN --cert-dir \"$CERT_DIR\""

    INSTALL_CMD="/root/.acme.sh/acme.sh --installcert -d $DOMAIN \
        --key-file \"$KEY_FILE\" \
        --fullchain-file \"$FULLCHAIN_FILE\" \
        --reloadcmd \"$RELOAD_CMD\""

    echo "[执行SSL证书安装命令]: $INSTALL_CMD"
    eval "$INSTALL_CMD"
    echo "[证书安装完成]: $DOMAIN"
else
    echo "[跳过证书安装]"
fi

echo "[所有操作完成]"