package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Fetch instructor insights and student risk context",
}

var insightsCourseCmd = &cobra.Command{
	Use:   "course <course>",
	Short: "Weekly instructor insight summary for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runInsightsCourse,
}

var insightsStudentFlags struct {
	enrollment string
}

var insightsStudentCmd = &cobra.Command{
	Use:   "student <course>",
	Short: "Risk-factor history for a student enrollment",
	Args:  cobra.ExactArgs(1),
	RunE:  runInsightsStudent,
}

func init() {
	insightsStudentCmd.Flags().StringVar(&insightsStudentFlags.enrollment, "enrollment", "", "enrollment UUID (required)")
	_ = insightsStudentCmd.MarkFlagRequired("enrollment")

	insightsCmd.AddCommand(insightsCourseCmd, insightsStudentCmd)
	rootCmd.AddCommand(insightsCmd)
}

func runInsightsCourse(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchCourseInsights(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		WeekOf         string `json:"weekOf"`
		WorkingWell    []struct {
			SignalKey string `json:"signalKey"`
			Title     string `json:"title"`
		} `json:"workingWell"`
		NeedsAttention []struct {
			SignalKey string `json:"signalKey"`
			Title     string `json:"title"`
		} `json:"needsAttention"`
		GeneratedAt string `json:"generatedAt"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "week=%s generated=%s\n", out.WeekOf, out.GeneratedAt)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "working_well:")
	for _, s := range out.WorkingWell {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", s.Title, s.SignalKey)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "needs_attention:")
	for _, s := range out.NeedsAttention {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", s.Title, s.SignalKey)
	}
	return nil
}

func runInsightsStudent(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	scores, raw, err := fetchEnrollmentAtRiskHistory(c, args[0], insightsStudentFlags.enrollment)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "DATE\tSCORE\tFACTOR")
	for _, s := range scores {
		_, _ = fmt.Fprintf(w, "%s\t%.1f\t%s\n", s.Date, s.Score, s.TopFactor)
	}
	return w.Flush()
}