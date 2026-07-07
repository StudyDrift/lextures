package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var hallPassCmd = &cobra.Command{
	Use:   "hall-pass",
	Short: "Issue and track digital hall passes",
}

var hallPassListFlags struct {
	section string
}

var hallPassListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List active hall passes for a section",
	Args:  cobra.ExactArgs(1),
	RunE:  runHallPassList,
}

var hallPassIssueFlags struct {
	section       string
	destination   string
	estimatedMins int
	approve       bool
}

var hallPassIssueCmd = &cobra.Command{
	Use:   "issue <course>",
	Short: "Request a hall pass (authenticated student) in a section",
	Args:  cobra.ExactArgs(1),
	RunE:  runHallPassIssue,
}

var hallPassReturnFlags struct {
	pass string
}

var hallPassReturnCmd = &cobra.Command{
	Use:   "return",
	Short: "Mark a hall pass as returned",
	RunE:  runHallPassReturn,
}

func init() {
	hallPassListCmd.Flags().StringVar(&hallPassListFlags.section, "section", "", "section UUID or code (required)")
	_ = hallPassListCmd.MarkFlagRequired("section")

	hallPassIssueCmd.Flags().StringVar(&hallPassIssueFlags.section, "section", "", "section UUID or code (required)")
	_ = hallPassIssueCmd.MarkFlagRequired("section")
	hallPassIssueCmd.Flags().StringVar(&hallPassIssueFlags.destination, "destination", "bathroom", "destination: bathroom, office, library, nurse, other")
	hallPassIssueCmd.Flags().IntVar(&hallPassIssueFlags.estimatedMins, "estimated-mins", 5, "estimated minutes away (1-120)")
	hallPassIssueCmd.Flags().BoolVar(&hallPassIssueFlags.approve, "approve", false, "approve immediately after request (teacher)")

	hallPassReturnCmd.Flags().StringVar(&hallPassReturnFlags.pass, "pass", "", "hall pass UUID (required)")
	_ = hallPassReturnCmd.MarkFlagRequired("pass")

	hallPassCmd.AddCommand(hallPassListCmd, hallPassIssueCmd, hallPassReturnCmd)
	rootCmd.AddCommand(hallPassCmd)
}

func runHallPassList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sec, err := resolveSectionForCourse(c, args[0], hallPassListFlags.section)
	if err != nil {
		return err
	}
	passes, err := listActiveHallPasses(c, sec.ID)
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(passes)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTUDENT\tDESTINATION\tSTATUS\tREQUESTED")
	for _, pass := range passes {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			pass.ID, pass.StudentID, pass.Destination, pass.Status, pass.RequestedAt)
	}
	return w.Flush()
}

func runHallPassIssue(cmd *cobra.Command, args []string) error {
	dest := strings.ToLower(strings.TrimSpace(hallPassIssueFlags.destination))
	if dest == "" {
		return fmt.Errorf("--destination is required")
	}
	mins := hallPassIssueFlags.estimatedMins
	if mins <= 0 || mins > 120 {
		return fmt.Errorf("--estimated-mins must be between 1 and 120")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	sec, err := resolveSectionForCourse(c, args[0], hallPassIssueFlags.section)
	if err != nil {
		return err
	}

	pass, err := issueHallPass(c, sec.ID, dest, &mins)
	if err != nil {
		return err
	}

	if hallPassIssueFlags.approve && pass.Status == "requested" {
		pass, err = updateHallPassStatus(c, pass.ID, "approved")
		if err != nil {
			return err
		}
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(pass)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Hall pass %s requested to %s (status: %s)\n",
		pass.ID, pass.Destination, pass.Status)
	return nil
}

func runHallPassReturn(cmd *cobra.Command, args []string) error {
	pass, err := updateHallPassStatus(client.New(Cfg.Server, Cfg.APIKey), hallPassReturnFlags.pass, "returned")
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(pass)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Hall pass %s marked returned\n", pass.ID)
	return nil
}