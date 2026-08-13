package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// InviteesOptions holds the options for getting meeting invitees.
type InviteesOptions struct {
	tmeet     *internal.Tmeet
	MeetingID string // Meeting ID
	Pos       int    // Starting position value for paginated retrieval of invited members, deprecated use PageToken instead
	PageToken string // Page token for pagination
	PageSize  int    // Page size for pagination, max 30
}

// newInviteesCmd gets meeting invitees.
func newInviteesCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &InviteesOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "invitees-list",
		Short: "get meeting invitees",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingInviteList)),
			middleWare.WithCompact(tmeet),
		),
	}

	// 填充参数
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().IntVar(&opts.Pos, "pos", -1, ""+
		"pos starting position value for retrieving the list of invited members in pagination. (deprecated, use --page-token instead)")
	cmd.Flags().StringVar(&opts.PageToken, "page-token", "", "page token for pagination")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 30, "page size for pagination, max 30, default 30")

	// mark required
	_ = cmd.MarkFlagRequired("meeting-id")
	// mark deprecated
	_ = cmd.Flags().MarkDeprecated("pos", "use --page-token instead")

	return cmd
}

func (o *InviteesOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.InviteesList(cmd.Context(), o.tmeet, meeting.InviteesListParams{
		MeetingID: o.MeetingID,
		Pos:       o.Pos,
		PageToken: o.PageToken,
		PageSize:  o.PageSize,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
