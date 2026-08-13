# tmeet contact 命令接口文档

> 基于 `cmd/contact/` 源码整理，共 3 个子命令。
> 全局参数（所有子命令继承）：`--format json|json-pretty`、`--compact`。

---

## 命令总览

| 命令 | 方法 | API URI | ApiCmd | 说明 |
|------|------|---------|--------|------|
| `contact search` | GET | `/v1/contacts/members/search` | `contact_search` | 搜索联系人 |
| `contact lookup-by-phone` | POST | `/v1/contacts/members/lookup-by-phone` | `contact_lookup_by_phone` | 按手机号查询用户 |
| `contact lookup-by-email` | POST | `/v1/contacts/members/lookup-by-email` | `contact_lookup_by_email` | 按邮箱查询用户 |

---

## search — 搜索联系人

- **ApiCmd**：`contact_search`
- **HTTP**：`GET /v1/contacts/members/search`
- **Query 固定字段**：`operator_id`、`operator_id_type=2`
- **支持 `--compact`**，启用 `WithContactSearchLogic`

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--username` | string | ✅ | `""` | 用户名关键词 |
| `--job-title` | string | 否 | `""` | 按职位过滤（用户名匹配过多时缩小范围） |
| `--department-name` | string | 否 | `""` | 按部门名称过滤（用户名匹配过多时缩小范围） |

---

## lookup-by-phone — 按手机号查询用户

- **ApiCmd**：`contact_lookup_by_phone`
- **HTTP**：`POST /v1/contacts/members/lookup-by-phone`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`users_phone`
- **支持 `--compact`**
- **校验**：手机号格式（`utils.ValidatePhone`），最多 50 个

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--phones` | []string | ✅ | `[]` | 手机号列表，逗号分隔或重复传参，最多 50 个 |

---

## lookup-by-email — 按邮箱查询用户

- **ApiCmd**：`contact_lookup_by_email`
- **HTTP**：`POST /v1/contacts/members/lookup-by-email`
- **Body 固定字段**：`operator_id`、`operator_id_type=2`、`users_email`
- **支持 `--compact`**
- **校验**：邮箱格式（`utils.ValidateEmail`），最多 50 个

### Flags

| Flag | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--emails` | []string | ✅ | `[]` | 邮箱列表，逗号分隔或重复传参，最多 50 个 |
