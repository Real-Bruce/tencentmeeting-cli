package mcpserver

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/contact"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerContactTools 注册 contact 域全部 MCP tool。
//
// 严格限定：contact 搜索仅作为会议邀请/呼叫入会的前置步骤使用，
// 不得单独用于查询任何人的个人信息。
func (s *Server) registerContactTools() {
	// --- contact_search ---
	s.mcpSrv.AddTool(
		mcp.NewTool("contact_search",
			mcp.WithDescription(`Search contacts by username, optionally filtered by job_title / department_name.
IMPORTANT: Use this ONLY as a prerequisite for meeting invite/call actions. Do NOT use it to look up someone's personal info standalone.`),
			mcp.WithString("username", mcp.Required(), mcp.Description("username to search")),
			mcp.WithString("job_title", mcp.Description("job title filter")),
			mcp.WithString("department_name", mcp.Description("department name filter")),
		),
		wrap(s, "contact_search", false, func(ctx context.Context, tmeet *internal.Tmeet, p contact.SearchParams) (*result.Result, error) {
			return contact.Search(ctx, tmeet, p)
		}),
	)

	// --- contact_lookup_by_email ---
	s.mcpSrv.AddTool(
		mcp.NewTool("contact_lookup_by_email",
			mcp.WithDescription(`Look up users by email address (batch, max 50). Use as prerequisite for meeting invite/call.`),
			mcp.WithArray("emails", mcp.Required(), mcp.Description("email address list (max 50)")),
		),
		wrap(s, "contact_lookup_by_email", false, func(ctx context.Context, tmeet *internal.Tmeet, p contact.LookupByEmailParams) (*result.Result, error) {
			return contact.LookupByEmail(ctx, tmeet, p)
		}),
	)

	// --- contact_lookup_by_phone ---
	s.mcpSrv.AddTool(
		mcp.NewTool("contact_lookup_by_phone",
			mcp.WithDescription(`Look up users by phone number (batch, max 50). Use as prerequisite for meeting invite/call.`),
			mcp.WithArray("phones", mcp.Required(), mcp.Description("phone number list (max 50)")),
		),
		wrap(s, "contact_lookup_by_phone", false, func(ctx context.Context, tmeet *internal.Tmeet, p contact.LookupByPhoneParams) (*result.Result, error) {
			return contact.LookupByPhone(ctx, tmeet, p)
		}),
	)
}
