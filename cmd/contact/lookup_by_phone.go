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

// LookupByPhoneOptions holds the options for looking up users by phone.
type LookupByPhoneOptions struct {
	tmeet  *internal.Tmeet
	Phones []string // Phone numbers array
}

// newLookupByPhoneCmd creates a command to look up users by phone number.
func newLookupByPhoneCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &LookupByPhoneOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "lookup-by-phone",
		Short: "look up users by phone number",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdContactLookupByPhone)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringSliceVar(&opts.Phones, "phones", []string{}, fmt.Sprintf(
		"phone numbers to look up, comma-separated or repeat the flag, max %d"+
			"\neg. --phones 13800138000,13900139000 or "+
			"--phones 13800138000 --phones 13900139000 (required)", contact.LookupByPhoneMax))

	_ = cmd.MarkFlagRequired("phones")

	return cmd
}

// Run executes the lookup by phone command.
func (o *LookupByPhoneOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := contact.LookupByPhone(cmd.Context(), o.tmeet, contact.LookupByPhoneParams{
		Phones: o.Phones,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
