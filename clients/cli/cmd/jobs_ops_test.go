package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestJobTerminalStatus(t *testing.T) {
	if !jobTerminalStatus("completed") || !jobTerminalStatus("failed") {
		t.Fatal("expected terminal statuses")
	}
	if jobTerminalStatus("running") {
		t.Fatal("running should not be terminal")
	}
}

func TestJobsGet_WaitComplete(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "running"
		if calls >= 2 {
			status = "completed"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jobs": []adminJobRow{{ID: "job-1", JobType: "import", Status: status, MaxAttempts: 5, Attempts: 1}},
			"stats": map[string]int{},
		})
	}))
	defer srv.Close()

	jobsGetFlags.wait = true
	jobsGetFlags.timeout = 5 * jobPollInterval
	defer func() {
		jobsGetFlags.wait = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	jobsGetCmd.SetOut(&out)
	if err := jobsGetCmd.RunE(jobsGetCmd, []string{"job-1"}); err != nil {
		t.Fatalf("jobs get --wait: %v", err)
	}
	if !strings.Contains(out.String(), "finished") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestJobsRetry_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "redrive") {
			_ = json.NewEncoder(w).Encode(map[string]string{"jobId": "job-2"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	jobsRetryCmd.SetOut(&out)
	if err := jobsRetryCmd.RunE(jobsRetryCmd, []string{"dl-1"}); err != nil {
		t.Fatalf("jobs retry: %v", err)
	}
	if !strings.Contains(out.String(), "job-2") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestQuarantineRelease_RequiresYes(t *testing.T) {
	Cfg = &config.Config{Server: "http://127.0.0.1:9", APIKey: "test-key"}
	quarantineReleaseFlags.yes = false
	if err := quarantineReleaseCmd.RunE(quarantineReleaseCmd, []string{"obj-1"}); err == nil {
		t.Fatal("expected --yes requirement")
	}
}

func TestSchedulerList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/scheduler" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{{"name": "nightly", "spec": "0 0 * * *", "enabled": true}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	schedulerListCmd.SetOut(&out)
	if err := schedulerListCmd.RunE(schedulerListCmd, nil); err != nil {
		t.Fatalf("scheduler list: %v", err)
	}
	if !strings.Contains(out.String(), "nightly") {
		t.Fatalf("output = %q", out.String())
	}
}