package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var licensesCmd = &cobra.Command{
	Use:   "licenses",
	Short: "Manage organization licenses and seat consumption",
}

var licensesListFlags struct {
	limit int
}

var licensesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List licenses across organizations",
	RunE:  runLicensesList,
}

var licensesStatusFlags struct {
	org string
}

var licensesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show seat usage for the current or specified org",
	RunE:  runLicensesStatus,
}

var licensesApplyFlags struct {
	org     string
	key     string
	keyFile string
	file    string
	tier    string
	maxSeats int
}

var licensesApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply license configuration to an organization",
	RunE:  runLicensesApply,
}

var licensesSeatsCmd = &cobra.Command{
	Use:   "seats",
	Short: "Show seat consumption by organization",
	RunE:  runLicensesSeats,
}

var licensesResyncCmd = &cobra.Command{
	Use:   "resync <org>",
	Short: "Refresh used-seat counters for an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runLicensesResync,
}

func init() {
	licensesListCmd.Flags().IntVar(&licensesListFlags.limit, "limit", 100, "maximum results")
	licensesStatusCmd.Flags().StringVar(&licensesStatusFlags.org, "org", "", "org UUID or slug")
	licensesApplyCmd.Flags().StringVar(&licensesApplyFlags.org, "org", "", "target org UUID (required)")
	licensesApplyCmd.Flags().StringVar(&licensesApplyFlags.key, "key", "", "license tier or JSON payload (not echoed)")
	licensesApplyCmd.Flags().StringVar(&licensesApplyFlags.keyFile, "key-file", "", "license JSON file")
	licensesApplyCmd.Flags().StringVar(&licensesApplyFlags.file, "file", "", "license JSON file")
	licensesApplyCmd.Flags().StringVar(&licensesApplyFlags.tier, "tier", "", "license tier")
	licensesApplyCmd.Flags().IntVar(&licensesApplyFlags.maxSeats, "max-seats", 0, "maximum seats")
	_ = licensesApplyCmd.MarkFlagRequired("org")

	licensesCmd.AddCommand(licensesListCmd, licensesStatusCmd, licensesApplyCmd, licensesSeatsCmd, licensesResyncCmd)
	rootCmd.AddCommand(licensesCmd)
}

func runLicensesList(cmd *cobra.Command, args []string) error {
	items, raw, err := fetchLicensesList(client.New(Cfg.Server, Cfg.APIKey), licensesListFlags.limit, 0)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ORG\tTIER\tSEATS\tUSED")
	for _, lic := range items {
		seats := "unlimited"
		if !lic.Unlimited && lic.MaxSeats >= 0 {
			seats = fmt.Sprintf("%d", lic.MaxSeats)
		}
		name := lic.OrgName
		if name == "" {
			name = lic.OrgID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", name, lic.Tier, seats, lic.UsedSeats)
	}
	return w.Flush()
}

func runLicensesStatus(cmd *cobra.Command, args []string) error {
	lic, raw, err := fetchLicenseStatus(client.New(Cfg.Server, Cfg.APIKey), licensesStatusFlags.org)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(lic)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "org=%s tier=%s %s\n", lic.OrgID, lic.Tier, formatSeatUsage(lic))
	return nil
}

func runLicensesApply(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload map[string]any
	var err error
	switch {
	case licensesApplyFlags.file != "":
		payload, err = loadJSONFile(licensesApplyFlags.file)
	case licensesApplyFlags.keyFile != "":
		payload, err = loadJSONFile(licensesApplyFlags.keyFile)
	default:
		payload, err = readLicenseKeyMaterial(licensesApplyFlags.key, "")
	}
	if err != nil {
		return err
	}
	if licensesApplyFlags.tier != "" {
		payload["tier"] = licensesApplyFlags.tier
	}
	if licensesApplyFlags.maxSeats > 0 {
		payload["maxSeats"] = licensesApplyFlags.maxSeats
	}
	lic, err := patchLicense(c, licensesApplyFlags.org, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(redactLicensePayload(map[string]any{
			"orgId":     lic.OrgID,
			"tier":      lic.Tier,
			"maxSeats":  lic.MaxSeats,
			"usedSeats": lic.UsedSeats,
		}))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Applied license to %s: %s\n", lic.OrgID, formatSeatUsage(lic))
	return nil
}

func runLicensesSeats(cmd *cobra.Command, args []string) error {
	return runLicensesList(cmd, args)
}

func runLicensesResync(cmd *cobra.Command, args []string) error {
	lic, err := resyncLicenseSeats(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(lic)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resynced seats for %s: %s\n", lic.OrgID, formatSeatUsage(lic))
	return nil
}