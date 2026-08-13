package result

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
