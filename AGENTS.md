# AGENTS.md

> 入口文档。面向 AI Agent 与人类协作者，按需跳转 [docs/](docs/) 下的细分文档。
> 与 [README.md](README.md)（终端用户）、[CONTRIBUTING.md](CONTRIBUTING.md)（开源贡献者）互补。

## 项目

`tmeet` 是腾讯会议官方 CLI，基于 OAuth2 暴露会议 / 录制 / 报告 / 通讯录 / 会中控制能力，主要面向 AI Agent 调用，通过 npm 包 `@tencentcloud/tmeet` 分发。

## 技术栈

- Go（见 `go.mod`）+ `github.com/spf13/cobra`
- 凭证存储：`go-keyring`（macOS）/ 自研 AES-256-GCM（Linux/Windows）
- 分发：单二进制 + npm wrapper（`scripts/tmeet.js` 按 platform/arch 选 `dist/` 产物）
- 当前版本写入 `package.json`，构建时通过 `-ldflags "-X tmeet/cmd.Version=..."` 注入

## 构建与测试

```bash
go build -o tmeet .                       # 开发自测编译（不注入版本号）
go test ./...                             # 全量单元测试
go test -tags=integration ./...           # 集成测试（涉及真实 keychain/文件系统）
make build VERSION=v1.0.x                 # 全平台交叉编译到 dist/（发布用，必须传 VERSION）
make sync-version VERSION=v1.0.x          # 同步 package.json 版本号
```

> 国内开发：`go env -w GOPROXY=https://goproxy.cn,direct`（默认 `proxy.golang.org` 不可达）。详见 [docs/debugging.md](docs/debugging.md)。

## 按需查阅

| 任务 | 文档 |
|---|---|
| 改 `internal/` 核心（架构 / 请求链路 / 文件位置） | [docs/architecture.md](docs/architecture.md) |
| **修改安全 / token / 日志 / 分页 / 输出不变量** | [docs/invariants.md](docs/invariants.md) ← 改前必读 |
| 新增子命令 | [docs/new-command.md](docs/new-command.md) |
| 错误码与日志调用 | [docs/conventions.md](docs/conventions.md) |
| 配置文件 / 环境变量 | [docs/config.md](docs/config.md) |
| 写测试 | [docs/testing.md](docs/testing.md) |
| 调试 / VSCode / GOPROXY | [docs/debugging.md](docs/debugging.md) |
| 排错速查 | [docs/troubleshooting.md](docs/troubleshooting.md) |
| 发版 | [docs/release.md](docs/release.md) |

## 全局约束

- `internal/` 禁止被外部 import。
- 修改 `internal/` 核心前必读 [docs/invariants.md](docs/invariants.md)（破坏不变量会引发连锁问题）。
- 注释使用中文，遵循 Go doc 规范。
- 提交流程、分支策略、敏感信息处理见 [CONTRIBUTING.md](CONTRIBUTING.md)。
