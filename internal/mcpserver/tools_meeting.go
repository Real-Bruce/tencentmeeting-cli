package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/meeting"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerMeetingTools 注册 meeting 域全部 MCP tool。
func (s *Server) registerMeetingTools() {
	// --- meeting_list ---
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
			if p.PageSize == 0 {
				p.PageSize = 20
			}
			return meeting.List(ctx, tmeet, p)
		}),
	)

	// --- meeting_get ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_get",
			mcp.WithDescription(`Get meeting details by meeting_id OR meeting_code (one is required).
meeting_id is internal; show meeting_code to users. Returns subject, time range, status, type, recurring rule, etc.`),
			mcp.WithString("meeting_id", mcp.Description("meeting ID (one of meeting_id/meeting_code required)")),
			mcp.WithString("meeting_code", mcp.Description("meeting code (one of meeting_id/meeting_code required)")),
		),
		wrap(s, "meeting_get", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.GetParams) (*result.Result, error) {
			return meeting.Get(ctx, tmeet, p)
		}),
	)

	// --- meeting_list_ended ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_list_ended",
			mcp.WithDescription(`List ended/historical meetings of the authorized user within a time range.
Returns meeting_info_list with meeting_code, subject, time range, meeting_type.`),
			mcp.WithString("start", mcp.Description("query start time (ISO 8601)")),
			mcp.WithString("end", mcp.Description("query end time (ISO 8601)")),
			mcp.WithInteger("page", mcp.Description("page number from 1 (deprecated, use page_token)")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
		),
		wrap(s, "meeting_list_ended", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.ListEndedParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return meeting.ListEnded(ctx, tmeet, p)
		}),
	)

	// --- meeting_search ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_search",
			mcp.WithDescription(`Search meetings by keyword, meeting code, time range, or status.
query_field: subject/creator/note/all. start/end define a time window (ISO 8601) matching scheduled/actual/join time.`),
			mcp.WithString("query", mcp.Description("search keyword (max 50 chars)")),
			mcp.WithString("query_field", mcp.Description("search field: subject/creator/note/all (default all)")),
			mcp.WithString("meeting_code", mcp.Description("filter by meeting code, exact match (digits only)")),
			mcp.WithString("start", mcp.Description("lower bound of time window (ISO 8601)")),
			mcp.WithString("end", mcp.Description("upper bound of time window (ISO 8601)")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
		),
		wrap(s, "meeting_search", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.SearchParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return meeting.Search(ctx, tmeet, p)
		}),
	)

	// --- meeting_create ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_create",
			mcp.WithDescription(`Create a meeting. subject/start/end are required (ISO 8601 times with timezone).
For recurring meetings set meeting_type=1 and provide recurring_type/until_type/until_count[/until_date].
Optional settings (water_mark_type, audio_watermark, auto_record_type, auto_asr) are only applied when provided.`),
			mcp.WithString("subject", mcp.Required(), mcp.Description("meeting subject")),
			mcp.WithString("start", mcp.Required(), mcp.Description("meeting start time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")),
			mcp.WithString("end", mcp.Required(), mcp.Description("meeting end time (ISO 8601, e.g. 2026-03-12T15:00+08:00)")),
			mcp.WithString("password", mcp.Description("meeting password (4-6 digits)")),
			mcp.WithString("timezone", mcp.Description("timezone, e.g. Asia/Shanghai")),
			mcp.WithInteger("meeting_type", mcp.Description("0-normal (default), 1-recurring")),
			mcp.WithInteger("join_type", mcp.Description("join restriction: 1-all, 2-invited only, 3-internal only")),
			mcp.WithBoolean("waiting_room", mcp.Description("enable waiting room")),
			mcp.WithInteger("recurring_type", mcp.Description("recurring type (meeting_type=1): 0-daily, 1-weekdays, 2-weekly, 3-biweekly, 4-monthly")),
			mcp.WithInteger("until_type", mcp.Description("recurring end type: 0-by date, 1-by count")),
			mcp.WithInteger("until_count", mcp.Description("recurring count (max 500 daily/weekly, 50 biweekly/monthly, default 7)")),
			mcp.WithString("until_date", mcp.Description("recurring end date (ISO 8601)")),
			mcp.WithArray("invitees", mcp.Description("invitee open_id list (max 100)")),
			mcp.WithInteger("water_mark_type", mcp.Description("text watermark: 0-single row, 1-double row, 2-off (default). Only applied when provided.")),
			mcp.WithBoolean("audio_watermark", mcp.Description("audio watermark. Only applied when provided.")),
			mcp.WithString("auto_record_type", mcp.Description("auto record: none/local/cloud. Only applied when provided.")),
			mcp.WithBoolean("auto_asr", mcp.Description("auto speech recognition. Only applied when provided.")),
		),
		wrap(s, "meeting_create", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.CreateParams) (*result.Result, error) {
			return meeting.Create(ctx, tmeet, p)
		}),
	)

	// --- meeting_update ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_update",
			mcp.WithDescription(`Update a meeting by meeting_id. Only provided fields are changed.
For recurring meetings: set meeting_type=1 with recurring_rule fields, OR set sub_meeting_id to update a single sub-meeting's time (mutually exclusive).
invitees + invitees_type (replace/add/remove) must be provided together.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithString("subject", mcp.Description("meeting subject")),
			mcp.WithString("start", mcp.Description("meeting start time (ISO 8601)")),
			mcp.WithString("end", mcp.Description("meeting end time (ISO 8601)")),
			mcp.WithString("password", mcp.Description("meeting password (4-6 digits)")),
			mcp.WithString("timezone", mcp.Description("timezone")),
			mcp.WithInteger("meeting_type", mcp.Description("0-normal, 1-recurring. Only applied when provided.")),
			mcp.WithInteger("join_type", mcp.Description("join restriction: 1-all, 2-invited, 3-internal")),
			mcp.WithBoolean("waiting_room", mcp.Description("enable waiting room")),
			mcp.WithInteger("recurring_type", mcp.Description("recurring type (meeting_type=1)")),
			mcp.WithInteger("until_type", mcp.Description("recurring end type")),
			mcp.WithInteger("until_count", mcp.Description("recurring count")),
			mcp.WithString("until_date", mcp.Description("recurring end date (ISO 8601)")),
			mcp.WithString("sub_meeting_id", mcp.Description("sub-meeting ID for single sub-meeting time update (mutually exclusive with recurring fields)")),
			mcp.WithArray("invitees", mcp.Description("invitee open_id list")),
			mcp.WithString("invitees_type", mcp.Description("invitees strategy: replace/add/remove (required when invitees provided)")),
			mcp.WithInteger("water_mark_type", mcp.Description("text watermark: 0-single, 1-double, 2-off. Only applied when provided.")),
			mcp.WithBoolean("audio_watermark", mcp.Description("audio watermark. Only applied when provided.")),
			mcp.WithString("auto_record_type", mcp.Description("auto record: none/local/cloud. Only applied when provided.")),
			mcp.WithBoolean("auto_asr", mcp.Description("auto speech recognition. Only applied when provided.")),
		),
		wrap(s, "meeting_update", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.UpdateParams) (*result.Result, error) {
			return meeting.Update(ctx, tmeet, p)
		}),
	)

	// --- meeting_cancel ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_cancel",
			mcp.WithDescription(`Cancel a meeting by meeting_id. For recurring meetings, set meeting_type=1 to cancel the entire series, or provide sub_meeting_id to cancel one instance.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithString("sub_meeting_id", mcp.Description("sub-meeting ID for canceling a specific recurring instance")),
			mcp.WithInteger("meeting_type", mcp.Description("0-normal/single sub-meeting, 1-entire recurring meeting")),
		),
		wrap(s, "meeting_cancel", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.CancelParams) (*result.Result, error) {
			return meeting.Cancel(ctx, tmeet, p)
		}),
	)

	// --- meeting_invitees_list ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_invitees_list",
			mcp.WithDescription(`List invitees of a meeting by meeting_id.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithInteger("pos", mcp.Description("starting position (deprecated, use page_token)")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
		),
		wrap(s, "meeting_invitees_list", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.InviteesListParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return meeting.InviteesList(ctx, tmeet, p)
		}),
	)

	// --- meeting_invitees_add ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_invitees_add",
			mcp.WithDescription(`Add invitees to a meeting by meeting_id. Provide a list of open_id values.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithArray("invitees", mcp.Required(), mcp.Description("invitee open_id list to add (max 100)")),
		),
		wrap(s, "meeting_invitees_add", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.InviteesAddParams) (*result.Result, error) {
			return meeting.InviteesAdd(ctx, tmeet, p)
		}),
	)

	// --- meeting_invitees_remove ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_invitees_remove",
			mcp.WithDescription(`Remove invitees from a meeting by meeting_id. Provide a list of open_id values.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithArray("invitees", mcp.Required(), mcp.Description("invitee open_id list to remove (max 100)")),
		),
		wrap(s, "meeting_invitees_remove", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.InviteesRemoveParams) (*result.Result, error) {
			return meeting.InviteesRemove(ctx, tmeet, p)
		}),
	)

	// --- meeting_invitees_replace ---
	s.mcpSrv.AddTool(
		mcp.NewTool("meeting_invitees_replace",
			mcp.WithDescription(`Replace all invitees of a meeting with a new list by meeting_id. Leave invitees empty to remove all.`),
			mcp.WithString("meeting_id", mcp.Required(), mcp.Description("meeting ID")),
			mcp.WithArray("invitees", mcp.Description("new invitee open_id list (empty to remove all, max 100)")),
		),
		wrap(s, "meeting_invitees_replace", false, func(ctx context.Context, tmeet *internal.Tmeet, p meeting.InviteesReplaceParams) (*result.Result, error) {
			return meeting.InviteesReplace(ctx, tmeet, p)
		}),
	)
}
