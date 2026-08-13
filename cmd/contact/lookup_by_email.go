package contact

import (
	"fmt"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	contact "tmeet/internal/contact"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// LookupByEmailOptions holds the options for looking up users by email.
type LookupByEmailOptions struct {
	tmeet  *internal.Tmeet
	Emails []string // Email addresses array
}

// newLookupByEmailCmd creates a command to look up users by email address.
func newLookupByEmailCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &LookupByEmailOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "lookup-by-email",
		Short: "look up users by email address",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdContactLookupByEmail)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringSliceVar(&opts.Emails, "emails", []string{}, fmt.Sprintf(
		"email addresses to look up, comma-separated or repeat the flag, max %d"+
			"\neg. --emails user1@example.com,user2@example.com or "+
			"--emails user1@example.com --emails user2@example.com (required)", contact.LookupByEmailMax))

	_ = cmd.MarkFlagRequired("emails")

	return cmd
}

// Run executes the lookup by email command.
func (o *LookupByEmailOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := contact.LookupByEmail(cmd.Context(), o.tmeet, contact.LookupByEmailParams{
		Emails: o.Emails,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
