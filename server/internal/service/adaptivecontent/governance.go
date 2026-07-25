package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

// AC.8 governance policy thresholds (centralized; fail-closed on error).
const (
	// ContestAutoPauseThreshold pauses a variant after N open contests (open question lean-yes).
	ContestAutoPauseThreshold = 3

	EventGateBlock       = "gate_block"
	EventContestOpened   = "contest_opened"
	EventContestResolved = "contest_resolved"
	EventQuarantined     = "unit_quarantined"
	EventUnquarantined   = "unit_unquarantined"
	EventKillSwitch      = "kill_switch_changed"
	EventOrgToggle       = "org_adaptive_content_toggled"
	EventFairnessFlag    = "fairness_disparity_flagged"

	ServeReasonGateBlock   ServeReason = "gate_block"
	ServeReasonQuarantine  ServeReason = "quarantined"
	ServeReasonOrgDisabled ServeReason = "org_disabled"
)

// Blocking a11y flag codes that prevent serving (AC.8 FR-2 / AC-1).
var BlockingA11yFlags = map[string]struct{}{
	"image_missing_alt": {},
	"heading_level_skip": {},
}

var (
	ErrContestNotFound   = errors.New("contest not found")
	ErrContestNotOpen    = errors.New("contest is not open")
	ErrInvalidContestStatus = errors.New("status must be reviewed, resolved, or dismissed")
	ErrQuarantineTarget  = errors.New("provide unitId or courseId")
)

// durableKillSwitch is the admin-engaged durable kill-switch (OR'd with env).
var durableKillSwitch atomic.Bool

// SetDurableKillSwitch updates the process-local durable kill-switch cache.
func SetDurableKillSwitch(engaged bool) {
	durableKillSwitch.Store(engaged)
	RefreshKillSwitchMetric()
}

// DurableKillSwitchCached reports the last-known durable kill-switch state.
func DurableKillSwitchCached() bool {
	return durableKillSwitch.Load()
}

// SyncDurableKillSwitchFromDB loads the durable kill-switch into process cache.
func SyncDurableKillSwitchFromDB(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	engaged, err := acrepo.GetDurableKillSwitch(ctx, pool)
	if err != nil {
		return
	}
	SetDurableKillSwitch(engaged)
}

// HasBlockingA11yFlag reports whether any a11y flag blocks serving.
func HasBlockingA11yFlag(flags []string) bool {
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if _, ok := BlockingA11yFlags[f]; ok {
			return true
		}
	}
	return false
}

// DecodeStringFlags unmarshals a JSON string array (safety/a11y flags column).
func DecodeStringFlags(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// GateCheckInput is the serve-time re-check payload for a candidate variant.
type GateCheckInput struct {
	Status        string
	FidelityScore *float64
	MinFidelity   float64
	SafetyFlags   []string
	A11yFlags     []string
}

// GateBlockReason explains why a variant failed serve-time gates (empty = pass).
func GateBlockReason(in GateCheckInput) string {
	status := strings.TrimSpace(in.Status)
	if status != "approved" && status != "auto_served" {
		return "status_not_servable"
	}
	minFid := in.MinFidelity
	if minFid <= 0 {
		minFid = DefaultMinFidelity
	}
	if in.FidelityScore == nil || *in.FidelityScore < minFid {
		return "fidelity_below_threshold"
	}
	for _, f := range in.SafetyFlags {
		if strings.TrimSpace(f) != "" {
			return "safety_flag"
		}
	}
	if HasBlockingA11yFlag(in.A11yFlags) {
		return "blocking_a11y"
	}
	return ""
}

// VariantPassesServeGates is true when the variant may be served to a student.
func VariantPassesServeGates(in GateCheckInput) bool {
	return GateBlockReason(in) == ""
}

// SoftGateFailed reports fidelity/safety/a11y soft failures that can be instructor-overridden.
// Hard key-term failures are excluded (never overridable). Blocking a11y flags fail soft gates (AC.8).
func SoftGateFailed(fidelityScore *float64, minFidelity float64, safetyFlags, a11yFlags []string) bool {
	if fidelityScore != nil && minFidelity > 0 && *fidelityScore < minFidelity {
		return true
	}
	for _, f := range safetyFlags {
		if f != "" && !strings.HasPrefix(f, "missing_key_term:") {
			return true
		}
	}
	return HasBlockingA11yFlag(a11yFlags)
}

// ForceInstructorApproval reports whether policy requires pre-serve human sign-off.
// True for COPPA-gated minors and when org/jurisdiction policy demands it (EU AI Act Art. 14).
func ForceInstructorApproval(coppaMinor bool, euHighRiskPolicy bool) bool {
	return coppaMinor || euHighRiskPolicy
}

// EnvEUHighRisk enables jurisdiction policy that forces instructor approval before serve.
const EnvEUHighRisk = "ACE_EU_AI_ACT_HIGH_RISK"

// EUHighRiskPolicyEnabled reports whether EU AI Act high-risk human-oversight policy is on.
func EUHighRiskPolicyEnabled() bool {
	v := strings.TrimSpace(os.Getenv(EnvEUHighRisk))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// OrgACEAllowed is false when the org has affirmatively disabled ACE (adaptive_content_org_enabled=false).
// NULL/missing means no opinion — course flag still controls enablement.
// Read errors fail open (treat as no opinion) so a transient settings lookup cannot brick serving;
// serve-time variant gates remain fail-closed separately.
func OrgACEAllowed(ctx context.Context, pool *pgxpool.Pool) bool {
	if pool == nil {
		return true
	}
	enabled, err := acrepo.GetOrgAdaptiveContentEnabled(ctx, pool)
	if err != nil || enabled == nil {
		return true
	}
	return *enabled
}

// CourseQuarantined reports whether the course is under ACE incident quarantine.
func CourseQuarantined(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) bool {
	if pool == nil || courseID == uuid.Nil {
		return false
	}
	q, err := acrepo.IsCourseQuarantined(ctx, pool, courseID)
	if err != nil {
		return true // fail-closed
	}
	return q
}

// EngageKillSwitch sets durable kill-switch on and syncs process cache.
func EngageKillSwitch(ctx context.Context, pool *pgxpool.Pool, actor uuid.UUID, engage bool) error {
	if pool == nil {
		return errors.New("adaptivecontent: nil pool")
	}
	if err := acrepo.SetDurableKillSwitch(ctx, pool, engage); err != nil {
		return err
	}
	SetDurableKillSwitch(engage)
	_ = actor // audited via admin HTTP access logs; events table is course-scoped
	IncKillSwitchToggle()
	return nil
}

// QuarantineUnit marks a unit quarantined (serving → base only).
func QuarantineUnit(ctx context.Context, pool *pgxpool.Pool, courseID, unitID, actor uuid.UUID, reason string) error {
	ok, err := acrepo.SetUnitQuarantine(ctx, pool, courseID, unitID, true, reason, actor)
	if err != nil {
		return err
	}
	if !ok {
		return ErrQuarantineTarget
	}
	uid := unitID
	_ = acrepo.InsertEvent(ctx, pool, courseID, &uid, &actor, nil, EventQuarantined, map[string]any{
		"unitId": unitID,
		"reason": reason,
	})
	IncQuarantine()
	return nil
}

// QuarantineCourse marks all ACE serving for a course as quarantined.
func QuarantineCourse(ctx context.Context, pool *pgxpool.Pool, courseID, actor uuid.UUID, reason string) error {
	if err := acrepo.SetCourseQuarantine(ctx, pool, courseID, true, reason); err != nil {
		return err
	}
	_ = acrepo.InsertEvent(ctx, pool, courseID, nil, &actor, nil, EventQuarantined, map[string]any{
		"courseId": courseID,
		"reason":   reason,
		"scope":    "course",
	})
	IncQuarantine()
	return nil
}

// ValidContestResolveStatus accepts reviewed|resolved|dismissed.
func ValidContestResolveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "reviewed", "resolved", "dismissed":
		return true
	default:
		return false
	}
}
