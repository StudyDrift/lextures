package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var signalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Classroom signal queues (anonymous questions, hall passes)",
}

var signalsCourseFlags struct {
	includeAddressed bool
}

var signalsCourseCmd = &cobra.Command{
	Use:   "course <course>",
	Short: "List anonymous question queue for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runSignalsCourse,
}

func init() {
	signalsCourseCmd.Flags().BoolVar(&signalsCourseFlags.includeAddressed, "include-addressed", false, "include addressed questions")

	signalsCmd.AddCommand(signalsCourseCmd)
	rootCmd.AddCommand(signalsCmd)
}

func runSignalsCourse(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	questions, raw, err := fetchCourseSignals(c, args[0], signalsCourseFlags.includeAddressed)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"questions": questions})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tADDRESSED\tCREATED\tQUESTION")
	for _, q := range questions {
		addr := "no"
		if q.Addressed {
			addr = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", q.ID, addr, q.CreatedAt, q.Question)
	}
	return w.Flush()
}