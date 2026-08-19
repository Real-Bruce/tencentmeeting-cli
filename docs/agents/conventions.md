# 错误码与日志约定

## 错误码

错误集中在 `../../internal/exception`：

- `errors.go` — `TmeetError{Code, Message}`，`With(format, args...)` 返回新错误（不可变，原错误保留）
- `client_code.go` — 客户端错误码 2xxx/3xxx + 9000（panic）
- `server_code.go` — 服务器错误码

新增错误：在 `errors.go` 的 `var (...)` 块里加 `NewTmeetError(ClientCodeXxx, "message")`，提示信息要带上**用户可操作的恢复建议**（如 "please use 'tmeet auth login'"）。

## 日志

- 文件日志位于 `<config_dir>/logs/tmeet-YYYY-MM-DD.log`，10MB 轮转，保留 7 天
- 调用方式：`log.Infof(ctx, "...", ...)` / `log.Errorf` / `log.Warnf` / `log.Debugf`
- **必须传 `ctx`**：context 里有 `traceID`（`log.CtxTraceIDKey`），会被写入日志行，与请求头 `Tmeet-Trace` 关联
- 级别过滤在 `send` 里做，低于设定级别直接 return，不格式化
- panic 时通过 `crash.RecoverAndReport` 上报到 `/v1/cli/crash`，1 秒超时，失败静默降级

> 日志的写入模型（O_APPEND / 跨进程锁 / channel 非阻塞）属于不变量，见 [invariants.md](invariants.md#日志写入模型)。
