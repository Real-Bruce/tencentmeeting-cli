package report

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	report "tmeet/internal/report"

	"github.com/spf13/cobra"
)

// ParticipantsOptions holds the options for participants.
type ParticipantsOptions struct {
	tmeet        *internal.Tmeet
	MeetingId    string // Meeting ID
	SubMeetingId string // Sub-meeting ID for recurring meetings
	Pos          int    // Query start position for paginated retrieval, deprecated use PageToken instead
	Size         int    // Number of participants per page, max 100, deprecated use PageSize instead
	StartTime    string // Query start time, ISO 8601, e.g. 2026-03-12T14:00+08:00
	EndTime      string // Query end time, ISO 8601, e.g. 2026-03-12T14:00+08:00
	PageToken    string // Page token
	PageSize     int    // Page size
}

// newParticipantsCmd gets the participants list.
func newParticipantsCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &ParticipantsOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "participants",
		Short: "get meeting participants",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdReportParticipants)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingId, "meeting-id", "", "meeting id (required)")
	cmd.Flags().StringVar(&opts.SubMeetingId, "sub-meeting-id", "", "sub meeting id for recurring meeting")
	cmd.Flags().IntVar(&opts.Pos, "pos", -1, "query start position, default 0, (deprecated, use --page-token instead)")
	cmd.Flags().IntVar(&opts.Size, "size", 0,
		"number of participants per page, max 100, (deprecated, use --page-size instead)")
	cmd.Flags().StringVar(&opts.StartTime, "start", "", "query start time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "query end time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 100, "page size, default 100, max 100")

	_ = cmd.MarkFlagRequired("meeting-id")
	_ = cmd.Flags().MarkDeprecated("pos", "use --page-token instead")
	_ = cmd.Flags().MarkDeprecated("size", "use --page-size instead")

	return cmd
}

func (o *ParticipantsOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := report.Participants(cmd.Context(), o.tmeet, report.ParticipantsParams{
		MeetingID:    o.MeetingId,
		SubMeetingID: o.SubMeetingId,
		Pos:          o.Pos,
		Size:         o.Size,
		StartTime:    o.StartTime,
		EndTime:      o.EndTime,
		PageToken:    o.PageToken,
		PageSize:     o.PageSize,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
