package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const jobPollInterval = 2 * time.Second

type adminJobRow struct {
	ID          string  `json:"id"`
	JobType     string  `json:"jobType"`
	Status      string  `json:"status"`
	Priority    int     `json:"priority"`
	Attempts    int     `json:"attempts"`
	MaxAttempts int     `json:"maxAttempts"`
	ErrorLog    *string `json:"errorLog,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

type deadLetterRow struct {
	ID       string  `json:"id"`
	JobType  string  `json:"jobType"`
	Attempts int     `json:"attempts"`
	ErrorLog *string `json:"errorLog,omitempty"`
	Redriven bool    `json:"redriven"`
}

func fetchAdminJobs(c *client.Client, statusFilter string) ([]adminJobRow, map[string]int, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("listing jobs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Stats map[string]int  `json:"stats"`
		Jobs  []adminJobRow   `json:"jobs"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, nil, body, fmt.Errorf("decoding response: %w", err)
	}
	jobs := out.Jobs
	if statusFilter != "" {
		filtered := make([]adminJobRow, 0, len(jobs))
		for _, j := range jobs {
			if strings.EqualFold(j.Status, statusFilter) {
				filtered = append(filtered, j)
			}
		}
		jobs = filtered
	}
	return jobs, out.Stats, body, nil
}

func fetchAdminJobByID(c *client.Client, jobID string) (adminJobRow, error) {
	jobs, _, _, err := fetchAdminJobs(c, "")
	if err != nil {
		return adminJobRow{}, err
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return j, nil
		}
	}
	return adminJobRow{}, fmt.Errorf("job %q not found", jobID)
}

func cancelAdminJob(c *client.Client, jobID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/jobs/"+url.PathEscape(jobID), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("cancelling job: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func redriveDeadLetterJob(c *client.Client, deadLetterID string) (string, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/jobs/dead-letters/"+url.PathEscape(deadLetterID)+"/redrive", nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", fmt.Errorf("retrying job: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	return out.JobID, nil
}

func fetchDeadLetters(c *client.Client) ([]deadLetterRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/jobs/dead-letters", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing dead letters: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		DeadLetters []deadLetterRow `json:"deadLetters"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.DeadLetters, body, nil
}

func jobTerminalStatus(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "complete", "failed", "dead_letter", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func jobSucceeded(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "complete":
		return true
	default:
		return false
	}
}

func waitForAdminJob(c *client.Client, jobID string, timeout time.Duration, onTick func(adminJobRow)) (adminJobRow, error) {
	deadline := time.Now().Add(timeout)
	var last adminJobRow
	for {
		job, err := fetchAdminJobByID(c, jobID)
		if err != nil {
			return last, err
		}
		last = job
		if onTick != nil {
			onTick(job)
		}
		if jobTerminalStatus(job.Status) {
			if jobSucceeded(job.Status) {
				return job, nil
			}
			return job, fmt.Errorf("job %s finished with status %s", jobID, job.Status)
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("job %s did not complete within %s", jobID, timeout)
		}
		time.Sleep(jobPollInterval)
	}
}