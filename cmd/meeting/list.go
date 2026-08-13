package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// ListOptions holds the options for listing meetings.
type ListOptions struct {
	tmeet                *internal.Tmeet
	Pos                  string // Query start position, default 0. ISO 8601 start time for paginated meeting list, e.g. 2026-03-12T15:00+08:00
	Cursory              string // Query end position. ISO 8601 end time for paginated meeting list, e.g. 2026-03-12T15:00+08:00
	IsShowAllSubMeetings int    // Whether to show all sub-meetings. 0: no, 1: yes, default 0
	PageToken            string // Page token for pagination
	PageSize             int    // Page size for pagination, max 20
}

// newListCmd lists meetings.
func newListCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &ListOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list pending/in-progress meetings",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingList)),
			middleWare.WithCompact(tmeet),
		),
	}

	// 填充参数
	cmd.Flags().StringVar(&opts.Pos, "start", "", "query start position (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().StringVar(&opts.Cursory, "end", "", "query end position (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().IntVar(&opts.IsShowAllSubMeetings, "show-all-sub", 0, "show all sub-meetings: 0-no, 1-yes")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 20, "page size for pagination, max 20, default 20")

	return cmd
}

func (o *ListOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.List(cmd.Context(), o.tmeet, meeting.ListParams{
		Start:      o.Pos,
		End:        o.Cursory,
		ShowAllSub: o.IsShowAllSubMeetings,
		PageToken:  o.PageToken,
		PageSize:   o.PageSize,
	})
	if err != nil {
		return err
	}

	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
