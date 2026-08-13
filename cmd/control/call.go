package control

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	control "tmeet/internal/control"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// CallOptions holds the options for calling meeting members.
type CallOptions struct {
	tmeet     *internal.Tmeet
	MeetingID string   // Meeting ID
	Users     []string // List of users to call, filled with user open_id
}

// newCallCmd calls users into a meeting.
func newCallCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &CallOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "call",
		Short: "call meeting members (in-meeting invite call)",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdControlCall)),
		),
	}

	cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID (required)")
	cmd.Flags().StringSliceVar(&opts.Users, "users", nil,
		"user open_id list to call, comma-separated or repeat the flag, max 20"+
			"eg. --users open_id1,open_id2 or --users open_id1 --users open_id2 (required)")

	_ = cmd.MarkFlagRequired("meeting-id")
	_ = cmd.MarkFlagRequired("users")

	return cmd
}

// Run executes the call command.
func (o *CallOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := control.Call(cmd.Context(), o.tmeet, control.CallParams{
		MeetingID: o.MeetingID,
		Users:     o.Users,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
