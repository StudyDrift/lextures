package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var entitlementsCmd = &cobra.Command{
	Use:   "entitlements",
	Short: "Audit user and organization entitlements",
}

var entitlementsListFlags struct {
	org  string
	user string
}

var entitlementsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List entitlements for the caller (or filtered context)",
	RunE:  runEntitlementsList,
}

func init() {
	entitlementsListCmd.Flags().StringVar(&entitlementsListFlags.org, "org", "", "org context (informational)")
	entitlementsListCmd.Flags().StringVar(&entitlementsListFlags.user, "user", "", "user context (defaults to caller)")

	entitlementsCmd.AddCommand(entitlementsListCmd)
	rootCmd.AddCommand(entitlementsCmd)
}

func runEntitlementsList(cmd *cobra.Command, args []string) error {
	if entitlementsListFlags.user != "" && entitlementsListFlags.user != "me" {
		return fmt.Errorf("listing another user's entitlements requires the admin console; omit --user to list caller entitlements")
	}
	items, raw, err := fetchEntitlements(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tCOURSE")
	for _, e := range items {
		course := ""
		if e.CourseID != nil {
			course = *e.CourseID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.ID, e.EntitlementType, e.Status, course)
	}
	if entitlementsListFlags.org != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(org filter %s is informational for renewal planning)\n", entitlementsListFlags.org)
	}
	return w.Flush()
}