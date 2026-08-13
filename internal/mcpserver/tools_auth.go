package mcpserver

import (
	"context"
	"strconv"
	"time"
	"tmeet/internal/auth"
	"tmeet/internal/config"
	"tmeet/internal/core/filelock"
	"tmeet/internal/exception"
	"tmeet/internal/log"

	"github.com/mark3labs/mcp-go/mcp"
)

// authLoginURLResult 是 auth_login_url tool 的返回结构。
type authLoginURLResult struct {
	AuthorizeURL string `json:"authorize_url"`
	Code         string `json:"code"`
	AlreadyLogin bool   `json:"already_login,omitempty"`
	Message      string `json:"message,omitempty"`
}

// authLoginPollResult 是 auth_login_poll tool 的返回结构。
type authLoginPollResult struct {
	Status  string `json:"status"` // "pending" | "success" | "error"
	Message string `json:"message,omitempty"`
}

// registerAuthTools 注册 auth 域 MCP tool。
// 登录拆成双 tool：auth_login_url（取 URL，立即返回）+ auth_login_poll（单次轮询，非阻塞），
// 模型引导用户打开浏览器后重复调用 poll 直至 success。
func (s *Server) registerAuthTools() {
	// --- auth_login_url ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_login_url",
			mcp.WithDescription(`Get the OAuth authorization URL for Tencent Meeting login.
Returns authorize_url (open in browser) and code (pass to auth_login_poll). If already logged in, returns already_login=true and no URL needed.
After getting the URL, ask the user to open it in their browser to authorize, then call auth_login_poll with the code.`),
		),
		s.handleAuthLoginURL,
	)

	// --- auth_login_poll ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_login_poll",
			mcp.WithDescription(`Poll once for OAuth login completion. Pass the code from auth_login_url.
Returns status: "pending" (user hasn't completed authorization yet, retry after a few seconds), "success" (login complete), or error.
On pending, ask the user to complete authorization in browser and retry this tool.`),
			mcp.WithString("code", mcp.Required(), mcp.Description("authorization code from auth_login_url")),
		),
		s.handleAuthLoginPoll,
	)
}

// handleAuthLoginURL 获取授权 URL。单次调用立即返回，不阻塞。
func (s *Server) handleAuthLoginURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
	s.tmeet.CmdPath = "mcp/auth_login_url"

	lockPath := config.GetTokenLockPath()
	var r authLoginURLResult
	if err := filelock.WithLock(lockPath, func() error {
		config.ResetCache()
		userConfig, _ := config.GetUserConfig()
		if userConfig != nil && userConfig.RefreshTokenExpires > time.Now().Unix() {
			r = authLoginURLResult{AlreadyLogin: true, Message: "already logged in, no need to authorize again"}
			return nil
		}
		tmeetAuth := auth.NewTmeetAuth(s.tmeet)
		url, code, err := tmeetAuth.GeneralOauth2Address(ctx)
		if err != nil {
			return err
		}
		r = authLoginURLResult{AuthorizeURL: url, Code: code}
		return nil
	}); err != nil {
		return errorResult(err), nil
	}

	out, _ := emitJSON(r)
	return mcp.NewToolResultText(out), nil
}

// handleAuthLoginPoll 单次轮询授权结果。pending 时不阻塞，返回让模型重试。
func (s *Server) handleAuthLoginPoll(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
	s.tmeet.CmdPath = "mcp/auth_login_poll"

	code, err := req.RequireString("code")
	if err != nil {
		return errorResult(exception.InvalidArgsError.With("code is required: %v", err)), nil
	}

	tmeetAuth := auth.NewTmeetAuth(s.tmeet)
	tokenData, err := tmeetAuth.PollOauth2Once(ctx, code)
	if err != nil {
		// 轮询出错（可能是网络问题或服务端拒绝），返回可重试的错误
		return errorResult(err), nil
	}
	if tokenData == nil || tokenData.OpenID == "" {
		// 用户尚未完成授权，返回 pending 让模型引导重试
		out, _ := emitJSON(authLoginPollResult{
			Status:  "pending",
			Message: "user has not completed authorization yet, ask the user to open the authorize_url in browser and scan the QR code, then retry this tool",
		})
		return mcp.NewToolResultText(out), nil
	}

	// 授权成功，保存凭证
	userConfig := &config.UserConfig{
		SdkId:        tokenData.SdkId,
		OpenId:       tokenData.OpenID,
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
	}
	userConfig.Expires, _ = strconv.ParseInt(tokenData.AccessTokenExpireTime, 10, 64)
	userConfig.RefreshTokenExpires, _ = strconv.ParseInt(tokenData.RefreshTokenExpireTime, 10, 64)

	lockPath := config.GetTokenLockPath()
	if err := filelock.WithLock(lockPath, func() error {
		config.ResetCache()
		return config.SaveUserConfig(userConfig)
	}); err != nil {
		return errorResult(exception.AuthorizationFailedError.With("save user config failed: %v", err)), nil
	}

	// 同步到内存中的 tmeet，后续 tool 即可用新凭证
	s.tmeet.UserConfig = userConfig

	out, _ := emitJSON(authLoginPollResult{
		Status:  "success",
		Message: "login successful, other tools are now available",
	})
	return mcp.NewToolResultText(out), nil
}
