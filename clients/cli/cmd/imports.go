package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var importsCmd = &cobra.Command{
	Use:   "imports",
	Short: "Manage bulk user import jobs",
}

var importsSubmitFlags struct {
	file          string
	org           string
	dryRun        bool
	mergeStrategy string
	profile       string
}

var importsSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a CSV import job",
	RunE:  runImportsSubmit,
}

var importsStatusFlags struct {
	org    string
	wait   bool
	timeout time.Duration
}

var importsStatusCmd = &cobra.Command{
	Use:   "status <job>",
	Short: "Get import job status",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportsStatus,
}

var importsListFlags struct {
	org string
}

var importsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent import jobs",
	RunE:  runImportsList,
}

func init() {
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.file, "file", "", "CSV file (required)")
	_ = importsSubmitCmd.MarkFlagRequired("file")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.org, "org", "", "target org UUID (global admin)")
	importsSubmitCmd.Flags().BoolVar(&importsSubmitFlags.dryRun, "dry-run", false, "validate only")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.mergeStrategy, "merge-strategy", "upsert", "merge strategy")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.profile, "profile", "default", "import profile")

	importsStatusCmd.Flags().StringVar(&importsStatusFlags.org, "org", "", "target org UUID (global admin)")
	importsStatusCmd.Flags().BoolVar(&importsStatusFlags.wait, "wait", false, "poll until the job completes")
	importsStatusCmd.Flags().DurationVar(&importsStatusFlags.timeout, "timeout", 10*time.Minute, "max wait time")

	importsListCmd.Flags().StringVar(&importsListFlags.org, "org", "", "target org UUID (global admin)")

	importsCmd.AddCommand(importsSubmitCmd, importsStatusCmd, importsListCmd)
	rootCmd.AddCommand(importsCmd)
}

func runImportsSubmit(cmd *cobra.Command, args []string) error {
	out, err := submitImportJob(
		client.New(Cfg.Server, Cfg.APIKey),
		importsSubmitFlags.org,
		importsSubmitFlags.file,
		importsSubmitFlags.dryRun,
		importsSubmitFlags.mergeStrategy,
		importsSubmitFlags.profile,
	)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	jobID, _ := out["jobId"].(string)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submitted import job %s\n", jobID)
	return nil
}

func runImportsStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	jobID := args[0]
	if importsStatusFlags.wait {
		status, err := waitForImportJob(c, importsStatusFlags.org, jobID, importsStatusFlags.timeout, func(s importJobStatus) {
			if !globalFlags.jsonOut {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status=%s processed=%d errors=%d\n",
					s.Status, s.ProcessedRows, s.ErrorRows)
			}
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Job %s finished: status=%s created=%d updated=%d skipped=%d errors=%d\n",
			jobID, status.Status, status.CreatedCount, status.UpdatedCount, status.SkippedCount, status.ErrorRows)
		return nil
	}
	status, raw, err := fetchImportJobStatus(c, importsStatusFlags.org, jobID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tprocessed=%d\terrors=%d\n", status.Status, status.ProcessedRows, status.ErrorRows)
	return nil
}

func runImportsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin-console/imports"
	if importsListFlags.org != "" {
		path += "?orgId=" + url.QueryEscape(importsListFlags.org)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var result struct {
		Items []importJobStatus `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "JOB\tSTATUS\tPROCESSED\tERRORS")
	for _, item := range result.Items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", item.JobID, item.Status, item.ProcessedRows, item.ErrorRows)
	}
	return w.Flush()
}