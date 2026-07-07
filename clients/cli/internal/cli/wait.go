package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JobStatus is a minimal async job shape shared across domains.
type JobStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// JobPoller fetches current job status by id.
type JobPoller func(jobID string) (JobStatus, error)

var terminalStatuses = map[string]struct{}{
	"completed": {}, "complete": {}, "done": {}, "success": {},
	"failed": {}, "error": {}, "dead_letter": {}, "cancelled": {}, "canceled": {},
}

var successStatuses = map[string]struct{}{
	"completed": {}, "complete": {}, "done": {}, "success": {},
}

// WaitForJob polls until the job reaches a terminal state or timeout.
// Returns exit code hint: 0 success, 2 failure/timeout.
func WaitForJob(poll JobPoller, jobID string, timeout time.Duration, interval time.Duration, onTick func(JobStatus)) (JobStatus, int, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last JobStatus
	for {
		st, err := poll(jobID)
		if err != nil {
			return last, 2, err
		}
		last = st
		if onTick != nil {
			onTick(st)
		}
		status := strings.ToLower(strings.TrimSpace(st.Status))
		if _, ok := terminalStatuses[status]; ok {
			if _, ok := successStatuses[status]; ok {
				return last, 0, nil
			}
			msg := st.Error
			if msg == "" {
				msg = fmt.Sprintf("job %s finished with status %s", jobID, st.Status)
			}
			return last, 2, fmt.Errorf("%s", msg)
		}
		if time.Now().After(deadline) {
			return last, 2, fmt.Errorf("job %s did not complete within %s", jobID, timeout)
		}
		time.Sleep(interval)
	}
}

// ParseJobStatus extracts status from a JSON job body.
func ParseJobStatus(body []byte) (JobStatus, error) {
	var st JobStatus
	if err := json.Unmarshal(body, &st); err != nil {
		return JobStatus{}, err
	}
	if st.Status == "" {
		var wrap map[string]json.RawMessage
		if err := json.Unmarshal(body, &wrap); err == nil {
			if raw, ok := wrap["job"]; ok {
				_ = json.Unmarshal(raw, &st)
			}
		}
	}
	return st, nil
}