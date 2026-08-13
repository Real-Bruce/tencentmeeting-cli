package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/record"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerRecordTools 注册 record 域全部 MCP tool。
func (s *Server) registerRecordTools() {
	// --- record_list ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_list",
			mcp.WithDescription(`List meeting recordings. One of these groups is required:
- start + end (time range, ISO 8601)
- meeting_id
- meeting_code
Returns record list with state, record_type, time range, meeting_record_id.`),
			mcp.WithString("start", mcp.Description("query start time (ISO 8601)")),
			mcp.WithString("end", mcp.Description("query end time (ISO 8601)")),
			mcp.WithInteger("page", mcp.Description("page number (deprecated, use page_token)")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
			mcp.WithString("meeting_id", mcp.Description("filter by meeting ID")),
			mcp.WithString("meeting_code", mcp.Description("filter by meeting code")),
		),
		wrap(s, "record_list", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.ListParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return record.List(ctx, tmeet, p)
		}),
	)

	// --- record_address ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_address",
			mcp.WithDescription(`Get download addresses for a recording by meeting_record_id.`),
			mcp.WithString("meeting_record_id", mcp.Required(), mcp.Description("meeting record ID")),
			mcp.WithInteger("page", mcp.Description("page number (deprecated, use page_token)")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
		),
		wrap(s, "record_address", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.AddressParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return record.Address(ctx, tmeet, p)
		}),
	)

	// --- record_search ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_search",
			mcp.WithDescription(`Search recordings by keyword, meeting code, meeting id or time range.
query_field: subject/creator/transcript_content/smart_minutes/timeline/all.
file_type: video/audio/transcript/upload/external/all.`),
			mcp.WithString("query", mcp.Description("search keyword (max 50 chars)")),
			mcp.WithString("query_field", mcp.Description("search field (default all)")),
			mcp.WithString("file_type", mcp.Description("file type: video/audio/transcript/upload/external/all (default all)")),
			mcp.WithString("meeting_id", mcp.Description("filter by meeting ID")),
			mcp.WithString("meeting_code", mcp.Description("filter by meeting code")),
			mcp.WithString("start", mcp.Description("time window lower bound (ISO 8601)")),
			mcp.WithString("end", mcp.Description("time window upper bound (ISO 8601)")),
			mcp.WithInteger("page_size", mcp.Description("page size, max 30, default 30")),
			mcp.WithString("page_token", mcp.Description("page token for pagination")),
		),
		wrap(s, "record_search", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.SearchParams) (*result.Result, error) {
			if p.PageSize == 0 {
				p.PageSize = 30
			}
			return record.Search(ctx, tmeet, p)
		}),
	)

	// --- record_smart_minutes ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_smart_minutes",
			mcp.WithDescription(`Get AI smart minutes (summary + todos) from a recording file by record_file_id.`),
			mcp.WithString("record_file_id", mcp.Required(), mcp.Description("record file ID")),
			mcp.WithString("lang", mcp.Description("language: default/zh/en/ja (default: default)")),
			mcp.WithString("pwd", mcp.Description("record file access password")),
		),
		wrap(s, "record_smart_minutes", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.SmartMinutesParams) (*result.Result, error) {
			return record.SmartMinutes(ctx, tmeet, p)
		}),
	)

	// --- record_transcript_get ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_transcript_get",
			mcp.WithDescription(`Get transcript details from a recording file by record_file_id. Supports pagination via pid + limit.`),
			mcp.WithString("record_file_id", mcp.Required(), mcp.Description("record file ID")),
			mcp.WithString("meeting_id", mcp.Description("meeting ID")),
			mcp.WithString("pid", mcp.Description("start paragraph ID")),
			mcp.WithString("limit", mcp.Description("number of paragraphs to query")),
		),
		wrap(s, "record_transcript_get", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.TranscriptGetParams) (*result.Result, error) {
			return record.TranscriptGet(ctx, tmeet, p)
		}),
	)

	// --- record_transcript_paragraphs ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_transcript_paragraphs",
			mcp.WithDescription(`Get transcript paragraphs from a recording file by record_file_id.`),
			mcp.WithString("record_file_id", mcp.Required(), mcp.Description("record file ID")),
			mcp.WithString("meeting_id", mcp.Description("meeting ID")),
		),
		wrap(s, "record_transcript_paragraphs", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.TranscriptParagraphsParams) (*result.Result, error) {
			return record.TranscriptParagraphs(ctx, tmeet, p)
		}),
	)

	// --- record_transcript_search ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_transcript_search",
			mcp.WithDescription(`Search transcript content within a recording file by record_file_id and text.`),
			mcp.WithString("record_file_id", mcp.Required(), mcp.Description("record file ID")),
			mcp.WithString("meeting_id", mcp.Description("meeting ID")),
			mcp.WithString("text", mcp.Required(), mcp.Description("search text")),
		),
		wrap(s, "record_transcript_search", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.TranscriptSearchParams) (*result.Result, error) {
			return record.TranscriptSearch(ctx, tmeet, p)
		}),
	)

	// --- record_permission_apply_prepare ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_permission_apply_prepare",
			mcp.WithDescription(`Preview a record permission application before committing (step 1 of 2).
Call this first, review the preview, then call record_permission_apply_commit after user confirmation.`),
			mcp.WithString("meeting_record_id", mcp.Required(), mcp.Description("meeting record ID")),
			mcp.WithString("meeting_id", mcp.Description("meeting ID")),
		),
		wrap(s, "record_permission_apply_prepare", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.PermissionApplyPrepareParams) (*result.Result, error) {
			return record.PermissionApplyPrepare(ctx, tmeet, p)
		}),
	)

	// --- record_permission_apply_commit ---
	s.mcpSrv.AddTool(
		mcp.NewTool("record_permission_apply_commit",
			mcp.WithDescription(`Commit a record permission application after user confirmation (step 2 of 2).
Call record_permission_apply_prepare first to preview, then this tool to submit.`),
			mcp.WithString("meeting_record_id", mcp.Required(), mcp.Description("meeting record ID")),
			mcp.WithString("meeting_id", mcp.Description("meeting ID")),
		),
		wrap(s, "record_permission_apply_commit", false, func(ctx context.Context, tmeet *internal.Tmeet, p record.PermissionApplyCommitParams) (*result.Result, error) {
			return record.PermissionApplyCommit(ctx, tmeet, p)
		}),
	)
}
