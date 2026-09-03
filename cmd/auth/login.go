package auth

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
	"tmeet/internal"
	"tmeet/internal/auth"
	"tmeet/internal/cmdutil"
	"tmeet/internal/config"
	"tmeet/internal/core/browser"
	"tmeet/internal/core/filelock"
	"tmeet/internal/exception"
	"tmeet/internal/log"
	"tmeet/internal/output"
	"tmeet/internal/utils/retry"

	"github.com/spf13/cobra"
)

// accountNamePattern mirrors internal/config.validateAccountName: the alias may
// only contain letters, digits, underscores and hyphens (e.g. token-qtj). Validated
// up front so an invalid name is rejected before any authorization URL is generated.
var accountNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// LoginOptions holds the login options.
type LoginOptions struct {
	tmeet     *internal.Tmeet
	Names     []string // account alias(es), e.g. token-qtj; at least one is required
	NoBrowser bool     // login without browser, true to disable auto-open browser
}

// newLoginCmd is the login command.
func newLoginCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &LoginOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "login [--name alias ...]",
		Short: "login to tmeet",
		Long:  "Login to tmeet by setting up configuration and credentials. --name is required to label each account (e.g. token-qtj) so every login is a distinct, switchable account. Multiple aliases may be passed at once (e.g. --name token-qtj token-qx token-dt) to batch-generate authorization URLs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			lockPath := config.GetTokenLockPath()
			return filelock.WithLock(lockPath, func() error {
				return opts.Run(cmd, args)
			})
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)

	cmd.Flags().StringSliceVar(&opts.Names, "name", nil, "account alias(es) for multi-account login (e.g. token-qtj); required. Accepts space-separated (--name a b c), comma-separated (--name a,b,c), or repeated (--name a --name b)")
	cmd.Flags().BoolVar(&opts.NoBrowser, "no-browser", false, "disable auto-opening the browser, only print the authorization URL(s)")

	return cmd
}

func (o *LoginOptions) Run(cmd *cobra.Command, args []string) error {
	// --name is required; aliases may also come from positional args so that the
	// space-separated form "--name a b c" works (pflag consumes only one value per
	// --name occurrence, leaving the rest as positional args).
	config.ResetCache()
	names := append([]string{}, o.Names...)
	names = append(names, args...)
	if len(names) == 0 {
		return exception.InvalidArgsError.With("--name is required (e.g. tmeet auth login --name token-qtj)")
	}
	// Dedupe + validate; an explicit duplicate is a clear mistake worth erroring on.
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			return exception.InvalidArgsError.With("duplicate account name: %q", name)
		}
		seen[name] = struct{}{}
		if name == "" || len(name) > 64 || !accountNamePattern.MatchString(name) {
			return exception.InvalidArgsError.With("invalid account name (only letters, digits, underscores, hyphens allowed, max 64 chars): %q", name)
		}
	}

	tmeetAuth := auth.NewTmeetAuth(o.tmeet)
	ctx := cmd.Context()

	// Phase 1: skip aliases that are already logged in (refresh token still valid);
	// for the rest, generate each authorize URL + auth code up front so all URLs can
	// be printed together.
	type pending struct {
		name string
		url  string
		code string
	}
	pendings := make([]pending, 0, len(names))
	for _, name := range names {
		if acct, _ := config.GetAccountConfig(name); acct != nil &&
			acct.RefreshTokenExpires > time.Now().Unix() {
			output.PrintInfof(cmd, "[%s] already logged in, use 'tmeet auth use %s' to switch", name, name)
			continue
		}
		authorizationLocation, authCode, err := tmeetAuth.GeneralOauth2Address(ctx)
		if err != nil {
			log.Errorf(ctx, "[%s] get authorization url failed: %v", name, err)
			return err
		}
		pendings = append(pendings, pending{name: name, url: authorizationLocation, code: authCode})
	}
	if len(pendings) == 0 {
		output.PrintInfof(cmd, "All specified accounts are already logged in. Nothing to do.")
		return nil
	}

	// Phase 2: print every authorize URL, labeled with its alias.
	output.PrintInfof(cmd, "")
	output.PrintInfof(cmd, "Please open the following URL(s) in your browser to authorize")
	output.PrintInfof(cmd, "")
	for _, p := range pendings {
		output.PrintInfof(cmd, "[%s] authorize url %s", p.name, p.url)
	}
	output.PrintInfof(cmd, "")
	output.PrintInfof(cmd, "waiting for authorization...")
	output.PrintInfof(cmd, "")

	// Auto-open the browser only for a single account; for multiple accounts just
	// print the URLs to avoid spawning a burst of browser tabs.
	if !o.NoBrowser && len(pendings) == 1 {
		_ = browser.Open(pendings[0].url)
	}

	// Phase 3: poll each auth code concurrently. Each poll is an independent HTTP
	// loop on its own auth code; tmeetAuth holds no mutable state, and the shared
	// context is only used read-only. Concurrent polling keeps the token-lock hold
	// at ~5 minutes regardless of how many accounts are being authorized.
	//
	// 结果通过带缓冲 channel 流式回传：单个账号轮询一完成即由 Phase 4 接收循环
	// 即时保存并打印成功状态，无需等待全部账号完成。closer goroutine 在所有轮询
	// goroutine 结束后关闭 channel，从而让接收循环自然退出。
	type pollResult struct {
		name  string
		token *auth.AuthTokenData
		err   error
	}
	resultCh := make(chan pollResult, len(pendings))
	var wg sync.WaitGroup
	for _, p := range pendings {
		wg.Add(1)
		go func(p pending) {
			defer wg.Done()
			td, err := tmeetAuth.PollingOauth2Result(ctx, p.code)
			resultCh <- pollResult{name: p.name, token: td, err: err}
		}(p)
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Phase 4: save + report each account as its poll completes. The receiver is
	// single-threaded, so config.SaveUserConfig's read-modify-write on config.json
	// stays serialized without an extra mutex. A per-account failure (e.g.
	// AccountNameConflictError) is reported and skipped without aborting the
	// remaining accounts. 每处理完一个账号，若仍有未完成账号，则再次打印
	// "waiting for authorization..." 提示用户继续等待其余账号授权。
	remaining := len(pendings)
	for r := range resultCh {
		if r.err != nil {
			log.Errorf(ctx, "[%s] polling authorization result failed: %v", r.name, r.err)
			output.PrintInfof(cmd, "[%s] login failed: %v", r.name, r.err)
		} else {
			userConfig := &config.UserConfig{
				Name:         r.name,
				SdkId:        r.token.SdkId,
				OpenId:       r.token.OpenID,
				AccessToken:  r.token.AccessToken,
				RefreshToken: r.token.RefreshToken,
			}
			userConfig.Expires, _ = strconv.ParseInt(r.token.AccessTokenExpireTime, 10, 64)
			userConfig.RefreshTokenExpires, _ = strconv.ParseInt(r.token.RefreshTokenExpireTime, 10, 64)
			if err := retry.Do(ctx, func(ctx context.Context) error {
				if err := config.SaveUserConfig(userConfig); err != nil {
					log.Errorf(ctx, "[%s] save user config failed: %v", r.name, err)
					return err
				}
				return nil
			}, retry.DefaultOptions); err != nil {
				output.PrintInfof(cmd, "[%s] login failed: %v", r.name, err)
			} else {
				output.PrintInfof(cmd, "[%s] login successful (openId: %s)", r.name, r.token.OpenID)
			}
		}
		remaining--
		if remaining > 0 {
			output.PrintInfof(cmd, "")
			output.PrintInfof(cmd, "waiting for authorization...")
			output.PrintInfof(cmd, "")
		}
	}

	// Record the caller's identity (AI-Agent / LLM model) in plaintext agent.json.
	// This is non-sensitive telemetry, independent of the user credential lifecycle;
	// a failure here is logged but does not fail the login. Saved once per command.
	agentCfg := &config.AgentConfig{
		Agent: os.Getenv("TMEET_AGENT"),
		Model: os.Getenv("TMEET_MODEL"),
	}
	if err := config.SaveAgentConfig(agentCfg); err != nil {
		log.Errorf(ctx, "save agent config failed: %v", err)
	}

	output.PrintInfof(cmd, "Login finished. Start managing your meetings using tmeet.")
	return nil
}
