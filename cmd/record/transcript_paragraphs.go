package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// TranscriptsParagraphsOptions holds the options for transcript paragraphs.
type TranscriptsParagraphsOptions struct {
	tmeet        *internal.Tmeet
	RecordFileID string // Record file ID
	MeetingID    string // Meeting ID
}

// newTranscriptParagraphsCmd gets transcript paragraphs.
func newTranscriptParagraphsCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &TranscriptsParagraphsOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "transcript-paragraphs",
		Short: "get transcript paragraphs",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordTranscriptParagraphs)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.RecordFileID, "record-file-id", "", "record file id (required)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting id")

	_ = cmd.MarkFlagRequired("record-file-id")

	return cmd
}

func (o *TranscriptsParagraphsOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.TranscriptParagraphs(cmd.Context(), o.tmeet, record.TranscriptParagraphsParams{
		RecordFileID: o.RecordFileID,
		MeetingID:    o.MeetingID,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
