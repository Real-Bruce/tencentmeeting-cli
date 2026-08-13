# tmeet control 命令接口文档

> 基于 `cmd/control/` 源码整理，共 2 个子命令（会中控制）。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `control call` | POST | `/v1/meetings/{meeting_id}/batch-call` | `control_call` | 会中邀请呼叫成员 |
| `control kick` | PUT | `/v1/real-control/meetings/{meeting_id}/kickout` | `control_kick` | 会中踢出成员 |

---

## call — 会中邀请呼叫成员

- **ApiCmd**：`control_call`
- **HTTP**：`POST /v1/meetings/{meeting_id}/batch-call`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`users`
- **校验**：经 `PackageMeetingControlUsers` 封装，最多 20 个

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | `""` | 会议 ID |
| `--users` | []string | ✅ | `nil` | 待呼叫的用户 open_id 列表，逗号分隔或重复传参，最多 20 个 |

---

## kick — 会中踢出成员

- **ApiCmd**：`control_kick`
- **HTTP**：`PUT /v1/real-control/meetings/{meeting_id}/kickout`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`instanceid=1`、`allow_rejoin`、`users`
- **校验**：`--users` / `--sip-users` / `--pstn-users` 三选一必填，总计最多 20 个（`MeetingControlUsersListMax`）

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--meeting-id` | string | ✅ | `""` | 会议 ID |
| `--allow-rejoin` | bool | 否 | `true` | 被踢出后是否允许重新入会；显式关闭用 `--allow-rejoin=false` |
| `--users` | []string | 三选一 | `nil` | 待踢出的用户 open_id 列表（不含 sip/pstn 设备），逗号分隔或重复传参 |
| `--sip-users` | []string | 三选一 | `nil` | 待踢出的 sip 用户 ms_open_id 列表 |
| `--pstn-users` | []string | 三选一 | `nil` | 待踢出的 pstn 用户 ms_open_id 列表 |

> `--users` / `--sip-users` / `--pstn-users` 至少提供一个，三者总计最多 20 个。
