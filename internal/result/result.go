package result

import (
	"encoding/json"
	"strings"
	"tmeet/internal/utils"
)

// Result 是命令业务逻辑层统一返回结构，供 cobra CLI 与 MCP server 共用。
//
// Data 为已经过字段转换（时间戳→日期、状态码→中文等）的 JSON 字符串，
// 由各域的纯函数 Execute 填充；cobra 层据此调用 output.FormatPrint，
// MCP 层据此直接返回 tool result。
type Result struct {
	// Data 已转换的响应 JSON 字符串。
	Data string
	// TraceId 服务端追踪 ID。
	TraceId string
	// Message 服务端返回的提示信息。
	Message string
}

// StripTotalCountIfZero 在 total_count==0 时剥离该字段。
// 服务器返回 0 通常是占位，不能展示给用户。
// 这是 output.WithTotalCountLogic 的纯函数版，供各域 Execute 复用。
func StripTotalCountIfZero(data string) string {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		return data
	}
	if tc, ok := root["total_count"].(float64); ok && tc == 0 {
		return string(utils.DeleteFields([]byte(data), 10, []string{"total_count"}))
	}
	return data
}

// MetaKeyFilterField 是服务端返回的特殊 meta key，其值为逗号分隔的额外要剥离的字段列表。
const MetaKeyFilterField = "filter_field"

// StripMetaFilterField 剥离 filter_field meta key 本身，以及其值列出的额外字段。
// 这是 output.WithMetaFieldFilter(MetaKeyFilterField) 的纯函数版。
func StripMetaFilterField(data string) string {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		return data
	}
	rawVal, hasMeta := root[MetaKeyFilterField]
	if !hasMeta {
		return data
	}
	fields := []string{MetaKeyFilterField}
	if raw, ok := rawVal.(string); ok {
		for _, item := range strings.Split(raw, ",") {
			if s := strings.TrimSpace(item); s != "" {
				fields = append(fields, s)
			}
		}
	}
	return string(utils.DeleteFields([]byte(data), 10, fields))
}

// ApplyContactSearchLogic 在 contact search 结果仅含 1 个用户时，只保留 open_id 字段。
// 这是 output.WithContactSearchLogic 的纯函数版。
func ApplyContactSearchLogic(data string) string {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		return data
	}
	if users, ok := root["users"].([]interface{}); ok && len(users) == 1 {
		return string(utils.KeepFields([]byte(data), 10, []string{"open_id"}))
	}
	return data
}
