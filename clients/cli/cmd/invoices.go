package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var invoicesCmd = &cobra.Command{
	Use:   "invoices",
	Short: "List and export invoices",
}

var invoicesListFlags struct {
	org   string
	month string
}

var invoicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List invoices from entitlements",
	RunE:  runInvoicesList,
}

var invoicesGetCmd = &cobra.Command{
	Use:   "get <invoice>",
	Short: "Download an invoice PDF",
	Args:  cobra.ExactArgs(1),
	RunE:  runInvoicesGet,
}

var invoicesExportFlags struct {
	org    string
	month  string
	out    string
	format string
	yes    bool
}

var invoicesExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export invoice records as CSV or JSON",
	RunE:  runInvoicesExport,
}

func init() {
	invoicesListCmd.Flags().StringVar(&invoicesListFlags.org, "org", "", "org context (informational)")
	invoicesListCmd.Flags().StringVar(&invoicesListFlags.month, "month", "", "filter by month (YYYY-MM)")

	invoicesExportCmd.Flags().StringVar(&invoicesExportFlags.org, "org", "", "org context (informational)")
	invoicesExportCmd.Flags().StringVar(&invoicesExportFlags.month, "month", "", "filter by month (YYYY-MM)")
	invoicesExportCmd.Flags().StringVar(&invoicesExportFlags.out, "out", ".", "output directory")
	invoicesExportCmd.Flags().StringVar(&invoicesExportFlags.format, "format", "csv", "csv or json")
	invoicesExportCmd.Flags().BoolVar(&invoicesExportFlags.yes, "yes", false, "confirm financial export")

	invoicesCmd.AddCommand(invoicesListCmd, invoicesGetCmd, invoicesExportCmd)
	rootCmd.AddCommand(invoicesCmd)
}

func runInvoicesList(cmd *cobra.Command, _ []string) error {
	items, raw, err := fetchMyEntitlements(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	items = filterEntitlementsByMonth(items, invoicesListFlags.month)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"invoices": items})
	}
	if len(items) == 0 && raw != nil && invoicesListFlags.month == "" {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "INVOICE_ID\tAMOUNT\tCURRENCY\tSTATUS\tVALID_FROM")
	for _, e := range items {
		inv := ""
		if e.InvoiceID != nil {
			inv = *e.InvoiceID
		}
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", inv, e.AmountPaidCents, e.Currency, e.Status, e.ValidFrom)
	}
	return w.Flush()
}

func runInvoicesGet(cmd *cobra.Command, args []string) error {
	body, err := fetchInvoicePDF(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(".", "invoice-"+args[0]+".pdf", body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "saved invoice to %s\n", path)
	return nil
}

func runInvoicesExport(cmd *cobra.Command, _ []string) error {
	if !invoicesExportFlags.yes {
		return fmt.Errorf("%s", financialExportWarning)
	}
	items, _, err := fetchMyEntitlements(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	items = filterEntitlementsByMonth(items, invoicesExportFlags.month)
	format := invoicesExportFlags.format
	if format == "json" {
		raw, err := json.MarshalIndent(map[string]any{"invoices": items}, "", "  ")
		if err != nil {
			return err
		}
		path, err := writeComplianceExport(invoicesExportFlags.out, "invoices.json", raw)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported %d invoices to %s\n", len(items), path)
		return nil
	}
	csvData, rows, err := entitlementsToCSV(items)
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(invoicesExportFlags.out, "invoices.csv", csvData)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported %d invoices (%d rows) to %s\n", len(items), rows-1, path)
	return nil
}