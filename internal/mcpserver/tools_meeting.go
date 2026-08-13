package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/meeting"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerMeetingTools 注册 meeting 域全部 MCP tool。
// 当前已实现：meeting_list。其余命令随 cobra 重构逐步补齐。
func (s *Server) registerMeetingTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_list",
			mcp.WithDescription(`List pending/in-progress meetings of the authorized user.

Time parameters use ISO 8601 with timezone, e.g. "2026-03-12T15:00+08:00".
Returns meeting_info_list with meeting_code (show this to users, NOT meeting_id), status, time range, etc.`),
			mcp.WithString("start", mcp.Description("query start time (ISO 8601, e.g. 2026-03-12T15:00+08:00)")),
			mcp.WithString("end", mcp.Description("query end time (ISO 8601, e.g. 2026-03-12T15:00+08:00)")),
			mcp.WithInteger("show_all_sub", mcp.Description("show all sub-meetings: 0-no (default), 1-yes")),
			mcp.WithString("page_token", mcp.Description("page token for pagination, returned from previous response")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 20, default 20")),
		),
		wrap(s, "meeting_list", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.ListParams) (*result.Result, error) {
			// MCP 默认 0 → 用 CLI 默认 20
			if p.PageSize == 0 {
				p.PageSize = 20
			}
			return meeting.List(ctx, tmeet, p)
		}),
	)
}
