package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var behaviorCmd = &cobra.Command{
	Use:   "behavior",
	Short: "List and export PBIS behavior records",
}

var behaviorListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List PBIS points and incident counts per student",
	Args:  cobra.ExactArgs(1),
	RunE:  runBehaviorList,
}

var behaviorExportFlags struct {
	out    string
	format string
	yes    bool
}

var behaviorExportCmd = &cobra.Command{
	Use:   "export <course>",
	Short: "Export PBIS awards and referrals (FERPA-gated)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBehaviorExport,
}

var behaviorAwardFlags struct {
	user     string
	points   int
	category string
	note     string
}

var behaviorAwardCmd = &cobra.Command{
	Use:   "award <course>",
	Short: "Award PBIS points to a student",
	Args:  cobra.ExactArgs(1),
	RunE:  runBehaviorAward,
}

func init() {
	behaviorExportCmd.Flags().StringVar(&behaviorExportFlags.out, "out", "", "write export to file instead of stdout")
	behaviorExportCmd.Flags().StringVar(&behaviorExportFlags.format, "format", "csv", "export format: csv or json")
	behaviorExportCmd.Flags().BoolVar(&behaviorExportFlags.yes, "yes", false, "confirm FERPA-covered behavior export")

	behaviorAwardCmd.Flags().StringVar(&behaviorAwardFlags.user, "user", "", "student UUID or email (required)")
	_ = behaviorAwardCmd.MarkFlagRequired("user")
	behaviorAwardCmd.Flags().IntVar(&behaviorAwardFlags.points, "points", 1, "points to award")
	behaviorAwardCmd.Flags().StringVar(&behaviorAwardFlags.category, "category", "", "category UUID or name (default: first positive category)")
	behaviorAwardCmd.Flags().StringVar(&behaviorAwardFlags.note, "note", "", "optional note")

	behaviorCmd.AddCommand(behaviorListCmd, behaviorExportCmd, behaviorAwardCmd)
	rootCmd.AddCommand(behaviorCmd)
}

func runBehaviorList(cmd *cobra.Command, args []string) error {
	rows, err := listCourseBehavior(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STUDENT\tNAME\tPOINTS\tAWARDS\tREFERRALS")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\n",
			row.StudentID, row.DisplayName, row.TotalPoints, row.AwardCount, row.Referrals)
	}
	return w.Flush()
}

func runBehaviorExport(cmd *cobra.Command, args []string) error {
	if err := confirmBehaviorExport(behaviorExportFlags.yes); err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(behaviorExportFlags.format))
	if format != "csv" && format != "json" {
		return fmt.Errorf("unsupported format %q: use csv or json", behaviorExportFlags.format)
	}

	rows, err := exportCourseBehavior(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}

	var w = cmd.OutOrStdout()
	if behaviorExportFlags.out != "" {
		file, err := os.Create(behaviorExportFlags.out)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		w = file
	}

	if format == "json" || globalFlags.jsonOut {
		return json.NewEncoder(w).Encode(rows)
	}
	return writeBehaviorExportCSV(w, rows)
}

func runBehaviorAward(cmd *cobra.Command, args []string) error {
	if behaviorAwardFlags.points <= 0 {
		return fmt.Errorf("--points must be positive")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	studentID, err := resolveStudentUserID(c, behaviorAwardFlags.user)
	if err != nil {
		return err
	}
	orgID, err := fetchCourseOrgID(c, args[0])
	if err != nil {
		return err
	}
	categories, err := fetchBehaviorCategories(c, orgID)
	if err != nil {
		return err
	}
	category, err := resolveBehaviorCategory(categories, behaviorAwardFlags.category)
	if err != nil {
		return err
	}

	saved, err := awardPBISPoints(c, studentID, category.ID, behaviorAwardFlags.points, behaviorAwardFlags.note)
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"studentId":    studentID,
			"categoryId":   category.ID,
			"categoryName": category.Name,
			"points":       behaviorAwardFlags.points,
			"saved":        saved,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Awarded %d point(s) to %s in category %s\n",
		behaviorAwardFlags.points, behaviorAwardFlags.user, category.Name)
	return nil
}