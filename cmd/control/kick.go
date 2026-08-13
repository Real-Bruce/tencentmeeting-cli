package control

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	control "tmeet/internal/control"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// KickOptions holds the options for kicking members out of a meeting.
type KickOptions struct {
	tmeet       *internal.Tmeet
	MeetingID   string   // Meeting ID
	AllowRejoin bool     // Allow re join after kick out
	Users       []string // List of users to kick out, filled with user open_id
	SipUsers    []string // List of sip users to kick out, filled with user ms_open_id
	PstnUsers   []string // List of pstn users to kick out, filled with user ms_open_id
}

// newKickCmd kicks users out of a meeting.
func newKickCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &KickOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "kick",
		Short: "kick meeting members out (in-meeting kick-out)",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdControlKick)),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().BoolVar(&opts.AllowRejoin, "allow-rejoin", true,
		"allow kicked-out members to rejoin the meeting, default true. "+
			"pass --allow-rejoin=false to disallow rejoin")
	cmd.Flags().StringSliceVar(&opts.Users, "users", nil,
		"user open_id list to kick out, not contains sip/pstn device, comma-separated or repeat the flag, "+
			"the total number of --users/--sip-users/--pstn-users is max 20. "+
			"eg. --users open_id1,open_id2 or --users open_id1 --users open_id2")
	cmd.Flags().StringSliceVar(&opts.SipUsers, "sip-users", nil,
		"sip user ms_open_id list to kick out, comma-separated or repeat the flag, "+
			"the total number of --users/--sip-users/--pstn-users is max 20. "+
			"eg. --sip-users ms_open_id1,ms_open_id2 or --sip-users ms_open_id1 --sip-users ms_open_id2")
	cmd.Flags().StringSliceVar(&opts.PstnUsers, "pstn-users", nil,
		"pstn user ms_open_id list to kick out, comma-separated or repeat the flag, "+
			"the total number of --users/--sip-users/--pstn-users is max 20. "+
			"eg. --pstn-users ms_open_id1,ms_open_id2 or --pstn-users ms_open_id1 --pstn-users ms_open_id2")

	_ = cmd.MarkFlagRequired("meeting-id")
	cmd.MarkFlagsOneRequired("users", "sip-users", "pstn-users")

	return cmd
}

// Run executes the kick command.
func (o *KickOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := control.Kick(cmd.Context(), o.tmeet, control.KickParams{
		MeetingID:   o.MeetingID,
		AllowRejoin: o.AllowRejoin,
		Users:       o.Users,
		SipUsers:    o.SipUsers,
		PstnUsers:   o.PstnUsers,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
