package meeting

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	meeting "tmeet/internal/meeting"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// InviteesAddOptions holds the options for adding invitees to a meeting.
type InviteesAddOptions struct {
	tmeet     *internal.Tmeet
	MeetingID string   // Meeting ID
	Invitees  []string // List of invitees to add, filled with user open_id
}

// newInviteesAddCmd adds invitees to a meeting.
func newInviteesAddCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &InviteesAddOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "invitees-add",
		Short: "add meeting invitees",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingInviteAdd)),
		),
	}

	// fill flags
	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().StringSliceVar(&opts.Invitees, "invitees", nil,
		"new invitee open_id list, comma-separated or repeat the flag, max 100"+
			"eg. --invitees open_id1,open_id2 or --invitees open_id1 --invitees open_id2 (required)")

	// mark required
	_ = cmd.MarkFlagRequired("meeting-id")
	_ = cmd.MarkFlagRequired("invitees")

	return cmd
}

// Run executes the invite add command.
func (o *InviteesAddOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := meeting.InviteesAdd(cmd.Context(), o.tmeet, meeting.InviteesAddParams{
		MeetingID: o.MeetingID,
		Invitees:  o.Invitees,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
