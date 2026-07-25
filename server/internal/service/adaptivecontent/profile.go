package adaptivecontent

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Named thresholds for profile computation (overridable per course via settings later).
const (
	// LowGapThreshold: all concept gaps ≤ this ⇒ compress.
	LowGapThreshold = 0.2
	// HighGapThreshold: mean gap ≥ this (or no prior mastery) ⇒ introduce.
	HighGapThreshold = 0.6
	// GapBucketSize: gaps are rounded to this size for signature stability.
	GapBucketSize = 0.1
	// DefaultMasteryFreshnessDays used when unit has no override.
	DefaultMasteryFreshnessDays = 30
	// NeutralSignature is the cache key for base/fallback content.
	NeutralSignature = "base"
)

// Emphasis modes (closed set).
const (
	EmphasisIntroduce = "introduce"
	EmphasisReinforce = "reinforce"
	EmphasisCompress  = "compress"
	EmphasisRemediate = "remediate"
)

// Trigger modes on a unit.
const (
	TriggerPreQuiz              = "pre_quiz"
	TriggerDiagnosticFirstVisit = "diagnostic_first_visit"
	TriggerMasterySnapshot      = "mastery_snapshot"
)

// EventProfileComputed is written to adaptive_content_events on each recompute.
const EventProfileComputed = "profile_computed"

// ConceptGap is one concept's gap (1 - mastery) for explainability / signature.
type ConceptGap struct {
	ConceptID uuid.UUID `json:"conceptId"`
	Gap       float64   `json:"gap"`
}

// ProfilePayload is the explainability snapshot stored in payload_json.
// No free-text PII or demographic attributes.
type ProfilePayload struct {
	ConceptGaps   []ConceptGap `json:"conceptGaps"`
	Misconceptions []string    `json:"misconceptions"` // misconception UUIDs as strings
	MeanGap       float64      `json:"meanGap"`
	PriorRecord   bool         `json:"priorRecord"`
}

// ProfileInput is the pure-rule input for computing an adaptation profile.
type ProfileInput struct {
	UnitID             uuid.UUID
	// ConceptMastery maps concept_id → effective mastery in [0,1].
	// Missing keys mean no prior mastery for that concept.
	ConceptMastery map[uuid.UUID]float64
	// ConceptIDs is the ordered set of concepts the unit adapts around.
	// Gaps are computed for every id even when mastery is missing (gap=1).
	ConceptIDs []uuid.UUID
	// MisconceptionIDs from the attempt (or recent events).
	MisconceptionIDs []uuid.UUID
	// AxisSet is the effective adaptation axes for this unit.
	AxisSet []string
	// ReadingLevelPref / ModalityPref (default "default").
	ReadingLevelPref string
	ModalityPref     string
	// TargetBloom override; empty ⇒ derived from emphasis.
	TargetBloom string
	// Thresholds (zero ⇒ package defaults).
	LowGap  float64
	HighGap float64
	Bucket  float64
}

// ProfileResult is the deterministic adaptation decision.
type ProfileResult struct {
	EmphasisMode     string
	TargetBloom      string
	ProfileSignature string
	IsNeutral        bool
	ReadingLevelPref string
	ModalityPref     string
	AxisSet          []string
	Payload          ProfilePayload
}

// thresholds returns effective low/high/bucket with defaults.
func (in ProfileInput) thresholds() (low, high, bucket float64) {
	low = in.LowGap
	if low <= 0 {
		low = LowGapThreshold
	}
	high = in.HighGap
	if high <= 0 {
		high = HighGapThreshold
	}
	bucket = in.Bucket
	if bucket <= 0 {
		bucket = GapBucketSize
	}
	return low, high, bucket
}

// BucketGap rounds gap to nearest bucket (e.g. 0.1) clamped to [0,1].
// Uses milli-precision integer arithmetic so 0.15 → 0.2 under a 0.1 bucket.
func BucketGap(gap, bucket float64) float64 {
	if bucket <= 0 {
		bucket = GapBucketSize
	}
	g := gap
	if g < 0 {
		g = 0
	}
	if g > 1 {
		g = 1
	}
	// Work in thousandths to avoid binary float noise (0.15/0.1 == 1.499...).
	const scale = 1000.0
	gMilli := int(math.Round(g * scale))
	bMilli := int(math.Round(bucket * scale))
	if bMilli <= 0 {
		bMilli = int(math.Round(GapBucketSize * scale))
	}
	// Nearest bucket: (g + bucket/2) / bucket, half rounds up.
	steps := (gMilli + bMilli/2) / bMilli
	outMilli := steps * bMilli
	if outMilli < 0 {
		outMilli = 0
	}
	if outMilli > int(scale) {
		outMilli = int(scale)
	}
	return float64(outMilli) / scale
}

// NeutralProfile returns a safe base fallback profile (never blocks the learner).
func NeutralProfile(unitID uuid.UUID, axisSet []string, readingPref, modalityPref string) ProfileResult {
	if readingPref == "" {
		readingPref = "default"
	}
	if modalityPref == "" {
		modalityPref = "default"
	}
	if axisSet == nil {
		axisSet = []string{}
	}
	return ProfileResult{
		EmphasisMode:     EmphasisIntroduce,
		TargetBloom:      "remember",
		ProfileSignature: NeutralSignature,
		IsNeutral:        true,
		ReadingLevelPref: readingPref,
		ModalityPref:     modalityPref,
		AxisSet:          NormalizeAxes(axisSet),
		Payload: ProfilePayload{
			ConceptGaps:    []ConceptGap{},
			Misconceptions: []string{},
			MeanGap:        0,
			PriorRecord:    false,
		},
	}
}

// ComputeProfile applies the deterministic emphasis / bloom / signature rules.
// Pure function — no I/O. Callers must pass pre-loaded mastery and misconceptions.
func ComputeProfile(in ProfileInput) ProfileResult {
	reading := strings.TrimSpace(in.ReadingLevelPref)
	if reading == "" {
		reading = "default"
	}
	modality := strings.TrimSpace(in.ModalityPref)
	if modality == "" {
		modality = "default"
	}
	axes := NormalizeAxes(in.AxisSet)

	// No concepts and no misconceptions ⇒ neutral base (nothing to adapt on).
	if len(in.ConceptIDs) == 0 && len(in.MisconceptionIDs) == 0 {
		return NeutralProfile(in.UnitID, axes, reading, modality)
	}

	low, high, bucket := in.thresholds()

	// Build per-concept gaps (missing mastery ⇒ gap 1.0, priorRecord false for that concept).
	gaps := make([]ConceptGap, 0, len(in.ConceptIDs))
	var sum float64
	priorAny := false
	allLow := len(in.ConceptIDs) > 0
	for _, cid := range in.ConceptIDs {
		mastery, ok := in.ConceptMastery[cid]
		gap := 1.0
		if ok {
			priorAny = true
			if mastery < 0 {
				mastery = 0
			}
			if mastery > 1 {
				mastery = 1
			}
			gap = 1.0 - mastery
		}
		bg := BucketGap(gap, bucket)
		gaps = append(gaps, ConceptGap{ConceptID: cid, Gap: bg})
		sum += bg
		if bg > low {
			allLow = false
		}
	}
	// Stable order by concept id for signature.
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].ConceptID.String() < gaps[j].ConceptID.String()
	})

	meanGap := 0.0
	if len(gaps) > 0 {
		meanGap = sum / float64(len(gaps))
		meanGap = math.Round(meanGap*1000) / 1000
	}

	// Misconception ids as sorted unique strings.
	misSet := make(map[string]struct{}, len(in.MisconceptionIDs))
	for _, mid := range in.MisconceptionIDs {
		if mid == uuid.Nil {
			continue
		}
		misSet[mid.String()] = struct{}{}
	}
	misIDs := make([]string, 0, len(misSet))
	for s := range misSet {
		misIDs = append(misIDs, s)
	}
	sort.Strings(misIDs)

	// Emphasis selection (deterministic). Misconception wins over high mastery (open Q2).
	var emphasis string
	switch {
	case len(misIDs) > 0:
		emphasis = EmphasisRemediate
	case len(gaps) > 0 && allLow:
		emphasis = EmphasisCompress
	case !priorAny || meanGap >= high:
		emphasis = EmphasisIntroduce
	case meanGap > low && meanGap < high:
		emphasis = EmphasisReinforce
	default:
		// meanGap == low boundary with not-all-low is rare; treat as reinforce.
		emphasis = EmphasisReinforce
	}

	bloom := strings.TrimSpace(in.TargetBloom)
	if bloom == "" {
		bloom = DefaultBloomForEmphasis(emphasis)
	}

	payload := ProfilePayload{
		ConceptGaps:    gaps,
		Misconceptions: misIDs,
		MeanGap:        meanGap,
		PriorRecord:    priorAny,
	}

	sig, err := ComputeProfileSignature(in.UnitID, gaps, misIDs, emphasis, bloom, reading, modality, axes)
	if err != nil {
		// Extremely unlikely (JSON marshal of primitives); fall back to neutral.
		return NeutralProfile(in.UnitID, axes, reading, modality)
	}

	return ProfileResult{
		EmphasisMode:     emphasis,
		TargetBloom:      bloom,
		ProfileSignature: sig,
		IsNeutral:        false,
		ReadingLevelPref: reading,
		ModalityPref:     modality,
		AxisSet:          axes,
		Payload:          payload,
	}
}

// DefaultBloomForEmphasis maps emphasis mode to a default Bloom target.
func DefaultBloomForEmphasis(emphasis string) string {
	switch emphasis {
	case EmphasisCompress:
		return "analyze"
	case EmphasisReinforce:
		return "apply"
	case EmphasisRemediate:
		return "understand"
	default:
		return "remember"
	}
}

// signatureMaterial is the canonical JSON object hashed for profile_signature.
// Field order is fixed by the struct tags for stable hashing.
type signatureMaterial struct {
	UnitID           string            `json:"unitId"`
	ConceptGaps      []sigConceptGap   `json:"conceptGaps"`
	Misconceptions   []string          `json:"misconceptions"`
	EmphasisMode     string            `json:"emphasisMode"`
	TargetBloom      string            `json:"targetBloom"`
	ReadingLevelPref string            `json:"readingLevelPref"`
	ModalityPref     string            `json:"modalityPref"`
	AxisSet          []string          `json:"axisSet"`
}

type sigConceptGap struct {
	ConceptID string  `json:"conceptId"`
	Gap       float64 `json:"gap"`
}

// ComputeProfileSignature builds the stable cache key for (unit, adaptation inputs).
// Gaps must already be bucketed. Identical learners collide into one key.
func ComputeProfileSignature(
	unitID uuid.UUID,
	gaps []ConceptGap,
	misconceptionIDs []string,
	emphasis, bloom, reading, modality string,
	axisSet []string,
) (string, error) {
	// Copy + sort for determinism.
	gCopy := make([]sigConceptGap, 0, len(gaps))
	for _, g := range gaps {
		gCopy = append(gCopy, sigConceptGap{ConceptID: g.ConceptID.String(), Gap: g.Gap})
	}
	sort.Slice(gCopy, func(i, j int) bool { return gCopy[i].ConceptID < gCopy[j].ConceptID })

	mCopy := append([]string(nil), misconceptionIDs...)
	sort.Strings(mCopy)

	axes := NormalizeAxes(axisSet)
	sort.Strings(axes)

	mat := signatureMaterial{
		UnitID:           unitID.String(),
		ConceptGaps:      gCopy,
		Misconceptions:   mCopy,
		EmphasisMode:     emphasis,
		TargetBloom:      bloom,
		ReadingLevelPref: reading,
		ModalityPref:     modality,
		AxisSet:          axes,
	}
	return ProfileSignature(mat)
}

// MasteryIsFresh reports whether lastSeen is within freshnessDays of now.
// nil lastSeen is never fresh (no prior record).
func MasteryIsFresh(lastSeen *time.Time, freshnessDays int, now time.Time) bool {
	if lastSeen == nil {
		return false
	}
	if freshnessDays < 0 {
		freshnessDays = DefaultMasteryFreshnessDays
	}
	cutoff := now.Add(-time.Duration(freshnessDays) * 24 * time.Hour)
	return !lastSeen.Before(cutoff)
}

// ValidateTriggerMode checks unit trigger_mode enum.
func ValidateTriggerMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case "", TriggerPreQuiz, TriggerDiagnosticFirstVisit, TriggerMasterySnapshot:
		return nil
	default:
		return fmt.Errorf("triggerMode must be pre_quiz, diagnostic_first_visit, or mastery_snapshot")
	}
}

// NormalizeTriggerMode returns a valid trigger mode (default pre_quiz).
func NormalizeTriggerMode(mode string) string {
	m := strings.TrimSpace(mode)
	if m == "" {
		return TriggerPreQuiz
	}
	return m
}

// ValidateModalityPref checks modality preference enum (empty = default).
func ValidateModalityPref(pref string) error {
	switch strings.TrimSpace(pref) {
	case "", "default", "text", "worked_example", "visual":
		return nil
	default:
		return fmt.Errorf("modalityPref must be text, worked_example, visual, or default")
	}
}
