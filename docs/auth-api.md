# auth 命令接口文档

> 腾讯会议 CLI(`tmeet`)auth 命令下所有子命令的 HTTP 接口、入参、出参及描述信息。
> 依据源码整理,代码版本 v1.0.14。

## 接口总览

| 子命令 | HTTP 接口数 | 代理类型 | 域名 |
|---|---|---|---|
| `auth login` | 2 个 CGI 接口 + 1 个浏览器授权页 | `cgiProxy` | `work.medialab.qq.com` |
| `auth logout` | 0(纯本地操作) | — | — |
| `auth status` | 1 个 REST 接口 + 1 个隐式刷新接口 | `restProxy` / `cgiProxy` | `api.meeting.qq.com` / `work.medialab.qq.com` |

## 通用请求头

### CGI 代理接口(login / refresh-token)

使用 `DefaultJsonAuthenticator`(`JsonAuthenticator`),**不携带 OAuth2 token**,仅设置:

| Header | 值 |
|---|---|
| `Content-Type` | `application/json; charset=utf-8` |

### REST 代理接口(status / get-user-info)

使用 `OAuth2Authenticator`,携带完整鉴权头:

| Header | 值 | 说明 |
|---|---|---|
| `Content-Type` | `application/json; charset=utf-8` | |
| `AccessToken` | `<access_token>` | OAuth2 认证 token |
| `OpenId` | `<open_id>` | 用户唯一标识 |
| `X-TC-Nonce` | `<随机正整数>` | 参与签名,自动生成 |
| `X-TC-Timestamp` | `<Unix 秒级时间戳>` | 请求发起时间,与服务端偏差 ≤5 分钟 |
| `Tmeet-Unique-ID` | `<openId>*<machineId>` | 设备唯一标识 |
| `Tmeet-Device-Info` | `<os>;<agent>;<model>` | 设备信息 |
| `Tmeet-Open-Source` | `CLI` | 固定值 |
| `Tmeet-Cli-Ver` | `<version>` | CLI 版本号 |
| `Tmeet-Trace` | `<cmdPath>;<traceId>` | 命令路径 + 链路追踪 ID |
| `Tmeet-Cli-Name` | `<apiCmd>` | API 命令标识 |

## CGI 响应信封结构

CGI 代理接口统一使用 `ProxyRsp[T]` 泛型信封(`internal/proxy/cgi-proxy/proxy.go:12`):

```json
{
  "code": 0,
  "message": "...",
  "nonce": "<Gw-Trace-Id>",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `code` | int32 | 业务码,0=成功 |
| `message` | string | 业务描述 |
| `nonce` | string | 取自响应头 `Gw-Trace-Id` |
| `data` | T | 泛型数据体,代理层完成强类型反序列化 |

## REST 响应信封结构

REST 代理接口使用 `ProxyRsp`(`internal/proxy/rest-proxy/proxy.go:27`):

```json
{
  "data": "<原始响应体字符串>",
  "trace_id": "<X-TC-Trace>",
  "message": "success"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `data` | string | 原始响应体字符串,由调用方自行反序列化 |
| `trace_id` | string | 取自响应头 `X-TC-Trace` |
| `message` | string | 成功时为 `"success"` |

---

## 接口 1:获取设备授权码

| 项目 | 内容 |
|---|---|
| **地址** | `POST https://work.medialab.qq.com/v2/oauth2/oauth/cli-oauth-init` |
| **代码位置** | `internal/auth/auth_code.go:18` |
| **触发命令** | `tmeet auth login` |
| **触发条件** | 总是执行(登录流程第一步) |

### 入参(Body)

```json
{
  "device_id": "<MachineID>"
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `device_id` | string | 是 | 48 位随机字符,缓存于 `~/.tmeet/.machine_id`,首次运行自动生成 |

### 出参(data)

```json
{
  "auth_code": "<authCode>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `auth_code` | string | 设备授权码,用于拼接浏览器授权页 URL 和接口 2 的入参 |

### 描述

登录流程第一步。向服务端申请一个设备授权码 `auth_code`,后续用此码拼接浏览器授权页 URL。设备 ID 在本地缓存,保证同一设备的跨次登录身份一致。

**错误处理:** `code!=0 || data==nil` → `GetAuthCodeError`;HTTP≠200 → `NetworkError`。

---

## 接口 2:轮询换取 token

| 项目 | 内容 |
|---|---|
| **地址** | `POST https://work.medialab.qq.com/v2/oauth2/oauth/cli-oauth-poll` |
| **代码位置** | `internal/auth/auth_token.go:22` |
| **触发命令** | `tmeet auth login` |
| **触发条件** | 获取 auth_code 后,每 5 秒轮询一次,最多 5 分钟超时 |

### 入参(Body)

```json
{
  "device_id": "<MachineID>",
  "auth_code": "<authCode>"
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `device_id` | string | 是 | 48 位随机字符,同接口 1 |
| `auth_code` | string | 是 | 来自接口 1 的响应 |

### 出参(data)

```json
{
  "access_token": "<accessToken>",
  "refresh_token": "<refreshToken>",
  "access_token_expire_time": "<秒级时间戳字符串>",
  "refresh_token_expire_time": "<秒级时间戳字符串>",
  "open_id": "<openId>",
  "sdk_id": "<sdkId>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `access_token` | string | 访问令牌,有效期约 30 天 |
| `refresh_token` | string | 刷新令牌,有效期约 90 天 |
| `access_token_expire_time` | string | access_token 过期时间(Unix 秒级时间戳字符串) |
| `refresh_token_expire_time` | string | refresh_token 过期时间(Unix 秒级时间戳字符串) |
| `open_id` | string | 用户唯一标识;未扫码时为空 → 触发下一轮轮询 |
| `sdk_id` | string | 应用 ID |

### 描述

登录流程第二步。用户在浏览器扫码授权前,响应中 `open_id` 为空,CLI 继续等待;授权成功后返回完整 token 信息。成功后 `SaveUserConfig` 将这些字段 AES-256-GCM 加密写入 keychain(Windows 注册表),明文不落盘。

**轮询机制:** 每 5 秒一次(`loopPollTime`),最多 5 分钟(`authorizationTimeout`)。

**错误处理:** `code!=0` → `AuthorizationFailedError`;超时 → `AuthorizationTimeoutError`。

---

## 接口 3:获取用户信息

| 项目 | 内容 |
|---|---|
| **地址** | `POST https://api.meeting.qq.com/v1/cli/get-user-info` |
| **代码位置** | `cmd/auth/status.go:133` |
| **触发命令** | `tmeet auth status` |
| **触发条件** | 仅当 RefreshToken 未过期时才发请求;已过期或未登录则跳过 |

### 入参(Body)

```json
{
  "operator_id": "<OpenId>",
  "operator_id_type": 2
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `operator_id` | string | 是 | 用户的 OpenId,来自本地加密凭证 |
| `operator_id_type` | int | 是 | 固定值 2,表示 OpenId 类型 |

### 出参(data)

```json
{
  "user_name": "<用户名>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `user_name` | string | 用户显示名称,用于 status 输出中展示 |

### 描述

`auth status` 命令在确认 RefreshToken 未过期后,调用此接口获取用户名并展示。请求经 REST 代理,会自动触发 token 刷新逻辑(见接口 4)。此接口不使用 `FormatPrint` 结构化输出,而是用 `PrintInfof` 打印自由文本格式的登录状态(OpenId / UserName / AccessToken 过期状态 / RefreshToken 过期状态)。

---

## 接口 4:刷新 token(隐式触发)

| 项目 | 内容 |
|---|---|
| **地址** | `POST https://work.medialab.qq.com/v2/oauth2/oauth/cli-refresh-token` |
| **代码位置** | `internal/auth/auth_token.go:38` |
| **触发命令** | `tmeet auth status`(以及其他所有走 rest-proxy 的业务命令) |
| **触发条件** | AccessToken 距过期 ≤60 秒(`accessTokenRefreshLeeway`);若 RefreshToken 已过期则不刷新,直接清除本地凭证 |

### 入参(Body)

```json
{
  "device_id": "<MachineID>",
  "refresh_token": "<refreshToken>",
  "open_id": "<OpenId>",
  "sdk_id": "<sdkId>"
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `device_id` | string | 是 | 48 位随机字符 |
| `refresh_token` | string | 是 | 当前的 refresh_token |
| `open_id` | string | 是 | 用户的 OpenId |
| `sdk_id` | string | 是 | 应用 ID |

### 出参(data)

```json
{
  "access_token": "<新accessToken>",
  "refresh_token": "<新refreshToken>",
  "access_token_expire_time": "<新的过期时间>",
  "refresh_token_expire_time": "<新的过期时间>",
  "open_id": "<openId>",
  "sdk_id": "<sdkId>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `access_token` | string | 新的访问令牌 |
| `refresh_token` | string | 新的刷新令牌 |
| `access_token_expire_time` | string | 新 access_token 的过期时间 |
| `refresh_token_expire_time` | string | 新 refresh_token 的过期时间 |
| `open_id` | string | 用户唯一标识 |
| `sdk_id` | string | 应用 ID |

### 描述

由 rest-proxy 的 `RequestProxy` 在发起业务请求前自动调用(`internal/auth/auth.go:72`)。采用 `token.lock` 文件锁防多进程并发刷新,锁内双重检查(重读配置判断其他进程是否已刷新)。刷新成功后 `SaveUserConfig` 写回 keychain,主密钥不变。刷新失败则 `ClearUserConfig` 清除本地凭证。

**错误处理:** `code!=0` → `RefreshTokenFailedError`,并清除本地凭证;RefreshToken 已过期 → `UserIdentityExpiredError`。

---

## 本地操作:登出(无 HTTP 接口)

| 项目 | 内容 |
|---|---|
| **触发命令** | `tmeet auth logout` |
| **代码位置** | `cmd/auth/logout.go:36` → `internal/config/user.go:145` `ClearUserConfig` |
| **HTTP 接口** | 无 |

### 操作内容

| 步骤 | 说明 |
|---|---|
| ① 删除加密凭证 | 从 keychain 删除 `<open_id>` 条目(Windows:删除注册表 `HKCU\Software\TmeetCli\keychain\<open_id>` 值) |
| ② 删除元数据 | 删除 `~/.tmeet/config.json` |
| ③ 清空内存缓存 | `userConfig = nil` |

### 不变量

| 保留项 | 原因 |
|---|---|
| 主密钥(`master.key`) | 一次生成永不更新,供其他用户继续使用 |
| `agent.json` | AI-Agent 身份信息,与用户凭证生命周期独立 |
| `~/.tmeet/.machine_id` | 设备标识,跨登录会话保持一致 |

### 描述

纯本地操作,不通知服务端。用 `retry.Do` 默认重试策略包裹。成功输出 `Logout success, welcome to use it again.`,失败返回 `LogoutFailedError`。

---

## 完整接口清单

| # | 子命令 | 方法 | 地址 | 代理类型 | 触发条件 |
|---|---|---|---|---|---|
| 1 | `auth login` | POST | `https://work.medialab.qq.com/v2/oauth2/oauth/cli-oauth-init` | CGI | 总是 |
| 2 | `auth login` | POST | `https://work.medialab.qq.com/v2/oauth2/oauth/cli-oauth-poll` | CGI | 每 5s 轮询,最长 5min |
| 3 | `auth status` | POST | `https://api.meeting.qq.com/v1/cli/get-user-info` | REST | RefreshToken 未过期时 |
| 4 | `auth status`(隐式) | POST | `https://work.medialab.qq.com/v2/oauth2/oauth/cli-refresh-token` | CGI | AccessToken ≤60s 过期时 |
| — | `auth logout` | — | 无(纯本地删除) | — | 总是 |

---

## 关键设计要点

| 关注点 | 说明 |
|---|---|
| **login 用 CGI 代理** | 登录是获取 token 的过程,不能用需要 token 的 REST 代理;CGI 代理用 `JsonAuthenticator`,不携带 token |
| **status 用 REST 代理** | status 是使用 token 的业务命令,走标准 REST 代理链路,自动刷新 token + OAuth2 鉴权头 |
| **凭证零落盘明文** | UserConfig 经 AES-256-GCM 加密后写入 keychain(Windows 注册表),`config.json` 仅存 `active_open_id` 明文指针 |
| **主密钥一次生成永不更新** | `MasterKeyAccount = "master.key"`;token 刷新只更新加密数据,不动主密钥,重启后仍可解密 |
| **提前 60 秒刷新** | `accessTokenRefreshLeeway` 吸收时钟偏移和网络延迟,防止请求到达服务端时 token 已过期 |
| **文件锁 + 双重检查** | `token.lock` 防多进程并发刷新互相覆盖;锁内重读配置判断其他进程是否已刷新 |
| **token 过期即清凭证** | 服务端返回 token 过期时,rest-proxy 自动 `ClearUserConfig`,避免用户陷入死循环 |
