package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// ListEndedOptions holds the options for listing ended meetings.
type ListEndedOptions struct {
	tmeet     *internal.Tmeet
	StartTime string // Query start time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	EndTime   string // Query end time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	Page      int    // Page number, starting from 1, deprecated use PageToken instead
	PageSize  int    // Page size
	PageToken string // Page token for pagination
}

// newListEndedCmd lists ended meetings.
func newListEndedCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &ListEndedOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "list-ended",
		Short: "list ended meetings",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingListEnded)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.StartTime, "start", "", "query start time (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "query end time (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().IntVar(&opts.Page, "page", 0, "page number, starting from 1, (deprecated, use --page-token instead)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size, default 30, max 30")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")

	// mark deprecated
	_ = cmd.Flags().MarkDeprecated("page", "use --page-token instead")
	return cmd
}

func (o *ListEndedOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.ListEnded(cmd.Context(), o.tmeet, meeting.ListEndedParams{
		StartTime: o.StartTime,
		EndTime:   o.EndTime,
		Page:      o.Page,
		PageSize:  o.PageSize,
		PageToken: o.PageToken,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
