package auth

import (
	"context"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/config"
	"tmeet/internal/exception"
	"tmeet/internal/log"
	"tmeet/internal/output"
	"tmeet/internal/utils/retry"

	"github.com/spf13/cobra"
)

// LogoutOptions holds the logout options.
type LogoutOptions struct {
	tmeet *internal.Tmeet
	Name  string // account alias to log out; required unless --all is set
	All   bool   // when true, log out every registered account
}

// newLogoutCmd is the logout command.
func newLogoutCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &LogoutOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "logout from tmeet",
		Long:  "Logout and clear the local credentials of the account specified by --name, or all accounts with --all.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd, args)
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)

	cmd.Flags().StringVar(&opts.Name, "name", "", "account alias to log out (required unless --all is set)")
	cmd.Flags().BoolVar(&opts.All, "all", false, "log out all registered accounts")

	return cmd
}

func (o *LogoutOptions) Run(cmd *cobra.Command, args []string) error {
	// --all and --name are mutually exclusive; exactly one is required.
	if o.All && o.Name != "" {
		return exception.InvalidArgsError.With("--all and --name are mutually exclusive")
	}
	if !o.All && o.Name == "" {
		return exception.InvalidArgsError.With("either --name or --all is required")
	}

	if o.All {
		err := retry.Do(cmd.Context(), func(ctx context.Context) error {
			return config.ClearAllAccounts()
		}, retry.DefaultOptions)
		if err != nil {
			log.Errorf(cmd.Context(), "Logout all failed: %v", err)
			return exception.LogoutFailedError
		}
		output.PrintInfof(cmd, "All accounts logged out, welcome to use it again.")
		return nil
	}

	err := retry.Do(cmd.Context(), func(ctx context.Context) error {
		return config.ClearAccount(o.Name)
	}, retry.DefaultOptions)
	if err != nil {
		log.Errorf(cmd.Context(), "Logout failed: %v", err)
		return exception.LogoutFailedError
	}
	output.PrintInfof(cmd, "Logout success, welcome to use it again.")
	return nil
}
