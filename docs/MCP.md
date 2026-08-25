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

MCP server 启动时读取本地已保存的 OAuth 凭证，**必须先在终端完成一次登录**：

```bash
tmeet auth login
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
| `--transport` | `streamable-http` | 传输模式，固定 streamable-http |
| `--addr` | `0.0.0.0:8347` | 监听地址。本地测试用 `127.0.0.1:8347`，远程部署用 `0.0.0.0:8347` |
| `--api-key` | — | Bearer Token 鉴权密钥。生产环境必填 |
| `--tls-cert` / `--tls-key` | — | 直接启用 TLS（无需 Nginx） |
| `--base-url` | — | 反向代理场景下告知客户端外部可访问的 URL |

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

```bash
# 在服务器终端执行，浏览器扫码授权
/opt/tmeet/tmeet auth login
```

> 服务器没有浏览器时，使用 `--no-browser` 获取授权 URL，在本地浏览器打开：
> ```bash
> /opt/tmeet/tmeet auth login --no-browser
> ```

### 4. 配置 systemd 守护进程

创建 `/etc/systemd/system/hdtmeet-mcp.service`：

```ini
[Unit]
Description=hdtmeet-mcp
After=network.target

[Service]
Type=simple
ExecStart=/opt/tmeet/tmeet mcp \
  --transport streamable-http \
  --addr 0.0.0.0:8347 \
  --api-key your-secret-key
WorkingDirectory=/opt/tmeet
Environment=TMEET_CLI_CONFIG_DIR=/opt/tmeet/.tmeet
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
  /opt/hdtmeet/tmeet auth login --no-browser
```

> 换实例名重复每个。`--no-browser` 输出授权 URL，发给对应用户在本地浏览器扫码。refresh_token 约 30 天到期后用同命令重登该实例。

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

共 32 个 tool，覆盖腾讯会议全部能力。

### auth — OAuth 登录与多账号（4 个）

| Tool | 说明 |
|------|------|
| `auth_login_url` | 获取授权 URL，返回 `{authorize_url, code}`。用户在浏览器打开授权。支持可选 `name` 参数为账号指定别名（多账号登录） |
| `auth_login_poll` | 单次轮询登录结果。传入 `code`，返回 `pending`/`success`/`error`。支持可选 `name` 参数，需与 `auth_login_url` 一致 |
| `auth_list` | 列出全部已登录账号（别名/openId/是否活跃/refresh token 状态）。无需登录 |
| `auth_use` | 按别名切换活跃账号，传入 `name`。切换后后续 tool 均作用于新活跃账号 |

> 多账号：`auth_login_url` 与 `auth_login_poll` 都接受可选 `name`，需保持一致；不传时别名默认为服务端返回的 OpenId。
> 远程部署场景下，可在服务器上先 `tmeet auth login --name <别名>` 完成登录，无需使用登录类 tool。它们主要用于模型引导用户完成登录。

### meeting — 会议管理（11 个）

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

### report — 会议报告（2 个）

| Tool | 说明 |
|------|------|
| `report_participants` | 查询参会人列表 |
| `report_waiting_room_log` | 查询等候室成员记录 |

### contact — 通讯录（3 个）

| Tool | 说明 |
|------|------|
| `contact_search` | 按用户名/职位/部门搜索通讯录成员 |
| `contact_lookup_by_email` | 按邮箱批量查询用户（最多 50） |
| `contact_lookup_by_phone` | 按手机号批量查询用户（最多 50） |

> **严格限定**：contact 搜索仅作为会议邀请/呼叫入会的前置步骤，不得单独用于查询个人信息。

### control — 会中控制（2 个）

| Tool | 说明 |
|------|------|
| `control_call` | 呼叫成员入会 |
| `control_kick` | 踢出会议成员（支持普通/sip/pstn 用户） |

### tshoot — 问题排查（1 个）

| Tool | 说明 |
|------|------|
| `tshoot_feedback` | 向平台反馈工具缺失/失败/能力不足等问题 |

> `tshoot log`（导出本地日志）涉及文件系统操作，保留 CLI-only，未暴露为 MCP tool。

---

## OAuth 登录流程

### 方式一：终端预登录（推荐）

在部署 tmeet 的机器上执行一次 `tmeet auth login --name <别名>`，凭证加密保存。后续 MCP server 直接复用。多账号可重复执行，用 `tmeet auth use <别名>` 切换活跃账号。

### 方式二：双 tool 登录

模型通过两个 MCP tool 引导用户完成 OAuth：

1. 调用 `auth_login_url`（可选传 `name`）→ 获取 `{authorize_url, code}`
2. 引导用户在浏览器中打开 `authorize_url` 完成扫码授权
3. 调用 `auth_login_poll` 传入 `code`（传相同 `name`）→ 返回 `pending`（用户尚未授权，重试）或 `success`（登录成功）

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

MCP server 未找到本地凭证。先在终端执行 `tmeet auth login` 完成 OAuth 登录，再启动 MCP server。

### Q: 客户端报 "Invalid session ID"

Streamable HTTP 传输要求先 `initialize` 获取 `Mcp-Session-Id`，后续请求携带该 ID。检查客户端是否正确处理了 session 握手。

### Q: 远程部署后凭证失效

OAuth refresh_token 有效期约 30 天。过期后需在服务器上重新 `tmeet auth login`。MCP server 内部会自动刷新 access_token（提前 60 秒），无需手动干预。

### Q: 多用户场景如何部署

每个用户/部门需独立实例（不同端口 + 独立凭证目录 + 独立 api-key）。完整步骤见 [多实例部署](#多实例部署) 章节：使用 systemd 模板单元 `hdtmeet-mcp@<name>.service`，一份单元管理 N 个实例，靠 `TMEET_CLI_CONFIG_DIR` + `TMEET_CLI_DATA_DIR` 双 env 隔离凭证。

### Q: tool 返回的数据里时间/状态已是中文/ISO 格式

MCP tool 返回的数据已做字段转换（时间戳→ISO 8601 日期、状态码→中文、base64→明文），与 CLI 输出一致。模型可直接使用，无需自行转换。

### Q: 如何限制 AI 只读不写

MCP 协议本身不提供 tool 级权限控制。如需限制，可在 `internal/mcpserver/server.go` 的 `registerTools()` 中注释掉写操作 tool（如 `meeting_create`/`meeting_cancel`/`control_kick` 等），重新编译部署。
