package learningevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
	"github.com/lextures/lextures/server/internal/repos/lrsconfig"
	"github.com/lextures/lextures/server/internal/repos/lrsforwardjobs"
	"github.com/lextures/lextures/server/internal/repos/xapistatements"
)

var lrsRetryDelays = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

const lrsMaxAttempts = 5

// SweepForwardJobs processes due LRS forwarding jobs.
func SweepForwardJobs(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.XAPIEmissionEnabled || pool == nil {
		return
	}
	jobs, err := lrsforwardjobs.ListDue(ctx, pool, 50, now)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, job := range jobs {
		if err := deliverLRSJob(ctx, pool, cfg, client, job, now); err != nil {
			// logged inside
			_ = err
		}
	}
}

func deliverLRSJob(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, client *http.Client, job lrsforwardjobs.Job, now time.Time) error {
	row, err := xapistatements.GetByID(ctx, pool, job.StatementID)
	if err != nil || row == nil {
		return err
	}
	ep, sec, err := lrsconfig.GetForForward(ctx, pool, job.LRSEndpointID)
	if err != nil || ep == nil || !ep.Enabled {
		return err
	}
	var payload Payload
	if err := json.Unmarshal(row.FullJSON, &payload); err != nil {
		return err
	}
	body := payload.XAPI
	if len(body) == 0 {
		body = row.FullJSON
	}
	url := strings.TrimRight(ep.EndpointURL, "/")
	if !strings.HasSuffix(url, "/statements") {
		url += "/statements"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Experience-API-Version", "1.0.3")
	if ep.AuthType == "basic" && len(cfg.PlatformSecretsKey) == 32 && sec != nil {
		plain, derr := appsecrets.Decrypt(sec.PasswordCiphertext, cfg.PlatformSecretsKey)
		if derr == nil && ep.Username != nil && len(plain) > 0 {
			req.SetBasicAuth(*ep.Username, string(plain))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return markLRSFailure(ctx, pool, job, now, 0, err.Error())
	}
	defer resp.Body.Close()
	snip, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return lrsforwardjobs.MarkSent(ctx, pool, job.ID, now, resp.StatusCode, string(snip))
	}
	return markLRSFailure(ctx, pool, job, now, resp.StatusCode, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(snip)))
}

func markLRSFailure(ctx context.Context, pool *pgxpool.Pool, job lrsforwardjobs.Job, now time.Time, httpStatus int, msg string) error {
	attempts := job.Attempts + 1
	dead := attempts >= lrsMaxAttempts
	var next time.Time
	if !dead {
		idx := attempts - 1
		if idx >= len(lrsRetryDelays) {
			idx = len(lrsRetryDelays) - 1
		}
		next = now.Add(lrsRetryDelays[idx])
	}
	if err := lrsforwardjobs.MarkRetry(ctx, pool, job.ID, attempts, next, dead, httpStatus, msg); err != nil {
		return err
	}
	if dead {
		_ = lrsforwardjobs.InsertDeadLetter(ctx, pool, job.StatementID, job.StatementStoredAt, job.LRSEndpointID, msg)
	}
	return fmt.Errorf("%s", msg)
}
