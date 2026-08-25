package auth

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/config"
	"tmeet/internal/log"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// UseOptions holds the account switch options.
type UseOptions struct {
	tmeet *internal.Tmeet
}

// newUseCmd is the auth use command.
func newUseCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &UseOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Switch the active account",
		Long:  "Switch the active account to a previously logged-in account by its alias.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd, args)
		},
	}
	cmdutil.InjectSkipPreCheckAnnotation(cmd)
	return cmd
}

func (o *UseOptions) Run(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := config.SwitchAccount(name); err != nil {
		log.Errorf(cmd.Context(), "switch account failed: %v", err)
		return err
	}
	// Sync the in-memory tmeet config so subsequent operations in this process use it.
	if cfg, _ := config.GetUserConfig(); cfg != nil {
		o.tmeet.UserConfig = cfg
	}
	output.PrintInfof(cmd, "Switched to account %q. Use 'tmeet auth status' to verify.", name)
	return nil
}
