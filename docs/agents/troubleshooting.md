# 排错速查

| 现象 | 原因 | 解决 |
|---|---|---|
| VSCode F5 报 `user has been login` | 已登录且 refresh_token 未过期，`auth login` 拒绝覆盖 | 调试别的命令，或先 `tmeet auth logout` |
| VSCode 调试 `meeting list` 报 token 过期 | 本机凭证失效 | 命令行跑一次 `tmeet auth login` 重新授权 |
| `tmeet auth login` 卡住 5 分钟 | OAuth 轮询超时（`authorizationTimeout`） | 浏览器没扫码 / 网络异常，重试 |
| 改了 `cmd.Version` 不生效 | 没用 ldflags 注入 | `go build -ldflags "-X tmeet/cmd.Version=v1.0.x"` 或 `make build VERSION=...` |
| 新命令 `--compact` 不生效 | 没注册 ApiCmd annotation / 远端没有该 schema | 确认 `WithApiCmd` 中间件已加 + ApiCmd 常量已声明 |
| 新增分页命令旧 `--page` 不兼容 | 没用 `ChoosePageOrToken` | 改用 `../../internal/cmdutil/pagination.go` 的统一函数 |
| 日志跨进程交错 | 改用 `bufio` 缓冲了 | 还原为直接 `O_APPEND` 写，参考 `internal/log/logging.go:342` |
| 凭证文件被路径穿越攻击 | OpenId 没过白名单 | 调用 `validateOpenId`，只允许 `[a-zA-Z0-9_]` |

> GOPROXY / `go install` 超时等开发环境问题见 [debugging.md](debugging.md)。
