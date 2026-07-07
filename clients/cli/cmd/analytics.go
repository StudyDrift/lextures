package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Fetch course and platform analytics",
}

var analyticsCourseCmd = &cobra.Command{
	Use:   "course <course>",
	Short: "Course engagement overview analytics",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyticsCourse,
}

var analyticsPlatformFlags struct {
	from string
	to   string
}

var analyticsPlatformCmd = &cobra.Command{
	Use:   "platform",
	Short: "Platform learning-activity analytics",
	RunE:  runAnalyticsPlatform,
}

func init() {
	analyticsPlatformCmd.Flags().StringVar(&analyticsPlatformFlags.from, "from", "", "range start (RFC3339)")
	analyticsPlatformCmd.Flags().StringVar(&analyticsPlatformFlags.to, "to", "", "range end (RFC3339)")

	analyticsCmd.AddCommand(analyticsCourseCmd, analyticsPlatformCmd)
	rootCmd.AddCommand(analyticsCmd)
}

func runAnalyticsCourse(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchCourseEngagementOverview(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Students []struct {
			EnrollmentID     string   `json:"enrollmentId"`
			DisplayName      string   `json:"displayName"`
			LoginsLast7Days  int      `json:"loginsLast7Days"`
			AvgTimeOnTaskMin float64  `json:"avgTimeOnTaskMin"`
			EngagementScore  float64  `json:"engagementScore"`
			AvgVideoWatchPct *float64 `json:"avgVideoWatchPct"`
		} `json:"students"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ENROLLMENT\tNAME\tLOGINS_7D\tTIME_MIN\tSCORE")
	for _, s := range out.Students {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%.1f\t%.1f\n",
			s.EnrollmentID, s.DisplayName, s.LoginsLast7Days, s.AvgTimeOnTaskMin, s.EngagementScore)
	}
	return w.Flush()
}

func runAnalyticsPlatform(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	report, raw, err := fetchLearningActivityReport(c, analyticsPlatformFlags.from, analyticsPlatformFlags.to)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "events=%d users=%d courses=%d days=%d kinds=%d\n",
		report.Summary.TotalEvents, report.Summary.UniqueUsers, report.Summary.UniqueCourses,
		len(report.ByDay), len(report.ByEventKind))
	return nil
}