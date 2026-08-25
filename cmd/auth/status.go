package auth

import (
	"fmt"
	"time"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/config"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// StatusOptions holds the status query options.
type StatusOptions struct {
	tmeet *internal.Tmeet
	Name  string // account alias to query; required
}

// newStatusCmd is the status query command.
func newStatusCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &StatusOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status of an account",
		Long:  "Display the login status and credential information of the account specified by --name.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd, args)
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)

	cmd.Flags().StringVar(&opts.Name, "name", "", "account alias to query (required)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func (o *StatusOptions) Run(cmd *cobra.Command, args []string) error {
	// Load the named account's stored credentials without touching the active account
	// pointer or triggering a token refresh, so a status check never silently switches
	// the active account.
	userConfig, err := config.GetAccountConfig(o.Name)
	if err != nil {
		return err
	}
	if userConfig == nil {
		output.PrintInfof(cmd, "Account %q is not logged in. Use 'tmeet auth login --name %s' to authenticate.", o.Name, o.Name)
		return nil
	}

	activeMarker := ""
	if config.GetActiveAccountName() == o.Name {
		activeMarker = " (active)"
	}

	// Show login status.
	output.PrintInfof(cmd, "Logged in%s", activeMarker)
	output.PrintInfof(cmd, "  Account:  %s", o.Name)
	output.PrintInfof(cmd, "  OpenId:  %s", userConfig.OpenId)
	now := time.Now().Unix()

	// AccessToken expiry status.
	if userConfig.Expires > 0 {
		expiresTime := time.Unix(userConfig.Expires, 0)
		if now >= userConfig.Expires {
			output.PrintInfof(cmd, "  AccessToken:  expired (at %s)", expiresTime.Format(time.DateTime))
		} else {
			remaining := time.Duration(userConfig.Expires-now) * time.Second
			output.PrintInfof(cmd, "  AccessToken:  valid (expires at %s, remaining %s)",
				expiresTime.Format(time.DateTime), formatDuration(remaining))
		}
	}

	// RefreshToken expiry status.
	if userConfig.RefreshTokenExpires > 0 {
		refreshExpiresTime := time.Unix(userConfig.RefreshTokenExpires, 0)
		if now >= userConfig.RefreshTokenExpires {
			output.PrintInfof(cmd, "  RefreshToken: expired (at %s)", refreshExpiresTime.Format(time.DateTime))
		} else {
			remaining := time.Duration(userConfig.RefreshTokenExpires-now) * time.Second
			output.PrintInfof(cmd, "  RefreshToken: valid (expires at %s, remaining %s)",
				refreshExpiresTime.Format(time.DateTime), formatDuration(remaining))
		}
	}

	return nil
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
