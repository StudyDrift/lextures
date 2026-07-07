package cmd

import (
	"fmt"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var taxCmd = &cobra.Command{
	Use:   "tax",
	Short: "Organization tax configuration and reports",
}

var taxGetFlags struct {
	org string
}

var taxGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get org tax settings",
	RunE:  runTaxGet,
}

var taxSetFlags struct {
	org  string
	file string
}

var taxSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set org tax settings from JSON file",
	RunE:  runTaxSet,
}

var taxReportFlags struct {
	org          string
	period       string
	jurisdiction string
	format       string
	out          string
	yes          bool
}

var taxReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Export org tax report",
	RunE:  runTaxReport,
}

func init() {
	taxGetCmd.Flags().StringVar(&taxGetFlags.org, "org", "", "organization UUID (required)")
	_ = taxGetCmd.MarkFlagRequired("org")

	taxSetCmd.Flags().StringVar(&taxSetFlags.org, "org", "", "organization UUID (required)")
	taxSetCmd.Flags().StringVar(&taxSetFlags.file, "file", "", "tax settings JSON file (required)")
	_ = taxSetCmd.MarkFlagRequired("org")
	_ = taxSetCmd.MarkFlagRequired("file")

	taxReportCmd.Flags().StringVar(&taxReportFlags.org, "org", "", "organization UUID (required)")
	taxReportCmd.Flags().StringVar(&taxReportFlags.period, "period", "", "report period (YYYY-MM)")
	taxReportCmd.Flags().StringVar(&taxReportFlags.jurisdiction, "jurisdiction", "", "jurisdiction filter")
	taxReportCmd.Flags().StringVar(&taxReportFlags.format, "format", "csv", "csv or json")
	taxReportCmd.Flags().StringVar(&taxReportFlags.out, "out", ".", "output directory")
	taxReportCmd.Flags().BoolVar(&taxReportFlags.yes, "yes", false, "confirm financial export")
	_ = taxReportCmd.MarkFlagRequired("org")

	taxCmd.AddCommand(taxGetCmd, taxSetCmd, taxReportCmd)
	rootCmd.AddCommand(taxCmd)
}

func runTaxGet(cmd *cobra.Command, _ []string) error {
	body, err := fetchOrgTaxSettings(client.New(Cfg.Server, Cfg.APIKey), taxGetFlags.org)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTaxSet(cmd *cobra.Command, _ []string) error {
	payload, err := readComplianceInputFile(taxSetFlags.file)
	if err != nil {
		return err
	}
	body, err := putOrgTaxSettings(client.New(Cfg.Server, Cfg.APIKey), taxSetFlags.org, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTaxReport(cmd *cobra.Command, _ []string) error {
	if !taxReportFlags.yes {
		return fmt.Errorf("%s", financialExportWarning)
	}
	body, contentType, err := fetchOrgTaxReport(client.New(Cfg.Server, Cfg.APIKey),
		taxReportFlags.org, taxReportFlags.period, taxReportFlags.jurisdiction, taxReportFlags.format)
	if err != nil {
		return err
	}
	filename := "tax-report.csv"
	if taxReportFlags.format == "json" || contentType == "application/json" {
		filename = "tax-report.json"
	}
	path, err := writeComplianceExport(taxReportFlags.out, filename, body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported tax report to %s\n", path)
	return nil
}