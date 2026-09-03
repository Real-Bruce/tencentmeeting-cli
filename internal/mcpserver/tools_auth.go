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

// accountEntryResult is the JSON shape returned by auth_list for each account.
type accountEntryResult struct {
	Name       string `json:"name"`
	OpenId     string `json:"open_id"`
	Active     bool   `json:"active"`
	TokenState string `json:"token_state,omitempty"`
}

// registerAuthTools 注册 auth 域 MCP tool。
// 登录拆成双 tool：auth_login_url（取 URL，立即返回）+ auth_login_poll（单次轮询，非阻塞），
// 模型引导用户打开浏览器后重复调用 poll 直至 success。两个 tool 均支持可选 name 参数用于多账号。
func (s *Server) registerAuthTools() {
	// --- auth_login_url ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_login_url",
			mcp.WithDescription(`Get the OAuth authorization URL for Tencent Meeting login.
Returns authorize_url (open in browser) and code (pass to auth_login_poll). If the named account is already logged in, returns already_login=true and no URL.
After getting the URL, ask the user to open it in their browser to authorize, then call auth_login_poll with the code and the same name.`),
			mcp.WithString("name", mcp.Required(), mcp.Description("account alias for this login, e.g. token-qtj; pass the same value to auth_login_poll")),
		),
		s.handleAuthLoginURL,
	)

	// --- auth_login_poll ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_login_poll",
			mcp.WithDescription(`Poll once for OAuth login completion. Pass the code from auth_login_url.
Returns status: "pending" (user hasn't completed authorization yet, retry after a few seconds), "success" (login complete), or error.
On pending, ask the user to complete authorization in browser and retry this tool. Pass the same name used in auth_login_url to label the account.`),
			mcp.WithString("code", mcp.Required(), mcp.Description("authorization code from auth_login_url")),
			mcp.WithString("name", mcp.Required(), mcp.Description("account alias for this login, must match the name passed to auth_login_url")),
		),
		s.handleAuthLoginPoll,
	)

	// --- auth_list ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_list",
			mcp.WithDescription(`List all logged-in accounts with their alias, openId, active marker, and refresh-token state. No login required.`),
		),
		s.handleAuthList,
	)

	// --- auth_use ---
	s.mcpSrv.AddTool(
		mcp.NewTool("auth_use",
			mcp.WithDescription(`Switch the active account to a previously logged-in account by its alias. Subsequent tools operate against the switched account.`),
			mcp.WithString("name", mcp.Required(), mcp.Description("account alias to switch to (see auth_list)")),
		),
		s.handleAuthUse,
	)
}

// handleAuthLoginURL 获取授权 URL。单次调用立即返回，不阻塞。
func (s *Server) handleAuthLoginURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
	s.tmeet.CmdPath = "mcp/auth_login_url"

	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(exception.InvalidArgsError.With("name is required: %v", err)), nil
	}

	lockPath := config.GetTokenLockPath()
	var r authLoginURLResult
	if err := filelock.WithLock(lockPath, func() error {
		config.ResetCache()
		// Multi-account: only treat as already-logged-in for the targeted account.
		existing, _ := config.GetAccountConfig(name)
		if existing != nil && existing.RefreshTokenExpires > time.Now().Unix() {
			r = authLoginURLResult{AlreadyLogin: true, Message: "account " + name + " already logged in, no need to authorize again"}
			return nil
		}
		tmeetAuth := auth.NewTmeetAuth(s.tmeet)
		url, code, gerr := tmeetAuth.GeneralOauth2Address(ctx)
		if gerr != nil {
			return gerr
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
	name, nerr := req.RequireString("name")
	if nerr != nil {
		return errorResult(exception.InvalidArgsError.With("name is required: %v", nerr)), nil
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
		Name:         name,
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

	accountName := name
	out, _ := emitJSON(authLoginPollResult{
		Status:  "success",
		Message: "login successful for account " + accountName + ", other tools are now available",
	})
	return mcp.NewToolResultText(out), nil
}

// handleAuthList 列出全部已登录账号。
func (s *Server) handleAuthList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
	s.tmeet.CmdPath = "mcp/auth_list"

	accounts, err := config.ListAccounts()
	if err != nil {
		return errorResult(err), nil
	}
	activeName := config.GetActiveAccountName()
	now := time.Now().Unix()

	out := make([]accountEntryResult, 0, len(accounts))
	for _, a := range accounts {
		entry := accountEntryResult{
			Name:   a.Name,
			OpenId: a.OpenId,
			Active: a.Name == activeName,
		}
		if acct, _ := config.GetAccountConfig(a.Name); acct != nil && acct.RefreshTokenExpires > 0 {
			if now >= acct.RefreshTokenExpires {
				entry.TokenState = "refresh-token expired"
			} else {
				remaining := time.Duration(acct.RefreshTokenExpires-now) * time.Second
				entry.TokenState = "refresh-token valid (remaining " + remaining.String() + ")"
			}
		}
		out = append(out, entry)
	}

	data, _ := emitJSON(out)
	return mcp.NewToolResultText(data), nil
}

// handleAuthUse 切换活跃账号。
func (s *Server) handleAuthUse(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
	s.tmeet.CmdPath = "mcp/auth_use"

	name, err := req.RequireString("name")
	if err != nil {
		return errorResult(exception.InvalidArgsError.With("name is required: %v", err)), nil
	}

	if err := config.SwitchAccount(name); err != nil {
		return errorResult(err), nil
	}
	// 同步到内存中的 tmeet，确保长进程持有新活跃账号凭证。
	// 切换已生效（active 指针已持久化）；若该账号凭证读不出来（master key 轮换后自愈仍
	// 失败或数据损坏），立即返回可操作错误引导重登，而非静默推迟到下一次 tool 调用。
	if cfg, err := config.GetUserConfig(); err != nil {
		return errorResult(exception.GetUserConfigUnknownError.With(
			"switched to %q but its credentials are unreadable: %v; if this persists, re-login via auth_login_url + auth_login_poll to repair",
			name, err)), nil
	} else if cfg != nil {
		s.tmeet.UserConfig = cfg
	}

	out, _ := emitJSON(map[string]string{
		"status":  "success",
		"message": "switched to account " + name,
	})
	return mcp.NewToolResultText(out), nil
}
