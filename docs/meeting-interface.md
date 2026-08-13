# tmeet meeting 命令接口文档

> 基于 `cmd/meeting/` 源码自动整理，共 11 个子命令。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`；根命令另有 `-V`/`--version`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `meeting create` | POST | `/v1/meetings` | `meeting_create` | 创建会议 |
| `meeting update` | PUT | `/v1/meetings/{meetingId}` | `meeting_update` | 更新会议 |
| `meeting cancel` | POST | `/v1/meetings/{meetingId}/cancel` | `meeting_cancel` | 取消会议 |
| `meeting get` | GET | `/v1/meetings/{meetingId}` 或 `/v1/meetings` | `meeting_get_by_id` / `meeting_get_by_code` | 获取会议详情 |
| `meeting list` | GET | `/v1/meetings` | `meeting_list` | 获取待开始/进行中会议列表 |
| `meeting list-ended` | GET | `/v1/history/meetings/{userid}` | `meeting_list_ended` | 获取已结束会议列表 |
| `meeting search` | GET | `/v1/meetings/search-meetings` | `meeting_search` | 搜索会议 |
| `meeting invitees-list` | GET | `/v1/meetings/{meeting_id}/invitees` | `meeting_invite_list` | 获取会议受邀者 |
| `meeting invitees-add` | PUT | `/v1/meetings/{meeting_id}/modify-invitees` | `meeting_invite_add` | 添加受邀成员 |
| `meeting invitees-remove` | PUT | `/v1/meetings/{meeting_id}/modify-invitees` | `meeting_invite_remove` | 移除受邀成员 |
| `meeting invitees-replace` | PUT | `/v1/meetings/{meeting_id}/invitees` | `meeting_invite_replace` | 替换受邀成员列表 |

---

## create — 创建会议

- **ApiCmd**：`meeting_create`
- **HTTP**：`POST /v1/meetings`
- **Body 固定字段**：`type=0`（预约会议）、`instanceid=1`（PC）、`userid`（当前登录用户 OpenId）

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--subject` | string | ✅ | — | 会议主题 |
| `--start` | string | ✅ | — | 开始时间（ISO 8601，含时区） |
| `--end` | string | ✅ | — | 结束时间（ISO 8601，含时区） |
| `--password` | string | 否 | `""` | 会议密码（4~6 位数字） |
| `--timezone` | string | 否 | `""` | 时区，如 `Asia/Shanghai` |
| `--meeting-type` | int | 否 | `0` | `0`-普通，`1`-周期性 |
| `--join-type` | int | 否 | `0` | `1`-所有成员，`2`-仅受邀，`3`-仅企业内部 |
| `--waiting-room` | bool | 否 | `false` | 开启等候室 |
| `--recurring-type` | int | 否 | `0` | 周期重复：`0`-每天，`1`-工作日，`2`-每周，`3`-每两周，`4`-每月 |
| `--until-type` | int | 否 | `0` | `0`-按日期结束，`1`-按次数结束 |
| `--until-count` | int | 否 | `7` | 重复次数（每天/工作日/每周≤500，每两周/每月≤50） |
| `--until-date` | string | 否 | `""` | 结束日期（ISO 8601） |
| `--invitees` | []string | 否 | `nil` | 邀请成员 openid 列表，逗号分隔或重复传参，≤100 |
| `--water-mark-type` | int | 否 | `2` | `0`-单排水印，`1`-双排，`2`-关闭 |
| `--audio-watermark` | bool | 否 | `false` | 音频水印 |
| `--auto-record-type` | string | 否 | `none` | `none`/`local`/`cloud` |
| `--auto-asr` | bool | 否 | `false` | 自动文字转写 |

---

## update — 更新会议

- **ApiCmd**：`meeting_update`
- **HTTP**：`PUT /v1/meetings/{meetingId}`
- **Body 固定字段**：`instanceid=1`、`userid`
- **特殊**：`--invitees` 与 `--invitees-type` 必须同时使用；`--sub-meeting-id` 与周期规则字段互斥

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--subject` | string | 否 | `""` | 会议主题 |
| `--start` | string | 否 | `""` | 开始时间（ISO 8601） |
| `--end` | string | 否 | `""` | 结束时间（ISO 8601） |
| `--password` | string | 否 | `""` | 会议密码 |
| `--timezone` | string | 否 | `""` | 时区 |
| `--meeting-type` | int | 否 | `0` | `0`-普通，`1`-周期性（改周期性会议必须传 `1`） |
| `--join-type` | int | 否 | `0` | 入会限制 |
| `--waiting-room` | bool | 否 | `false` | 开启等候室 |
| `--recurring-type` | int | 否 | `0` | 周期重复类型 |
| `--until-type` | int | 否 | `0` | 结束类型 |
| `--until-count` | int | 否 | `7` | 重复次数 |
| `--until-date` | string | 否 | `""` | 结束日期 |
| `--sub-meeting-id` | string | 否 | `""` | 子会议 ID（仅改该场时间，与周期规则字段互斥） |
| `--invitees` | []string | 与 `--invitees-type` 同时 | `nil` | 待变更的邀请成员 openid 列表 |
| `--invitees-type` | string | 与 `--invitees` 同时 | `""` | `replace`/`add`/`remove` |
| `--water-mark-type` | int | 否 | `2` | 文字水印 |
| `--audio-watermark` | bool | 否 | `false` | 音频水印 |
| `--auto-record-type` | string | 否 | `none` | 自动录制 |
| `--auto-asr` | bool | 否 | `false` | 自动文字转写 |

---

## cancel — 取消会议

- **ApiCmd**：`meeting_cancel`
- **HTTP**：`POST /v1/meetings/{meetingId}/cancel`
- **Body 固定字段**：`reason_code=0`、`instanceid=1`、`userid`、`meetingId`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--sub-meeting-id` | string | 否 | `""` | 子会议 ID（取消周期性会议的某场） |
| `--meeting-type` | int | 否 | `0` | `0`-普通/单场，`1`-整场周期性会议 |

---

## get — 获取会议详情

- **ApiCmd**：按 flag 路由 — `--meeting-id` → `meeting_get_by_id`；`--meeting-code` → `meeting_get_by_code`
- **HTTP**：`GET`
  - 按 ID：`/v1/meetings/{meetingId}`，Query 固定 `instanceid=1`、`operator_id`、`operator_id_type=2`
  - 按会议码：`/v1/meetings`，Query 固定 `userid`、`instanceid=1`、`meeting_code`（经 `FormatMeetingCode` 归一化）

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | 二选一 | `""` | 会议 ID（优先级更高） |
| `--meeting-code` | string | 二选一 | `""` | 会议码 |

---

## list — 获取待开始/进行中会议列表

- **ApiCmd**：`meeting_list`
- **HTTP**：`GET /v1/meetings`
- **Query 固定字段**：`userid`、`instanceid=1`、`page_type=token`、`page_size`
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--start` | string | 否 | `""` | 查询起始时间（ISO 8601），用于 `pos` 及客户端过滤 |
| `--end` | string | 否 | `""` | 查询结束时间（ISO 8601），仅客户端过滤 |
| `--show-all-sub` | int | 否 | `0` | `0`-不展示子会议，`1`-展示 |
| `--page-token` | string | 否 | `""` | 分页游标 |
| `--page-size` | int | 否 | `20` | 每页数量，最大 20 |

---

## list-ended — 获取已结束会议列表

- **ApiCmd**：`meeting_list_ended`
- **HTTP**：`GET /v1/history/meetings/{userid}`
- **分页兼容**：`ChoosePageOrToken`（旧 `--page` 已弃用）
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--start` | string | 否 | `""` | 查询起始时间（ISO 8601） |
| `--end` | string | 否 | `""` | 查询结束时间（ISO 8601） |
| `--page` | int | 否 | `0` | ⚠️ 已弃用，页码（从 1 开始），请用 `--page-token` |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |
| `--page-token` | string | 否 | `""` | 分页游标 |

---

## search — 搜索会议

- **ApiCmd**：`meeting_search`
- **HTTP**：`GET /v1/meetings/search-meetings`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`
- **支持 `--compact`**，并启用 `WithTotalCountLogic`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--query` | string | 否 | `""` | 搜索关键词（≤50 字符） |
| `--query-field` | string | 否 | `all` | `subject`/`creator`/`note`/`all` |
| `--meeting-code` | string | 否 | `""` | 按会议码精确搜索（经 `FormatMeetingCode` 归一化） |
| `--start` | string | 否 | `""` | 时间窗口下界（ISO 8601） |
| `--end` | string | 否 | `""` | 时间窗口上界（ISO 8601），需晚于 `--start` |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |
| `--page-token` | string | 否 | `""` | 分页游标 |

---

## invitees-list — 获取会议受邀者

- **ApiCmd**：`meeting_invite_list`
- **HTTP**：`GET /v1/meetings/{meeting_id}/invitees`
- **分页兼容**：`ChoosePosOrToken`（旧 `--pos` 已弃用）
- **支持 `--compact`**

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--pos` | int | 否 | `-1` | ⚠️ 已弃用，分页起始位置，请用 `--page-token` |
| `--page-token` | string | 否 | `""` | 分页游标 |
| `--page-size` | int | 否 | `30` | 每页数量，最大 30 |

---

## invitees-add — 添加受邀成员

- **ApiCmd**：`meeting_invite_add`
- **HTTP**：`PUT /v1/meetings/{meeting_id}/modify-invitees`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`instanceid=1`、`add_invitees`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--invitees` | []string | ✅ | `nil` | 待添加的 openid 列表，逗号分隔或重复传参，≤100 |

---

## invitees-remove — 移除受邀成员

- **ApiCmd**：`meeting_invite_remove`
- **HTTP**：`PUT /v1/meetings/{meeting_id}/modify-invitees`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`instanceid=1`、`delete_invitees`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--invitees` | []string | ✅ | `nil` | 待移除的 openid 列表，逗号分隔或重复传参，≤100 |

---

## invitees-replace — 替换受邀成员列表

- **ApiCmd**：`meeting_invite_replace`
- **HTTP**：`PUT /v1/meetings/{meeting_id}/invitees`
- **Body 固定字段**：`meeting_id`、`userid`、`instanceid=1`、`invitees`（可为空，空则移除全部）

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | — | 会议 ID |
| `--invitees` | []string | 否 | `nil` | 替换后的完整 openid 列表；留空则移除全部受邀者，≤100 |

---

## 全局参数

| Flag | 作用域 | 类型 | 默认值 | 说明 |
|------|--------|------|--------|------|
| `--format` | 所有子命令 | string | `json` | 输出格式：`json`（紧凑）| `json-pretty`（缩进） |
| `--compact` | 所有子命令 | bool | `false` | 精简输出，仅展示必要字段（字段表来自远端 schema） |
| `-V` / `--version` | 根命令 | bool | `false` | 显示版本号 |
