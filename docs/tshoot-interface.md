# tmeet tshoot 命令接口文档

> 基于 `cmd/tshoot/` 源码整理，共 2 个子命令（故障排查）。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `tshoot log` | GET + PUT + POST | `/v1/cli/tshoot/log/*` | `tshoot_log_upload` | 导出/上传本地日志 |
| `tshoot feedback` | POST | `/v1/api/feedback` | `tshoot_feedback` | 上报故障反馈 |

---

## log — 导出/上传本地日志

- **默认行为**：导出本地日志为 zip 文件，输出路径
- **上传模式**（`--upload`）：经三步上传到服务器，返回 log id
  1. `GET /v1/cli/tshoot/log/upload-token`（获取上传凭证，ApiCmd `tshoot_log_upload`）
  2. `PUT <upload_url>`（直传 COS，含 `Authorization`/`Content-MD5` 头）
  3. `POST /v1/cli/tshoot/log/upload-complete`（通知上传完成）
- **跳过 preCheck**：未加 `--upload` 时不要求登录

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--start` | string | 与 `--end` 同时 | `""` | 查询起始时间（ISO 8601） |
| `--end` | string | 与 `--start` 同时 | `""` | 查询结束时间（ISO 8601），需晚于 `--start` |
| `--upload` | bool | 否 | `false` | 上传日志到服务器（需登录） |

> `--start` 与 `--end` 必须同时提供或同时省略；省略时导出全部日志。

---

## feedback — 上报故障反馈

- **ApiCmd**：`tshoot_feedback`
- **HTTP**：`POST /v1/api/feedback`
- **Body 固定字段**：`operator_id`、`operator_id_type="2"`、`category`、`intent`、`actions_tried`、`result`、`tool_name`、`error_code`、`from_source="CLI"`
- **校验**：`--intent` ≤200 字符，`--actions-tried` / `--result` ≤500 字符

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--category` | string | ✅ | `""` | 反馈类别：`tool_not_found` / `tool_error` / `tool_inadequate` / `unexpected_result` / `suggestion` |
| `--intent` | string | ✅ | `""` | 原始意图（≤200 字符） |
| `--actions-tried` | string | 否 | `""` | 已尝试的操作（≤500 字符） |
| `--result` | string | 否 | `""` | 结果或阻塞点（≤500 字符） |
| `--tool-name` | string | 否 | `""` | 使用的工具/命令名 |
| `--error-code` | string | 否 | `""` | 工具返回的错误码 |
