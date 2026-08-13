package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// CreateOptions holds the options for creating a meeting.
type CreateOptions struct {
	tmeet             *internal.Tmeet
	Subject           string   // Meeting subject (required)
	StartTime         string   // Meeting start time, ISO 8601, e.g. 2026-03-12T14:00+08:00 (required)
	EndTime           string   // Meeting end time, ISO 8601, e.g. 2026-03-12T15:00+08:00 (required)
	Password          string   // Meeting password (4-6 digits), optional
	Timezone          string   // Timezone, see Oracle-TimeZone standard, e.g. Asia/Shanghai
	MeetingType       int      // Meeting type, default 0. 0: normal meeting, 1: recurring meeting
	OnlyUserJoinType  int      // Member join restriction. 1: all members, 2: invited only, 3: internal only
	AutoInWaitingRoom bool     // Whether to enable waiting room, default false.
	RecurringType     int      // Recurring meeting config (required when meetingType=1). Recurrence type, default 0. 0: daily, 1: weekdays, 2: weekly, 3: biweekly, 4: monthly
	UntilType         int      // Recurring meeting config (required when meetingType=1). End type, default 0. 0: end by date, 1: end by count
	UntilCount        int      // Recurring meeting config (required when meetingType=1). Max occurrences. Daily/weekday/weekly max 500; biweekly/monthly max 50. Default 7.
	UntilDate         string   // Recurring meeting config (required when meetingType=1). End date, ISO 8601, e.g. 2026-03-12T15:00+08:00
	Invitees          []string // Invited participants (openid list). Up to 100 users.
	WaterMarkType     int      // Text watermark, 0=single row, 1=double row, 2=off
	AudioWatermark    bool     // Audio watermark, true=on, false=off
	AutoRecordType    string   // Auto record when host joins, none=off, local=local recording, cloud=cloud recording
	AutoASR           bool     // Auto speech recognition, true=on, false=off
}

// newCreateCmd creates a meeting.
func newCreateCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &CreateOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create meeting",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingCreate)),
		),
	}

	// Fill in parameters.
	cmd.Flags().StringVar(&opts.Subject, "subject", "", "meeting subject (required)")
	cmd.Flags().StringVar(&opts.StartTime, "start", "", "meeting start time (ISO 8601, e.g. 2026-03-12T14:00+08:00) (required)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "meeting end time (ISO 8601, e.g. 2026-03-12T15:00+08:00) (required)")
	cmd.Flags().StringVar(&opts.Password, "password", "", "meeting password (4-6 digits)")
	cmd.Flags().StringVar(&opts.Timezone, "timezone", "", "timezone (e.g. Asia/Shanghai)")
	cmd.Flags().IntVar(&opts.MeetingType, "meeting-type", 0, "meeting type: 0-normal, 1-recurring")
	cmd.Flags().IntVar(&opts.OnlyUserJoinType, "join-type", 0, "join restriction: 1-all members, 2-invited only, 3-internal only")
	cmd.Flags().BoolVar(&opts.AutoInWaitingRoom, "waiting-room", false, "enable waiting room")
	cmd.Flags().IntVar(&opts.RecurringType, "recurring-type", 0, "recurring type (meeting-type=1): 0-daily, 1-weekdays, 2-weekly, 3-biweekly, 4-monthly")
	cmd.Flags().IntVar(&opts.UntilType, "until-type", 0, "recurring end type (meeting-type=1): 0-by date, 1-by count")
	cmd.Flags().IntVar(&opts.UntilCount, "until-count", 7, "recurring count (meeting-type=1, max: 500 for daily/weekdays/weekly, 50 for biweekly/monthly)")
	cmd.Flags().StringVar(&opts.UntilDate, "until-date", "", "recurring end date (meeting-type=1) e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().StringSliceVar(&opts.Invitees, "invitees", nil,
		"invited participants' openid list, comma-separated or repeat the flag (max 100, e.g. --invitees open_id1,open_id2)")
	cmd.Flags().IntVar(&opts.WaterMarkType, "water-mark-type", 2, "text watermark: 0=single row, 1=double row, 2=off")
	cmd.Flags().BoolVar(&opts.AudioWatermark, "audio-watermark", false, "audio watermark")
	cmd.Flags().StringVar(&opts.AutoRecordType, "auto-record-type", "none", "auto record when host joins: none=off, local=local recording, cloud=cloud recording")
	cmd.Flags().BoolVar(&opts.AutoASR, "auto-asr", false, "auto speech")

	_ = cmd.MarkFlagRequired("subject")
	_ = cmd.MarkFlagRequired("start")
	_ = cmd.MarkFlagRequired("end")
	return cmd
}

// Hints implements the Hints.HintProvider interface and returns hints for meeting settings.
// Lock detection is solely driven by the corp_lock_mask bitmask in the response.
func (o *CreateOptions) Hints(responseData string) []string {
	return cmdutil.GenerateMeetingSettingsHints(responseData)
}

func (o *CreateOptions) Run(cmd *cobra.Command, args []string) error {
	p := meeting.CreateParams{
		Subject:           o.Subject,
		StartTime:         o.StartTime,
		EndTime:           o.EndTime,
		Password:          o.Password,
		Timezone:          o.Timezone,
		MeetingType:       o.MeetingType,
		OnlyUserJoinType:  o.OnlyUserJoinType,
		AutoInWaitingRoom: o.AutoInWaitingRoom,
		RecurringType:     o.RecurringType,
		UntilType:         o.UntilType,
		UntilCount:        o.UntilCount,
		UntilDate:         o.UntilDate,
		Invitees:          o.Invitees,
	}
	// Changed() → 指针：CLI flag 显式设置才传给纯函数，与 MCP 的 nil 语义一致。
	if cmd.Flags().Changed("water-mark-type") {
		p.WaterMarkType = &o.WaterMarkType
	}
	if cmd.Flags().Changed("audio-watermark") {
		p.AudioWatermark = &o.AudioWatermark
	}
	if cmd.Flags().Changed("auto-record-type") {
		p.AutoRecordType = &o.AutoRecordType
	}
	if cmd.Flags().Changed("auto-asr") {
		p.AutoASR = &o.AutoASR
	}

	r, err := meeting.Create(cmd.Context(), o.tmeet, p)
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithHints(o.Hints))
	return nil
}
