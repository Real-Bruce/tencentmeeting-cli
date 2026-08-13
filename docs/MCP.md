# tmeet MCP Server

将腾讯会议的全部能力以 [MCP（Model Context Protocol）](https://modelcontextprotocol.io/) tool 形式暴露给 AI Agent。

支持三种传输模式：**stdio**（本地同机）、**streamable-http**（远程部署，推荐）、**sse**（旧版兼容）。

---

## 目录

- [快速开始](#快速开始)
- [传输模式](#传输模式)
  - [stdio（本地）](#stdio本地)
  - [streamable-http（远程部署）](#streamable-http远程部署)
  - [sse（旧版兼容）](#sse旧版兼容)
- [Linux 服务器部署](#linux-服务器部署)
- [AI 客户端接入](#ai-客户端接入)
- [鉴权](#鉴权)
- [TLS / HTTPS](#tls--https)
- [可用 Tool 列表](#可用-tool-列表)
- [OAuth 登录流程](#oauth-登录流程)
- [调试](#调试)
- [FAQ](#faq)

---

## 快速开始

### 前提：完成 OAuth 登录

MCP server 启动时读取本地已保存的 OAuth 凭证，**必须先在终端完成一次登录**：

```bash
tmeet auth login
```

浏览器扫码授权后凭证加密保存到本地，后续 MCP server 直接复用，无需重复登录。

### 启动 MCP server

```bash
# 本地 stdio 模式（Cursor / Claude Desktop 同机调用）
tmeet mcp

# 远程 HTTP 模式（Linux 服务器部署）
tmeet mcp --transport streamable-http --addr 0.0.0.0:8080
```

---

## 传输模式

### stdio（本地）

MCP client（Cursor / Claude Desktop / Cline 等）在本地 spawn `tmeet mcp` 进程，通过 stdin/stdout 通信。

**适用场景**：AI 应用与 tmeet 运行在同一台机器。

**启动方式**：不需要手动启动，AI 客户端配置后会自动 spawn 进程。

**客户端配置示例**：

```json
{
  "mcpServers": {
    "tmeet": {
      "command": "tmeet",
      "args": ["mcp"]
    }
  }
}
```

### streamable-http（远程部署）

MCP 2025-03-26 规范的 Streamable HTTP 传输。tmeet 作为长驻 HTTP 服务运行，AI 客户端通过 HTTP URL 远程接入。

**适用场景**：tmeet 部署在 Linux 服务器上，多个 AI 应用远程接入。**生产环境推荐**。

**启动方式**：

```bash
tmeet mcp --transport streamable-http --addr 0.0.0.0:8080
```

**客户端连接地址**：`http://<server-ip>:8080/mcp`

**会话管理**：客户端首次 `initialize` 请求会收到 `Mcp-Session-Id` 响应头，后续请求需在请求头中携带该 ID。

### sse（旧版兼容）

基于 Server-Sent Events 的传输，兼容不支持 Streamable HTTP 的旧版客户端。

**适用场景**：客户端不支持 Streamable HTTP 时的降级方案。

**启动方式**：

```bash
tmeet mcp --transport sse --addr 0.0.0.0:8080 --base-url https://tmeet.example.com
```

**客户端连接地址**：SSE 端点 `http://<server-ip>:8080/sse`

> `--base-url` 用于反向代理场景下告知客户端外部可访问的 URL。

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
# 上传二进制
scp tmeet user@your-server:/opt/tmeet/tmeet
ssh user@your-server
chmod +x /opt/tmeet/tmeet
```

### 3. 完成 OAuth 登录

```bash
# 在服务器终端执行，浏览器扫码授权
/opt/tmeet/tmeet auth login
```

> 如果服务器没有浏览器，使用 `--no-browser` 获取授权 URL，在本地浏览器打开：
> ```bash
> /opt/tmeet/tmeet auth login --no-browser
> ```

### 4. 配置 systemd 守护进程

创建 `/etc/systemd/system/tmeet-mcp.service`：

```ini
[Unit]
Description=tmeet MCP Server
After=network.target

[Service]
Type=simple
ExecStart=/opt/tmeet/tmeet mcp \
  --transport streamable-http \
  --addr 0.0.0.0:8080 \
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
sudo systemctl enable --now tmeet-mcp
sudo systemctl status tmeet-mcp    # 查看运行状态
sudo journalctl -u tmeet-mcp -f    # 查看实时日志
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
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        # SSE/streamable-http 需要长连接
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 300s;
    }
}
```

此时客户端连接地址为 `https://tmeet.example.com/mcp`。

---

## AI 客户端接入

### Cursor

项目级配置（项目根目录 `.cursor/mcp.json`）或全局配置（`~/.cursor/mcp.json`）：

**stdio 模式**：
```json
{
  "mcpServers": {
    "tmeet": {
      "command": "tmeet",
      "args": ["mcp"]
    }
  }
}
```

**远程 HTTP 模式**（Cursor 0.42+）：
```json
{
  "mcpServers": {
    "tmeet": {
      "url": "http://your-server:8080/mcp",
      "headers": {
        "Authorization": "Bearer your-secret-key"
      }
    }
  }
}
```

### Claude Desktop

配置文件路径：

| 系统 | 路径 |
|------|------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |

**stdio 模式**：
```json
{
  "mcpServers": {
    "tmeet": {
      "command": "/usr/local/bin/tmeet",
      "args": ["mcp"]
    }
  }
}
```

> Claude Desktop 目前仅支持 stdio 模式。远程 HTTP 部署请使用 Cursor 或其他支持 HTTP 传输的客户端。

### Cline（VSCode 插件）

在 VSCode settings.json 中配置：

```json
{
  "cline.mcpServers": {
    "tmeet": {
      "command": "tmeet",
      "args": ["mcp"]
    }
  }
}
```

### 通用 stdio 接入

任何支持 MCP stdio 协议的客户端，配置 `command` + `args` 即可：

```
command: tmeet
args:    mcp
```

### 通用 HTTP 接入

任何支持 MCP Streamable HTTP 传输的客户端，配置 URL：

```
URL:    http://<server-ip>:8080/mcp
Header: Authorization: Bearer <api-key>   （如启用鉴权）
```

---

## 鉴权

HTTP 模式（streamable-http / sse）支持基于 Bearer Token 的 API Key 鉴权。

```bash
# 启动时指定 api-key
tmeet mcp --transport streamable-http --addr 0.0.0.0:8080 --api-key my-secret-key
```

启用后，客户端必须在每个请求中携带 `Authorization: Bearer my-secret-key` 头，否则返回 `401 Unauthorized`。

> **生产环境务必设置 api-key**，否则任何能访问该端口的客户端都可使用你的腾讯会议凭证调用全部 tool。

---

## TLS / HTTPS

HTTP 模式支持直接启用 TLS（无需 Nginx）：

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

共 30 个 tool，覆盖腾讯会议全部能力。

### auth — OAuth 登录（2 个）

| Tool | 说明 |
|------|------|
| `auth_login_url` | 获取授权 URL，返回 `{authorize_url, code}`。用户在浏览器打开授权 |
| `auth_login_poll` | 单次轮询登录结果。传入 `code`，返回 `pending`/`success`/`error` |

> 远程部署场景下，可在服务器上先 `tmeet auth login` 完成登录，无需使用这两个 tool。它们主要用于 stdio 本地场景下模型引导用户登录。

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

### 方式一：终端预登录（推荐，适合远程部署）

在部署 tmeet 的机器上执行一次 `tmeet auth login`，凭证加密保存。后续 MCP server 直接复用。

### 方式二：双 tool 登录（适合 stdio 本地场景）

模型通过两个 MCP tool 引导用户完成 OAuth：

1. 调用 `auth_login_url` → 获取 `{authorize_url, code}`
2. 引导用户在浏览器中打开 `authorize_url` 完成扫码授权
3. 调用 `auth_login_poll` 传入 `code` → 返回 `pending`（用户尚未授权，重试）或 `success`（登录成功）

这种双 tool 方案避免了传统 CLI 登录的 5 分钟阻塞轮询，符合 MCP tool 快速返回的规范。

---

## 调试

### VSCode 调试

在 `.vscode/launch.json` 中选择 **"MCP server (interactive stdio)"** 配置：

1. F5 启动，tmeet mcp 在集成终端等待 stdin 输入
2. 打开 `.vscode/mcp-input.txt`，全选复制
3. 在集成终端粘贴 → 每行一条 JSON-RPC 消息逐条处理，命中断点
4. 修改 `mcp-input.txt` 里的 `tools/call` name + arguments 可测不同 tool

### 非调试快速验证

```bash
# 编译 + 管道输入文件验证
# VSCode: Ctrl+Shift+P → Run Task → "MCP server (pipe & run)"
# 或手动执行：
go build -o tmeet . && cat .vscode/mcp-input.txt | ./tmeet mcp
```

### curl 测试 HTTP 传输

```bash
# 1. initialize，获取 session-id
curl -s -D - http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}'
# 响应头中取 Mcp-Session-Id 的值

# 2. 发送 initialized 通知
curl -s http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id-from-step-1>" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# 3. 调用 tool
curl -s http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id-from-step-1>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"meeting_list","arguments":{}}}'
```

---

## FAQ

### Q: stdio 模式报 "user config is empty"

MCP server 未找到本地凭证。先在终端执行 `tmeet auth login` 完成 OAuth 登录，再启动 MCP server。

### Q: HTTP 模式客户端报 "Invalid session ID"

Streamable HTTP 传输要求先 `initialize` 获取 `Mcp-Session-Id`，后续请求携带该 ID。检查客户端是否正确处理了 session 握手。

### Q: 远程部署后凭证失效

OAuth refresh_token 有效期约 30 天。过期后需在服务器上重新 `tmeet auth login`。MCP server 内部会自动刷新 access_token（提前 60 秒），无需手动干预。

### Q: 多用户场景如何部署

当前 MCP server 持有单用户凭证。多用户场景需为每个用户部署独立的 tmeet 实例（不同端口/不同 config-dir）：

```bash
# 用户 A
TMEET_CLI_CONFIG_DIR=/opt/tmeet/userA tmeet mcp --addr 0.0.0.0:8081

# 用户 B
TMEET_CLI_CONFIG_DIR=/opt/tmeet/userB tmeet mcp --addr 0.0.0.0:8082
```

### Q: tool 返回的数据里时间/状态已是中文/ISO 格式

MCP tool 返回的数据已做字段转换（时间戳→ISO 8601 日期、状态码→中文、base64→明文），与 CLI 输出一致。模型可直接使用，无需自行转换。

### Q: 如何限制 AI 只读不写

MCP 协议本身不提供 tool 级权限控制。如需限制，可在 `internal/mcpserver/server.go` 的 `registerTools()` 中注释掉写操作 tool（如 `meeting_create`/`meeting_cancel`/`control_kick` 等），重新编译部署。
