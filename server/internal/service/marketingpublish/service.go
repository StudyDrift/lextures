// Package marketingpublish owns marketing-site rebuild coalescing and dispatch.
package marketingpublish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

const EventType = "marketing-content-publish"

type Build struct {
	ID              uuid.UUID  `json:"id"`
	Status          string     `json:"status"`
	Reason          string     `json:"reason"`
	Paths           []string   `json:"paths"`
	Urgent          bool       `json:"urgent"`
	NotBefore       time.Time  `json:"notBefore"`
	Deadline        time.Time  `json:"deadline"`
	DispatchedAt    *time.Time `json:"dispatchedAt"`
	CompletedAt     *time.Time `json:"completedAt"`
	ProviderRunID   *string    `json:"providerRunId"`
	ProviderRunURL  *string    `json:"providerRunUrl"`
	Error           *string    `json:"error"`
	RequestedBy     *uuid.UUID `json:"requestedBy"`
	CreatedAt       time.Time  `json:"createdAt"`
	EstimatedLiveAt *time.Time `json:"estimatedLiveAt"`
}
type Event struct {
	ID        uuid.UUID  `json:"id"`
	ArticleID *uuid.UUID `json:"articleId"`
	Path      string     `json:"path"`
	Action    string     `json:"action"`
	ActorID   *uuid.UUID `json:"actorId"`
	BuildID   *uuid.UUID `json:"buildId"`
	Error     *string    `json:"error"`
	CreatedAt time.Time  `json:"createdAt"`
}
type Settings struct {
	Provider, Repository, WorkflowRef string
	Token                             string
	QuietPeriod, MaxWait              time.Duration
}
type Dispatcher interface {
	Dispatch(context.Context, Settings, Build) error
	FindRun(context.Context, Settings, Build) (runID, runURL, status string, err error)
}
type Service struct {
	Pool       *pgxpool.Pool
	SecretsKey []byte
	Dispatcher Dispatcher
	Now        func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// RecordChange records the audit event and atomically coalesces a rebuild. It is
// intended to be called inside the article mutation transaction.
func RecordChange(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, path, action string, actor *uuid.UUID, urgent bool, now time.Time) (uuid.UUID, error) {
	var quiet, maxWait int
	if err := tx.QueryRow(ctx, `SELECT quiet_seconds,max_wait_seconds FROM marketing.content_build_settings WHERE singleton=true`).Scan(&quiet, &maxWait); err != nil {
		return uuid.Nil, err
	}
	reason := action
	if action == "scheduled_publish" {
		reason = "scheduled"
	}
	if action == "restore" {
		reason = "update"
	}
	notBefore := now.Add(time.Duration(quiet) * time.Second)
	if urgent {
		notBefore = now
	}
	var buildID uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_builds(reason,paths,urgent,not_before,deadline,requested_by)
	 VALUES($1,ARRAY[$2],$3,$4,$5,$6)
	 ON CONFLICT ((status)) WHERE status='pending' DO UPDATE SET
	 paths=(SELECT ARRAY(SELECT DISTINCT p FROM unnest(marketing.content_builds.paths || EXCLUDED.paths) p ORDER BY p)),
	 urgent=marketing.content_builds.urgent OR EXCLUDED.urgent,
	 reason=CASE WHEN EXCLUDED.urgent THEN EXCLUDED.reason ELSE marketing.content_builds.reason END,
	 not_before=CASE WHEN EXCLUDED.urgent THEN EXCLUDED.not_before ELSE GREATEST(marketing.content_builds.not_before,EXCLUDED.not_before) END
	 RETURNING id`, reason, path, urgent, notBefore, now.Add(time.Duration(maxWait)*time.Second), actor).Scan(&buildID)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO marketing.content_publish_events(article_id,path,action,actor_id,build_id) VALUES($1,$2,$3,$4,$5)`, articleID, path, action, actor, buildID)
	return buildID, err
}

func RecordEvent(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, path, action string, actor *uuid.UUID, eventError *string) error {
	_, err := tx.Exec(ctx, `INSERT INTO marketing.content_publish_events(article_id,path,action,actor_id,error) VALUES($1,$2,$3,$4,$5)`, articleID, path, action, actor, eventError)
	return err
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	var out Settings
	var cipher []byte
	var quiet, maxWait int
	err := s.Pool.QueryRow(ctx, `SELECT provider,repository,workflow_ref,token_ciphertext,quiet_seconds,max_wait_seconds FROM marketing.content_build_settings WHERE singleton=true`).Scan(&out.Provider, &out.Repository, &out.WorkflowRef, &cipher, &quiet, &maxWait)
	if err != nil {
		return out, err
	}
	out.QuietPeriod = time.Duration(quiet) * time.Second
	out.MaxWait = time.Duration(maxWait) * time.Second
	if len(cipher) > 0 {
		p, e := appsecrets.Decrypt(cipher, s.SecretsKey)
		if e != nil {
			return out, e
		}
		out.Token = string(p)
	}
	return out, nil
}

func (s *Service) UpdateSettings(ctx context.Context, in Settings, token *string) error {
	if in.Provider != "none" && in.Provider != "github" {
		return errors.New("provider must be none or github")
	}
	if in.QuietPeriod < 0 || in.QuietPeriod > 15*time.Minute || in.MaxWait <= 0 || in.MaxWait > time.Hour {
		return errors.New("invalid debounce settings")
	}
	var cipher any
	if token != nil {
		if *token != "" {
			c, e := appsecrets.Encrypt([]byte(*token), s.SecretsKey)
			if e != nil {
				return e
			}
			cipher = c
		}
	}
	_, err := s.Pool.Exec(ctx, `UPDATE marketing.content_build_settings SET provider=$1,repository=$2,workflow_ref=$3,quiet_seconds=$4,max_wait_seconds=$5,token_ciphertext=CASE WHEN $6::bytea IS NULL THEN token_ciphertext ELSE $6 END,updated_at=now() WHERE singleton=true`, in.Provider, in.Repository, in.WorkflowRef, int(in.QuietPeriod/time.Second), int(in.MaxWait/time.Second), cipher)
	return err
}

func scanBuild(row pgx.Row) (Build, error) {
	var b Build
	e := row.Scan(&b.ID, &b.Status, &b.Reason, &b.Paths, &b.Urgent, &b.NotBefore, &b.Deadline, &b.DispatchedAt, &b.CompletedAt, &b.ProviderRunID, &b.ProviderRunURL, &b.Error, &b.RequestedBy, &b.CreatedAt)
	if e == nil && (b.Status == "pending" || b.Status == "dispatched" || b.Status == "running") {
		estimate := b.NotBefore.Add(6 * time.Minute)
		if b.Urgent {
			estimate = b.CreatedAt.Add(6 * time.Minute)
		}
		b.EstimatedLiveAt = &estimate
	}
	return b, e
}

const buildCols = `id,status,reason,paths,urgent,not_before,deadline,dispatched_at,completed_at,provider_run_id,provider_run_url,error,requested_by,created_at`

func (s *Service) EnqueueManual(ctx context.Context, actor uuid.UUID) (Build, error) {
	now := s.now()
	var count int
	if e := s.Pool.QueryRow(ctx, `SELECT count(*) FROM marketing.content_builds WHERE reason='manual' AND requested_by=$1 AND created_at>$2`, actor, now.Add(-time.Hour)).Scan(&count); e != nil {
		return Build{}, e
	}
	if count >= 6 {
		return Build{}, ErrRateLimited
	}
	var id uuid.UUID
	e := s.Pool.QueryRow(ctx, `INSERT INTO marketing.content_builds(reason,paths,urgent,not_before,deadline,requested_by) VALUES('manual',ARRAY['/'],true,$1,$2,$3)
	 ON CONFLICT ((status)) WHERE status='pending' DO UPDATE SET urgent=true,not_before=EXCLUDED.not_before,paths=CASE WHEN marketing.content_builds.paths='{}' THEN ARRAY['/'] ELSE marketing.content_builds.paths END RETURNING id`, now, now.Add(30*time.Minute), actor).Scan(&id)
	if e != nil {
		return Build{}, e
	}
	return scanBuild(s.Pool.QueryRow(ctx, `SELECT `+buildCols+` FROM marketing.content_builds WHERE id=$1`, id))
}

var ErrRateLimited = errors.New("marketingpublish: manual rebuild rate limit exceeded")

func (s *Service) ListBuilds(ctx context.Context, limit int) ([]Build, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, e := s.Pool.Query(ctx, `SELECT `+buildCols+` FROM marketing.content_builds ORDER BY created_at DESC LIMIT $1`, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Build{}
	for rows.Next() {
		b, e := scanBuild(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *Service) ListEvents(ctx context.Context, article *uuid.UUID, limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, e := s.Pool.Query(ctx, `SELECT id,article_id,path,action,actor_id,build_id,error,created_at FROM marketing.content_publish_events WHERE ($1::uuid IS NULL OR article_id=$1) ORDER BY created_at DESC LIMIT $2`, article, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var v Event
		if e = rows.Scan(&v.ID, &v.ArticleID, &v.Path, &v.Action, &v.ActorID, &v.BuildID, &v.Error, &v.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Service) LatestForArticle(ctx context.Context, id uuid.UUID) (*Build, error) {
	b, e := scanBuild(s.Pool.QueryRow(ctx, `SELECT b.id,b.status,b.reason,b.paths,b.urgent,b.not_before,b.deadline,b.dispatched_at,b.completed_at,b.provider_run_id,b.provider_run_url,b.error,b.requested_by,b.created_at FROM marketing.content_builds b JOIN marketing.content_publish_events e ON e.build_id=b.id WHERE e.article_id=$1 ORDER BY e.created_at DESC LIMIT 1`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	return &b, e
}

// Run advances dispatches without holding a database lock during HTTP calls.
func (s *Service) Run(ctx context.Context) error {
	settings, e := s.Settings(ctx)
	if e != nil {
		return e
	}
	now := s.now()
	timedOut, _ := s.Pool.Exec(ctx, `UPDATE marketing.content_builds SET status='timed_out',completed_at=$1,error='Workflow did not complete within 30 minutes' WHERE status IN ('dispatched','running') AND dispatched_at<$2`, now, now.Add(-30*time.Minute))
	if timedOut.RowsAffected() > 0 {
		slog.Error("marketing_content_builds_total", "status", "timed_out", "count", timedOut.RowsAffected())
	}
	if settings.Provider == "none" {
		return nil
	}
	b, e := scanBuild(s.Pool.QueryRow(ctx, `UPDATE marketing.content_builds SET status='dispatched',dispatched_at=$1 WHERE id=(SELECT id FROM marketing.content_builds WHERE status='pending' AND (urgent OR not_before<=$1 OR deadline<=$1) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING `+buildCols, now))
	if e == nil {
		d := s.Dispatcher
		if d == nil {
			d = &GitHubDispatcher{Client: http.DefaultClient}
		}
		if e = d.Dispatch(ctx, settings, b); e != nil {
			_, _ = s.Pool.Exec(ctx, `UPDATE marketing.content_builds SET status='failed',completed_at=$2,error=$3 WHERE id=$1`, b.ID, now, e.Error())
			slog.Error("marketing_content_dispatch_failures_total", "build_id", b.ID, "err", e)
			return e
		}
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	rows, e := s.Pool.Query(ctx, `SELECT `+buildCols+` FROM marketing.content_builds WHERE status IN ('dispatched','running') ORDER BY dispatched_at LIMIT 20`)
	if e != nil {
		return e
	}
	defer rows.Close()
	var active []Build
	for rows.Next() {
		b, e := scanBuild(rows)
		if e != nil {
			return e
		}
		active = append(active, b)
	}
	for _, b := range active {
		d := s.Dispatcher
		if d == nil {
			d = &GitHubDispatcher{Client: http.DefaultClient}
		}
		rid, rurl, status, e := d.FindRun(ctx, settings, b)
		if e != nil {
			continue
		}
		if status == "" {
			continue
		}
		terminal := status == "succeeded" || status == "failed"
		_, e = s.Pool.Exec(ctx, `UPDATE marketing.content_builds SET status=$2,provider_run_id=NULLIF($3,''),provider_run_url=NULLIF($4,''),completed_at=CASE WHEN $5 THEN $6 ELSE completed_at END WHERE id=$1`, b.ID, status, rid, rurl, terminal, now)
		if e != nil {
			return e
		}
		if terminal {
			slog.Info("marketing_content_builds_total", "status", status, "build_id", b.ID, "marketing_content_build_latency_seconds", now.Sub(b.CreatedAt).Seconds())
		}
	}
	return nil
}

type GitHubDispatcher struct{ Client *http.Client }

func (g *GitHubDispatcher) request(ctx context.Context, s Settings, method, path string, body any) ([]byte, error) {
	if _, e := url.ParseRequestURI("https://api.github.com" + path); e != nil {
		return nil, e
	}
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, "https://api.github.com"+path, r)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, e := g.Client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github dispatch: status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
func (g *GitHubDispatcher) Dispatch(ctx context.Context, s Settings, b Build) error {
	sort.Strings(b.Paths)
	_, e := g.request(ctx, s, http.MethodPost, "/repos/"+s.Repository+"/dispatches", map[string]any{"event_type": EventType, "client_payload": map[string]any{"buildId": b.ID, "paths": b.Paths, "reason": b.Reason}})
	return e
}
func (g *GitHubDispatcher) FindRun(ctx context.Context, s Settings, b Build) (string, string, string, error) {
	q := url.Values{"event": {"repository_dispatch"}, "branch": {s.WorkflowRef}, "per_page": {"20"}}
	data, e := g.request(ctx, s, http.MethodGet, "/repos/"+s.Repository+"/actions/runs?"+q.Encode(), nil)
	if e != nil {
		return "", "", "", e
	}
	var v struct {
		Runs []struct {
			ID         json.Number `json:"id"`
			URL        string      `json:"html_url"`
			Status     string      `json:"status"`
			Conclusion string      `json:"conclusion"`
			Created    time.Time   `json:"created_at"`
		} `json:"workflow_runs"`
	}
	if e = json.Unmarshal(data, &v); e != nil {
		return "", "", "", e
	}
	for _, r := range v.Runs {
		if b.DispatchedAt != nil && r.Created.Before(b.DispatchedAt.Add(-15*time.Second)) {
			continue
		}
		status := "running"
		if r.Status == "completed" {
			status = "failed"
			if r.Conclusion == "success" {
				status = "succeeded"
			}
		}
		return r.ID.String(), r.URL, status, nil
	}
	return "", "", "", nil
}
