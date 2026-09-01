# CLAUDE.md

> Claude Code 项目记忆入口。本文件为**自包含快速参考**，加载即可上手；深度内容以 [`AGENTS.md`](AGENTS.md) 为单一事实来源，按需跳转 [`docs/agents/`](docs/agents/)。与 [README.md](README.md)（终端用户）互补。

## 项目概览

`tmeet` 是腾讯会议官方 CLI，基于 Go + `spf13/cobra`，模块名 `tmeet`（go 1.25.5），通过 npm 包 `@tencentcloud/tmeet` 分发（当前 v1.0.14），主要面向 AI Agent 调用，亦可作为 MCP server 运行。

- **能力域**：`auth` / `meeting` / `record` / `contact` / `report` / `control` / `tshoot` / `mcp`。
- **后端**（见 `internal/core/endpoints.go`）：Open API `api.meeting.qq.com`、CGI `work.medialab.qq.com`、Auth `meeting.tencent.com`。`tmeet` 本身**不托管 REST API**，只作出站客户端 + MCP server。
- **入口**：`main.go` → `cmd.Execute()`（`cmd/root.go`）；`cmd/` 是薄 cobra 壳，`internal/` 是业务核心（**禁止外部 import**）。

## 最近重点：多账号（`meeting-multiple` 分支）

新增多账号登录与按 OpenId 隔离的凭证存储，`--name` 改为必填。给 Claude 的心智模型：

- **设计**：凭证按 OpenId 隔离。明文 `config.json` 存 `{active_name, accounts[], active_open_id(legacy 兼容)}` 别名注册表 + active 指针；加密 secrets 按 `<dataDir>/<openId>/` 分目录，每账号独立 `master.key` + AES-256-GCM `data.enc`（AAD 绑定 openId，防跨账号 .enc 互换）。
- **`--name`**（commit `b248925`）：`auth login` / `auth status` 必填；`auth logout` 必填（除非 `--all`）；`auth use <name>` 为位置参数；MCP `auth_login_url` / `auth_login_poll` / `auth_use` 的 `name` 亦必填。别名按 openId 唯一——重登同 openId 更新别名/sdk，别名撞车不同 openId 报 `AccountNameConflictError`(1009)，账号不存在报 `AccountNotFoundError`(1008)。
- **存储路径**：`TMEET_CLI_CONFIG_DIR`（默认 `~/.tmeet/`）存明文 `config.json` / `agent.json` / `logs/` / `token.lock`；`TMEET_CLI_DATA_DIR` 存加密 keychain（`<openId>/master.key` + `data.enc`）。
- **迁移**：`NewTmeet`（`internal/tmeet.go`）启动时调 `config.MigrateLegacyAccounts()`，旧单账号共享 master.key 布局 → 新按账号布局，幂等、非破坏、失败仅记 stderr。active 解析回退 legacy `active_open_id`，故旧单账号安装无需重登。
- **刷新**：access token 提前 60s 刷新（`accessTokenRefreshLeeway`），file lock 串行；refresh token 过期则清本地凭证。MCP 长驻进程每次非 auth 工具调用前经 `requireLogin` 用 active 账号同步 `tmeet.UserConfig`，`auth_use` 后立即生效。

## 关键不变量（改前必读 [`docs/agents/invariants.md`](docs/agents/invariants.md)）

只列最易踩坑的几条，细节见该文档：

- `internal/` 禁止外部 import；改核心前必读 invariants.md。
- **凭证安全**：AES-256-GCM + AAD(openId)；master.key 一次性生成、不轮换；openId 白名单 `^[a-zA-Z0-9_]+$` 且禁保留名 `master.key`（防路径穿越/覆盖）；写操作原子化（tmp→Sync→Rename），读前 `filecheck.ValidateBeforeRead`（拒符号链接/异主文件）。
- **输出契约**：JSON 不 HTML 转义（`SetEscapeHTML(false)`）；`--compact` 字段表**远程拉取**（绝不硬编码，缓存在 `<config-dir>/cache/schema/`）；`total_count=0` 剥离。
- **分页**：统一 `--page-token` + `--page-size`（v1.0.5+，旧 `--page/--pos/--size` 已废弃），用 `cmdutil.ChoosePageOrToken` / `ChoosePosOrToken` + `ClampingPageSize`。
- **日志**：`log.Infof(ctx, ...)` 的 **ctx 必传**，携带 traceID 与 `Tmeet-Trace` 请求头关联。
- **panic**：`crash.RecoverAndReport` 兜底，上报 `/v1/cli/crash`，exit 2；其内不得再 `os.Exit`。

## 命令实现模式

`cmd/<domain>/<verb>.go` 是薄壳：`<Domain>Options` 结构体 → `newXxxCmd` 返回 `cobra.Command`，其 `RunE` 经 `middleWare.Chain(opts.Run, middleWare.WithApiCmd(...), middleWare.WithCompact(...))` → 调 `internal/<domain>` 纯函数 → `output.FormatPrint`。ApiCmd 字符串（如 `meeting_create`）经 `Tmeet-Cli-Name` 头上送、并驱动 compact 字段。**一条命令映射多接口**用 `cmdutil.FlagSwitch`（见 `cmd/meeting/get.go`：按 `meeting_id` 或 `meeting_code` 路由）。返回值统一 `result.Result{Data, TraceId, Message}`，CLI 与 MCP 共用。

## 构建与测试

```bash
go build -o tmeet .                  # 开发自测编译（不注入版本号）
go test ./...                        # 全量单元测试
go test -tags=integration ./...      # 集成测试（真实 keychain/文件系统）
make build VERSION=v1.0.x            # 交叉编译 5 目标到 dist/（发布用，必须传 VERSION）
make sync-version VERSION=v1.0.x     # 同步 package.json 版本号
go env -w GOPROXY=https://goproxy.cn,direct   # 国内开发
```

版本经 ldflags 注入：`-X tmeet/cmd.Version=... -X tmeet/cmd.BuildTime=...`。集成测试用 `_integration_test.go` + `//go:build integration`，mock keychain 经 `config.SetKeychain(mock)` 注入。

## MCP server 速查

`tmeet mcp`（`cmd/mcp.go` + `internal/mcpserver/`），本身 skipPreCheck：

- **传输**：`stdio`（默认）/ `streamable-http`（MCP 2025-03-26，默认 `:8347`，端点 `/mcp`）/ `sse`。HTTP 模式支持 `--api-key`（Bearer）+ `--tls-cert/--tls-key`。首个 `initialize` 返回 `Mcp-Session-Id` 头需回传。
- **工具**：33 个（auth 4 / meeting 12 / record 9 / report 2 / contact 3 / control 2 / tshoot 1），复用 `internal/<domain>` 同一套逻辑。`tshoot log` 仅 CLI（需文件系统访问）。详见 [`docs/MCP.md`](docs/MCP.md)。
- **多实例隔离**（每用户/部门）：port(`--addr`) + `TMEET_CLI_CONFIG_DIR` + `TMEET_CLI_DATA_DIR` + 随机 api-key + systemd `%i`。

## 约定与工作流

- 注释用**中文**，遵循 Go doc 规范（`//` 行注释，导出项必注）。
- 错误统一用 `internal/exception`（`TmeetError{Code, Message}`，不可变，`With(...)` 返回新实例）；新错误必须带**可操作恢复建议**。
- 每次改代码后**必须提交 git**；提交流程/分支/敏感信息见 [`CONTRIBUTING.md`](CONTRIBUTING.md)。
- **commit message 用英文**：subject + body 均用英文描述更改（遵循 Conventional Commits，如 `feat:`/`fix:`/`docs:`/`chore:`），保留 `Co-Authored-By` trailer；代码注释仍用中文。
- 改 `internal/` 核心前必读 [`docs/agents/invariants.md`](docs/agents/invariants.md)；新增子命令按 [`docs/agents/new-command.md`](docs/agents/new-command.md) 9 步清单。

## 深度文档索引

| 任务 | 文档 |
|---|---|
| 架构 / 请求链路 / 文件位置 | [docs/agents/architecture.md](docs/agents/architecture.md) |
| 安全 / token / 日志 / 分页 / 输出不变量 | [docs/agents/invariants.md](docs/agents/invariants.md) ← 改前必读 |
| 新增子命令 | [docs/agents/new-command.md](docs/agents/new-command.md) |
| 错误码与日志调用 | [docs/agents/conventions.md](docs/agents/conventions.md) |
| 配置文件 / 环境变量 | [docs/agents/config.md](docs/agents/config.md) |
| 写测试 | [docs/agents/testing.md](docs/agents/testing.md) |
| 调试 / VSCode / GOPROXY | [docs/agents/debugging.md](docs/agents/debugging.md) |
| 排错速查 | [docs/agents/troubleshooting.md](docs/agents/troubleshooting.md) |
| 发版 | [docs/agents/release.md](docs/agents/release.md) |
