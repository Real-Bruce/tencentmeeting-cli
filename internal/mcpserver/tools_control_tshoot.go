package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/control"
	"tmeet/internal/result"
	"tmeet/internal/tshoot"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerControlTools 注册 control 域全部 MCP tool（会中控制）。
func (s *Server) registerControlTools() {
	// --- control_call ---
	s.mcpSrv.AddTool(
		mcp.NewTool("control_call",
			mcp.WithDescription(`Call users into an in-progress meeting (in-meeting invite call) by meeting_id.
Provide a list of open_id values (max 20).`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithArray("users", mcp.Required(), mcp.Description("user open_id list to call (max 20)")),
		),
		wrap(s, "control_call", false, func(ctx context.Context, tmeet *internal.Tmeet, p control.CallParams) (*result.Result, error) {
			return control.Call(ctx, tmeet, p)
		}),
	)

	// --- control_kick ---
	s.mcpSrv.AddTool(
		mcp.NewTool("control_kick",
			mcp.WithDescription(`Kick members out of an in-progress meeting by meeting_id.
At least one of users/sip_users/pstn_users is required; total max 20. allow_rejoin defaults to true.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithBoolean("allow_rejoin", mcp.Description("allow kicked members to rejoin (default true)")),
			mcp.WithArray("users", mcp.Description("user open_id list to kick (not sip/pstn)")),
			mcp.WithArray("sip_users", mcp.Description("sip user ms_open_id list to kick")),
			mcp.WithArray("pstn_users", mcp.Description("pstn user ms_open_id list to kick")),
		),
		wrap(s, "control_kick", false, func(ctx context.Context, tmeet *internal.Tmeet, p control.KickParams) (*result.Result, error) {
			// MCP 未提供 allow_rejoin 时默认 false，但 CLI 语义默认 true。
			// MCP 协议无法区分 "false" 和 "未提供"，这里保持模型传入值，
			// 若需要 true 应显式传 true。
			return control.Kick(ctx, tmeet, p)
		}),
	)
}

// registerTshootTools 注册 tshoot 域 MCP tool。
// tshoot/log 涉及本地文件系统操作（读取日志、打包 zip、上传 COS），MCP 化收益低，保留 CLI-only。
func (s *Server) registerTshootTools() {
	// --- tshoot_feedback ---
	s.mcpSrv.AddTool(
		mcp.NewTool("tshoot_feedback",
			mcp.WithDescription(`Report troubleshooting feedback to the platform. Use when encountering tool not found, tool error, tool inadequate, unexpected result, or have a suggestion.
category: tool_not_found/tool_error/tool_inadequate/unexpected_result/suggestion.`),
			mcp.WithString("category", mcp.Required(), mcp.Description("feedback category")),
			mcp.WithString("intent", mcp.Required(), mcp.Description("original intent (max 200 chars)")),
			mcp.WithString("actions_tried", mcp.Description("actions already tried (max 500 chars)")),
			mcp.WithString("result", mcp.Description("result or blocker (max 500 chars)")),
			mcp.WithString("tool_name", mcp.Description("tool/command name used")),
			mcp.WithString("error_code", mcp.Description("error code returned by the tool")),
		),
		wrap(s, "tshoot_feedback", false, func(ctx context.Context, tmeet *internal.Tmeet, p tshoot.FeedbackParams) (*result.Result, error) {
			return tshoot.Feedback(ctx, tmeet, p)
		}),
	)
}
