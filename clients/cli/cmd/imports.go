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
	Short: "Manage user CSV imports and content package imports",
}

var importsSubmitFlags struct {
	file          string
	org           string
	courseID      string
	dryRun        bool
	mergeStrategy string
	profile       string
	wait          bool
	timeout       time.Duration
}

var importsSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a user CSV or content package import job",
	RunE:  runImportsSubmit,
}

var importsStatusFlags struct {
	org      string
	content  bool
	wait     bool
	timeout  time.Duration
}

var importsStatusCmd = &cobra.Command{
	Use:   "status <job>",
	Short: "Get import job status",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportsStatus,
}

var importsListFlags struct {
	org      string
	courseID string
	content  bool
}

var importsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent import jobs",
	RunE:  runImportsList,
}

func init() {
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.file, "file", "", "CSV or package file (.imscc, .zip, .xml)")
	_ = importsSubmitCmd.MarkFlagRequired("file")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.org, "org", "", "target org UUID for user CSV imports")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.courseID, "course-id", "", "target course UUID for content package imports")
	importsSubmitCmd.Flags().BoolVar(&importsSubmitFlags.dryRun, "dry-run", false, "validate only (user CSV imports)")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.mergeStrategy, "merge-strategy", "upsert", "merge strategy")
	importsSubmitCmd.Flags().StringVar(&importsSubmitFlags.profile, "profile", "default", "import profile")
	importsSubmitCmd.Flags().BoolVar(&importsSubmitFlags.wait, "wait", false, "poll until the content import completes")
	importsSubmitCmd.Flags().DurationVar(&importsSubmitFlags.timeout, "timeout", 30*time.Minute, "max wait time for content imports")

	importsStatusCmd.Flags().StringVar(&importsStatusFlags.org, "org", "", "target org UUID (user CSV imports)")
	importsStatusCmd.Flags().BoolVar(&importsStatusFlags.content, "content", false, "query content package import status")
	importsStatusCmd.Flags().BoolVar(&importsStatusFlags.wait, "wait", false, "poll until the job completes")
	importsStatusCmd.Flags().DurationVar(&importsStatusFlags.timeout, "timeout", 10*time.Minute, "max wait time")

	importsListCmd.Flags().StringVar(&importsListFlags.org, "org", "", "target org UUID (user CSV imports)")
	importsListCmd.Flags().StringVar(&importsListFlags.courseID, "course-id", "", "filter content imports by course UUID")
	importsListCmd.Flags().BoolVar(&importsListFlags.content, "content", false, "list content package imports")

	importsCmd.AddCommand(importsSubmitCmd, importsStatusCmd, importsListCmd)
	rootCmd.AddCommand(importsCmd)
}

func runImportsSubmit(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if importsSubmitFlags.courseID != "" {
		jobID, raw, err := submitContentImport(c, importsSubmitFlags.courseID, importsSubmitFlags.file)
		if err != nil {
			return err
		}
		if importsSubmitFlags.wait {
			status, err := waitForContentImport(c, jobID, importsSubmitFlags.timeout, func(s contentImportJob) {
				if !globalFlags.jsonOut {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status=%s processed=%d/%d\n",
						s.Status, s.ProcessedItems, s.TotalItems)
				}
			})
			if err != nil {
				return err
			}
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Content import %s finished: succeeded=%d failed=%d\n",
				jobID, status.SucceededItems, status.FailedItems)
			return nil
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submitted content import %s\n", jobID)
		return nil
	}
	out, err := submitImportJob(
		c,
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
	if importsStatusFlags.content {
		if importsStatusFlags.wait {
			status, err := waitForContentImport(c, jobID, importsStatusFlags.timeout, func(s contentImportJob) {
				if !globalFlags.jsonOut {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status=%s processed=%d\n", s.Status, s.ProcessedItems)
				}
			})
			if err != nil {
				return err
			}
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Job %s: %s (succeeded=%d failed=%d)\n",
				jobID, status.Status, status.SucceededItems, status.FailedItems)
			return nil
		}
		status, raw, err := fetchContentImportStatus(c, jobID)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\tprocessed=%d\tsucceeded=%d\n",
			status.Status, status.ProcessedItems, status.SucceededItems)
		return nil
	}
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
	if importsListFlags.content {
		rows, raw, err := listContentImports(c, importsListFlags.courseID)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "JOB\tTYPE\tFILE\tSTATUS")
		for _, row := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.ID, row.ImportType, row.OriginalFilename, row.Status)
		}
		return w.Flush()
	}
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