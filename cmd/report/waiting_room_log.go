package report

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	report "tmeet/internal/report"

	"github.com/spf13/cobra"
)

// WaitingRoomOptions holds the options for waiting room.
type WaitingRoomOptions struct {
	tmeet     *internal.Tmeet
	MeetingId string // Meeting ID
	PageSize  int    // Page size
	Page      int    // Page number, deprecated use PageToken instead
	PageToken string // Page token
}

// newWaitingRoomCmd gets the waiting room members list.
func newWaitingRoomCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &WaitingRoomOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "waiting-room-log",
		Short: "get waiting room members",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdReportWaitingRoomLog)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingId, "meeting-id", "", "meeting id (required)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 100, "page size, default 100, max 100")
	cmd.Flags().IntVar(&opts.Page, "page", 0, "page number, (deprecated, use --page-token instead)")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token")

	_ = cmd.MarkFlagRequired("meeting-id")
	_ = cmd.Flags().MarkDeprecated("page", "use --page-token instead")

	return cmd
}

func (o *WaitingRoomOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := report.WaitingRoomLog(cmd.Context(), o.tmeet, report.WaitingRoomLogParams{
		MeetingID: o.MeetingId,
		PageSize:  o.PageSize,
		Page:      o.Page,
		PageToken: o.PageToken,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
