# StreamAgent

StreamAgent 是一个用于流媒体解锁检测与路由处理的 Linux 后台服务。

它会读取 `-c` 指定的配置文件，按配置定时执行任务，并把日志输出到标准输出/标准错误，由 `systemd` 和 `journalctl` 接管查看。

## 运行方式

服务启动后会立即执行一次，之后按 `scheduler` 的 cron 表达式周期运行。

日志查看：

```bash
journalctl -xf -u agent
```

## 配置文件

启动时必须通过 `-c` 指定配置文件路径。

示例配置：

```yaml
debug: false
mode: client # client, server
api: https://stream.airtools.cc
token: 61ebaa11-00da-40cc-a01b-2f799ece5af2
node: 1 # server 配置
stack: default # ipv4, ipv6, default client配置
scheduler: "0 30 */3 * * *"
exclude: []
```

### 字段说明

- `debug`：是否开启调试日志。
- `mode`：运行模式，`client` 或 `server`。
- `api`：服务端 API 地址。
- `token`：认证 token。
- `node`：`server` 模式下使用的节点编号。
- `stack`：`client` 模式下节点探测使用的地址选择方式。
  - `default`：直接使用 API 返回的 `host`
  - `ipv4`：先解析出 IPv4，再用解析结果探测
  - `ipv6`：先解析出 IPv6，再用解析结果探测
- `scheduler`：定时任务的 cron 表达式。
- `exclude`：`server` 模式下需要排除的流媒体平台列表。

## 模式说明

### client

`client` 模式对应原来的客户端脚本能力，主要负责：

- 拉取 `/api/unlock` 返回的节点和平台映射
- 按 `stack` 决定节点探测时使用原始域名、IPv4 还是 IPv6
- 调用 UnlockTests 识别当前出口哪些平台可解锁
- 只为被锁平台生成 soga 路由配置

### server

`server` 模式对应原来的上报脚本能力，主要负责：

- 调用 UnlockTests 做流媒体解锁检测
- 过滤 `exclude` 中指定的平台
- 将结果通过 `/api/upload?token=...` 上报到服务端

## 调度说明

`scheduler` 使用 6 段 cron 表达式，包含秒字段。

例如：

- `0 */30 * * * *`：每 30 分钟执行一次
- `0 30 */3 * * *`：每 3 小时的第 30 分钟执行一次

服务内部会固定执行以下行为：

- 启动后立即运行一次
- 单次任务超时固定为 10 分钟
- 上一轮任务未结束时跳过下一轮

## UnlockTests

流媒体解锁检测使用 `UnlockTests` 的 Go 结构化接口，而不是 shell 调用。

在本项目里，它只负责“当前出口哪些平台可解锁”的判定，不负责节点解析，也不负责路由写入。

## 目录约定

```text
/opt/stream/
  config.yml
```

## 从旧脚本迁移

如果你之前使用的是 `client.sh` 和 `service.sh`：

- `client.sh` 的逻辑会迁移到 `client` 模式
- `service.sh` 的逻辑会迁移到 `server` 模式
- 不再需要 shell 作为入口

## 备注

- 该服务面向 Linux 环境。
- 日志不单独落文件，统一通过 `journalctl` 查看。
- 配置尽量保持精简，保护性参数由服务内部固定处理。

## 编译

```bash
./build.sh
```

也可以单独打某个架构：

```bash
./build.sh amd64
./build.sh arm64
```

## 启动

```bash
./agent -c /opt/stream/config.yml
```
