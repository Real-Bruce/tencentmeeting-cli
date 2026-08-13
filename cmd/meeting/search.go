package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// Character length limits for search fields.
const (
	maxQueryLen = 50
)

// SearchOptions is the options for search command.
type SearchOptions struct {
	tmeet       *internal.Tmeet
	Query       string // search keyword
	QueryField  string // search field for Query
	MeetingCode string // meeting code
	From        string // start time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	To          string // end time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	PageSize    int    // Page size
	PageToken   string // Page token for pagination
}

// newSearchCmd searches meetings.
func newSearchCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &SearchOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search meetings by keyword, meeting code, time range or status",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingSearch)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.Query, "query", "", "search keyword")
	cmd.Flags().StringVar(&opts.QueryField, "query-field", "all", `search field for --query (e.g.
		subject: meeting subject;
		creator: creator's nickname/remark name,
		note: user's note on the meeting
		all: search all fields)`)
	cmd.Flags().StringVar(&opts.MeetingCode, "meeting-code", "", "filter by meeting code, exact match (digits only, no dashes)")
	cmd.Flags().StringVar(&opts.From, "start", "", "lower bound of search time window (ISO 8601). Matches if meeting's scheduled start time OR actual start time OR user's join time falls within the window (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().StringVar(&opts.To, "end", "", "upper bound of search time window (ISO 8601). Same semantics as above (ISO 8601, e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size, default 30, max 30")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")

	return cmd
}

// Run executes the search command.
func (o *SearchOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.Search(cmd.Context(), o.tmeet, meeting.SearchParams{
		Query:       o.Query,
		QueryField:  o.QueryField,
		MeetingCode: o.MeetingCode,
		From:        o.From,
		To:          o.To,
		PageSize:    o.PageSize,
		PageToken:   o.PageToken,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
