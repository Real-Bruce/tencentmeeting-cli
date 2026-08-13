# tmeet report 命令接口文档

> 基于 `cmd/report/` 源码整理，共 2 个子命令（会议报告）。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `report participants` | GET | `/v1/meetings/{meeting_id}/participants` | `report_participants` | 获取参会成员列表 |
| `report waiting-room-log` | GET | `/v1/meeting/{meeting_id}/waiting-room` | `report_waiting_room_log` | 获取等候室成员列表 |

---

## participants — 获取参会成员列表

- **ApiCmd**：`report_participants`
- **HTTP**：`GET /v1/meetings/{meeting_id}/participants`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`
- **分页兼容**：`ChoosePosOrToken`（旧 `--pos` / `--size` 已弃用）
- **支持 `--compact`**，启用 `WithMetaFieldFilter`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | `""` | 会议 ID |
| `--sub-meeting-id` | string | 否 | `""` | 子会议 ID（周期性会议） |
| `--pos` | int | 否 | `-1` | ⚠️ 已弃用，分页起始位置，请用 `--page-token` |
| `--size` | int | 否 | `0` | ⚠️ 已弃用，每页数量，请用 `--page-size` |
| `--start` | string | 否 | `""` | 查询起始时间（ISO 8601） |
| `--end` | string | 否 | `""` | 查询结束时间（ISO 8601） |
| `--page-token` | string | 否 | `""` | 分页游标 |
| `--page-size` | int | 否 | `100` | 每页数量，最大 100 |

---

## waiting-room-log — 获取等候室成员列表

- **ApiCmd**：`report_waiting_room_log`
- **HTTP**：`GET /v1/meeting/{meeting_id}/waiting-room`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`
- **分页兼容**：`ChoosePageOrToken`（旧 `--page` 已弃用）
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | `""` | 会议 ID |
| `--page-size` | int | 否 | `100` | 每页数量，最大 100 |
| `--page` | int | 否 | `0` | ⚠️ 已弃用，页码，请用 `--page-token` |
| `--page-token` | string | 否 | `""` | 分页游标 |
