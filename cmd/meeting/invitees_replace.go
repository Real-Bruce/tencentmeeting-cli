package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// InviteesReplaceOptions holds the options for replacing meeting invitees.
type InviteesReplaceOptions struct {
	tmeet     *internal.Tmeet
	MeetingID string   // Meeting ID
	Invitees  []string // New list of invitees to replace with, filled with user open_id
}

// newInviteesReplaceCmd replaces meeting invitees with the given list.
func newInviteesReplaceCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &InviteesReplaceOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "invitees-replace",
		Short: "replace meeting invitees",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingInviteReplace)),
		),
	}

	// fill flags
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().StringSliceVar(&opts.Invitees, "invitees", nil,
		"new invitee open_id list, comma-separated or repeat the flag, leave empty to remove all invitees, max 100"+
			"eg. --invitees open_id1,open_id2 or --invitees open_id1 --invitees open_id2")

	// mark required
	_ = cmd.MarkFlagRequired("meeting-id")

	return cmd
}

// Run executes the invite replace command.
func (o *InviteesReplaceOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.InviteesReplace(cmd.Context(), o.tmeet, meeting.InviteesReplaceParams{
		MeetingID: o.MeetingID,
		Invitees:  o.Invitees,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
