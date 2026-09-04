# StreamAgent

StreamAgent 是一个用于流媒体解锁检测与路由处理的后台服务。

它会读取 `/opt/stream/config.yml`，按配置定时执行任务，并把路由或上报结果写回对应服务端。

## 安装

在支持的 Linux 系统上直接执行安装脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/install.sh | bash
```

也可以指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/HuTuTuOnO/StreamAgent/main/script/install.sh | bash -s -- v0.0.1
```

安装脚本需要使用 root 权限，并支持 Alpine/OpenRC 和 systemd。

安装后会放置这些文件：

```text
/opt/stream/stream-agent
/opt/stream/config.yml
/usr/bin/stream
```

系统会自动识别 `systemd` 或 `OpenRC`，并安装对应服务文件。

## 管理命令

```bash
stream
stream start
stream stop
stream restart
stream status
stream enable
stream disable
stream log
stream update
stream install
stream uninstall
stream version
```

说明：

- `stream`：显示交互菜单
- `stream start`：启动服务
- `stream stop`：停止服务
- `stream restart`：重启服务
- `stream status`：查看运行状态
- `stream enable`：设置开机自启
- `stream disable`：取消开机自启
- `stream log`：查看日志
- `stream update`：更新到最新版
- `stream install`：安装服务
- `stream uninstall`：卸载服务
- `stream version`：查看版本

## 配置

默认配置模板位于 `script/extras/config.yml`。

示例：

```yaml
debug: false
mode: client # client, server
api: example.com
token: 00000000-0000-0000-0000-000000000000
node: 1 # server 配置
stack: default # ipv4, ipv6, default client配置
scheduler: "0 30 */3 * * *"
exclude: []
```

字段说明：

- `debug`：开启调试日志
- `mode`：运行模式，`client` 或 `server`
- `api`：服务端 API 地址
- `token`：认证 token
- `node`：`server` 模式下的节点编号
- `stack`：`client` 模式下的地址选择方式
- `scheduler`：6 段 cron 表达式，包含秒字段
- `exclude`：`server` 模式下要排除的平台列表

## 模式

### client

`client` 模式会：

- 调用 `/api/unlock`
- 检测当前出口可解锁的平台
- 生成 `/etc/soga/routes.toml`

### server

`server` 模式会：

- 检测当前出口可解锁的平台
- 过滤 `exclude` 中的平台
- 调用 `/api/upload?token=...` 上报结果

## 构建

```bash
./build.sh
./build.sh amd64
./build.sh arm64
```

## 发布包

GitHub Release 会生成：

- `agent-v版本-linux-amd64.tar.gz`
- `agent-v版本-linux-arm64.tar.gz`

安装脚本会从 Release 下载对应架构的包，并安装服务和管理命令。

## Git 更新版本

项目使用 Git tag 发布版本。版本 tag 建议使用以下格式：

```text
v主版本号.次版本号.修订号
```

例如：`v0.0.3`。

完成代码修改后，执行：

```bash
# 查看修改内容
git status
git diff

# 提交代码
git add .
git commit -m "release: v0.0.3"

# 推送代码
git push origin main

# 创建并推送版本 tag
git tag v0.0.3
git push origin v0.0.3
```

推送 `v*` 格式的 tag 后，GitHub Actions 会自动执行：

1. 编译 Linux `amd64` 和 `arm64` 二进制文件
2. 生成 Release 压缩包
3. 生成 `SHA256SUMS`
4. 创建 GitHub Release 并上传文件

发布完成后，可以在 GitHub 仓库的 `Actions` 页面查看构建状态，在 `Releases` 页面查看发布文件：

```text
agent-v0.0.3-linux-amd64.tar.gz
agent-v0.0.3-linux-arm64.tar.gz
```

如果 tag 已经推送，但工作流没有自动执行，可以在 GitHub 的 `Actions` 页面手动运行 `Release` 工作流，并在 `tag` 参数中填写对应的 tag，例如：

```text
v0.0.3
```

注意：同一个 tag 不能重复创建。如果需要重新发布同一版本，应先删除远程 Release 和 tag，再重新创建；通常更建议直接递增修订号发布新版本。
