package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// PermissionApplyPrepareOptions holds the options for previewing a record permission application.
type PermissionApplyPrepareOptions struct {
	tmeet           *internal.Tmeet
	MeetingID       string // Meeting ID
	MeetingRecordID string // Meeting record ID
}

// newPermissionApplyPrepareCmd previews the record permission application before commit.
func newPermissionApplyPrepareCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &PermissionApplyPrepareOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "permission-apply-prepare",
		Short: "preview record permission application before commit",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordPermissionApplyPrepare)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingRecordID, "meeting-record-id", "", "meeting record id (required)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting id")

	_ = cmd.MarkFlagRequired("meeting-record-id")

	return cmd
}

// Run run cmd
func (o *PermissionApplyPrepareOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.PermissionApplyPrepare(cmd.Context(), o.tmeet, record.PermissionApplyPrepareParams{
		MeetingRecordID: o.MeetingRecordID,
		MeetingID:       o.MeetingID,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
