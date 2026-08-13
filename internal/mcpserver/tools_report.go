package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/report"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerReportTools 注册 report 域全部 MCP tool。
func (s *Server) registerReportTools() {
	// --- report_participants ---
	s.mcpSrv.AddTool(
		mcp.NewTool("report_participants",
			mcp.WithDescription(`Get meeting participants by meeting_id. Supports time range filtering and pagination.
Returns participant list with join/left time, user_name, user_role, instanceid, etc.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithString("sub_meeting_id", mcp.Description("sub-meeting ID for recurring meetings")),
			mcp.WithInteger("pos", mcp.Description("start position (deprecated, use page_token)")),
			mcp.WithInteger("size", mcp.Description("page size (deprecated, use page_size)")),
			mcp.WithString("start", mcp.Description("query start time (ISO 8601)")),
			mcp.WithString("end", mcp.Description("query end time (ISO 8601)")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 100, default 100")),
		),
		wrap(s, "report_participants", false, func(ctx context.Context, tmeet *internal.Tmeet, p report.ParticipantsParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 100
			}
			return report.Participants(ctx, tmeet, p)
		}),
	)

	// --- report_waiting_room_log ---
	s.mcpSrv.AddTool(
		mcp.NewTool("report_waiting_room_log",
			mcp.WithDescription(`Get waiting room members log by meeting_id.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 100, default 100")),
			mcp.WithInteger("page", mcp.Description("page number (deprecated, use page_token)")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
		),
		wrap(s, "report_waiting_room_log", false, func(ctx context.Context, tmeet *internal.Tmeet, p report.WaitingRoomLogParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 100
			}
			return report.WaitingRoomLog(ctx, tmeet, p)
		}),
	)
}
