package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// CancelOptions holds the options for canceling a meeting.
type CancelOptions struct {
	tmeet        *internal.Tmeet
	MeetingID    string // Meeting ID
	SubMeetingID string // Sub-meeting ID for canceling a specific recurring sub-meeting
	MeetingType  int    // Meeting type, default 0. 0: normal meeting, 1: recurring meeting
}

// newCancelCmd cancels a meeting.
func newCancelCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &CancelOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "cancel meeting",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingCancel)),
		),
	}

	// 填充参数
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().StringVar(&opts.SubMeetingID, "sub-meeting-id", "", "sub-meeting ID for canceling a specific recurring meeting")
	cmd.Flags().IntVar(&opts.MeetingType, "meeting-type", 0, "meeting type: 0-normal/single sub-meeting, 1-entire recurring meeting")

	// 设置必填参数
	_ = cmd.MarkFlagRequired("meeting-id")

	return cmd
}

func (o *CancelOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.Cancel(cmd.Context(), o.tmeet, meeting.CancelParams{
		MeetingID:    o.MeetingID,
		SubMeetingID: o.SubMeetingID,
		MeetingType:  o.MeetingType,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
