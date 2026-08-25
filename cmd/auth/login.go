package auth

import (
	"context"
	"os"
	"strconv"
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

// LoginOptions holds the login options.
type LoginOptions struct {
	tmeet     *internal.Tmeet
	Name      string // account alias, e.g. token-qtj; empty defaults to the server-returned OpenId
	NoBrowser bool   // login without browser, true to disable auto-open browser
}

// newLoginCmd is the login command.
func newLoginCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &LoginOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login to tmeet",
		Long:  "Login to tmeet by setting up configuration and credentials. Use --name to label the account (e.g. token-qtj) when logging in multiple accounts; the alias defaults to the server-returned OpenId when omitted.",
		RunE: func(cmd *cobra.Command, args []string) error {
			lockPath := config.GetTokenLockPath()
			return filelock.WithLock(lockPath, func() error {
				return opts.Run(cmd, args)
			})
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)

	cmd.Flags().StringVar(&opts.Name, "name", "", "account alias for multi-account login (e.g. token-qtj); defaults to OpenId when empty")
	cmd.Flags().BoolVar(&opts.NoBrowser, "no-browser", false, "disable auto-opening the browser, only print the authorization URL")

	return cmd
}

func (o *LoginOptions) Run(cmd *cobra.Command, args []string) error {
	// Multi-account login: only block when a named account is already authenticated.
	// Without --name we cannot know which account is about to log in, so allow proceeding
	// (this lets users add accounts with a bare `tmeet auth login`).
	if o.Name != "" {
		config.ResetCache()
		if acct, _ := config.GetAccountConfig(o.Name); acct != nil &&
			acct.RefreshTokenExpires > time.Now().Unix() {
			return exception.UserHasBeenInitializedError.
				With("account %q is already logged in, use 'tmeet auth use %s' to switch to it", o.Name, o.Name)
		}
	} else {
		// Re-read the active account so the in-memory cache reflects on-disk state.
		config.ResetCache()
	}
	// Get authorization URL.
	tmeetAuth := auth.NewTmeetAuth(o.tmeet)
	authorizationLocation, authCode, err := tmeetAuth.GeneralOauth2Address(cmd.Context())
	if err != nil {
		log.Errorf(cmd.Context(), "get authorization url failed: %v", err)
		return err
	}
	output.PrintInfof(cmd, "")
	output.PrintInfof(cmd, "Please open the following URL in your browser to authorize")
	output.PrintInfof(cmd, "")
	output.PrintInfof(cmd, "authorize url: %s", authorizationLocation)
	output.PrintInfof(cmd, "")
	output.PrintInfof(cmd, "waiting for authorization...")
	output.PrintInfof(cmd, "")

	if !o.NoBrowser {
		// Try to open the default browser automatically
		_ = browser.Open(authorizationLocation)
	}

	// Poll for authorization result.
	authTokenData, err := tmeetAuth.PollingOauth2Result(cmd.Context(), authCode)
	if err != nil {
		log.Errorf(cmd.Context(), "polling authorization result failed: %v", err)
		return err
	}

	userConfig := &config.UserConfig{
		Name:         o.Name,
		SdkId:        authTokenData.SdkId,
		OpenId:       authTokenData.OpenID,
		AccessToken:  authTokenData.AccessToken,
		RefreshToken: authTokenData.RefreshToken,
	}
	userConfig.Expires, _ = strconv.ParseInt(authTokenData.AccessTokenExpireTime, 10, 64)
	userConfig.RefreshTokenExpires, _ = strconv.ParseInt(authTokenData.RefreshTokenExpireTime, 10, 64)
	err = retry.Do(cmd.Context(), func(ctx context.Context) error {
		err = config.SaveUserConfig(userConfig)
		if err != nil {
			log.Errorf(ctx, "save user config failed: %v", err)
			return exception.AuthorizationFailedError
		}
		return nil
	}, retry.DefaultOptions)

	if err != nil {
		return err
	}

	// Record the caller's identity (AI-Agent / LLM model) in plaintext agent.json.
	// This is non-sensitive telemetry, independent of the user credential lifecycle;
	// a failure here is logged but does not fail the login.
	agentCfg := &config.AgentConfig{
		Agent: os.Getenv("TMEET_AGENT"),
		Model: os.Getenv("TMEET_MODEL"),
	}
	if err := config.SaveAgentConfig(agentCfg); err != nil {
		log.Errorf(cmd.Context(), "save agent config failed: %v", err)
	}

	accountName := o.Name
	if accountName == "" {
		accountName = authTokenData.OpenID
	}
	output.PrintInfof(cmd, "Login successful. Start managing your meetings using tmeet.")
	output.PrintInfof(cmd, "  Account: %s (openId: %s)", accountName, authTokenData.OpenID)
	return nil
}
