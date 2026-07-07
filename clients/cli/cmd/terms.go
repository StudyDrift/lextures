package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var termsCmd = &cobra.Command{
	Use:   "terms",
	Short: "Manage academic terms for an organization",
}

var termsListCmd = &cobra.Command{
	Use:   "list <org>",
	Short: "List terms for an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runTermsList,
}

var termsCreateFlags struct {
	name   string
	typ    string
	start  string
	end    string
	status string
}

var termsCreateCmd = &cobra.Command{
	Use:   "create <org>",
	Short: "Create an academic term",
	Args:  cobra.ExactArgs(1),
	RunE:  runTermsCreate,
}

var termsUpdateFlags struct {
	name   string
	typ    string
	start  string
	end    string
	status string
}

var termsUpdateCmd = &cobra.Command{
	Use:   "update <org> <term>",
	Short: "Update a term",
	Args:  cobra.ExactArgs(2),
	RunE:  runTermsUpdate,
}

var termsDeleteCmd = &cobra.Command{
	Use:   "delete <org> <term>",
	Short: "Delete a term",
	Args:  cobra.ExactArgs(2),
	RunE:  runTermsDelete,
}

func init() {
	termsCreateCmd.Flags().StringVar(&termsCreateFlags.name, "name", "", "term name (required)")
	_ = termsCreateCmd.MarkFlagRequired("name")
	termsCreateCmd.Flags().StringVar(&termsCreateFlags.typ, "type", "term", "term type")
	termsCreateCmd.Flags().StringVar(&termsCreateFlags.start, "start", "", "start date (YYYY-MM-DD)")
	termsCreateCmd.Flags().StringVar(&termsCreateFlags.end, "end", "", "end date (YYYY-MM-DD)")
	termsCreateCmd.Flags().StringVar(&termsCreateFlags.status, "status", "active", "term status")

	termsUpdateCmd.Flags().StringVar(&termsUpdateFlags.name, "name", "", "term name")
	termsUpdateCmd.Flags().StringVar(&termsUpdateFlags.typ, "type", "", "term type")
	termsUpdateCmd.Flags().StringVar(&termsUpdateFlags.start, "start", "", "start date")
	termsUpdateCmd.Flags().StringVar(&termsUpdateFlags.end, "end", "", "end date")
	termsUpdateCmd.Flags().StringVar(&termsUpdateFlags.status, "status", "", "term status")

	termsCmd.AddCommand(termsListCmd, termsCreateCmd, termsUpdateCmd, termsDeleteCmd)
	rootCmd.AddCommand(termsCmd)
}

func runTermsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchOrgTerms(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Terms) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No terms.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tSTART\tEND\tSTATUS")
	for _, t := range body.Terms {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, t.Name, t.StartDate, t.EndDate, t.Status)
	}
	return w.Flush()
}

func runTermsCreate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{
		"name":      termsCreateFlags.name,
		"termType":  termsCreateFlags.typ,
		"startDate": termsCreateFlags.start,
		"endDate":   termsCreateFlags.end,
		"status":    termsCreateFlags.status,
	}
	body, err := createOrgTerm(client.New(Cfg.Server, Cfg.APIKey), args[0], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Created term")
}

func runTermsUpdate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{}
	if termsUpdateFlags.name != "" {
		payload["name"] = termsUpdateFlags.name
	}
	if termsUpdateFlags.typ != "" {
		payload["termType"] = termsUpdateFlags.typ
	}
	if termsUpdateFlags.start != "" {
		payload["startDate"] = termsUpdateFlags.start
	}
	if termsUpdateFlags.end != "" {
		payload["endDate"] = termsUpdateFlags.end
	}
	if termsUpdateFlags.status != "" {
		payload["status"] = termsUpdateFlags.status
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one update flag is required")
	}
	body, err := patchOrgTerm(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Updated term")
}

func runTermsDelete(cmd *cobra.Command, args []string) error {
	if err := deleteOrgTerm(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, "Deleted term")
}