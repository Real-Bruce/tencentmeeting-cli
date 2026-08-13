package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// TranscriptsGetOptions holds the options for transcript-related commands (details, paragraphs, search).
type TranscriptsGetOptions struct {
	tmeet        *internal.Tmeet
	RecordFileID string // Record file ID
	MeetingID    string // Meeting ID
	Pid          string // Starting paragraph ID for query
	Limit        string // Number of paragraphs to query
}

// newTranscriptGetCmd gets transcript details.
func newTranscriptGetCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &TranscriptsGetOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "transcript-get",
		Short: "get transcript detail",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordTranscriptGet)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.RecordFileID, "record-file-id", "", "record file id (required)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting id")
	cmd.Flags().StringVar(&opts.Pid, "pid", "", "start paragraph id")
	cmd.Flags().StringVar(&opts.Limit, "limit", "", "number of paragraphs to query")

	_ = cmd.MarkFlagRequired("record-file-id")

	return cmd
}

func (o *TranscriptsGetOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.TranscriptGet(cmd.Context(), o.tmeet, record.TranscriptGetParams{
		RecordFileID: o.RecordFileID,
		MeetingID:    o.MeetingID,
		Pid:          o.Pid,
		Limit:        o.Limit,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
