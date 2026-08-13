# tmeet record 命令接口文档

> 基于 `cmd/record/` 源码整理，共 9 个子命令（录制管理）。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `record list` | GET | `/v1/mcp/records` | `record_list` | 获取录制列表 |
| `record address` | GET | `/v1/mcp/addresses` | `record_address` | 获取录制下载地址 |
| `record search` | GET | `/v1/mcp/records/search` | `record_search` | 搜索录制 |
| `record smart-minutes` | GET | `/v1/smart/minutes/{record_file_id}` | `record_smart_minutes` | 获取智能纪要 |
| `record transcript-get` | GET | `/v1/records/mcp/transcripts/details` | `record_transcript_get` | 获取转写详情 |
| `record transcript-paragraphs` | GET | `/v1/records/mcp/transcripts/paragraphs` | `record_transcript_paragraphs` | 获取转写段落 |
| `record transcript-search` | GET | `/v1/records/mcp/transcripts/search` | `record_transcript_search` | 搜索转写内容 |
| `record permission-apply-prepare` | GET | `/v1/records/mcp/permission-apply/prepare` | `record_permission_apply_prepare` | 预览录制权限申请 |
| `record permission-apply-commit` | GET | `/v1/records/mcp/permission-apply/commit` | `record_permission_apply_commit` | 提交录制权限申请 |

---

## list — 获取录制列表

- **ApiCmd**：`record_list`
- **HTTP**：`GET /v1/mcp/records`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`
- **分页兼容**：`ChoosePageOrToken`（旧 `--page` 已弃用）
- **支持 `--compact`**，启用 `WithTotalCountLogic`
- **入参要求**：三组参数至少提供一组：`--start + --end` / `--meeting-id` / `--meeting-code`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--start` | string | 三选一 | `""` | 查询起始时间（ISO 8601），需与 `--end` 同时使用 |
| `--end` | string | 三选一 | `""` | 查询结束时间（ISO 8601），需与 `--start` 同时使用 |
| `--meeting-id` | string | 三选一 | `""` | 会议 ID |
| `--meeting-code` | string | 三选一 | `""` | 会议码（经 `FormatMeetingCode` 归一化） |
| `--page` | int | 否 | `0` | ⚠️ 已弃用，页码，请用 `--page-token` |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |
| `--page-token` | string | 否 | `""` | 分页游标 |

> `--start + --end` / `--meeting-id` / `--meeting-code` 三组参数至少提供一组。

---

## address — 获取录制下载地址

- **ApiCmd**：`record_address`
- **HTTP**：`GET /v1/mcp/addresses`
- **Query 固定字段**：`meeting_record_id`、`operator_id`、`operator_id_type=2`
- **分页兼容**：`ChoosePageOrToken`（旧 `--page` 已弃用）
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-record-id` | string | ✅ | `""` | 录制 ID |
| `--page` | int | 否 | `0` | ⚠️ 已弃用，页码，请用 `--page-token` |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |
| `--page-token` | string | 否 | `""` | 分页游标 |

---

## search — 搜索录制

- **ApiCmd**：`record_search`
- **HTTP**：`GET /v1/mcp/records/search`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`file_type`
- **支持 `--compact`**，启用 `WithTotalCountLogic`
- **校验**：`--query` ≤50 字符；`--start` 需早于 `--end`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--query` | string | 否 | `""` | 搜索关键词（≤50 字符） |
| `--query-field` | string | 否 | `all` | `subject`/`creator`/`transcript_content`/`smart_minutes`/`timeline`/`all` |
| `--start` | string | 否 | `""` | 时间窗口下界（ISO 8601） |
| `--end` | string | 否 | `""` | 时间窗口上界（ISO 8601），需晚于 `--start` |
| `--file-type` | string | 否 | `all` | `video`/`audio`/`transcript`/`upload`/`external`/`all` |
| `--meeting-id` | string | 否 | `""` | 按会议 ID 过滤 |
| `--meeting-code` | string | 否 | `""` | 按会议码精确过滤（经 `FormatMeetingCode` 归一化） |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |
| `--page-token` | string | 否 | `""` | 分页游标 |

---

## smart-minutes — 获取智能纪要

- **ApiCmd**：`record_smart_minutes`
- **HTTP**：`GET /v1/smart/minutes/{record_file_id}`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`lang`
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--record-file-id` | string | ✅ | `""` | 录制文件 ID |
| `--lang` | string | 否 | `default` | 翻译语言：`default`（不翻译）/ `zh` / `en` / `ja` |
| `--pwd` | string | 否 | `""` | 录制文件访问密码 |

---

## transcript-get — 获取转写详情

- **ApiCmd**：`record_transcript_get`
- **HTTP**：`GET /v1/records/mcp/transcripts/details`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`record_file_id`
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--record-file-id` | string | ✅ | `""` | 录制文件 ID |
| `--meeting-id` | string | 否 | `""` | 会议 ID |
| `--pid` | string | 否 | `""` | 起始段落 ID |
| `--limit` | string | 否 | `""` | 查询段落数量 |

---

## transcript-paragraphs — 获取转写段落

- **ApiCmd**：`record_transcript_paragraphs`
- **HTTP**：`GET /v1/records/mcp/transcripts/paragraphs`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`record_file_id`
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--record-file-id` | string | ✅ | `""` | 录制文件 ID |
| `--meeting-id` | string | 否 | `""` | 会议 ID |

---

## transcript-search — 搜索转写内容

- **ApiCmd**：`record_transcript_search`
- **HTTP**：`GET /v1/records/mcp/transcripts/search`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`record_file_id`、`text`
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--record-file-id` | string | ✅ | `""` | 录制文件 ID |
| `--meeting-id` | string | 否 | `""` | 会议 ID |
| `--text` | string | ✅ | `""` | 搜索文本 |

---

## permission-apply-prepare — 预览录制权限申请

- **ApiCmd**：`record_permission_apply_prepare`
- **HTTP**：`GET /v1/records/mcp/permission-apply/prepare`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`meeting_record_id`
- **支持 `--compact`**
- **两步流第一步**：预览 → 用户确认 → `permission-apply-commit` 提交

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-record-id` | string | ✅ | `""` | 录制 ID |
| `--meeting-id` | string | 否 | `""` | 会议 ID |

---

## permission-apply-commit — 提交录制权限申请

- **ApiCmd**：`record_permission_apply_commit`
- **HTTP**：`GET /v1/records/mcp/permission-apply/commit`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`、`meeting_record_id`
- **支持 `--compact`**
- **两步流第二步**：用户确认 `prepare` 预览结果后执行提交

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-record-id` | string | ✅ | `""` | 录制 ID |
| `--meeting-id` | string | 否 | `""` | 会议 ID |
