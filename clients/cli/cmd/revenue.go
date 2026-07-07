package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var revenueCmd = &cobra.Command{
	Use:   "revenue",
	Short: "View revenue-share and creator earnings",
}

var revenueReportFlags struct {
	org string
	yes bool
}

var revenueReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show platform or creator revenue summary",
	RunE:  runRevenueReport,
}

var revenueShareCmd = &cobra.Command{
	Use:   "share",
	Short: "Alias for revenue report (creator earnings)",
	RunE:  runRevenueShare,
}

var affiliateCmd = &cobra.Command{
	Use:   "affiliate",
	Short: "Creator affiliate referral earnings",
}

var affiliateReportFlags struct {
	yes bool
}

var affiliateReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show affiliate codes and earnings ledger",
	RunE:  runAffiliateReport,
}

func init() {
	revenueReportCmd.Flags().StringVar(&revenueReportFlags.org, "org", "", "org context (informational)")
	revenueReportCmd.Flags().BoolVar(&revenueReportFlags.yes, "yes", false, "confirm export of financial data")

	revenueShareCmd.Flags().StringVar(&revenueReportFlags.org, "org", "", "org context (informational)")
	revenueShareCmd.Flags().BoolVar(&revenueReportFlags.yes, "yes", false, "confirm export of financial data")

	affiliateReportCmd.Flags().BoolVar(&affiliateReportFlags.yes, "yes", false, "confirm export of financial data")

	affiliateCmd.AddCommand(affiliateReportCmd)
	revenueCmd.AddCommand(revenueReportCmd, revenueShareCmd, affiliateCmd)
	rootCmd.AddCommand(revenueCmd)
}

func runAffiliateReport(cmd *cobra.Command, _ []string) error {
	if !affiliateReportFlags.yes && !globalFlags.jsonOut {
		return fmt.Errorf("affiliate data is financial; re-run with --yes to export")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	codes, err := fetchAffiliateCodes(c)
	if err != nil {
		return err
	}
	ledger, err := fetchCreatorEarningsLedger(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		var codesParsed any
		var ledgerParsed any
		_ = json.Unmarshal(codes, &codesParsed)
		_ = json.Unmarshal(ledger, &ledgerParsed)
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"affiliateCodes": codesParsed,
			"ledger":         ledgerParsed,
		})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "affiliate_codes:")
	_, _ = cmd.OutOrStdout().Write(codes)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ledger:")
	_, err = cmd.OutOrStdout().Write(ledger)
	return err
}

func runRevenueReport(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchRevenueSummary(c)
	if err != nil {
		body, err = fetchCreatorEarnings(c)
	}
	if err != nil {
		return err
	}
	if !revenueReportFlags.yes && !globalFlags.jsonOut {
		return fmt.Errorf("revenue data is financial; re-run with --yes to export")
	}
	if globalFlags.jsonOut {
		var parsed any
		if json.Unmarshal(body, &parsed) == nil {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(parsed)
		}
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runRevenueShare(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchCreatorEarnings(c)
	if err != nil {
		return err
	}
	if !revenueReportFlags.yes && !globalFlags.jsonOut {
		return fmt.Errorf("revenue data is financial; re-run with --yes to export")
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}