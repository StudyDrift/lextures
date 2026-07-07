package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var paymentsCmd = &cobra.Command{
	Use:   "payments",
	Short: "Payment transactions and checkout",
}

var paymentsTransactionsListCmd = &cobra.Command{
	Use:   "transactions list",
	Short: "List your payment transactions",
	RunE:  runPaymentsTransactionsList,
}

var paymentsTransactionsExportFlags struct {
	out    string
	format string
	yes    bool
}

var paymentsTransactionsExportCmd = &cobra.Command{
	Use:   "transactions export",
	Short: "Export payment transactions",
	RunE:  runPaymentsTransactionsExport,
}

var checkoutLinkCreateFlags struct {
	course     string
	plan       string
	successURL string
	cancelURL  string
}

var checkoutLinkCreateCmd = &cobra.Command{
	Use:   "checkout link create",
	Short: "Create a hosted Stripe checkout link",
	RunE:  runCheckoutLinkCreate,
}

func init() {
	paymentsTransactionsExportCmd.Flags().StringVar(&paymentsTransactionsExportFlags.out, "out", ".", "output directory")
	paymentsTransactionsExportCmd.Flags().StringVar(&paymentsTransactionsExportFlags.format, "format", "csv", "csv or json")
	paymentsTransactionsExportCmd.Flags().BoolVar(&paymentsTransactionsExportFlags.yes, "yes", false, "confirm financial export")

	checkoutLinkCreateCmd.Flags().StringVar(&checkoutLinkCreateFlags.course, "course", "", "course UUID")
	checkoutLinkCreateCmd.Flags().StringVar(&checkoutLinkCreateFlags.plan, "plan", "", "subscription plan id")
	checkoutLinkCreateCmd.Flags().StringVar(&checkoutLinkCreateFlags.successURL, "success-url", "http://localhost:5173/me/billing", "checkout success redirect")
	checkoutLinkCreateCmd.Flags().StringVar(&checkoutLinkCreateFlags.cancelURL, "cancel-url", "http://localhost:5173/me/billing", "checkout cancel redirect")

	paymentsCmd.AddCommand(paymentsTransactionsListCmd, paymentsTransactionsExportCmd, checkoutLinkCreateCmd)
	rootCmd.AddCommand(paymentsCmd)
}

func runPaymentsTransactionsList(cmd *cobra.Command, _ []string) error {
	items, raw, err := fetchMyTransactions(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"transactions": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPROVIDER\tAMOUNT\tCURRENCY\tSTATUS\tCREATED")
	for _, tx := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			tx.ID, tx.Provider, tx.AmountCents, tx.Currency, tx.Status, tx.CreatedAt)
	}
	return w.Flush()
}

func runPaymentsTransactionsExport(cmd *cobra.Command, _ []string) error {
	if !paymentsTransactionsExportFlags.yes {
		return fmt.Errorf("%s", financialExportWarning)
	}
	items, _, err := fetchMyTransactions(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if paymentsTransactionsExportFlags.format == "json" {
		raw, err := json.MarshalIndent(map[string]any{"transactions": items}, "", "  ")
		if err != nil {
			return err
		}
		path, err := writeComplianceExport(paymentsTransactionsExportFlags.out, "transactions.json", raw)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported %d transactions to %s\n", len(items), path)
		return nil
	}
	csvData, rows, err := transactionsToCSV(items)
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(paymentsTransactionsExportFlags.out, "transactions.csv", csvData)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported %d transactions (%d rows) to %s\n", len(items), rows-1, path)
	return nil
}

func runCheckoutLinkCreate(cmd *cobra.Command, _ []string) error {
	payload := map[string]any{
		"successUrl": checkoutLinkCreateFlags.successURL,
		"cancelUrl":  checkoutLinkCreateFlags.cancelURL,
	}
	if checkoutLinkCreateFlags.course != "" {
		payload["courseId"] = checkoutLinkCreateFlags.course
	}
	if checkoutLinkCreateFlags.plan != "" {
		payload["plan"] = checkoutLinkCreateFlags.plan
	}
	body, err := createCheckoutLink(client.New(Cfg.Server, Cfg.APIKey), payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}