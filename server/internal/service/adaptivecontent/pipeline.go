package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
	"github.com/lextures/lextures/server/internal/repos/concepts"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	coppasvc "github.com/lextures/lextures/server/internal/service/coppa"
)

// Job priority bands (higher = preferred at claim).
const (
	PriorityOnDemand  int16 = 10 // cache-miss / serve path
	PriorityActivation int16 = 5  // unit activated pre-warm
	PriorityPrewarm   int16 = 3  // instructor pre-warm / cohort growth
	PriorityRegen     int16 = 4  // base content edit regen
)

// EventJobEnqueued / pipeline audit events.
const (
	EventJobEnqueued     = "generation_job_enqueued"
	EventJobCompleted    = "generation_job_completed"
	EventJobFailed       = "generation_job_failed"
	EventJobDeadLetter   = "generation_job_dead_letter"
	EventPipelinePaused  = "generation_pipeline_paused"
	EventPrewarmStarted  = "prewarm_started"
)

// Retry backoff schedule for transient model failures (AC.4 FR-6).
var jobBackoff = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

// JobBackoff returns the wait after the given attempt number (1-based after claim).
func JobBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	idx := attempt - 1
	if idx >= len(jobBackoff) {
		idx = len(jobBackoff) - 1
	}
	return jobBackoff[idx]
}

// IsTransientGenError reports whether a generation error should be retried.
func IsTransientGenError(err error) bool {
	if err == nil {
		return false
	}
	// Permanent / non-retry
	if errors.Is(err, ErrBudgetExhausted) ||
		errors.Is(err, ErrGatewayDenied) ||
		errors.Is(err, ErrRejectedFidelity) ||
		errors.Is(err, ErrRejectedSafety) {
		return false
	}
	// Default: model/parse/network treated as transient when wrapped as ErrGenerationFailed
	// or any other error (timeouts, 5xx).
	return true
}

// Enqueue enqueues generation for (unit, signature, version) with dedupe (FR-2).
// Skips neutral/base signatures and when a ready variant already exists.
func Enqueue(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID uuid.UUID,
	signature string,
	contentVersion int32,
	priority int16,
) (jobID uuid.UUID, created bool, err error) {
	signature = strings.TrimSpace(signature)
	if signature == "" || signature == NeutralSignature || strings.EqualFold(signature, "neutral") {
		return uuid.Nil, false, nil
	}
	if contentVersion <= 0 {
		contentVersion = 1
	}
	// Skip if ready cache exists.
	ready, err := acrepo.HasReadyVariant(ctx, pool, unitID, signature, contentVersion)
	if err != nil {
		return uuid.Nil, false, err
	}
	if ready {
		IncCacheHit()
		return uuid.Nil, false, nil
	}
	id, created, err := acrepo.EnqueueJob(ctx, pool, acrepo.EnqueueJobParams{
		UnitID:           unitID,
		ProfileSignature: signature,
		ContentVersion:   contentVersion,
		Priority:         priority,
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	if created {
		RefreshQueueMetrics(ctx, pool)
	}
	return id, created, nil
}

// EnqueueOnCacheMiss is the serve-path helper (AC.4 FR-3 / AC.6): never blocks;
// enqueues for next time. Returns base-serve instruction always.
func EnqueueOnCacheMiss(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, signature string, contentVersion int32) {
	IncCacheMiss()
	_, _, err := Enqueue(ctx, pool, unitID, signature, contentVersion, PriorityOnDemand)
	if err != nil {
		slog.Warn("adaptivecontent: enqueue on cache miss failed", "unit_id", unitID, "err", err)
	}
}

// LookupReadyVariant returns an approved/auto_served variant for serve (AC.6 prep).
// On miss, optionally enqueues generation when enqueueOnMiss is true.
func LookupReadyVariant(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID uuid.UUID,
	signature string,
	contentVersion int32,
	enqueueOnMiss bool,
) (*acrepo.VariantRow, error) {
	start := time.Now()
	defer func() {
		// serve lookup latency is observed via cache hit path; keep light
		_ = start
	}()
	v, err := acrepo.GetVariantBySignature(ctx, pool, unitID, signature, contentVersion)
	if err != nil {
		return nil, err
	}
	if v != nil && (v.Status == "approved" || v.Status == "auto_served") {
		IncCacheHit()
		return v, nil
	}
	if enqueueOnMiss {
		EnqueueOnCacheMiss(ctx, pool, unitID, signature, contentVersion)
	} else {
		IncCacheMiss()
	}
	return nil, nil
}

// PrewarmUnit enqueues generation for the top-N cohort signatures (FR-8).
func PrewarmUnit(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow, maxN int, priority int16) (enqueued int, err error) {
	if maxN <= 0 {
		settings, _ := acrepo.GetSettings(ctx, pool, unit.CourseID)
		maxN = 12
		if settings != nil && settings.MaxPrewarmVariants > 0 {
			maxN = int(settings.MaxPrewarmVariants)
		}
	}
	if maxN > 100 {
		maxN = 100
	}
	sigs, err := acrepo.ListTopSignatures(ctx, pool, unit.ID, maxN)
	if err != nil {
		return 0, err
	}
	// Deduplicate signatures (same sig may appear under different emphasis labels in aggregate).
	seen := map[string]struct{}{}
	for _, s := range sigs {
		if _, ok := seen[s.ProfileSignature]; ok {
			continue
		}
		seen[s.ProfileSignature] = struct{}{}
		_, created, eerr := Enqueue(ctx, pool, unit.ID, s.ProfileSignature, unit.ContentVersion, priority)
		if eerr != nil {
			slog.Warn("adaptivecontent: prewarm enqueue failed", "unit_id", unit.ID, "sig", s.ProfileSignature, "err", eerr)
			continue
		}
		if created {
			enqueued++
		}
		if len(seen) >= maxN {
			break
		}
	}
	if enqueued > 0 {
		uid := unit.ID
		_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, nil, nil, EventPrewarmStarted, map[string]any{
			"enqueued":       enqueued,
			"maxPrewarm":     maxN,
			"contentVersion": unit.ContentVersion,
		})
	}
	return enqueued, nil
}

// EnqueueRegenForUnit re-enqueues signatures after a base-content edit (AC-7).
func EnqueueRegenForUnit(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow) (int, error) {
	settings, _ := acrepo.GetSettings(ctx, pool, unit.CourseID)
	maxN := 12
	if settings != nil && settings.MaxPrewarmVariants > 0 {
		maxN = int(settings.MaxPrewarmVariants)
	}
	sigs, err := acrepo.ListSignaturesNeedingRegen(ctx, pool, unit.ID, unit.ContentVersion, maxN)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, sig := range sigs {
		_, created, eerr := Enqueue(ctx, pool, unit.ID, sig, unit.ContentVersion, PriorityRegen)
		if eerr != nil {
			continue
		}
		if created {
			n++
		}
	}
	return n, nil
}

// WorkerDeps are process dependencies for the generation worker.
type WorkerDeps struct {
	Pool     *pgxpool.Pool
	Client   aiprovider.ScopedCompleter
	WorkerID string
	// MaxAttempts defaults to acrepo.DefaultJobMaxAttempts.
	MaxAttempts int
	// Concurrency is how many jobs to claim per tick.
	Concurrency int
	// Limiter optional; defaults to GlobalModelLimiter().
	Limiter *GlobalRateLimiter
	// Now optional clock for tests.
	Now func() time.Time
}

func (d WorkerDeps) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d WorkerDeps) workerID() string {
	if d.WorkerID != "" {
		return d.WorkerID
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "worker"
	}
	return "ac-" + host
}

// RunOnce claims and processes up to Concurrency jobs.
func (d WorkerDeps) RunOnce(ctx context.Context) (processed int) {
	if d.Pool == nil {
		return 0
	}
	if KillSwitchEngaged() {
		return 0
	}
	paused, err := acrepo.GetPlatformGenerationPaused(ctx, d.Pool)
	if err != nil {
		slog.Warn("adaptivecontent: platform pause read failed", "err", err)
	}
	if paused {
		return 0
	}

	limit := d.Concurrency
	if limit <= 0 {
		limit = 2
	}
	jobs, err := acrepo.ClaimJobs(ctx, d.Pool, d.workerID(), limit, d.now(), acrepo.DefaultJobVisibilityTimeout)
	if err != nil {
		slog.Warn("adaptivecontent: claim jobs failed", "err", err)
		return 0
	}
	SetInflight(float64(len(jobs)))
	for _, job := range jobs {
		d.processJob(ctx, job)
		processed++
	}
	RefreshQueueMetrics(ctx, d.Pool)
	return processed
}

func (d WorkerDeps) processJob(ctx context.Context, job acrepo.JobRow) {
	start := time.Now()
	defer func() {
		ObserveJobLatency(float64(time.Since(start).Milliseconds()))
	}()

	unit, err := acrepo.GetUnitByID(ctx, d.Pool, job.UnitID)
	if err != nil || unit == nil {
		_ = acrepo.CancelJob(ctx, d.Pool, job.ID, "unit_missing", d.now())
		return
	}

	// Course / platform gates.
	enabled, _ := acrepo.AdaptiveContentEnabledForCourse(ctx, d.Pool, unit.CourseID)
	if !ActiveForCourse(enabled) {
		_ = acrepo.CancelJob(ctx, d.Pool, job.ID, "course_disabled_or_kill_switch", d.now())
		return
	}
	settings, _ := acrepo.GetSettings(ctx, d.Pool, unit.CourseID)
	if settings != nil && settings.GenerationPaused {
		// Release without burning attempts; retry later.
		_ = acrepo.ReleaseJobToPending(ctx, d.Pool, job.ID, d.now().Add(30*time.Second), d.now())
		return
	}
	// Content version drift: cancel stale job.
	if unit.ContentVersion != job.ContentVersion {
		_ = acrepo.CancelJob(ctx, d.Pool, job.ID, "content_version_stale", d.now())
		return
	}
	// Already ready?
	ready, _ := acrepo.HasReadyVariant(ctx, d.Pool, unit.ID, job.ProfileSignature, job.ContentVersion)
	if ready {
		IncCacheHit()
		_ = acrepo.CompleteJob(ctx, d.Pool, job.ID, d.now())
		return
	}

	// Budget check.
	check, err := CheckCourseBudget(ctx, d.Pool, unit.CourseID, DefaultEstimateTokens, d.now())
	if err != nil {
		d.failTransient(ctx, job, fmt.Errorf("budget check: %w", err))
		return
	}
	if !check.Allowed {
		IncBudgetExhausted()
		uid := unit.ID
		RecordBudgetExhaustedEvent(ctx, d.Pool, unit.CourseID, &uid, map[string]any{
			"profileSignature": job.ProfileSignature,
			"contentVersion":   job.ContentVersion,
			"tokensUsed":       check.Used,
			"budget":           check.Budget,
		})
		_ = acrepo.CancelJob(ctx, d.Pool, job.ID, "budget_exhausted", d.now())
		return
	}

	// Global rate limit.
	lim := d.Limiter
	if lim == nil {
		lim = GlobalModelLimiter()
	}
	if !lim.TryAcquire() {
		IncJobRetry()
		_ = acrepo.ReleaseJobToPending(ctx, d.Pool, job.ID, d.now().Add(5*time.Second), d.now())
		return
	}

	// Load base content.
	base, err := coursemodulecontent.GetForCourseItem(ctx, d.Pool, unit.CourseID, unit.BaseContentItemID)
	if err != nil || base == nil {
		d.failTransient(ctx, job, fmt.Errorf("base content: %v", err))
		return
	}

	profile := d.profileForJob(ctx, *unit, job.ProfileSignature)
	if profile.IsNeutral || profile.ProfileSignature == NeutralSignature {
		_ = acrepo.CompleteJob(ctx, d.Pool, job.ID, d.now())
		return
	}

	keyTerms, _ := acrepo.ListKeyTerms(ctx, d.Pool, unit.ID)
	termStrs := make([]string, 0, len(keyTerms))
	for _, kt := range keyTerms {
		if kt.MustAppear {
			termStrs = append(termStrs, kt.Term)
		}
	}
	conceptLabels, misLabels := loadLabels(ctx, d.Pool, unit.CourseID, profile)

	axes := unit.AllowedAxes
	if len(axes) == 0 && settings != nil {
		axes = settings.AllowedAxes
	}
	requireApproval := settings != nil && settings.RequireInstructorApproval
	// AC.8: force instructor approval for COPPA minors and EU AI Act high-risk policy.
	coppaMinor := false
	if row, err := acrepo.GetAnyProfileBySignature(ctx, d.Pool, unit.ID, job.ProfileSignature); err == nil && row != nil && row.UserID != uuid.Nil {
		if status, err := coppasvc.GetUserConsentStatus(ctx, d.Pool, row.UserID); err == nil && status.CoppaMinor {
			coppaMinor = true
		}
	}
	if ForceInstructorApproval(coppaMinor, EUHighRiskPolicyEnabled()) {
		requireApproval = true
	}

	sysPrompt := DefaultSystemPrompt
	if s, err := systemprompts.GetByKey(ctx, d.Pool, PromptKey); err == nil && strings.TrimSpace(s) != "" {
		sysPrompt = s
	}
	model := userai.DefaultCourseSetupModelID

	genIn := GenerateInput{
		BaseMarkdown:              base.Markdown,
		BaseTitle:                 base.Title,
		Profile:                   profile,
		AllowedAxes:               axes,
		KeyTerms:                  termStrs,
		ConceptLabels:             conceptLabels,
		MisconceptionLabels:       misLabels,
		Model:                     model,
		SystemPrompt:              sysPrompt,
		PromptVersion:             PromptVersionCurrent,
		MinFidelity:               unit.MinFidelity,
		ContentVersion:            unit.ContentVersion,
		RequireInstructorApproval: requireApproval,
		GatewayAllowed:            true, // pipeline is instructor/platform side; student opt-out is serve-time (AC.6)
		BudgetExhausted:           false,
	}

	variant, callMeta, genErr := GenerateVariant(ctx, d.Client, genIn)

	// Log usage when the model was involved.
	tokens := int64(variant.PromptTokens + variant.CompletionTokens)
	if callMeta.Usage.TotalTokens > 0 {
		tokens = int64(callMeta.Usage.TotalTokens)
	}
	if tokens > 0 || callMeta.Usage.HasData() {
		courseID := unit.CourseID
		_ = aiusage.Insert(ctx, d.Pool, aiusage.EntryFromCallMeta(
			nil, &courseID, acrepo.FeatureAdaptiveContent, callMeta, callMeta.Usage,
			genErr == nil || errors.Is(genErr, ErrRejectedFidelity) || errors.Is(genErr, ErrRejectedSafety),
		))
		_ = RecordTokenUsage(ctx, d.Pool, unit.CourseID, tokens, d.now())
	}

	// Permanent rejection: store variant and complete job (no retry).
	if errors.Is(genErr, ErrRejectedFidelity) || errors.Is(genErr, ErrRejectedSafety) {
		d.persistVariant(ctx, *unit, variant)
		uid := unit.ID
		_ = acrepo.InsertEvent(ctx, d.Pool, unit.CourseID, &uid, nil, nil, EventVariantRejected, map[string]any{
			"profileSignature": variant.ProfileSignature,
			"status":           variant.Status,
			"fidelityScore":    variant.FidelityScore,
			"jobId":            job.ID,
		})
		_ = acrepo.CompleteJob(ctx, d.Pool, job.ID, d.now())
		return
	}

	// Budget exhausted from engine (defensive).
	if errors.Is(genErr, ErrBudgetExhausted) {
		IncBudgetExhausted()
		_ = acrepo.CancelJob(ctx, d.Pool, job.ID, "budget_exhausted", d.now())
		return
	}

	// Transient failure → retry with backoff.
	if genErr != nil && IsTransientGenError(genErr) {
		d.failTransient(ctx, job, genErr)
		return
	}

	// Success (or non-fallback permanent path).
	if !variant.IsNeutralLike() && variant.ProfileSignature != "" {
		d.persistVariant(ctx, *unit, variant)
		uid := unit.ID
		eventType := EventVariantGenerated
		if variant.Status == "rejected" {
			eventType = EventVariantRejected
		}
		_ = acrepo.InsertEvent(ctx, d.Pool, unit.CourseID, &uid, nil, nil, eventType, map[string]any{
			"profileSignature": variant.ProfileSignature,
			"status":           variant.Status,
			"fidelityScore":    variant.FidelityScore,
			"jobId":            job.ID,
			"model":            variant.Model,
		})
	}
	_ = acrepo.CompleteJob(ctx, d.Pool, job.ID, d.now())
	_ = acrepo.InsertEvent(ctx, d.Pool, unit.CourseID, &unit.ID, nil, nil, EventJobCompleted, map[string]any{
		"jobId":            job.ID,
		"profileSignature": job.ProfileSignature,
		"status":           variant.Status,
	})
}

func (d WorkerDeps) failTransient(ctx context.Context, job acrepo.JobRow, genErr error) {
	IncJobRetry()
	max := d.MaxAttempts
	if max <= 0 {
		max = acrepo.DefaultJobMaxAttempts
	}
	msg := genErr.Error()
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	dead, err := acrepo.FailJob(ctx, d.Pool, job.ID, job.Attempts, max, msg, d.now(), JobBackoff(int(job.Attempts)))
	if err != nil {
		slog.Warn("adaptivecontent: fail job", "job_id", job.ID, "err", err)
		return
	}
	unit, _ := acrepo.GetUnitByID(ctx, d.Pool, job.UnitID)
	if unit != nil {
		uid := unit.ID
		ev := EventJobFailed
		if dead {
			ev = EventJobDeadLetter
		}
		_ = acrepo.InsertEvent(ctx, d.Pool, unit.CourseID, &uid, nil, nil, ev, map[string]any{
			"jobId":            job.ID,
			"profileSignature": job.ProfileSignature,
			"attempts":         job.Attempts,
			"error":            msg,
			"deadLetter":       dead,
		})
	}
}

func (d WorkerDeps) persistVariant(ctx context.Context, unit acrepo.UnitRow, variant Variant) {
	fid := variant.FidelityScore
	row := acrepo.VariantRow{
		UnitID:           unit.ID,
		ProfileSignature: variant.ProfileSignature,
		AxesApplied:      variant.AxesApplied,
		VariantMarkdown:  variant.Markdown,
		Model:            variant.Model,
		FidelityScore:    &fid,
		SafetyFlags:      acrepo.FlagsJSON(variant.SafetyFlags),
		Status:           variant.Status,
		PromptVersion:    variant.PromptVersion,
		ContentVersion:   variant.ContentVersion,
		PromptTokens:     int32(variant.PromptTokens),
		CompletionTokens: int32(variant.CompletionTokens),
		A11yFlags:        acrepo.FlagsJSON(variant.A11yFlags),
	}
	if _, err := acrepo.UpsertVariant(ctx, d.Pool, row); err != nil {
		slog.Warn("adaptivecontent: persist variant failed", "unit_id", unit.ID, "err", err)
	}
}

func (d WorkerDeps) profileForJob(ctx context.Context, unit acrepo.UnitRow, signature string) ProfileResult {
	row, err := acrepo.GetAnyProfileBySignature(ctx, d.Pool, unit.ID, signature)
	if err == nil && row != nil {
		return ProfileResultFromRow(row)
	}
	// Minimal reconstruction when only the signature is known.
	return ProfileResult{
		EmphasisMode:     EmphasisReinforce,
		TargetBloom:      "understand",
		ProfileSignature: signature,
		IsNeutral:        false,
		ReadingLevelPref: "default",
		ModalityPref:     "default",
		AxisSet:          unit.AllowedAxes,
		Payload:          ProfilePayload{},
	}
}

// ProfileResultFromRow rebuilds a ProfileResult from a stored adaptation_profiles row.
func ProfileResultFromRow(row *acrepo.ProfileRow) ProfileResult {
	if row == nil {
		return ProfileResult{}
	}
	emphasis := EmphasisReinforce
	if row.EmphasisMode != nil && *row.EmphasisMode != "" {
		emphasis = *row.EmphasisMode
	}
	bloom := "understand"
	if row.TargetBloom != nil && *row.TargetBloom != "" {
		bloom = *row.TargetBloom
	}
	reading := "default"
	if row.ReadingLevelPref != nil && *row.ReadingLevelPref != "" {
		reading = *row.ReadingLevelPref
	}
	modality := "default"
	if row.ModalityPref != nil && *row.ModalityPref != "" {
		modality = *row.ModalityPref
	}
	payload := ProfilePayload{}
	if len(row.PayloadJSON) > 0 {
		_ = json.Unmarshal(row.PayloadJSON, &payload)
	}
	return ProfileResult{
		EmphasisMode:     emphasis,
		TargetBloom:      bloom,
		ProfileSignature: row.ProfileSignature,
		IsNeutral:        row.IsNeutral,
		ReadingLevelPref: reading,
		ModalityPref:     modality,
		AxisSet:          row.AxisSet,
		Payload:          payload,
	}
}

func loadLabels(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, profile ProfileResult) (map[uuid.UUID]string, map[string]string) {
	conceptLabels := map[uuid.UUID]string{}
	ids := make([]uuid.UUID, 0, len(profile.Payload.ConceptGaps))
	for _, g := range profile.Payload.ConceptGaps {
		ids = append(ids, g.ConceptID)
	}
	if len(ids) > 0 {
		if rows, err := concepts.LoadConceptsByIDs(ctx, pool, ids); err == nil {
			for _, c := range rows {
				conceptLabels[c.ID] = c.Name
			}
		}
	}
	misLabels := map[string]string{}
	for _, mid := range profile.Payload.Misconceptions {
		id, err := uuid.Parse(mid)
		if err != nil {
			misLabels[mid] = mid
			continue
		}
		var name string
		err = pool.QueryRow(ctx, `
SELECT name FROM course.misconceptions WHERE id = $1 AND course_id = $2
`, id, courseID).Scan(&name)
		if err != nil || name == "" {
			misLabels[mid] = mid
		} else {
			misLabels[mid] = name
		}
	}
	return conceptLabels, misLabels
}

// RefreshQueueMetrics updates queue_depth and inflight gauges.
func RefreshQueueMetrics(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	if n, err := acrepo.CountPendingJobs(ctx, pool); err == nil {
		SetQueueDepth(float64(n))
	}
	if n, err := acrepo.CountGeneratingJobs(ctx, pool); err == nil {
		SetInflight(float64(n))
	}
}

// MaybeEnqueueAfterProfile is called after a profile upsert (pre-assessment path).
func MaybeEnqueueAfterProfile(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow, result ProfileResult) {
	if result.IsNeutral || result.ProfileSignature == NeutralSignature {
		return
	}
	if unit.Status != "active" {
		return
	}
	_, created, err := Enqueue(ctx, pool, unit.ID, result.ProfileSignature, unit.ContentVersion, PriorityOnDemand)
	if err != nil {
		slog.Warn("adaptivecontent: enqueue after profile failed", "unit_id", unit.ID, "err", err)
		return
	}
	if created {
		uid := unit.ID
		_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, nil, nil, EventJobEnqueued, map[string]any{
			"profileSignature": result.ProfileSignature,
			"contentVersion":   unit.ContentVersion,
			"source":           "pre_assessment",
		})
	}
}
