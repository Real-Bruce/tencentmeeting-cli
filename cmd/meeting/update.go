package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// UpdateOptions holds the options for updating a meeting.
type UpdateOptions struct {
	tmeet             *internal.Tmeet
	MeetingID         string   // Meeting ID
	Subject           string   // Meeting subject
	StartTime         string   // Meeting start time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	EndTime           string   // Meeting end time, ISO 8601, e.g. 2026-03-12T15:00+08:00
	Password          string   // Meeting password (4-6 digits)
	Timezone          string   // Timezone, e.g. Asia/Shanghai
	MeetingType       int      // Meeting type. 0: normal meeting, 1: recurring meeting
	OnlyUserJoinType  int      // Member join restriction. 1: all members, 2: invited only, 3: internal only
	AutoInWaitingRoom bool     // Whether to enable waiting room
	RecurringType     int      // Recurring meeting config (required when meetingType=1). Recurrence type, default 0. 0: daily, 1: weekdays, 2: weekly, 3: biweekly, 4: monthly
	UntilType         int      // Recurring meeting config (required when meetingType=1). End type, default 0. 0: end by date, 1: end by count
	UntilCount        int      // Recurring meeting config (required when meetingType=1). Max occurrences. Daily/weekday/weekly max 500; biweekly/monthly max 50. Default 7.
	UntilDate         string   // Recurring meeting config (required when meetingType=1). End date
	SubMeetingID      string   // Recurring meeting config (optional, when meetingType=1). Sub-meeting ID: modify a single sub-meeting's time only; mutually exclusive with recurring-rule fields (recurring-type/until-type/until-count/until-date). If empty, the whole recurring meeting is updated.
	Invitees          []string // Invited participants (openid list).
	InviteesType      string   // Invitees mutation strategy: replace / add / remove.
	WaterMarkType     int      // Text watermark, 0=single row, 1=double row, 2=off
	AudioWatermark    bool     // Audio watermark, true=on, false=off
	AutoRecordType    string   // Auto record when host joins, none=off, local=local recording, cloud=cloud recording
	AutoASR           bool     // Auto speech recognition, true=on, false=off
}

// newUpdateCmd updates a meeting.
func newUpdateCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &UpdateOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update meeting",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingUpdate)),
		),
	}

	// 填充参数
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().StringVar(&opts.Subject, "subject", "", "meeting subject")
	cmd.Flags().StringVar(&opts.StartTime, "start", "", "meeting start time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.EndTime, "end", "", "meeting end time (ISO 8601, e.g. 2026-03-12T14:00+08:00)")
	cmd.Flags().StringVar(&opts.Password, "password", "", "meeting password (4-6 digits)")
	cmd.Flags().StringVar(&opts.Timezone, "timezone", "", "timezone (e.g. Asia/Shanghai)")
	cmd.Flags().IntVar(&opts.MeetingType, "meeting-type", 0, "meeting type: 0-normal, 1-recurring")
	cmd.Flags().IntVar(&opts.OnlyUserJoinType, "join-type", 0, "join restriction: 1-all members, 2-invited only, 3-internal only")
	cmd.Flags().BoolVar(&opts.AutoInWaitingRoom, "waiting-room", false, "enable waiting room")
	cmd.Flags().IntVar(&opts.RecurringType, "recurring-type", 0, "recurring type (0-daily, 1-weekdays, 2-weekly, 3-biweekly, 4-monthly)")
	cmd.Flags().IntVar(&opts.UntilType, "until-type", 0, "until type (0-date, 1-count)")
	cmd.Flags().IntVar(&opts.UntilCount, "until-count", 7, "until count")
	cmd.Flags().StringVar(&opts.UntilDate, "until-date", "", "until date e.g. 2026-03-12T15:00+08:00)")
	cmd.Flags().StringVar(&opts.SubMeetingID, "sub-meeting-id", "",
		"sub-meeting ID: modify a single sub-meeting's time only (recurring meeting). "+
			"mutually exclusive with --recurring-type / --until-type / --until-count / --until-date. "+
			"if not set, the whole recurring meeting is updated")
	cmd.Flags().StringSliceVar(&opts.Invitees, "invitees", nil,
		"invitee openid list, comma-separated or repeat the flag (used together with --invitees-type)")
	cmd.Flags().StringVar(&opts.InviteesType, "invitees-type", "",
		"invitees mutation strategy: replace | add | remove (required when --invitees is set)")
	cmd.Flags().IntVar(&opts.WaterMarkType, "water-mark-type", 2, "text watermark: 0=single row, 1=double row, 2=off")
	cmd.Flags().BoolVar(&opts.AudioWatermark, "audio-watermark", false, "audio watermark")
	cmd.Flags().StringVar(&opts.AutoRecordType, "auto-record-type", "none", "auto record when host joins: none=off, local=local recording, cloud=cloud recording")
	cmd.Flags().BoolVar(&opts.AutoASR, "auto-asr", false, "auto speech recognition")

	// Set required parameters.
	_ = cmd.MarkFlagRequired("meeting-id")

	return cmd
}

// Hints implements the Hints.HintProvider interface and returns hints for meeting settings.
// Lock detection is solely driven by the corp_lock_mask bitmask in the response.
func (o *UpdateOptions) Hints(responseData string) []string {
	return cmdutil.GenerateMeetingSettingsHints(responseData)
}

func (o *UpdateOptions) Run(cmd *cobra.Command, args []string) error {
	p := meeting.UpdateParams{
		MeetingID:         o.MeetingID,
		Subject:           o.Subject,
		StartTime:         o.StartTime,
		EndTime:           o.EndTime,
		Password:          o.Password,
		Timezone:          o.Timezone,
		OnlyUserJoinType:  o.OnlyUserJoinType,
		AutoInWaitingRoom: o.AutoInWaitingRoom,
		SubMeetingID:      o.SubMeetingID,
		Invitees:          o.Invitees,
		InviteesType:      o.InviteesType,
	}
	// Changed() → 指针：CLI flag 显式设置才传给纯函数，与 MCP 的 nil 语义一致。
	if cmd.Flags().Changed("meeting-type") {
		p.MeetingType = &o.MeetingType
	}
	if cmd.Flags().Changed("recurring-type") {
		p.RecurringType = &o.RecurringType
	}
	if cmd.Flags().Changed("until-type") {
		p.UntilType = &o.UntilType
	}
	if cmd.Flags().Changed("until-count") {
		p.UntilCount = &o.UntilCount
	}
	if cmd.Flags().Changed("until-date") {
		p.UntilDate = &o.UntilDate
	}
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

	r, err := meeting.Update(cmd.Context(), o.tmeet, p)
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithHints(o.Hints))
	return nil
}
