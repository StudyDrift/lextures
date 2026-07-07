package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Inspect and control background jobs",
}

var jobsListFlags struct {
	status string
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent background jobs",
	RunE:  runJobsList,
}

var jobsGetFlags struct {
	wait    bool
	timeout time.Duration
}

var jobsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a job by id",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsGet,
}

var jobsRetryCmd = &cobra.Command{
	Use:   "retry <id>",
	Short: "Retry a dead-letter job (redrive)",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsRetry,
}

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel <id>",
	Short: "Cancel a pending job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsCancel,
}

var jobsDeadLetterCmd = &cobra.Command{
	Use:   "dead-letter",
	Short: "Inspect the job dead-letter queue",
}

var jobsDeadLetterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dead-letter jobs",
	RunE:  runJobsDeadLetterList,
}

var jobsDeadLetterRetryCmd = &cobra.Command{
	Use:   "retry <id>",
	Short: "Redrive a dead-letter job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsDeadLetterRetry,
}

func init() {
	jobsListCmd.Flags().StringVar(&jobsListFlags.status, "status", "", "filter by status")
	jobsGetCmd.Flags().BoolVar(&jobsGetFlags.wait, "wait", false, "poll until the job completes")
	jobsGetCmd.Flags().DurationVar(&jobsGetFlags.timeout, "timeout", 10*time.Minute, "max wait time")

	jobsDeadLetterCmd.AddCommand(jobsDeadLetterListCmd, jobsDeadLetterRetryCmd)
	jobsCmd.AddCommand(jobsListCmd, jobsGetCmd, jobsRetryCmd, jobsCancelCmd, jobsDeadLetterCmd)
	rootCmd.AddCommand(jobsCmd)
}

func runJobsList(cmd *cobra.Command, args []string) error {
	jobs, _, raw, err := fetchAdminJobs(client.New(Cfg.Server, Cfg.APIKey), jobsListFlags.status)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tATTEMPTS")
	for _, j := range jobs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d/%d\n", j.ID, j.JobType, j.Status, j.Attempts, j.MaxAttempts)
	}
	return w.Flush()
}

func runJobsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	jobID := args[0]
	if jobsGetFlags.wait {
		job, err := waitForAdminJob(c, jobID, jobsGetFlags.timeout, func(j adminJobRow) {
			if !globalFlags.jsonOut {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status=%s attempts=%d/%d\n", j.Status, j.Attempts, j.MaxAttempts)
			}
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(job)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Job %s finished: status=%s\n", jobID, job.Status)
		return nil
	}
	job, err := fetchAdminJobByID(c, jobID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(job)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\tattempts=%d/%d\n", job.ID, job.Status, job.Attempts, job.MaxAttempts)
	return nil
}

func runJobsRetry(cmd *cobra.Command, args []string) error {
	newID, err := redriveDeadLetterJob(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"jobId": newID})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Queued retry job %s\n", newID)
	return nil
}

func runJobsCancel(cmd *cobra.Command, args []string) error {
	if err := cancelAdminJob(client.New(Cfg.Server, Cfg.APIKey), args[0]); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled job")
	return nil
}

func runJobsDeadLetterList(cmd *cobra.Command, args []string) error {
	rows, raw, err := fetchDeadLetters(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tATTEMPTS\tREDRIVEN")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%v\n", row.ID, row.JobType, row.Attempts, row.Redriven)
	}
	return w.Flush()
}

func runJobsDeadLetterRetry(cmd *cobra.Command, args []string) error {
	return runJobsRetry(cmd, args)
}