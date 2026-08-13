package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// Character length limits for search fields.
const (
	maxQueryLen = 50
)

// SearchOptions holds the options for searching records.
type SearchOptions struct {
	tmeet       *internal.Tmeet
	Query       string // search keyword
	QueryField  string // search field for Query
	FileType    string // file type: video / audio / transcript / upload / external / all (default all)
	MeetingID   string // meeting id
	MeetingCode string // meeting code
	From        string // start time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	To          string // end time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	PageSize    int    // page size
	PageToken   string // page token for pagination
}

// newSearchCmd searches records.
func newSearchCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &SearchOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search records by keyword, meeting code, meeting id or time range",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordSearch)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.Query, "query", "", "search keyword")
	cmd.Flags().StringVar(&opts.QueryField, "query-field", "all", `search field for --query (e.g.
		subject: record subject;
		creator: nickname/remark name of the meeting creator;
		transcript_content: original transcript content within the file;
		smart_minutes: smart minutes content within the file (summary + todos);
		timeline: timeline content within the file;
		all: search all fields)`)
	cmd.Flags().StringVar(&opts.From, "start", "", "query start time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.To, "end", "", "query end time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.FileType, "file-type", "all", "file type: video / audio / transcript / upload / external / all (default all)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "filter by meeting id")
	cmd.Flags().StringVar(&opts.MeetingCode, "meeting-code", "", "filter by meeting code, exact match (digits only, no dashes)")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size, default 30, max 30")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")

	return cmd
}

// Run executes the search command.
func (o *SearchOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.Search(cmd.Context(), o.tmeet, record.SearchParams{
		Query:       o.Query,
		QueryField:  o.QueryField,
		FileType:    o.FileType,
		MeetingID:   o.MeetingID,
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
