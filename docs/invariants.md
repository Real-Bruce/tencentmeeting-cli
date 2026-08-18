# 关键不变量（修改前必读）

> 这些是项目刻意维持的设计约束，破坏会引发连锁问题。修改 `internal/` 核心代码前请完整阅读本文件。

## 凭证安全

- **UserConfig 整体 AES-256-GCM 加密后存入 keychain**，明文不落盘。`config.json` 只存 `active_open_id`（用于定位 keychain 条目）。
- **Master key 一次生成永不更新**（`internal/core/keychain/keychain.go:35` `MasterKeyAccount = "master.key"`）。token 刷新只动加密数据，不动 keychain。
- **OpenId 白名单校验**（`internal/config/user.go:50` `validateOpenId`）：只允许字母+数字+下划线，且不能等于 `master.key`，防路径穿越 / 覆盖主密钥。
- **写配置走原子写**：`saveMeta` / `SaveAgentConfig` 都是 `tmp 文件 → Sync → Rename`，不能改成直接 `os.WriteFile`。

## Token 刷新

- **提前 60 秒刷新**（`internal/auth/auth.go:26` `accessTokenRefreshLeeway`）：吸收时钟漂移 + 网络延迟。改小会导致请求到达服务器时 token 已过期，rest-proxy 视为致命错误清掉本地凭证。
- **刷新必须加文件锁**（`config.GetTokenLockPath()` → `filelock.WithLock`）：多进程并发刷新会互相覆盖。锁内必须 `ResetCache` + 重读配置，判断"别的进程刚刷过"就直接返回。
- **服务器返回 token 过期 → 清本地凭证**（`rest-proxy/proxy.go:65`）：不能改成静默忽略，否则用户陷入死循环。

## Panic 兜底与日志关闭顺序

`cmd/root.go:47-71` 的 defer 顺序是刻意设计的：

```go
if logErr := log.Init(...); logErr == nil { defer log.Close() }  // 先注册
defer func() { crash.RecoverAndReport(...) }                    // 后注册
```

LIFO 下 `log.Close()` 最后执行。**不能颠倒**：写已关闭的 logger 会 panic，被 crash reporter 内层 recover 静默吞掉。`crash.RecoverAndReport` 也**禁止调用 `os.Exit`**，否则会跳过 `log.Close()` 丢失日志。

## 日志写入模型

- **直接 `O_APPEND` 写文件，无 bufio 缓冲**（`internal/log/logging.go:342`）：单次 `write()` 原子（POSIX 保证），多进程并发不交错。改用 `bufio` 会破坏跨进程原子性。
- **轮转加跨进程锁**（`rotateWithLock` → `filelock.WithLock`）：防止多进程同时创建 `.1.log`。
- **send 用 channel 非阻塞**（`send` 的 `select default`）：channel 满时丢弃日志，绝不阻塞调用方。

## 分页兼容

v1.0.5 起统一 `--page-token` + `--page-size`，旧参数 `--page` / `--pos` / `--size` 标记 deprecated 但仍兼容。`ChoosePageOrToken` / `ChoosePosOrToken`（`internal/cmdutil/pagination.go`）实现优先级：**有 token 用 token，否则用旧参数，都没有就 token 模式**。新增分页命令必须复用这两个函数，不能自造逻辑。

## ApiCmd 路由

部分命令按 flag 路由到不同后端 ApiCmd（如 `meeting get` 按 `--meeting-id` / `--meeting-code` 分别走 `meeting_get_by_id` / `meeting_get_by_code`）。通过 `cmdutil.FlagSwitch` + `cmdutil.FlagCase` 声明，**短路求值，先匹配先赢**，高优先级 case 必须放前面（见 `cmd/meeting/get.go:32-41`）。

## 输出契约

- **JSON 不转义 HTML**（`internal/output/print.go:15` `marshalNoEscape`）：`SetEscapeHTML(false)`，否则 URL 里的 `&` 会被转义成 `\u0026`。
- **`--compact` 裁剪字段表来自远程**（`WithCompact` → `GetAPISchema`）：不能硬编码字段列表，否则 schema 更新后失效。
- **`total_count=0` 被剥离**（`WithTotalCountLogic`）：服务器返回 0 通常是占位，不能展示给用户。
