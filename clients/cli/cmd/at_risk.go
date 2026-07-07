package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var atRiskCmd = &cobra.Command{
	Use:   "at-risk",
	Short: "List and recompute at-risk student alerts",
}

var atRiskListFlags struct {
	org              string
	threshold        string
	factor           string
	export           bool
	out              string
	yes              bool
	includeResolved  bool
}

var atRiskListCmd = &cobra.Command{
	Use:   "list [course]",
	Short: "List at-risk alerts for a course or organization",
	RunE:  runAtRiskList,
}

var atRiskRecomputeFlags struct {
	course string
	org    string
}

var atRiskRecomputeCmd = &cobra.Command{
	Use:   "recompute",
	Short: "Trigger at-risk scoring for a course or the platform",
	RunE:  runAtRiskRecompute,
}

var atRiskFactorsFlags struct {
	user string
}

var atRiskFactorsCmd = &cobra.Command{
	Use:   "factors <course>",
	Short: "Show risk-factor history for an enrollment",
	Args:  cobra.ExactArgs(1),
	RunE:  runAtRiskFactors,
}

func init() {
	atRiskListCmd.Flags().StringVar(&atRiskListFlags.org, "org", "", "organization UUID (lists across org courses)")
	atRiskListCmd.Flags().StringVar(&atRiskListFlags.threshold, "threshold", "", "filter by risk level: high, medium, low, or numeric score")
	atRiskListCmd.Flags().StringVar(&atRiskListFlags.factor, "factor", "", "filter by top risk factor key")
	atRiskListCmd.Flags().BoolVar(&atRiskListFlags.export, "export", false, "write cohort to CSV in --out")
	atRiskListCmd.Flags().StringVar(&atRiskListFlags.out, "out", "", "output directory for --export")
	atRiskListCmd.Flags().BoolVar(&atRiskListFlags.yes, "yes", false, "confirm exporting FERPA-covered at-risk data")
	atRiskListCmd.Flags().BoolVar(&atRiskListFlags.includeResolved, "include-resolved", false, "include resolved and snoozed alerts")

	atRiskRecomputeCmd.Flags().StringVar(&atRiskRecomputeFlags.course, "course", "", "course code to recompute")
	atRiskRecomputeCmd.Flags().StringVar(&atRiskRecomputeFlags.org, "org", "", "organization UUID (informational)")

	atRiskFactorsCmd.Flags().StringVar(&atRiskFactorsFlags.user, "user", "", "enrollment UUID (required)")
	_ = atRiskFactorsCmd.MarkFlagRequired("user")

	atRiskCmd.AddCommand(atRiskListCmd, atRiskRecomputeCmd, atRiskFactorsCmd)
	rootCmd.AddCommand(atRiskCmd)
}

func runAtRiskList(cmd *cobra.Command, args []string) error {
	if (atRiskListFlags.export || atRiskListFlags.out != "") && !atRiskListFlags.yes {
		return fmt.Errorf("%s", ferpaAtRiskExportWarning)
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	var alerts []atRiskAlert
	var raw []byte
	var err error

	switch {
	case len(args) > 0:
		alerts, raw, err = fetchCourseAtRiskAlerts(c, args[0], atRiskListFlags.includeResolved)
	case atRiskListFlags.org != "":
		alerts, err = fetchOrgAtRiskAlerts(c, atRiskListFlags.org, atRiskListFlags.includeResolved)
	default:
		return fmt.Errorf("provide a course or --org")
	}
	if err != nil {
		return err
	}

	minScore, hasThreshold, err := parseAtRiskThreshold(atRiskListFlags.threshold)
	if err != nil {
		return err
	}
	alerts = filterAtRiskAlerts(alerts, minScore, hasThreshold, atRiskListFlags.factor)

	if atRiskListFlags.export || atRiskListFlags.out != "" {
		csvData, rows, err := atRiskAlertsToCSV(alerts)
		if err != nil {
			return err
		}
		outDir := atRiskListFlags.out
		if outDir == "" {
			outDir = "."
		}
		path := resolveExportPath(outDir, "at-risk-cohort.csv")
		if err := writeExportOutput(path, csvData); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported %d rows to %s\n", rows-1, path)
		return nil
	}

	if globalFlags.jsonOut {
		if len(args) > 0 && raw != nil {
			var parsed any
			if json.Unmarshal(raw, &parsed) == nil {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"alerts": alerts})
			}
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"alerts": alerts})
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "COURSE\tUSER\tNAME\tSCORE\tFACTOR\tSTATUS")
	for _, a := range alerts {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%.1f\t%s\t%s\n",
			a.CourseCode, a.UserID, a.DisplayName, a.Score, a.TopFactor, a.Status)
	}
	return w.Flush()
}

func runAtRiskRecompute(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := postAtRiskRecompute(c, atRiskRecomputeFlags.course)
	if err != nil {
		return err
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

func runAtRiskFactors(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	scores, raw, err := fetchEnrollmentAtRiskHistory(c, args[0], atRiskFactorsFlags.user)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"scores": scores})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "DATE\tSCORE\tFACTOR\tINACTIVE_DAYS")
	for _, s := range scores {
		_, _ = fmt.Fprintf(w, "%s\t%.1f\t%s\t%d\n", s.Date, s.Score, s.TopFactor, s.DaysInactive)
	}
	return w.Flush()
}