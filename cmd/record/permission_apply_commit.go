package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// PermissionApplyCommitOptions holds the options for committing a record permission application.
type PermissionApplyCommitOptions struct {
	tmeet           *internal.Tmeet
	MeetingID       string // Meeting ID
	MeetingRecordID string // Meeting record ID
}

// newPermissionApplyCommitCmd commits the record permission application after user confirmation.
func newPermissionApplyCommitCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &PermissionApplyCommitOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "permission-apply-commit",
		Short: "commit record permission application after user confirmation",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordPermissionApplyCommit)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingRecordID, "meeting-record-id", "", "meeting record id (required)")
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting id")

	_ = cmd.MarkFlagRequired("meeting-record-id")

	return cmd
}

// Run run cmd
func (o *PermissionApplyCommitOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.PermissionApplyCommit(cmd.Context(), o.tmeet, record.PermissionApplyCommitParams{
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
