package record

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	record "tmeet/internal/record"

	"github.com/spf13/cobra"
)

// SmartMinutesOptions holds the options for smart minutes.
type SmartMinutesOptions struct {
	tmeet        *internal.Tmeet
	RecordFileId string // Record file ID
	Lang         string // Language for translation: default (no translation), zh (Simplified Chinese), en (English), ja (Japanese)
	Pwd          string // Record file access password
}

// newSmartMinutesCmd gets smart minutes.
func newSmartMinutesCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &SmartMinutesOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "smart-minutes",
		Short: "get smart minutes from record",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdRecordSmartMinutes)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.RecordFileId, "record-file-id", "", "record file id (required)")
	cmd.Flags().StringVar(&opts.Lang, "lang", "default", "language: default, zh, en, ja")
	cmd.Flags().StringVar(&opts.Pwd, "pwd", "", "record file access password")

	_ = cmd.MarkFlagRequired("record-file-id")

	return cmd
}

// Run run cmd
func (o *SmartMinutesOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := record.SmartMinutes(cmd.Context(), o.tmeet, record.SmartMinutesParams{
		RecordFileID: o.RecordFileId,
		Lang:         o.Lang,
		Pwd:          o.Pwd,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
