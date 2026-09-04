#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

version="v1.0.0"

[[ $EUID -ne 0 ]] && echo -e "${red}错误: ${plain} 必须使用root用户运行此脚本！\n" && exit 1

if [[ -f /etc/alpine-release ]]; then
    release="alpine"
elif [[ -d /run/systemd/system ]]; then
    release="systemd"
else
    echo -e "${red}未检测到系统版本或系统不支持，请联系脚本作者！${plain}\n" && exit 1
fi

confirm() {
    if [[ $# > 1 ]]; then
        echo && read -p "$1 [默认$2]: " temp
        [[ -z "$temp" ]] && temp=$2
    else
        read -p "$1 [y/n]: " temp
    fi
    [[ "$temp" == "y" || "$temp" == "Y" ]] && return 0 || return 1
}

before_show_menu() {
    echo && echo -n -e "${yellow}按回车返回主菜单: ${plain}" && read temp
    show_menu
}

install() {
    bash <(curl -Ls https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/install.sh)
    [[ $? == 0 ]] && start 0
}

update() {
    if [[ $# == 0 ]]; then
        echo && echo -n -e "输入指定版本(默认最新版): " && read stream_version
        [[ -z "$stream_version" ]] && stream_version="latest"
    else
        stream_version="${2:-latest}"
    fi
    bash <(curl -Ls https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/install.sh) "$stream_version"
    if [[ $? == 0 ]]; then
        restart 0
        echo -e "${green}更新完成，已自动重启 Stream，请使用 stream log 查看运行日志${plain}"
        exit
    fi

    [[ $# == 0 ]] && before_show_menu
}

uninstall() {
    confirm "确定要卸载 Stream 吗?" "n"
    if [[ $? != 0 ]]; then
        [[ $# == 0 ]] && show_menu
        return 0
    fi

    if [[ $release == "alpine" ]]; then
        rc-service stream stop >/dev/null 2>&1
        rc-update del stream >/dev/null 2>&1
        rm /etc/init.d/stream -f
    else
        systemctl stop stream >/dev/null 2>&1
        systemctl disable stream >/dev/null 2>&1
        rm /etc/systemd/system/stream.service -f
        systemctl daemon-reload
        systemctl reset-failed
    fi
    rm /opt/stream/ -rf

    echo ""
    echo -e "卸载成功，如果你想删除此脚本，则退出脚本后运行 ${green}rm /usr/bin/stream -f${plain} 进行删除"
    echo ""

    [[ $# == 0 ]] && before_show_menu
}

start() {
    check_status
    if [[ $? == 0 ]]; then
        echo ""
        echo -e "${green}Stream已运行，无需再次启动，如需重启请选择重启${plain}"
    else
        if [[ $release == "alpine" ]]; then
            rc-service stream start >/dev/null 2>&1
        else
            systemctl reset-failed stream >/dev/null 2>&1
            systemctl start stream >/dev/null 2>&1
        fi
        sleep 2
        check_status
        if [[ $? == 0 ]]; then
            echo -e "${green}Stream 启动成功，请使用 stream log 查看运行日志${plain}"
        else
            echo -e "${red}Stream可能启动失败，请稍后使用 stream log 查看日志信息${plain}"
        fi
    fi

    [[ $# == 0 ]] && before_show_menu
}

stop() {
    if [[ $release == "alpine" ]]; then
        rc-service stream stop >/dev/null 2>&1
    else
        systemctl stop stream >/dev/null 2>&1
    fi
    sleep 2
    check_status
    if [[ $? == 1 ]]; then
        echo -e "${green}Stream 停止成功${plain}"
    else
        echo -e "${red}Stream停止失败，可能是因为停止时间超过了两秒，请稍后查看日志信息${plain}"
    fi

    [[ $# == 0 ]] && before_show_menu
}

restart() {
    if [[ $release == "alpine" ]]; then
        rc-service stream restart >/dev/null 2>&1
    else
        systemctl reset-failed stream >/dev/null 2>&1
        systemctl restart stream >/dev/null 2>&1
    fi
    sleep 2
    check_status
    if [[ $? == 0 ]]; then
        echo -e "${green}Stream 重启成功，请使用 stream log 查看运行日志${plain}"
    else
        echo -e "${red}Stream可能启动失败，请稍后使用 stream log 查看日志信息${plain}"
    fi
    [[ $# == 0 ]] && before_show_menu
}

enable() {
    if [[ $release == "alpine" ]]; then
        rc-update add stream default
    else
        systemctl enable stream
    fi
    if [[ $? == 0 ]]; then
        echo -e "${green}Stream 设置开机自启成功${plain}"
    else
        echo -e "${red}Stream 设置开机自启失败${plain}"
    fi

    [[ $# == 0 ]] && before_show_menu
}

disable() {
    if [[ $release == "alpine" ]]; then
        rc-update del stream default
    else
        systemctl disable stream
    fi
    if [[ $? == 0 ]]; then
        echo -e "${green}Stream 取消开机自启成功${plain}"
    else
        echo -e "${red}Stream 取消开机自启失败${plain}"
    fi

    [[ $# == 0 ]] && before_show_menu
}

show_log() {
    if [[ $release == "alpine" ]]; then
        tail -f /var/log/stream.log
    else
        journalctl -u stream.service -e --no-pager -f
    fi
    [[ $# == 0 ]] && before_show_menu
}

check_status() {
    if [[ $release == "alpine" ]]; then
        [[ ! -f /etc/init.d/stream ]] && return 2
        rc-service stream status 2>&1 | grep -q "started" && return 0
        return 1
    fi
    [[ ! -f /etc/systemd/system/stream.service ]] && return 2
    systemctl is-active --quiet stream.service && return 0
    return 1
}

check_enabled() {
    if [[ $release == "alpine" ]]; then
        rc-status | grep -q 'stream'
        [[ $? == 0 ]] && return 0 || return 1
    fi
    temp=$(systemctl is-enabled stream 2>/dev/null)
    [[ "$temp" == "enabled" ]] && return 0 || return 1
}

check_uninstall() {
    check_status
    if [[ $? != 2 ]]; then
        echo ""
        echo -e "${red}Stream已安装，请不要重复安装${plain}"
        [[ $# == 0 ]] && before_show_menu
        return 1
    fi
    return 0
}

check_install() {
    check_status
    if [[ $? == 2 ]]; then
        echo ""
        echo -e "${red}请先安装 Stream${plain}"
        [[ $# == 0 ]] && before_show_menu
        return 1
    fi
    return 0
}

show_status() {
    check_status
    case $? in
        0)
            echo -e "Stream状态: ${green}已运行${plain}"
            show_enable_status
            ;;
        1)
            echo -e "Stream状态: ${yellow}未运行${plain}"
            show_enable_status
            ;;
        2)
            echo -e "Stream状态: ${red}未安装${plain}"
            ;;
    esac
}

show_enable_status() {
    check_enabled
    if [[ $? == 0 ]]; then
        echo -e "是否开机自启: ${green}是${plain}"
    else
        echo -e "是否开机自启: ${red}否${plain}"
    fi
}

show_stream_version() {
    echo -n "Stream 版本："
    /opt/stream/stream-agent -version
    echo ""
    [[ $# == 0 ]] && before_show_menu
}

show_usage() {
    echo "Stream 管理脚本使用方法: "
    echo "------------------------------------------"
    echo "stream                    - 显示管理菜单"
    echo "stream start              - 启动 Stream"
    echo "stream stop               - 停止 Stream"
    echo "stream restart            - 重启 Stream"
    echo "stream status             - 查看 Stream 状态"
    echo "stream enable             - 设置 Stream 开机自启"
    echo "stream disable            - 取消 Stream 开机自启"
    echo "stream log                - 查看 Stream 日志"
    echo "stream update             - 更新 Stream 最新版"
    echo "stream update x.x.x       - 更新 Stream 指定版本"
    echo "stream install            - 安装 Stream"
    echo "stream uninstall          - 卸载 Stream"
    echo "stream version            - 查看 Stream 版本"
    echo "------------------------------------------"
}

show_menu() {
    echo -e "
  ${green}Stream 后端管理脚本${plain}

  ${green}0.${plain} 退出脚本
————————————————
  ${green}1.${plain} 安装 Stream
  ${green}2.${plain} 更新 Stream
  ${green}3.${plain} 卸载 Stream
————————————————
  ${green}4.${plain} 启动 Stream
  ${green}5.${plain} 停止 Stream
  ${green}6.${plain} 重启 Stream
  ${green}7.${plain} 查看 Stream 日志
————————————————
  ${green}8.${plain} 设置 Stream 自启
  ${green}9.${plain} 取消 Stream 自启
————————————————
 ${green}10.${plain} 查看 Stream 状态
 ${green}11.${plain} 查看 Stream 版本
 "
    show_status
    echo && read -p "请输入选择 [0-11]: " num

    case "$num" in
        0) exit 0 ;;
        1) check_uninstall && install ;;
        2) check_install && update ;;
        3) check_install && uninstall ;;
        4) check_install && start ;;
        5) check_install && stop ;;
        6) check_install && restart ;;
        7) check_install && show_log ;;
        8) check_install && enable ;;
        9) check_install && disable ;;
        10) show_status ;;
        11) check_install && show_stream_version ;;
        *) echo -e "${red}请输入正确的数字 [0-11]${plain}" ;;
    esac
}

if [[ $# > 0 ]]; then
    case "$1" in
        start) check_install 0 && start 0 ;;
        stop) check_install 0 && stop 0 ;;
        restart) check_install 0 && restart 0 ;;
        status) show_status ;;
        enable) check_install 0 && enable 0 ;;
        disable) check_install 0 && disable 0 ;;
        log) check_install 0 && show_log 0 ;;
        update)
            check_install 0 && update 0 "${2:-latest}"
            ;;
        install) check_uninstall 0 && install 0 ;;
        uninstall) check_install 0 && uninstall 0 ;;
        version) check_install 0 && show_stream_version 0 ;;
        *) show_usage ;;
    esac
else
    show_menu
fi
