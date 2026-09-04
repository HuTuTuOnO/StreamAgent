#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)
install_dir="/opt/stream"
config_file="${install_dir}/config.yml"
service_name="stream"

[[ $EUID -ne 0 ]] && echo -e "${red}错误：${plain} 必须使用root用户运行此脚本！\n" && exit 1

if [[ -f /etc/alpine-release ]]; then
    release="alpine"
elif [[ -d /run/systemd/system ]]; then
    release="systemd"
else
    echo -e "${red}未检测到系统版本或系统不支持，请联系脚本作者！${plain}\n" && exit 1
fi

arch=$(arch)
if [[ $arch == "x86_64" || $arch == "x64" || $arch == "amd64" ]]; then
    arch="amd64"
elif [[ $arch == "aarch64" || $arch == "arm64" ]]; then
    arch="arm64"
else
    echo -e "${red}检测架构失败，当前架构不受支持: ${arch}${plain}"
    exit 1
fi

echo "架构: ${arch}"

if [[ $(getconf LONG_BIT) != '64' ]]; then
    echo "本软件不支持 32 位系统，请使用 64 位系统"
    exit 2
fi

is_cmd_exist() {
    local cmd="$1"
    [[ -n "$cmd" ]] && command -v "$cmd" >/dev/null 2>&1
}

install_base() {
    if [[ $release == "alpine" ]]; then
        apk add --no-cache wget curl tar tzdata openrc
    elif is_cmd_exist apt-get; then
        apt-get update
        apt-get install -y wget curl tar tzdata
    fi
}

check_status() {
    if [[ $release == "alpine" ]]; then
        [[ ! -f /etc/init.d/${service_name} ]] && return 2
        rc-service "$service_name" status 2>&1 | grep -q "started" && return 0
        return 1
    fi
    [[ ! -f /etc/systemd/system/${service_name}.service ]] && return 2
    systemctl is-active --quiet "${service_name}.service" && return 0
    return 1
}

install_stream() {
    local version="${1:-latest}"
    local temp_dir
    local archive
    local url
    local binary
    local new_install=0

    mkdir -p "$install_dir"
    if [[ ! -f $config_file ]]; then
        if [[ -f "${cur_dir}/script/extras/config.yml" ]]; then
            cp "${cur_dir}/script/extras/config.yml" "$config_file"
        else
            curl -fsSL -o "$config_file" "https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/extras/config.yml"
        fi
        chmod 0644 "$config_file"
        new_install=1
    fi

    temp_dir=$(mktemp -d)
    archive="${temp_dir}/stream-agent.tar.gz"
    if [[ $version == "latest" ]]; then
        url="https://github.com/HuTuTuOnO/StreamAgent/releases/latest/download/stream-agent-linux-${arch}.tar.gz"
    else
        version=${version#v}
        url="https://github.com/HuTuTuOnO/StreamAgent/releases/download/v${version}/stream-agent-linux-${arch}.tar.gz"
    fi

    echo -e "开始安装 StreamAgent ${version}"
    curl -fL --retry 3 -o "$archive" "$url"
    if [[ $? -ne 0 ]]; then
        echo -e "${red}下载 StreamAgent 失败，请确保你的服务器能够下载 Github 的文件${plain}"
        rm -rf "$temp_dir"
        exit 1
    fi

    mkdir -p "${temp_dir}/unpack"
    tar zxvf "$archive" -C "${temp_dir}/unpack" >/dev/null
    binary=$(find "${temp_dir}/unpack" -type f -name stream-agent -print -quit)
    [[ -z $binary ]] && echo -e "${red}安装包中未找到 StreamAgent 文件${plain}" && rm -rf "$temp_dir" && exit 1
    install -m 0755 "$binary" "${install_dir}/stream-agent"
    rm -rf "$temp_dir"

    if [[ $release == "alpine" ]]; then
        curl -fsSL -o /etc/init.d/stream "https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/services/stream"
        chmod +x /etc/init.d/stream
        rc-update add stream default
    else
        curl -fsSL -o /etc/systemd/system/stream.service "https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/services/stream.service"
        chmod 0644 /etc/systemd/system/stream.service
        systemctl daemon-reload
        systemctl enable stream.service
    fi

    if [[ $new_install -eq 0 ]]; then
        restart
    else
        echo -e "${green}StreamAgent 安装完成，已设置开机自启${plain}"
        echo -e "请先配置 ${config_file}，然后运行 stream start"
    fi
}

install_command() {
    curl -fsSL -o /usr/bin/stream "https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/stream.sh"
    if [[ $? -ne 0 ]]; then
        echo -e "${red}下载 StreamAgent 管理脚本失败，请确保你的服务器能够下载 Github 的文件${plain}"
        exit 1
    fi
    chmod +x /usr/bin/stream
}

install() {
    install_stream "$1"
}

update() {
    if [[ $# -eq 0 ]]; then
        echo && echo -n -e "输入指定版本(默认最新版): " && read version
        [[ -z $version ]] && version="latest"
    else
        version="$1"
    fi
    install_stream "$version"
}

start() {
    if [[ $release == "alpine" ]]; then
        rc-service stream start
    else
        systemctl start stream.service
    fi
}

restart() {
    if [[ $release == "alpine" ]]; then
        rc-service stream restart
    else
        systemctl restart stream.service
    fi
}

main() {
    echo -e "${green}开始安装${plain}"
    install_base
    install_stream "$1"
    install_command
}

main "$@"
