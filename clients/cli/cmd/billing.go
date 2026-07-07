package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Billing status and subscriptions",
}

var billingStatusFlags struct {
	org string
}

var billingStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active entitlements (subscription state)",
	RunE:  runBillingStatus,
}

var billingSubscriptionGetCmd = &cobra.Command{
	Use:   "subscription get",
	Short: "Alias for billing status (entitlements)",
	RunE:  runBillingStatus,
}

func init() {
	billingStatusCmd.Flags().StringVar(&billingStatusFlags.org, "org", "", "org context (informational)")

	billingCmd.AddCommand(billingStatusCmd, billingSubscriptionGetCmd)
	rootCmd.AddCommand(billingCmd)
}

func runBillingStatus(cmd *cobra.Command, _ []string) error {
	items, raw, err := fetchMyEntitlements(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"entitlements": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tAMOUNT\tCURRENCY\tSTATUS")
	for _, e := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", e.ID, e.EntitlementType, e.AmountPaidCents, e.Currency, e.Status)
	}
	return w.Flush()
}