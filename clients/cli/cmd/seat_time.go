package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var seatTimeCmd = &cobra.Command{
	Use:   "seat-time",
	Short: "Seat-time and CEU compliance reports",
}

var seatTimeReportFlags struct {
	user string
}

var seatTimeReportCmd = &cobra.Command{
	Use:   "report <course>",
	Short: "Print per-student seat-time minutes for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runSeatTimeReport,
}

func init() {
	seatTimeReportCmd.Flags().StringVar(&seatTimeReportFlags.user, "user", "", "filter to one student UUID or email")

	seatTimeCmd.AddCommand(seatTimeReportCmd)
	rootCmd.AddCommand(seatTimeCmd)
}

func runSeatTimeReport(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, err := fetchSeatTimeReport(c, args[0])
	if err != nil {
		return err
	}

	if userFilter := strings.TrimSpace(seatTimeReportFlags.user); userFilter != "" {
		userID, err := resolveStudentUserID(c, userFilter)
		if err != nil {
			return err
		}
		filtered := make([]seatTimeStudentRow, 0)
		for _, row := range rows {
			if row.UserID == userID {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USER\tNAME\tMINUTES\tHOURS\tCEU\tREQUIRED_HOURS")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%.2f\t%t\t%.2f\n",
			row.UserID, row.DisplayName, row.TotalMinutes, row.ContactHours, row.CEUEarned, row.RequiredHours)
	}
	return w.Flush()
}