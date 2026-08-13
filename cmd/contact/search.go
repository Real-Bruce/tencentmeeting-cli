package contact

import (
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	middleWare "tmeet/internal/cmdutil/middleware"
	contact "tmeet/internal/contact"
	"tmeet/internal/output"

	"github.com/spf13/cobra"
)

// SearchOptions holds the options for searching contacts.
type SearchOptions struct {
	tmeet          *internal.Tmeet
	Username       string // Username to search
	JobTitle       string // Job title to search
	DepartmentName string // DepartmentName to search
}

// newSearchCmd searches contacts.
func newSearchCmd(tmeet *internal.Tmeet) *cobra.Command {
	opts := &SearchOptions{tmeet: tmeet}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "search contacts",
		RunE: middleWare.Chain(
			opts.Run,
			middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdContactSearch)),
			middleWare.WithCompact(tmeet),
		),
	}

	cmd.Flags().StringVar(&opts.Username, "username", "", "username to search (required)")
	cmd.Flags().StringVar(&opts.JobTitle, "job-title", "",
		"job title to filter results when the username search returns too many matches")
	cmd.Flags().StringVar(&opts.DepartmentName, "department-name", "",
		"department name to filter results when the username search returns too many matches")

	_ = cmd.MarkFlagRequired("username")

	return cmd
}

// Run executes the search contacts command.
func (o *SearchOptions) Run(cmd *cobra.Command, args []string) error {
	r, err := contact.Search(cmd.Context(), o.tmeet, contact.SearchParams{
		Username:       o.Username,
		JobTitle:       o.JobTitle,
		DepartmentName: o.DepartmentName,
	})
	if err != nil {
		return err
	}
	output.FormatPrint(cmd, r.TraceId, r.Message, r.Data,
		output.WithCompact(middleWare.GetCompactFields(cmd.Context())))
	return nil
}
