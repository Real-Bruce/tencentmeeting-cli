package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// GetOptions holds the options for getting meeting details.
type GetOptions struct {
	tmeet       *internal.Tmeet
	MeetingID   string // Meeting ID, higher priority than meeting code
	MeetingCode string // Meeting code
}

// newGetCmd gets meeting details.
func newGetCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &GetOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get meeting details",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.FlagSwitch(
				cmdutil.FlagCase{
					When:   cmdutil.WhenStringFlagSet("meeting-id"),
					ApiCmd: cmdutil.ApiCmdMeetingGetById,
				},
				cmdutil.FlagCase{
					When:   cmdutil.WhenStringFlagSet("meeting-code"),
					ApiCmd: cmdutil.ApiCmdMeetingGetByCode,
				},
			)),
		),
	}

	// 填充参数
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID, higher priority than `meeting_code`")
	cmd.Flags().StringVar(&opts.MeetingCode, "meeting-code", "", "meeting code")

	cmd.MarkFlagsOneRequired("meeting-id", "meeting-code")

	return cmd
}

func (o *GetOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.Get(cmd.Context(), o.tmeet, meeting.GetParams{
		MeetingID:   o.MeetingID,
		MeetingCode: o.MeetingCode,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
