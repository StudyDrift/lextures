package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Manage scheduled background tasks",
}

var schedulerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled tasks",
	RunE:  runSchedulerList,
}

var schedulerRunNowCmd = &cobra.Command{
	Use:   "run-now <task>",
	Short: "Trigger a scheduled task immediately",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchedulerRunNow,
}

var schedulerEnableCmd = &cobra.Command{
	Use:   "enable <task>",
	Short: "Enable a scheduled task",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchedulerEnable,
}

var schedulerDisableCmd = &cobra.Command{
	Use:   "disable <task>",
	Short: "Disable a scheduled task",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchedulerDisable,
}

func init() {
	schedulerCmd.AddCommand(schedulerListCmd, schedulerRunNowCmd, schedulerEnableCmd, schedulerDisableCmd)
	rootCmd.AddCommand(schedulerCmd)
}

func runSchedulerList(cmd *cobra.Command, args []string) error {
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodGet, "/api/v1/admin/scheduler", nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Jobs []struct {
			Name       string `json:"name"`
			Spec       string `json:"spec"`
			Enabled    bool   `json:"enabled"`
			LastStatus string `json:"lastStatus"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tENABLED\tSPEC\tLAST")
	for _, j := range out.Jobs {
		_, _ = fmt.Fprintf(w, "%s\t%v\t%s\t%s\n", j.Name, j.Enabled, j.Spec, j.LastStatus)
	}
	return w.Flush()
}

func runSchedulerRunNow(cmd *cobra.Command, args []string) error {
	return postSchedulerAction(args[0], "trigger")
}

func runSchedulerEnable(cmd *cobra.Command, args []string) error {
	return postSchedulerAction(args[0], "enable")
}

func runSchedulerDisable(cmd *cobra.Command, args []string) error {
	return postSchedulerAction(args[0], "disable")
}

func postSchedulerAction(name, action string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/scheduler/" + url.PathEscape(name) + "/" + action
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if action == "trigger" {
		var out struct {
			JobID string `json:"jobId"`
		}
		if json.Unmarshal(body, &out) == nil && out.JobID != "" {
			fmt.Printf("Triggered %s (job %s)\n", name, out.JobID)
		}
	}
	return nil
}