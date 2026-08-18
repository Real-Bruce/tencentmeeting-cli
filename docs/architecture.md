# 架构总览

> 入口：[AGENTS.md](../AGENTS.md) | 修改核心前先读 [invariants.md](invariants.md)

## 目录结构

```
main.go                        入口，仅调用 cmd.Execute()
├── cmd/                       cobra 命令层（薄壳，仅参数解析 + 拼请求）
│   ├── root.go                根命令、全局 flag、panic 兜底、子命令注册
│   ├── auth/ meeting/ record/ contact/ report/ control/ tshoot/
│   └── 每个子命令文件命名：<verb>.go（如 create.go / list.go）
│
└── internal/                  业务核心（禁止被外部 import）
    ├── tmeet.go               Tmeet 客户端结构，持有 RestClient + CGIClient + UserConfig + SystemInfo
    ├── config/                用户凭证（加密）+ agent.json（明文）
    ├── auth/                  OAuth2 设备码 + token 刷新
    ├── proxy/
    │   ├── rest-proxy/        REST 请求代理：token 刷新 → 重试 → 统一错误码
    │   └── cgi-proxy/         CGI 请求代理：泛型 Response[T]，无 token 刷新
    ├── core/
    │   ├── endpoints.go       三类后端：Open / CGI / Auth
    │   ├── keychain/          跨平台加密凭证（master key + AES-256-GCM）
    │   ├── thttp/             HTTP 客户端 + 请求/响应封装
    │   ├── browser/           跨平台开浏览器
    │   ├── filelock/          跨进程文件锁（token 刷新 / 日志轮转用）
    │   ├── filecache/         API schema 本地缓存
    │   └── filecheck/         文件校验
    ├── cmdutil/               命令层公共工具
    │   ├── annotations.go     cobra annotation：apiCmd / skipPreCheck / skipPreCheckFlag
    │   ├── api_cmd_resolver.go FlagSwitch 路由（同一命令按 flag 选不同 ApiCmd）
    │   ├── api_schema.go      远程 compact schema 拉取 + 缓存
    │   ├── api_schema_cache.go 本地磁盘缓存
    │   ├── pagination.go      统一分页：page-token（新）+ page/pos（旧，deprecated）
    │   ├── meeting_help.go    FormatMeetingCode：归一化 `#腾讯会议：295-150-176` → `295150176`
    │   ├── middleware/        洋葱模型中间件：WithApiCmd / WithCompact
    │   ├── apicmdctx/         context 透传 ApiCmd
    │   └── hints/             输出 hints
    ├── output/                JSON / json-pretty 输出 + compact 裁剪 + 字段转换
    ├── exception/             统一错误码（client_code 2xxx/3xxx/9000 + server_code）
    ├── log/                   文件日志：日切 + 10MB 轮转 + 7 天清理 + 跨进程锁
    ├── crash/                 panic 兜底：捕获 → 上报服务器 → exit code 2
    └── utils/                 converter / filter / time / string / validate / retry
```

## 请求链路（理解这个就理解了 80% 代码）

以 `tmeet meeting list` 为例：

```
用户输入
  → cobra 解析 flags
  → PersistentPreRunE (cmd/root.go:78)
      ├─ commandOperationLog  → 写 INFO 日志（命令 + flags）
      ├─ injectCmdPathIntoTmeet → 设置 tmeet.CmdPath（用于请求头）
      └─ preCheck → 校验 ~/.tmeet/config.json 存在且已登录
  → middleware.Chain（洋葱模型，cmd/meeting/list.go:34）
      ├─ [pre] WithApiCmd → 在 cmd 上注 apiCmd=meeting_list
      ├─ [pre] WithCompact → 若 --compact，远程拉 compact 字段表塞进 ctx
      │
      └─ opts.Run（cmd/meeting/list.go:51）
          ├─ 组装 thttp.Request{ApiURI, QueryParams, PathParams}
          ├─ restProxy.RequestProxy(ctx, GET, tmeet, req) ← internal/proxy/rest-proxy/proxy.go:46
          │   ├─ auth.RefreshToken  ← internal/auth/auth.go:72
          │   │   ├─ 检查 access_token 剩余 > 60s → 直接返回
          │   │   ├─ 检查 refresh_token 过期 → 清凭证 + 报错
          │   │   └─ filelock.WithLock(token.lock) → 拿锁 → 重读配置 → 刷新 → SaveUserConfig
          │   ├─ retry.Do（带 RetryIf：token 过期 / 不可重试错误立即停）
          │   │   └─ requestProxy → doRequest → RestClient.Get/Post/...
          │   ├─ 非 200 → 解析 ProxyError → TokenExpired 时清本地凭证
          │   └─ 返回 ProxyRsp{Data, TraceId, Message}
          │
          ├─ （可选）utils.FilterMeetingsByTimeRange 客户端过滤
          ├─ utils.ConvertFields（时间戳/状态码/枚举转中文）
          └─ output.FormatPrint → marshalNoEscape → stdout
```

## 关键文件速查表

| 改什么 | 看哪里 |
|---|---|
| 全局 flag（`--format`/`--compact`/`-V`） | `cmd/root.go:90-96` |
| 子命令注册 | `cmd/<domain>/base.go` |
| 后端域名 | `internal/core/endpoints.go` |
| OAuth 流程 | `internal/auth/auth.go` + `auth_code.go` + `auth_token.go` |
| token 刷新逻辑 | `internal/auth/auth.go:72` `RefreshToken` |
| 凭证加解密 | `internal/core/keychain/`（平台分支：`keychain_darwin.go` / `keychain_windows.go` / `keychain_other.go`） |
| HTTP 客户端 | `internal/core/thttp/client.go` + `request.go` + `response.go` |
| 统一请求代理 | `internal/proxy/rest-proxy/proxy.go`（REST）/ `cgi-proxy/proxy.go`（CGI） |
| 错误码定义 | `internal/exception/errors.go` + `client_code.go` + `server_code.go` |
| 分页 | `internal/cmdutil/pagination.go` |
| ApiCmd 路由 | `internal/cmdutil/api_cmd_resolver.go` + `api_schema.go` |
| 中间件 | `internal/cmdutil/middleware/middle_ware.go` + `compact.go` + `api_cmd.go` |
| 输出格式化 | `internal/output/print.go` + `options.go` |
| 字段转换（时间戳→日期，状态码→中文） | `internal/utils/converter.go` + `enumerate/` |
| 日志系统 | `internal/log/logging.go` + `trace_id.go` |
| panic 兜底 | `internal/crash/recover.go` + `reporter.go` |
| 跨进程锁 | `internal/core/filelock/` |
| API schema 缓存 | `internal/cmdutil/api_schema_cache.go` + `internal/core/filecache/` |
| npm 入口 wrapper | `scripts/tmeet.js`（按平台选 `dist/` 下二进制） |
| 交叉编译 | `Makefile`（5 个 GOOS/GOARCH 目标） |
