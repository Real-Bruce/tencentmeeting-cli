package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// ListOptions holds the options for listing records.
type ListOptions struct {
	tmeet       *internal.Tmeet
	StartTime   string // Query start time, ISO 8601, e.g. 2026-03-12T14:00+08:00 (required)
	EndTime     string // Query end time, ISO 8601, e.g. 2026-03-12T14:00+08:00 (required)
	Page        int    // Page number, starting from 1, deprecated, use PageToken instead
	PageSize    int    // Page size
	PageToken   string // Page token
	MeetingID   string // Meeting ID
	MeetingCode string // Meeting code
}

// newListCmd lists records.
func newListCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &ListOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list records",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordList)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.StartTime, "start", "", "query start time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "query end time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().IntVar(&opts.Page, "page", 0, "page number, starting from 1, (deprecated, use --page-token instead)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size, default 30, max 30")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "",
		"meeting id, one of the following groups is required(--start + --end or --meeting-id or --meeting-code)")
	cmd.Flags().StringVar(&opts.MeetingCode, "meeting-code", "",
		"meeting code, one of the following groups is required(--start + --end or --meeting-id or --meeting-code)")

	_ = cmd.Flags().MarkDeprecated("page", "use --page-token instead")
	return cmd
}

func (o *ListOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.List(cmd.Context(), o.tmeet, record.ListParams{
		StartTime:   o.StartTime,
		EndTime:     o.EndTime,
		Page:        o.Page,
		PageSize:    o.PageSize,
		PageToken:   o.PageToken,
		MeetingID:   o.MeetingID,
		MeetingCode: o.MeetingCode,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
