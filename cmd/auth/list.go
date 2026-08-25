package auth

import (
	"time"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/config"
	"tmeet/internal/log"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// ListOptions holds the account list options.
type ListOptions struct {
	tmeet *internal.Tmeet
}

// newListCmd is the auth list command.
func newListCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &ListOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all logged-in accounts",
		Long:  "List every registered account (alias, openId, active marker, refresh-token expiry).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd, args)
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)
	return cmd
}

func (o *ListOptions) Run(cmd *cobra.Command, args []string) error {
	accounts, err := config.ListAccounts()
	if err != nil {
		log.Errorf(cmd.Context(), "failed to list accounts: %v", err)
		return err
	}
	if len(accounts) == 0 {
		output.PrintInfof(cmd, "No accounts logged in. Use 'tmeet auth login [--name <alias>]' to authenticate.")
		return nil
	}

	activeName := config.GetActiveAccountName()
	now := time.Now().Unix()

	for _, a := range accounts {
		acct, _ := config.GetAccountConfig(a.Name)
		marker := ""
		if a.Name == activeName {
			marker = " (active)"
		}
		expiry := "unknown"
		if acct != nil && acct.RefreshTokenExpires > 0 {
			if now >= acct.RefreshTokenExpires {
				expiry = "refresh-token expired"
			} else {
				remaining := time.Duration(acct.RefreshTokenExpires-now) * time.Second
				expiry = "refresh-token valid (remaining " + formatDuration(remaining) + ")"
			}
		}
		output.PrintInfof(cmd, "- %s%s", a.Name, marker)
		output.PrintInfof(cmd, "    openId: %s", a.OpenId)
		output.PrintInfof(cmd, "    status:  %s", expiry)
	}
	return nil
}
