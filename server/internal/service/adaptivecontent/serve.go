package adaptivecontent

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

// Event names for serving audit.
const (
	EventServed         = "adaptation_served"
	EventOptoutChanged  = "student_optout_changed"
	EventViewedOriginal = "viewed_original"
)

// ServeReason classifies why a particular payload was chosen.
type ServeReason string

const (
	ServeReasonAdapted     ServeReason = "adapted"
	ServeReasonHoldout     ServeReason = "holdout"
	ServeReasonOptout      ServeReason = "optout"
	ServeReasonNoUnit      ServeReason = "no_unit"
	ServeReasonInactive    ServeReason = "inactive"
	ServeReasonNoProfile   ServeReason = "no_profile"
	ServeReasonNeutral     ServeReason = "neutral"
	ServeReasonNoVariant   ServeReason = "no_variant"
	ServeReasonVersionMiss ServeReason = "version_mismatch"
	ServeReasonGatewayDeny ServeReason = "gateway_denied"
	ServeReasonKillSwitch  ServeReason = "kill_switch"
	ServeReasonNoEnrollment ServeReason = "no_enrollment"
	ServeReasonError       ServeReason = "error"
)

// ServeRequest is the input for ResolveServing on a content-page GET.
type ServeRequest struct {
	CourseID          uuid.UUID
	BaseContentItemID uuid.UUID
	UserID            uuid.UUID
	// BaseMarkdown is the authoritative base content (for originalMarkdown when adapted).
	BaseMarkdown string
	// CourseFlag is courses.adaptive_content_enabled.
	CourseFlag bool
	// GatewayAllowed is false when aigateway would deny AI features for this user (COPPA etc.).
	// When false, serve base with no adapted indicator.
	GatewayAllowed bool
	// EnqueueOnMiss when true enqueues an AC.4 generation job on cache miss (default true for students).
	EnqueueOnMiss bool
}

// ServeResult is the serving decision for the content-page response.
type ServeResult struct {
	// Applicable is false when the page is not under an active ACE unit (or ACE is off).
	// When false, the content-page response should omit the adaptive block entirely.
	Applicable bool

	UnitID          uuid.UUID
	IsAdapted       bool
	ServedVariantID *uuid.UUID
	AxesApplied     []string
	CanViewOriginal bool
	OptedOut        bool
	IsHoldout       bool
	WasFallback     bool
	// Markdown is the body to serve (variant or base). Empty means "use existing base".
	Markdown string
	// OriginalMarkdown is base content when IsAdapted; empty otherwise.
	OriginalMarkdown string
	// AdaptationReason is a short, non-judgmental label for the banner (e.g. "extra practice").
	AdaptationReason string
	// PreAssessmentItemID when the unit has a pre-check the student may still need.
	PreAssessmentItemID *uuid.UUID
	// RequiresPreAssessment is true when trigger is pre_quiz, no profile exists, and pre-assessment is set.
	RequiresPreAssessment bool
	// ProfileID / EnrollmentID for serving-record write.
	ProfileID    *uuid.UUID
	EnrollmentID *uuid.UUID
	ContentVersion int32
	Reason       ServeReason
	EmphasisMode string
}

// ResolveServing decides which content to show a student for a unit's base content page.
// Never blocks: any error or miss yields base content. Serving-record writes are best-effort.
func ResolveServing(ctx context.Context, pool *pgxpool.Pool, req ServeRequest) ServeResult {
	start := time.Now()
	defer func() {
		ObserveServeLatency(float64(time.Since(start).Milliseconds()))
	}()

	out := ServeResult{
		AxesApplied: []string{},
		Reason:      ServeReasonNoUnit,
	}

	if pool == nil || req.UserID == uuid.Nil || req.CourseID == uuid.Nil {
		return out
	}

	if KillSwitchEngaged() || !ActiveForCourse(req.CourseFlag) {
		out.Reason = ServeReasonKillSwitch
		return out
	}

	unit, err := acrepo.GetActiveUnitByBaseContentItem(ctx, pool, req.CourseID, req.BaseContentItemID)
	if err != nil {
		slog.Debug("adaptivecontent.serve: unit lookup failed", "err", err)
		out.Reason = ServeReasonError
		return out
	}
	if unit == nil {
		return out
	}

	out.Applicable = true
	out.UnitID = unit.ID
	out.ContentVersion = unit.ContentVersion
	if out.ContentVersion <= 0 {
		out.ContentVersion = 1
	}
	out.PreAssessmentItemID = unit.PreAssessmentItemID
	out.Markdown = req.BaseMarkdown

	enrollmentID, err := acrepo.GetEnrollmentIDForUser(ctx, pool, req.CourseID, req.UserID)
	if err != nil {
		slog.Debug("adaptivecontent.serve: enrollment lookup failed", "err", err)
		out.Reason = ServeReasonError
		out.WasFallback = true
		recordBaseFallback(ctx, pool, req, unit, out)
		return out
	}
	if enrollmentID == nil {
		// Staff / non-student viewers: no serving record; return base without adaptive chrome.
		out.Applicable = false
		out.Reason = ServeReasonNoEnrollment
		return out
	}
	out.EnrollmentID = enrollmentID

	// Settings (holdout + opt-out allowed).
	settings, err := acrepo.GetSettings(ctx, pool, req.CourseID)
	if err != nil {
		slog.Debug("adaptivecontent.serve: settings failed", "err", err)
		settings = nil
	}
	holdoutPct := int16(0)
	studentOptoutAllowed := true
	if settings != nil {
		holdoutPct = settings.HoldoutPercent
		studentOptoutAllowed = settings.StudentOptoutAllowed
	} else {
		def := acrepo.DefaultSettings(req.CourseID)
		holdoutPct = def.HoldoutPercent
		studentOptoutAllowed = def.StudentOptoutAllowed
	}

	// Student opt-out (when course allows it).
	optedOut, err := acrepo.IsOptedOut(ctx, pool, req.CourseID, req.UserID)
	if err != nil {
		slog.Debug("adaptivecontent.serve: optout read failed", "err", err)
		optedOut = false
	}
	if optedOut && studentOptoutAllowed {
		out.OptedOut = true
		out.WasFallback = true
		out.Reason = ServeReasonOptout
		IncServedFallback()
		// was_holdout=false: opt-out is an explicit preference, not experimental control.
		writeServingAsync(ctx, pool, req, unit, out, nil, nil, false, true)
		return out
	}

	// COPPA / gateway deny → base, no indicator (FR-7).
	if !req.GatewayAllowed {
		out.WasFallback = true
		out.Reason = ServeReasonGatewayDeny
		IncServedFallback()
		writeServingAsync(ctx, pool, req, unit, out, nil, nil, false, true)
		return out
	}

	// Holdout / control group (FR-3).
	if IsHoldout(*enrollmentID, unit.ID, holdoutPct) {
		out.IsHoldout = true
		out.WasFallback = false // control is intentional, not a miss fallback
		out.Reason = ServeReasonHoldout
		IncServedHoldout()
		// Still resolve profile id if present (for AC.7 attribution) without serving variant.
		profile, _ := acrepo.GetProfileForEnrollment(ctx, pool, unit.ID, *enrollmentID)
		var profileID *uuid.UUID
		if profile != nil {
			profileID = &profile.ID
			out.ProfileID = profileID
			out.EmphasisMode = stringOrEmpty(profile.EmphasisMode)
		}
		writeServingAsync(ctx, pool, req, unit, out, profileID, nil, true, false)
		return out
	}

	// Profile resolution.
	profile, err := acrepo.GetProfileForEnrollment(ctx, pool, unit.ID, *enrollmentID)
	if err != nil {
		slog.Debug("adaptivecontent.serve: profile failed", "err", err)
		profile = nil
	}

	// Lazy mastery/diagnostic profile when trigger allows and none exists.
	if profile == nil {
		mode := NormalizeTriggerMode(unit.TriggerMode)
		if mode == TriggerMasterySnapshot || mode == TriggerDiagnosticFirstVisit {
			profile, err = EnsureMasterySnapshotProfile(ctx, pool, *unit, *enrollmentID, req.UserID)
			if err != nil {
				slog.Debug("adaptivecontent.serve: ensure profile failed", "err", err)
				profile = nil
			}
		}
	}

	if profile == nil {
		out.WasFallback = true
		out.Reason = ServeReasonNoProfile
		if unit.PreAssessmentItemID != nil && NormalizeTriggerMode(unit.TriggerMode) == TriggerPreQuiz {
			out.RequiresPreAssessment = true
		}
		IncServedFallback()
		writeServingAsync(ctx, pool, req, unit, out, nil, nil, false, true)
		return out
	}

	out.ProfileID = &profile.ID
	out.EmphasisMode = stringOrEmpty(profile.EmphasisMode)

	if profile.IsNeutral || profile.ProfileSignature == "" || profile.ProfileSignature == NeutralSignature {
		out.WasFallback = true
		out.Reason = ServeReasonNeutral
		IncServedBase()
		writeServingAsync(ctx, pool, req, unit, out, &profile.ID, nil, false, true)
		return out
	}

	// Variant lookup (approved / auto_served + matching content_version).
	variant, err := acrepo.GetServableVariant(ctx, pool, unit.ID, profile.ProfileSignature, unit.ContentVersion)
	if err != nil {
		slog.Debug("adaptivecontent.serve: variant lookup failed", "err", err)
		variant = nil
	}
	if variant == nil {
		out.WasFallback = true
		out.Reason = ServeReasonNoVariant
		IncServedFallback()
		IncCacheMiss()
		if req.EnqueueOnMiss {
			_, _, enqErr := Enqueue(ctx, pool, unit.ID, profile.ProfileSignature, unit.ContentVersion, PriorityOnDemand)
			if enqErr != nil {
				slog.Debug("adaptivecontent.serve: enqueue failed", "err", enqErr)
			}
		}
		writeServingAsync(ctx, pool, req, unit, out, &profile.ID, nil, false, true)
		return out
	}

	// Serve adapted variant.
	out.IsAdapted = true
	out.ServedVariantID = &variant.ID
	out.AxesApplied = variant.AxesApplied
	if out.AxesApplied == nil {
		out.AxesApplied = []string{}
	}
	out.CanViewOriginal = true
	out.Markdown = variant.VariantMarkdown
	out.OriginalMarkdown = req.BaseMarkdown
	out.AdaptationReason = AdaptationReasonLabel(out.EmphasisMode, out.AxesApplied)
	out.Reason = ServeReasonAdapted
	IncServedVariant()
	IncCacheHit()
	writeServingAsync(ctx, pool, req, unit, out, &profile.ID, &variant.ID, false, false)
	return out
}

// AdaptationReasonLabel builds a short, non-judgmental banner subtitle.
func AdaptationReasonLabel(emphasis string, axes []string) string {
	switch strings.TrimSpace(emphasis) {
	case EmphasisIntroduce:
		return "building from the foundations"
	case EmphasisReinforce:
		return "extra practice on key ideas"
	case EmphasisCompress:
		return "a faster path through material you know"
	case EmphasisRemediate:
		return "clearing up common sticking points"
	}
	for _, a := range axes {
		switch strings.TrimSpace(a) {
		case "scaffolding":
			return "with extra scaffolding"
		case "reading_level":
			return "tuned to your reading level"
		case "misconception":
			return "addressing common misconceptions"
		}
	}
	return "matched to your progress"
}

func stringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func recordBaseFallback(ctx context.Context, pool *pgxpool.Pool, req ServeRequest, unit *acrepo.UnitRow, out ServeResult) {
	IncServedFallback()
	if out.EnrollmentID != nil {
		writeServingAsync(ctx, pool, req, unit, out, out.ProfileID, nil, out.IsHoldout, true)
	}
}

// writeServingAsync upserts the serving row; failures never block the response.
func writeServingAsync(
	ctx context.Context,
	pool *pgxpool.Pool,
	req ServeRequest,
	unit *acrepo.UnitRow,
	out ServeResult,
	profileID, variantID *uuid.UUID,
	wasHoldout, wasFallback bool,
) {
	if out.EnrollmentID == nil || unit == nil {
		return
	}
	enr := *out.EnrollmentID
	cv := unit.ContentVersion
	if cv <= 0 {
		cv = 1
	}
	// Use a detached-friendly context with a short timeout so request cancel doesn't drop the write.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	_, err := acrepo.UpsertServing(writeCtx, pool, acrepo.ServingUpsert{
		UnitID:         unit.ID,
		EnrollmentID:   enr,
		ProfileID:      profileID,
		VariantID:      variantID,
		WasHoldout:     wasHoldout,
		WasFallback:    wasFallback,
		ContentVersion: cv,
	})
	if err != nil {
		slog.Warn("adaptivecontent.serve: serving upsert failed", "err", err, "unitId", unit.ID)
		return
	}
	uid := unit.ID
	subj := req.UserID
	_ = acrepo.InsertEvent(writeCtx, pool, req.CourseID, &uid, &subj, &subj, EventServed, map[string]any{
		"unitId":         unit.ID,
		"enrollmentId":   enr,
		"variantId":      variantID,
		"profileId":      profileID,
		"wasHoldout":     wasHoldout,
		"wasFallback":    wasFallback,
		"contentVersion": cv,
		"isAdapted":      out.IsAdapted,
		"reason":         string(out.Reason),
	})
}

// RecordViewedOriginal increments the view-original counter and metrics (best-effort).
func RecordViewedOriginal(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, unitID, userID uuid.UUID,
) (clicks int32, err error) {
	enrollmentID, err := acrepo.GetEnrollmentIDForUser(ctx, pool, courseID, userID)
	if err != nil {
		return 0, err
	}
	if enrollmentID == nil {
		return 0, nil
	}
	unit, err := acrepo.GetUnit(ctx, pool, courseID, unitID)
	if err != nil {
		return 0, err
	}
	if unit == nil {
		return 0, nil
	}
	cv := unit.ContentVersion
	if cv <= 0 {
		cv = 1
	}
	n, err := acrepo.IncrementViewOriginalClicks(ctx, pool, unitID, *enrollmentID, cv)
	if err != nil {
		return 0, err
	}
	IncViewOriginalClicks()
	uid := unitID
	subj := userID
	_ = acrepo.InsertEvent(ctx, pool, courseID, &uid, &subj, &subj, EventViewedOriginal, map[string]any{
		"unitId":         unitID,
		"enrollmentId":   *enrollmentID,
		"contentVersion": cv,
		"clicks":         n,
	})
	return n, nil
}
