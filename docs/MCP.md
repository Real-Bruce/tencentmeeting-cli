# hdtmeet-mcp

将腾讯会议的全部能力以 [MCP（Model Context Protocol）](https://modelcontextprotocol.io/) tool 形式暴露给 AI Agent。

统一采用 **streamable-http** 传输模式，本地测试与远程部署同一套命令，差异仅在 `--addr` 绑定地址。

---

## 目录

- [快速开始](#快速开始)
- [启动参数](#启动参数)
- [Linux 服务器部署](#linux-服务器部署)
- [多实例部署](#多实例部署)
- [AI 客户端接入](#ai-客户端接入)
- [鉴权](#鉴权)
- [TLS / HTTPS](#tls--https)
- [可用 Tool 列表](#可用-tool-列表)
- [OAuth 登录流程](#oauth-登录流程)
- [调试](#调试)
- [FAQ](#faq)

---

## 快速开始

### 1. 完成 OAuth 登录

MCP server 启动时读取本地已保存的 OAuth 凭证，**必须先在终端完成一次登录**（`--name` 必填，用于为账号指定别名）：

```bash
tmeet auth login --name <别名>
```

浏览器扫码授权后凭证加密保存到本地，后续 MCP server 直接复用，无需重复登录。

### 2. 启动 MCP server

```bash
# 本地测试
tmeet mcp --transport streamable-http --addr 127.0.0.1:8347

# 远程部署（生产环境务必加 --api-key）
tmeet mcp --transport streamable-http --addr 0.0.0.0:8347 --api-key your-secret-key
```

客户端连接地址：`http://<server-ip>:8347/mcp`

> 首次 `initialize` 请求会收到 `Mcp-Session-Id` 响应头，后续请求需在请求头中携带该 ID。

---

## 启动参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `--transport` | `stdio` | 传输模式，支持 `stdio` / `streamable-http` / `sse`。远程部署用 `streamable-http` |
| `--addr` | `:8347` | HTTP 监听地址（streamable-http/sse 用）。本地测试用 `127.0.0.1:8347`，远程部署用 `0.0.0.0:8347` |
| `--endpoint` | `/mcp` | streamable-http 的端点路径 |
| `--api-key` | — | Bearer Token 鉴权密钥。HTTP 模式下生产环境必填 |
| `--tls-cert` / `--tls-key` | — | 直接启用 TLS（无需 Nginx），HTTP 模式可用 |
| `--base-url` | — | 外部可访问的基础 URL，主要用于 sse / 反向代理场景 |

---

## Linux 服务器部署

### 1. 编译

```bash
# 在开发机交叉编译 Linux 产物
GOOS=linux GOARCH=amd64 go build -ldflags "-X tmeet/cmd.Version=v1.0.14" -o tmeet .

# 或全平台交叉编译
make build VERSION=v1.0.14
# 产物在 dist/tmeet-linux-amd64
```

### 2. 部署到服务器

```bash
scp tmeet user@your-server:/opt/tmeet/tmeet
ssh user@your-server
chmod +x /opt/tmeet/tmeet
```

### 3. 完成 OAuth 登录

先建专用系统用户与凭证目录（一次性）。登录、systemd 服务都以此用户与目录操作，保证凭证写入与读取路径一致：

```bash
sudo useradd -r -s /usr/sbin/nologin -d /opt/tmeet tmeet
sudo install -d -o tmeet -g tmeet -m 700 /opt/tmeet/.tmeet
```

以 `tmeet` 用户、显式指定凭证目录执行登录（`--name` 必填，为账号指定别名）：

```bash
sudo -u tmeet env TMEET_CLI_CONFIG_DIR=/opt/tmeet/.tmeet \
  TMEET_CLI_DATA_DIR=/opt/tmeet/.tmeet \
  /opt/tmeet/tmeet auth login --name <别名> --no-browser
```

> 服务器没有浏览器时用 `--no-browser` 获取授权 URL，在本地浏览器打开扫码。
>
> ⚠️ 登录与 systemd unit 必须使用**同一套** `TMEET_CLI_CONFIG_DIR` + `TMEET_CLI_DATA_DIR`，且目录属主匹配服务的 `User=`。否则服务以其他用户/其他 HOME 启动时，会读到空目录，`auth_list` 查不到账号。

### 4. 配置 systemd 守护进程

创建 `/etc/systemd/system/hdtmeet-mcp.service`：

```ini
[Unit]
Description=hdtmeet-mcp
After=network.target

[Service]
Type=simple
User=tmeet
Group=tmeet
ExecStart=/opt/tmeet/tmeet mcp \
  --transport streamable-http \
  --addr 0.0.0.0:8347 \
  --api-key your-secret-key
WorkingDirectory=/opt/tmeet
Environment=TMEET_CLI_CONFIG_DIR=/opt/tmeet/.tmeet
Environment=TMEET_CLI_DATA_DIR=/opt/tmeet/.tmeet
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hdtmeet-mcp
sudo systemctl status hdtmeet-mcp    # 查看运行状态
sudo journalctl -u hdtmeet-mcp -f    # 查看实时日志
```

### 5. 反向代理（可选）

如果用 Nginx 反向代理并终止 TLS：

```nginx
server {
    listen 443 ssl;
    server_name tmeet.example.com;

    ssl_certificate     /etc/ssl/tmeet.crt;
    ssl_certificate_key /etc/ssl/tmeet.key;

    location /mcp {
        proxy_pass http://127.0.0.1:8347;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        # streamable-http 需要长连接
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 300s;
    }
}
```

此时客户端连接地址为 `https://tmeet.example.com/mcp`。

---

## 多实例部署

同一台服务器为多个用户/部门各自运行独立 MCP server：每个实例持有该用户的 OAuth 凭证，靠「端口 + 凭证目录 + 数据目录」三重隔离，共用一份二进制和一个 systemd 模板单元。

### 隔离维度

| 维度 | 机制 | 说明 |
|------|------|------|
| 端口 | `--addr` 各异 | 每实例独占端口 |
| OAuth 凭证目录 | `TMEET_CLI_CONFIG_DIR` | 独立 `config.json` + `token.lock` + `<openid>.enc` |
| 加密数据目录 | `TMEET_CLI_DATA_DIR` | 独立 `master.key` + `.enc` 文件 |
| api-key | 各自随机生成 | 每实例独立 Bearer Token |
| systemd 实例名 | `hdtmeet-mcp@<name>` | 模板单元 `%i` 取实例名 |

> 所有实例共用一个 `tmeet` 系统用户和一份二进制。若需跨人强隔离（一实例沦陷不波及他人），可改为每实例一个 OS 用户并在 unit 内设 `User=%i`。

### 目录结构

```
/opt/hdtmeet/tmeet              # 单一二进制，所有实例共用
/opt/hdtmeet/<name>/            # 每实例独立（config.json + token.lock + master.key + *.enc，0700）
/etc/hdtmeet/<name>.conf        # 每实例 env 文件（ADDR + API_KEY，0600）
/etc/systemd/system/hdtmeet-mcp@.service   # 模板单元，%i = 实例名
```

### 1. 建服务用户 + 基础目录

```bash
sudo useradd -r -s /usr/sbin/nologin -d /opt/hdtmeet tmeet
sudo mkdir -p /opt/hdtmeet /etc/hdtmeet
sudo chown tmeet:tmeet /opt/hdtmeet /etc/hdtmeet
sudo chmod 700 /etc/hdtmeet
sudo install -m 0755 -o tmeet -g tmeet tmeet /opt/hdtmeet/tmeet   # 装二进制
```

### 2. 写 systemd 模板单元

`/etc/systemd/system/hdtmeet-mcp@.service`：

```ini
[Unit]
Description=hdtmeet-mcp (%i)
After=network.target

[Service]
Type=simple
User=tmeet
Group=tmeet
EnvironmentFile=/etc/hdtmeet/%i.conf
ExecStart=/opt/hdtmeet/tmeet mcp --transport streamable-http --addr ${ADDR} --api-key ${API_KEY}
WorkingDirectory=/opt/hdtmeet
Environment=TMEET_CLI_CONFIG_DIR=/opt/hdtmeet/%i
Environment=TMEET_CLI_DATA_DIR=/opt/hdtmeet/%i
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

> `%i` 为实例名占位符。`systemctl start hdtmeet-mcp@qtj` 时 `%i=qtj`，自动读取 `/etc/hdtmeet/qtj.conf` 并使用 `/opt/hdtmeet/qtj/` 作为凭证目录。

### 3. 批量建实例目录 + env 文件

以 7 个部门为例（端口 8347–8353）：

```bash
port=8347
for u in qtj xfq qx dt gyl ls sy; do
  sudo install -d -o tmeet -g tmeet -m 700 "/opt/hdtmeet/$u"
  key=$(openssl rand -hex 24)
  sudo tee "/etc/hdtmeet/$u.conf" >/dev/null <<EOF
ADDR=0.0.0.0:$port
API_KEY=$key
EOF
  sudo chown tmeet:tmeet "/etc/hdtmeet/$u.conf"
  sudo chmod 600 "/etc/hdtmeet/$u.conf"
  echo "$u  http://<server-ip>:$port/mcp  key=$key"
  port=$((port+1))
done
```

> 记下输出中的 api-key，后续分发给各客户端。

### 4. 每实例 OAuth 登录

每实例单独登录一次，凭证落入各自目录：

```bash
sudo -u tmeet env TMEET_CLI_CONFIG_DIR=/opt/hdtmeet/qtj \
  TMEET_CLI_DATA_DIR=/opt/hdtmeet/qtj \
  /opt/hdtmeet/tmeet auth login --name qtj --no-browser
```

> 换实例名时同步替换 `--name`（与实例名一致便于辨认），重复每个实例。`--no-browser` 输出授权 URL，发给对应用户在本地浏览器扫码。refresh_token 约 30 天到期后用同命令重登该实例。

### 5. 防火墙开端口

```bash
sudo firewall-cmd --permanent --add-port=8347-8353/tcp
sudo firewall-cmd --reload
```

### 6. 启动 + 开机自启

```bash
sudo systemctl daemon-reload
for u in qtj xfq qx dt gyl ls sy; do
  sudo systemctl enable --now hdtmeet-mcp@$u
done
```

### 7. 验证与运维

```bash
sudo systemctl list-units 'hdtmeet-mcp@*'          # 一览全部实例
sudo systemctl status hdtmeet-mcp@qtj              # 单个状态
sudo journalctl -u hdtmeet-mcp@qtj -f              # 单个实时日志

sudo systemctl restart hdtmeet-mcp@qtj             # 重启某人（凭证续期后）
sudo systemctl restart 'hdtmeet-mcp@*'              # 全部重启
sudo systemctl disable --now hdtmeet-mcp@qtj       # 停 + 取消自启
sudo rm -rf /opt/hdtmeet/qtj /etc/hdtmeet/qtj.conf # 彻底清理某人
```

### 客户端分发

每实例一份连接信息：

- URL：`http://<server-ip>:<port>/mcp`
- Header：`Authorization: Bearer <该实例 api-key>`

### 升级二进制

```bash
sudo install -m 0755 -o tmeet -g tmeet tmeet /opt/hdtmeet/tmeet
sudo systemctl restart 'hdtmeet-mcp@*'   # 配置与凭证不受影响
```

---

## AI 客户端接入

### Cursor

项目级配置（项目根目录 `.cursor/mcp.json`）或全局配置（`~/.cursor/mcp.json`），Cursor 0.42+：

```json
{
  "mcpServers": {
    "hdtmeet-mcp": {
      "url": "http://your-server:8347/mcp",
      "headers": {
        "Authorization": "Bearer your-secret-key"
      }
    }
  }
}
```

### 通用 HTTP 接入

任何支持 MCP Streamable HTTP 传输的客户端，配置 URL：

```
URL:    http://<server-ip>:8347/mcp
Header: Authorization: Bearer <api-key>   （如启用鉴权）
```

---

## 鉴权

基于 Bearer Token 的 API Key 鉴权。

```bash
tmeet mcp --transport streamable-http --addr 0.0.0.0:8347 --api-key my-secret-key
```

启用后，客户端必须在每个请求中携带 `Authorization: Bearer my-secret-key` 头，否则返回 `401 Unauthorized`。

> **生产环境务必设置 api-key**，否则任何能访问该端口的客户端都可使用你的腾讯会议凭证调用全部 tool。

---

## TLS / HTTPS

直接启用 TLS（无需 Nginx）：

```bash
tmeet mcp --transport streamable-http \
  --addr 0.0.0.0:8443 \
  --tls-cert /etc/ssl/tmeet.crt \
  --tls-key /etc/ssl/tmeet.key \
  --api-key your-secret-key
```

客户端连接地址为 `https://<server-ip>:8443/mcp`。

> 如已使用 Nginx 反向代理终止 TLS，则不需要 tmeet 自身启用 TLS。

---

## 可用 Tool 列表

共 33 个 tool，覆盖腾讯会议全部能力。

### auth — OAuth 登录与多账号（4 个）

| Tool | 说明 |
|------|------|
| `auth_login_url` | 获取授权 URL，返回 `{authorize_url, code}`。用户在浏览器打开授权。**必填** `name` 参数为账号指定别名（多账号登录） |
| `auth_login_poll` | 单次轮询登录结果。传入 `code`，返回 `pending`/`success`/`error`。**必填** `name` 参数，需与 `auth_login_url` 一致 |
| `auth_list` | 列出全部已登录账号（别名/openId/是否活跃/refresh token 状态）。无需登录 |
| `auth_use` | 按别名切换活跃账号，传入 `name`。切换后后续 tool 均作用于新活跃账号 |

> 多账号：`auth_login_url` 与 `auth_login_poll` 的 `name` 均为必填，且需保持一致，用于为本次登录的账号指定别名。
> 远程部署场景下，可在服务器上先 `tmeet auth login --name <别名>` 完成登录，无需使用登录类 tool。它们主要用于模型引导用户完成登录。

#### auth_login_url

获取 OAuth 授权 URL，用户在浏览器打开完成授权。若该别名账号已登录（refresh token 未过期），直接返回 `already_login=true`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 账号别名，如 `token-qtj`；需在 `auth_login_poll` 传入相同值 |

**出参**

自定义结构（非 API 透传）：

| 字段 | 类型 | 说明 |
|------|------|------|
| authorize_url | string | OAuth 授权 URL，引导用户在浏览器打开 |
| code | string | 授权码，传给 `auth_login_poll` |
| already_login | bool | 仅在已登录时出现，为 `true` 表示无需再次授权 |
| message | string | 提示信息（如 `account <name> already logged in, no need to authorize again`） |

#### auth_login_poll

单次轮询登录结果。返回 `pending`（用户尚未完成授权，数秒后重试）或 `success`（登录成功）。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 来自 `auth_login_url` 的授权码 |
| name | string | 是 | 账号别名，必须与 `auth_login_url` 一致 |

**出参**

自定义结构（非 API 透传）：

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | `pending` / `success` / `error` |
| message | string | 提示信息（如 `login successful for account <name>, other tools are now available`） |

#### auth_list

列出全部已登录账号。无需登录即可调用。

**入参**

无。

**出参**

账号数组，每项字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 账号别名 |
| open_id | string | 账号 openId |
| active | bool | 是否为当前活跃账号 |
| token_state | string | refresh token 状态，如 `refresh-token valid (remaining 1h2m3s)` 或 `refresh-token expired` |

#### auth_use

按别名切换活跃账号，切换后后续 tool 均作用于新活跃账号。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 要切换到的账号别名（见 `auth_list`） |

**出参**

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | `success` |
| message | string | `switched to account <name>` |

### meeting — 会议管理（12 个）

| Tool | 说明 |
|------|------|
| `meeting_list` | 列出待开始/进行中的会议 |
| `meeting_list_ended` | 列出已结束的会议 |
| `meeting_search` | 按关键字/会议号/时间范围/状态搜索会议 |
| `meeting_get` | 查询会议详情（按 meeting_id 或 meeting_code） |
| `meeting_create` | 创建会议（支持周期性会议） |
| `meeting_update` | 更新会议（仅更新传入的字段） |
| `meeting_cancel` | 取消会议（支持取消整个周期性会议或单场） |
| `meeting_invitees_list` | 查询会议受邀者列表 |
| `meeting_invitees_add` | 添加会议受邀者 |
| `meeting_invitees_remove` | 移除会议受邀者 |
| `meeting_invitees_replace` | 替换全部受邀者 |
| `meeting_conflict` | 检测某时间段是否与已有待开始/进行中会议冲突，返回 `meet_conflict` 布尔值与冲突会议列表 |

> 以下各 tool 的出参统一为 `result.Result.Data`，即腾讯会议 REST API 响应经字段转换（时间戳→ISO 日期、状态码→中文、base64→明文、`total_count==0` 剥离等）后的 JSON 文本；`TraceId`/`Message` 不进入 MCP 输出。

#### meeting_list

列出当前用户的待开始/进行中会议。时间参数用带时区的 ISO 8601，如 `2026-03-12T15:00+08:00`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | string | 否 | 查询开始时间（ISO 8601） |
| end | string | 否 | 查询结束时间（ISO 8601） |
| show_all_sub | integer | 否 | 是否展示全部子会议：0-否（默认），1-是 |
| page_token | string | 否 | 分页 token，取自上一次响应 |
| page_size | integer | 否 | 分页大小，最大 20，默认 20 |

**出参**

会议列表 JSON，含 `meeting_info_list`（每项含 `meeting_code`（向用户展示此项而非 meeting_id）、`status`、`start_time`/`end_time`（已转 ISO 日期）、`meeting_type`、`recurring_type`、`until_date`、`until_type`、`join_meeting_role`、`only_user_join_type`、`is_show_all_sub_meetings` 等）及 `next_page_token`。

#### meeting_list_ended

列出当前用户在时间范围内的已结束会议。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | string | 否 | 查询开始时间（ISO 8601） |
| end | string | 否 | 查询结束时间（ISO 8601） |
| page | integer | 否 | 页码，从 1 开始（已废弃，用 `page_token`） |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |
| page_token | string | 否 | 分页 token |

**出参**

会议列表 JSON，含 `meeting_info_list`（每项含 `meeting_code`、`subject`、`start_time`/`end_time`（已转 ISO 日期）、`meeting_type` 等）及分页字段。

#### meeting_search

按关键字、会议号、时间范围、状态搜索会议。`query_field` 取 `subject`/`creator`/`note`/`all`；`start`/`end` 定义时间窗口。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | 否 | 搜索关键字（最长 50 字符） |
| query_field | string | 否 | 搜索字段：`subject`/`creator`/`note`/`all`（默认 `all`） |
| meeting_code | string | 否 | 按会议号精确过滤（仅数字） |
| start | string | 否 | 时间窗口下界（ISO 8601） |
| end | string | 否 | 时间窗口上界（ISO 8601） |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |
| page_token | string | 否 | 分页 token |

**出参**

搜索结果 JSON，含 `meetings`（已转换 `meeting_type`、`status`）；`total_count` 为 0 时剥离。

#### meeting_get

按 `meeting_id` 或 `meeting_code` 查询会议详情（二者必传其一）。`meeting_id` 为内部 ID，向用户展示 `meeting_code`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 否 | 会议 ID（运行时 `meeting_id`/`meeting_code` 至少传其一） |
| meeting_code | string | 否 | 会议号（运行时 `meeting_id`/`meeting_code` 至少传其一） |

**出参**

会议详情 JSON，含 `subject`、`start_time`/`end_time`（已转 ISO 日期）、`meeting_info_list.status`、`meeting_type`、`recurring_type`、`time_zone`（已 base64 解码）、`until_date`、`until_type` 等。

#### meeting_create

创建会议。`subject`/`start`/`end` 必填（ISO 8601 带时区）。周期性会议需 `meeting_type=1` 并提供 `recurring_type`/`until_type`/`until_count`[/`until_date`]。`water_mark_type`/`audio_watermark`/`auto_record_type`/`auto_asr` 仅在传入时生效。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| subject | string | 是 | 会议主题 |
| start | string | 是 | 开始时间（ISO 8601，如 `2026-03-12T14:00+08:00`） |
| end | string | 是 | 结束时间（ISO 8601） |
| password | string | 否 | 会议密码（4-6 位数字） |
| timezone | string | 否 | 时区，如 `Asia/Shanghai` |
| meeting_type | integer | 否 | 0-普通（默认），1-周期性 |
| join_type | integer | 否 | 入会限制：1-所有人，2-仅受邀，3-仅内部 |
| waiting_room | boolean | 否 | 是否启用等候室 |
| recurring_type | integer | 否 | 周期类型（`meeting_type=1` 时）：0-每日，1-工作日，2-每周，3-双周，4-每月 |
| until_type | integer | 否 | 周期结束方式：0-按日期，1-按次数 |
| until_count | integer | 否 | 周期次数（每日/每周≤500，双周/每月≤50，默认 7） |
| until_date | string | 否 | 周期结束日期（ISO 8601） |
| invitees | array&lt;string&gt; | 否 | 受邀者 open_id 列表（最多 100） |
| water_mark_type | integer | 否 | 文字水印：0-单行，1-双行，2-关闭（默认）。仅在传入时生效 |
| audio_watermark | boolean | 否 | 音频水印。仅在传入时生效 |
| auto_record_type | string | 否 | 自动录制：`none`/`local`/`cloud`。仅在传入时生效 |
| auto_asr | boolean | 否 | 自动语音转写。仅在传入时生效 |

**出参**

会议详情 JSON，含 `meeting_code`、`start_time`/`end_time`（已转 ISO 日期）、`only_user_join_type` 等。

#### meeting_update

按 `meeting_id` 更新会议，仅更新传入字段。周期会议：设 `meeting_type=1` 配周期字段，或设 `sub_meeting_id` 更新单场子会议时间（二者互斥）。`invitees` 与 `invitees_type`（replace/add/remove）必须同时传入。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| subject | string | 否 | 会议主题 |
| start | string | 否 | 开始时间（ISO 8601） |
| end | string | 否 | 结束时间（ISO 8601） |
| password | string | 否 | 会议密码（4-6 位数字） |
| timezone | string | 否 | 时区 |
| meeting_type | integer | 否 | 0-普通，1-周期性。仅在传入时生效 |
| join_type | integer | 否 | 入会限制：1-所有人，2-仅受邀，3-仅内部 |
| waiting_room | boolean | 否 | 是否启用等候室 |
| recurring_type | integer | 否 | 周期类型（`meeting_type=1` 时） |
| until_type | integer | 否 | 周期结束方式 |
| until_count | integer | 否 | 周期次数 |
| until_date | string | 否 | 周期结束日期（ISO 8601） |
| sub_meeting_id | string | 否 | 子会议 ID，用于更新单场子会议时间（与周期字段互斥） |
| invitees | array&lt;string&gt; | 否 | 受邀者 open_id 列表 |
| invitees_type | string | 否 | 受邀者策略：`replace`/`add`/`remove`（传 `invitees` 时必填） |
| water_mark_type | integer | 否 | 文字水印：0-单行，1-双行，2-关闭。仅在传入时生效 |
| audio_watermark | boolean | 否 | 音频水印。仅在传入时生效 |
| auto_record_type | string | 否 | 自动录制：`none`/`local`/`cloud`。仅在传入时生效 |
| auto_asr | boolean | 否 | 自动语音转写。仅在传入时生效 |

**出参**

会议详情 JSON，含 `meeting_code`、`start_time`/`end_time`（已转 ISO 日期）、`only_user_join_type` 等。

#### meeting_cancel

按 `meeting_id` 取消会议。周期会议：设 `meeting_type=1` 取消整个系列，或提供 `sub_meeting_id` 取消单场。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| sub_meeting_id | string | 否 | 子会议 ID，取消指定周期实例 |
| meeting_type | integer | 否 | 0-普通/单场子会议，1-整个周期会议 |

**出参**

取消结果 JSON（API 原样返回，无字段转换）。

#### meeting_invitees_list

按 `meeting_id` 查询会议受邀者列表。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| pos | integer | 否 | 起始位置（已废弃，用 `page_token`） |
| page_token | string | 否 | 分页 token |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |

**出参**

受邀者列表 JSON（API 原样返回）。

#### meeting_invitees_add

按 `meeting_id` 添加受邀者，传入 open_id 列表。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| invitees | array&lt;string&gt; | 是 | 待添加受邀者 open_id 列表（最多 100） |

**出参**

操作结果 JSON（API 原样返回）。

#### meeting_invitees_remove

按 `meeting_id` 移除受邀者，传入 open_id 列表。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| invitees | array&lt;string&gt; | 是 | 待移除受邀者 open_id 列表（最多 100） |

**出参**

操作结果 JSON（API 原样返回）。

#### meeting_invitees_replace

按 `meeting_id` 替换全部受邀者。`invitees` 留空表示移除全部。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| invitees | array&lt;string&gt; | 否 | 新受邀者 open_id 列表（空表示移除全部，最多 100） |

**出参**

操作结果 JSON（API 原样返回）。

#### meeting_conflict

检测某时间段是否与当前用户已有待开始/进行中会议冲突。仅检查待开始/进行中会议，不含已结束会议。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | string | 是 | 时间段起点（带时区 ISO 8601，如 `2026-08-31T12:30+08:00`） |
| end | string | 是 | 时间段终点（带时区 ISO 8601，须晚于 `start`） |

**出参**

自定义结构（非 API 透传）：

| 字段 | 类型 | 说明 |
|------|------|------|
| meet_conflict | boolean | 是否存在冲突 |
| total_count | integer | 冲突会议数 |
| conflict_meetings | array | 冲突会议列表，每项含已转换的 `start_time`/`end_time` 等，向用户展示 `meeting_code` |

### record — 录制管理（9 个）

| Tool | 说明 |
|------|------|
| `record_list` | 列出录制记录（按时间范围/会议 ID/会议号） |
| `record_search` | 搜索录制记录（按关键字/类型/时间范围） |
| `record_address` | 获取录制文件下载地址 |
| `record_smart_minutes` | 获取 AI 智能纪要 |
| `record_transcript_get` | 获取转写详情 |
| `record_transcript_paragraphs` | 获取转写段落列表 |
| `record_transcript_search` | 搜索转写内容 |
| `record_permission_apply_prepare` | 预览录制权限申请（两步流第一步） |
| `record_permission_apply_commit` | 提交录制权限申请（两步流第二步，用户确认后调用） |

#### record_list

列出录制记录。以下三组条件至少满足其一：`start`+`end`（时间范围）、`meeting_id`、`meeting_code`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | string | 否 | 查询开始时间（ISO 8601） |
| end | string | 否 | 查询结束时间（ISO 8601） |
| meeting_id | string | 否 | 按会议 ID 过滤 |
| meeting_code | string | 否 | 按会议号过滤 |
| page | integer | 否 | 页码（已废弃，用 `page_token`） |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |
| page_token | string | 否 | 分页 token |

**出参**

录制列表 JSON，含 `media_start_time`/`record_start_time`/`record_end_time`（已转 ISO 日期）、`state`（状态码→中文）、`record_type`（已转换）、`meeting_record_id` 等；`total_count` 为 0 时剥离。

#### record_search

按关键字、会议号、会议 ID 或时间范围搜索录制。`query_field` 取 `subject`/`creator`/`transcript_content`/`smart_minutes`/`timeline`/`all`；`file_type` 取 `video`/`audio`/`transcript`/`upload`/`external`/`all`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | 否 | 搜索关键字（最长 50 字符） |
| query_field | string | 否 | 搜索字段（默认 `all`） |
| file_type | string | 否 | 文件类型：`video`/`audio`/`transcript`/`upload`/`external`/`all`（默认 `all`） |
| meeting_id | string | 否 | 按会议 ID 过滤 |
| meeting_code | string | 否 | 按会议号过滤 |
| start | string | 否 | 时间窗口下界（ISO 8601） |
| end | string | 否 | 时间窗口上界（ISO 8601） |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |
| page_token | string | 否 | 分页 token |

**出参**

搜索结果 JSON（API 原样返回）；`total_count` 为 0 时剥离。

#### record_address

按 `meeting_record_id` 获取录制文件下载地址。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_record_id | string | 是 | 会议录制 ID |
| page | integer | 否 | 页码（已废弃，用 `page_token`） |
| page_size | integer | 否 | 分页大小，最大 30，默认 30 |
| page_token | string | 否 | 分页 token |

**出参**

下载地址列表 JSON（API 原样返回）。

#### record_smart_minutes

按 `record_file_id` 获取 AI 智能纪要（摘要 + 待办）。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| record_file_id | string | 是 | 录制文件 ID |
| lang | string | 否 | 语言：`default`/`zh`/`en`/`ja`（默认 `default`） |
| pwd | string | 否 | 录制文件访问密码 |

**出参**

智能纪要 JSON（API 原样返回）。

#### record_transcript_get

按 `record_file_id` 获取转写详情，支持 `pid`+`limit` 分页。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| record_file_id | string | 是 | 录制文件 ID |
| meeting_id | string | 否 | 会议 ID |
| pid | string | 否 | 起始段落数 ID |
| limit | string | 否 | 查询段落数 |

**出参**

转写详情 JSON，`start_time`/`end_time` 已转为 `HH:MM:SS`。

#### record_transcript_paragraphs

按 `record_file_id` 获取转写段落数据。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| record_file_id | string | 是 | 录制文件 ID |
| meeting_id | string | 否 | 会议 ID |

**出参**

段落数组 JSON，`start_time`/`end_time` 已转为 `HH:MM:SS`，`audio_detect` 已转换。

#### record_transcript_search

按 `record_file_id` 和 `text` 在转写内容中搜索。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| record_file_id | string | 是 | 录制文件 ID |
| meeting_id | string | 否 | 会议 ID |
| text | string | 是 | 搜索文本 |

**出参**

搜索结果 JSON，`start_time` 已转为 `HH:MM:SS`。

#### record_permission_apply_prepare

预览录制权限申请（两步流第一步）。先调用本 tool 预览，用户确认后再调 `record_permission_apply_commit`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_record_id | string | 是 | 会议录制 ID |
| meeting_id | string | 否 | 会议 ID |

**出参**

预览信息 JSON（API 原样返回）。

#### record_permission_apply_commit

提交录制权限申请（两步流第二步，用户确认后调用）。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_record_id | string | 是 | 会议录制 ID |
| meeting_id | string | 否 | 会议 ID |

**出参**

提交结果 JSON（API 原样返回）。

### report — 会议报告（2 个）

| Tool | 说明 |
|------|------|
| `report_participants` | 查询参会人列表 |
| `report_waiting_room_log` | 查询等候室成员记录 |

#### report_participants

按 `meeting_id` 查询参会人列表，支持时间范围过滤与分页。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| sub_meeting_id | string | 否 | 周期会议的子会议 ID |
| pos | integer | 否 | 起始位置（已废弃，用 `page_token`） |
| size | integer | 否 | 分页大小（已废弃，用 `page_size`） |
| start | string | 否 | 查询开始时间（ISO 8601） |
| end | string | 否 | 查询结束时间（ISO 8601） |
| page_token | string | 否 | 分页 token |
| page_size | integer | 否 | 分页大小，最大 100，默认 100 |

**出参**

参会人列表 JSON，含 `join_time`/`left_time`/`schedule_start_time`/`schedule_end_time`（已转 ISO 日期）、`instanceid`、`user_name`/`subject`（已 base64 解码）、`user_role` 等；`filter_field` meta 字段已剥离。

#### report_waiting_room_log

按 `meeting_id` 查询等候室成员记录。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| page_size | integer | 否 | 分页大小，最大 100，默认 100 |
| page | integer | 否 | 页码（已废弃，用 `page_token`） |
| page_token | string | 否 | 分页 token |

**出参**

等候室记录 JSON，含 `join_time`/`left_time`/`schedule_start_time`/`schedule_end_time`（已转 ISO 日期）、`instanceid`、`user_name`/`subject`（已 base64 解码）等。

### contact — 通讯录（3 个）

| Tool | 说明 |
|------|------|
| `contact_search` | 按用户名/职位/部门搜索通讯录成员 |
| `contact_lookup_by_email` | 按邮箱批量查询用户（最多 50） |
| `contact_lookup_by_phone` | 按手机号批量查询用户（最多 50） |

> **严格限定**：contact 搜索仅作为会议邀请/呼叫入会的前置步骤，不得单独用于查询个人信息。

#### contact_search

按用户名搜索通讯录成员，可选按职位/部门过滤。结果仅 1 个用户时只保留 `open_id` 字段。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名关键字 |
| job_title | string | 否 | 职位过滤 |
| department_name | string | 否 | 部门名过滤 |

**出参**

用户列表 JSON；当结果仅含 1 个用户时，只保留 `open_id` 字段。

#### contact_lookup_by_email

按邮箱批量查询用户（最多 50）。仅作为会议邀请/呼叫入会前置步骤。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| emails | array&lt;string&gt; | 是 | 邮箱列表（最多 50） |

**出参**

用户列表 JSON（API 原样返回）。

#### contact_lookup_by_phone

按手机号批量查询用户（最多 50）。仅作为会议邀请/呼叫入会前置步骤。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phones | array&lt;string&gt; | 是 | 手机号列表（最多 50） |

**出参**

用户列表 JSON（API 原样返回）。

### control — 会中控制（2 个）

| Tool | 说明 |
|------|------|
| `control_call` | 呼叫成员入会 |
| `control_kick` | 踢出会议成员（支持普通/sip/pstn 用户） |

#### control_call

呼叫成员加入进行中的会议（会中邀请呼叫），传入 open_id 列表（最多 20）。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| users | array&lt;string&gt; | 是 | 待呼叫用户 open_id 列表（最多 20） |

**出参**

呼叫结果 JSON（API 原样返回）。

#### control_kick

按 `meeting_id` 踢出会议成员。`users`/`sip_users`/`pstn_users` 至少传其一，合计最多 20。`allow_rejoin` 默认为 `true`，但 MCP 层未设默认值，需显式传 `true`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| meeting_id | string | 是 | 会议 ID |
| allow_rejoin | boolean | 否 | 是否允许被踢成员重新入会（默认 `true`，MCP 层需显式传值） |
| users | array&lt;string&gt; | 否 | 待踢普通用户 open_id 列表（非 sip/pstn） |
| sip_users | array&lt;string&gt; | 否 | 待踢 sip 用户 ms_open_id 列表 |
| pstn_users | array&lt;string&gt; | 否 | 待踢 pstn 用户 ms_open_id 列表 |

**出参**

踢出结果 JSON（API 原样返回）。

### tshoot — 问题排查（1 个）

| Tool | 说明 |
|------|------|
| `tshoot_feedback` | 向平台反馈工具缺失/失败/能力不足等问题 |

#### tshoot_feedback

向平台反馈工具缺失/失败/能力不足等问题。`category` 取 `tool_not_found`/`tool_error`/`tool_inadequate`/`unexpected_result`/`suggestion`。

**入参**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 是 | 反馈类别：`tool_not_found`/`tool_error`/`tool_inadequate`/`unexpected_result`/`suggestion` |
| intent | string | 是 | 原始意图（最长 200 字符） |
| actions_tried | string | 否 | 已尝试的操作（最长 500 字符） |
| result | string | 否 | 结果或阻塞点（最长 500 字符） |
| tool_name | string | 否 | 使用过的 tool/命令名 |
| error_code | string | 否 | tool 返回的错误码 |

**出参**

反馈提交结果 JSON（API 原样返回）。

> `tshoot log`（导出本地日志）涉及文件系统操作，保留 CLI-only，未暴露为 MCP tool。

---

## OAuth 登录流程

### 方式一：终端预登录（推荐）

在部署 tmeet 的机器上执行一次 `tmeet auth login --name <别名>`，凭证加密保存。后续 MCP server 直接复用。多账号可重复执行，用 `tmeet auth use <别名>` 切换活跃账号。

### 方式二：双 tool 登录

模型通过两个 MCP tool 引导用户完成 OAuth：

1. 调用 `auth_login_url` 传入 `name`（必填）→ 获取 `{authorize_url, code}`
2. 引导用户在浏览器中打开 `authorize_url` 完成扫码授权
3. 调用 `auth_login_poll` 传入 `code` 与相同的 `name` → 返回 `pending`（用户尚未授权，重试）或 `success`（登录成功）

这种双 tool 方案避免了传统 CLI 登录的 5 分钟阻塞轮询，符合 MCP tool 快速返回的规范。

### 多账号切换

登录多个账号后，用 `auth_list` 查看全部账号、`auth_use` 按别名切换活跃账号；切换后后续 tool 均作用于新活跃账号。

---

## 调试

### curl 测试 HTTP 传输

```bash
# 1. initialize，获取 session-id
curl -s -D - http://localhost:8347/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}'
# 响应头中取 Mcp-Session-Id 的值

# 2. 发送 initialized 通知
curl -s http://localhost:8347/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id-from-step-1>" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# 3. 调用 tool
curl -s http://localhost:8347/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id-from-step-1>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"meeting_list","arguments":{}}}'
```

---

## FAQ

### Q: 客户端报 "user config is empty"

MCP server 未找到本地凭证。先在终端执行 `tmeet auth login --name <别名>` 完成 OAuth 登录，再启动 MCP server。

### Q: 客户端报 "Invalid session ID"

Streamable HTTP 传输要求先 `initialize` 获取 `Mcp-Session-Id`，后续请求携带该 ID。检查客户端是否正确处理了 session 握手。

### Q: 远程部署后凭证失效

OAuth refresh_token 有效期约 30 天。过期后需在服务器上重新 `tmeet auth login --name <别名>`。MCP server 内部会自动刷新 access_token（提前 60 秒），无需手动干预。

### Q: 多用户场景如何部署

每个用户/部门需独立实例（不同端口 + 独立凭证目录 + 独立 api-key）。完整步骤见 [多实例部署](#多实例部署) 章节：使用 systemd 模板单元 `hdtmeet-mcp@<name>.service`，一份单元管理 N 个实例，靠 `TMEET_CLI_CONFIG_DIR` + `TMEET_CLI_DATA_DIR` 双 env 隔离凭证。

### Q: tool 返回的数据里时间/状态已是中文/ISO 格式

MCP tool 返回的数据已做字段转换（时间戳→ISO 8601 日期、状态码→中文、base64→明文），与 CLI 输出一致。模型可直接使用，无需自行转换。

### Q: 如何限制 AI 只读不写

MCP 协议本身不提供 tool 级权限控制。如需限制，可在 `internal/mcpserver/tools_meeting.go`、`tools_control_tshoot.go` 等 `tools_*.go` 文件中注释掉对应写 tool 的 `mcpSrv.AddTool(...)` 块（如 `meeting_create`/`meeting_cancel`/`control_kick` 等），重新编译部署。
