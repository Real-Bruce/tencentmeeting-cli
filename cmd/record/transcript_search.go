package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// TranscriptsSearchOptions holds the options for transcript search.
type TranscriptsSearchOptions struct {
	tmeet        *internal.Tmeet
	RecordFileID string // Record file ID
	MeetingID    string // Meeting ID
	Text         string // Search text
}

// newTranscriptSearchCmd searches transcript content.
func newTranscriptSearchCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &TranscriptsSearchOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "transcript-search",
		Short: "search transcript content",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordTranscriptSearch)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.RecordFileID, "record-file-id", "", "record file id (required)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting id")
	cmd.Flags().StringVar(&opts.Text, "text", "", "search text (required)")

	_ = cmd.MarkFlagRequired("record-file-id")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

func (o *TranscriptsSearchOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.TranscriptSearch(cmd.Context(), o.tmeet, record.TranscriptSearchParams{
		RecordFileID: o.RecordFileID,
		MeetingID:    o.MeetingID,
		Text:         o.Text,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
