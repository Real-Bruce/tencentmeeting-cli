package tshoot

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	"tmeet/internal/output"
	tshoot "tmeet/internal/tshoot"

	"github.com/spf13/cobra"
)

// FeedbackOptions is the options for feedback.
type FeedbackOptions struct {
	tmeet        *internal.Tmeet
	Category     string // Feedback category
	Intent       string // Original intent of the agent
	ActionsTried string // Actions the agent has tried
	Result       string // Result or blocker of the tried actions
	ToolName     string // Tool/command name used
	ErrorCode    string // Error code returned by the tool
}

// newFeedbackCmd reports troubleshooting feedback to the server.
func newFeedbackCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &FeedbackOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "feedback",
		Short: "report troubleshooting feedback to the server",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdTshootFeedback)),
		),
	}

	cmd.Flags().StringVar(&opts.Category, "category", "", `feedback category (required), options:
		tool_not_found: want to do something but cannot find a matching tool;
		tool_error: called a tool but it returned an error;
		tool_inadequate: tool exists but its capability/parameters are insufficient;
		unexpected_result: call succeeded but the result did not meet expectations;
		suggestion: general suggestion or improvement idea`,
	)
	cmd.Flags().StringVar(&opts.Intent, "intent", "", "original intent of the agent (required), max 200 characters")
	cmd.Flags().StringVar(&opts.ActionsTried, "actions-tried", "", "actions the agent has tried, max 500 characters")
	cmd.Flags().StringVar(&opts.Result, "result", "", "result or blocker of the tried actions, max 500 characters")
	cmd.Flags().StringVar(&opts.ToolName, "tool-name", "", "tool/command name used")
	cmd.Flags().StringVar(&opts.ErrorCode, "error-code", "", "error code returned by the tool")

	_ = cmd.MarkFlagRequired("category")
	_ = cmd.MarkFlagRequired("intent")

	return cmd
}

// Run executes the feedback command.
func (o *FeedbackOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := tshoot.Feedback(cmd.Context(), o.tmeet, tshoot.FeedbackParams{
		Category:     o.Category,
		Intent:       o.Intent,
		ActionsTried: o.ActionsTried,
		Result:       o.Result,
		ToolName:     o.ToolName,
		ErrorCode:    o.ErrorCode,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data)
	return nil
}
