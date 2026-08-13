package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// AddressOptions holds the options for getting record download addresses.
type AddressOptions struct {
	tmeet           *internal.Tmeet
	MeetingRecordID string // Meeting record ID
	Page            int    // Page number, starting from 1, deprecated use PageToken instead
	PageSize        int    // Page size
	PageToken       string // Page token
}

// newAddressCmd gets record download addresses.
func newAddressCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &AddressOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "address",
		Short: "get record download address",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordAddress)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingRecordID, "meeting-record-id", "", "meeting record id (required)")
	cmd.Flags().IntVar(&opts.Page, "page", 0, "page number, starting from 1, (deprecated, use --page-token instead)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size, default 30, max 30")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")

	_ = cmd.MarkFlagRequired("meeting-record-id")
	_ = cmd.Flags().MarkDeprecated("page", "use page-token instead")

	return cmd
}

func (o *AddressOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.Address(cmd.Context(), o.tmeet, record.AddressParams{
		MeetingRecordID: o.MeetingRecordID,
		Page:            o.Page,
		PageSize:        o.PageSize,
		PageToken:       o.PageToken,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
