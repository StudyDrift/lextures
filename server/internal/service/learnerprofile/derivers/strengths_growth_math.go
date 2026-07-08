package derivers

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/learnermodel"
)

const (
	strengthsGrowthDeriverVersion       = 1
	strengthsGrowthMinConcepts          = 3
	strengthsGrowthMaxPerCategory        = 5
	strengthsGrowthStrongThreshold       = 0.8
	strengthsGrowthWeakThreshold         = 0.5
	strengthsGrowthMinAttempts           = 2
	strengthsGrowthMisconceptionMinTriggers = 2
	strengthsGrowthSourceTable           = "course.learner_concept_states"
	strengthsGrowthMisconceptionTable    = "course.misconception_events"
)

type conceptCategory string

const (
	categoryStrength    conceptCategory = "strength"
	categoryGrowth      conceptCategory = "growth"
	categoryNeedsReview conceptCategory = "needs_review"
)

// StrengthItem is one ranked strength in the facet summary.
type StrengthItem struct {
	Concept      string  `json:"concept"`
	Mastery      float64 `json:"mastery"`
	Courses      int     `json:"courses"`
	AttemptCount int32   `json:"attemptCount,omitempty"`
}

// GrowthItem is one ranked growth area or recurring misconception.
type GrowthItem struct {
	Concept       string   `json:"concept,omitempty"`
	Mastery       *float64 `json:"mastery,omitempty"`
	Misconception string   `json:"misconception,omitempty"`
	Description   string   `json:"description,omitempty"`
	TriggerCount  int64    `json:"triggerCount,omitempty"`
}

// NeedsReviewItem is one concept whose mastery has decayed or is due for review.
type NeedsReviewItem struct {
	Concept       string  `json:"concept"`
	Mastery       float64 `json:"mastery"`
	LastSeenDays  int     `json:"lastSeenDays"`
	AttemptCount  int32   `json:"attemptCount,omitempty"`
}

// StrengthsGrowthSummary is the facet-level aggregate returned in summary JSON.
type StrengthsGrowthSummary struct {
	Strengths    []StrengthItem    `json:"strengths"`
	Growth       []GrowthItem      `json:"growth"`
	NeedsReview  []NeedsReviewItem `json:"needsReview"`
}

type conceptCourseRow struct {
	Slug          string
	Name          string
	ConceptID     uuid.UUID
	CourseID      uuid.UUID
	DecayLambda   float64
	StoredMastery float64
	AttemptCount  int32
	LastSeenAt    *time.Time
	NeedsReviewAt *time.Time
}

type aggregatedConcept struct {
	Slug             string
	Name             string
	EffectiveMastery float64
	StoredMastery    float64
	AttemptCount     int32
	CourseIDs        []uuid.UUID
	LastSeenAt       *time.Time
	NeedsReviewDue   bool
	LastSeenDays     int
}

type misconceptionRow struct {
	MisconceptionID uuid.UUID
	Name            string
	Description     *string
	ConceptName     *string
	CourseID        uuid.UUID
	TriggerCount    int64
}

type strengthsGrowthComputeInput struct {
	ConceptRows      []conceptCourseRow
	Misconceptions   []misconceptionRow
	Now              time.Time
}

type classifiedConcept struct {
	agg      aggregatedConcept
	category conceptCategory
	salience float64
}

func computeStrengthsGrowth(in strengthsGrowthComputeInput) (StrengthsGrowthSummary, bool, int) {
	concepts := aggregateConceptsByName(in.ConceptRows, in.Now)
	signalCount := countConceptSignals(concepts)
	if signalCount < strengthsGrowthMinConcepts {
		return StrengthsGrowthSummary{}, false, signalCount
	}

	var classified []classifiedConcept
	for _, agg := range concepts {
		cat, ok := classifyConcept(agg, in.Now)
		if !ok {
			continue
		}
		classified = append(classified, classifiedConcept{
			agg:      agg,
			category: cat,
			salience: conceptSalience(agg, cat),
		})
	}

	strengths := capRanked(classified, categoryStrength, strengthsGrowthMaxPerCategory)
	growthConcepts := capRanked(classified, categoryGrowth, strengthsGrowthMaxPerCategory)
	needsReview := capRanked(classified, categoryNeedsReview, strengthsGrowthMaxPerCategory)
	misconceptions := capMisconceptions(in.Misconceptions, strengthsGrowthMaxPerCategory)

	summary := StrengthsGrowthSummary{
		Strengths:   toStrengthItems(strengths),
		Growth:      append(toGrowthItems(growthConcepts), toMisconceptionGrowthItems(misconceptions)...),
		NeedsReview: toNeedsReviewItems(needsReview),
	}
	return summary, true, signalCount
}

func conceptAggregateKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func aggregateConceptsByName(rows []conceptCourseRow, now time.Time) []aggregatedConcept {
	type acc struct {
		agg           aggregatedConcept
		weightSum     float64
		weightedStore float64
		weightedEff   float64
	}
	byKey := make(map[string]*acc)

	for _, row := range rows {
		if row.AttemptCount <= 0 && row.StoredMastery <= 0 {
			continue
		}
		weight := float64(row.AttemptCount)
		if weight <= 0 {
			weight = 1
		}
		effective := learnermodel.DecayAdjustedMasteryAt(row.StoredMastery, row.LastSeenAt, row.DecayLambda, now)
		needsDue := needsReviewElapsed(row.NeedsReviewAt, now)
		key := conceptAggregateKey(row.Name)

		entry, ok := byKey[key]
		if !ok {
			entry = &acc{agg: aggregatedConcept{
				Slug: row.Slug,
				Name: row.Name,
			}}
			byKey[key] = entry
		}
		entry.weightSum += weight
		entry.weightedStore += row.StoredMastery * weight
		entry.weightedEff += effective * weight
		entry.agg.AttemptCount += row.AttemptCount
		entry.agg.CourseIDs = appendUniqueCourse(entry.agg.CourseIDs, row.CourseID)
		if row.LastSeenAt != nil && (entry.agg.LastSeenAt == nil || row.LastSeenAt.After(*entry.agg.LastSeenAt)) {
			t := row.LastSeenAt.UTC()
			entry.agg.LastSeenAt = &t
		}
		if needsDue {
			entry.agg.NeedsReviewDue = true
		}
		if effective > entry.agg.EffectiveMastery {
			entry.agg.EffectiveMastery = effective
			entry.agg.Name = row.Name
		}
	}

	out := make([]aggregatedConcept, 0, len(byKey))
	for _, entry := range byKey {
		if entry.weightSum > 0 {
			entry.agg.StoredMastery = entry.weightedStore / entry.weightSum
			entry.agg.EffectiveMastery = entry.weightedEff / entry.weightSum
		}
		if entry.agg.LastSeenAt != nil {
			entry.agg.LastSeenDays = int(now.Sub(*entry.agg.LastSeenAt).Hours() / 24)
		}
		out = append(out, entry.agg)
	}
	return out
}

func classifyConcept(agg aggregatedConcept, now time.Time) (conceptCategory, bool) {
	if agg.NeedsReviewDue {
		return categoryNeedsReview, true
	}
	if agg.StoredMastery >= strengthsGrowthStrongThreshold && agg.EffectiveMastery < strengthsGrowthStrongThreshold {
		return categoryNeedsReview, true
	}
	if agg.EffectiveMastery >= strengthsGrowthStrongThreshold && agg.AttemptCount >= strengthsGrowthMinAttempts {
		return categoryStrength, true
	}
	if agg.EffectiveMastery <= strengthsGrowthWeakThreshold {
		return categoryGrowth, true
	}
	return "", false
}

func needsReviewElapsed(at *time.Time, now time.Time) bool {
	return at != nil && !now.Before(at.UTC())
}

func countConceptSignals(concepts []aggregatedConcept) int {
	n := 0
	for _, c := range concepts {
		if c.AttemptCount > 0 || c.StoredMastery > 0 {
			n++
		}
	}
	return n
}

func conceptSalience(agg aggregatedConcept, cat conceptCategory) float64 {
	switch cat {
	case categoryStrength:
		return agg.EffectiveMastery*100 + float64(agg.AttemptCount)*0.01
	case categoryGrowth:
		return (1-agg.EffectiveMastery)*100 + float64(agg.AttemptCount)*0.01
	case categoryNeedsReview:
		return float64(agg.LastSeenDays)*2 + (1-agg.EffectiveMastery)*50
	default:
		return 0
	}
}

func capRanked(classified []classifiedConcept, cat conceptCategory, limit int) []aggregatedConcept {
	filtered := make([]classifiedConcept, 0)
	for _, item := range classified {
		if item.category == cat {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].salience == filtered[j].salience {
			return filtered[i].agg.Name < filtered[j].agg.Name
		}
		return filtered[i].salience > filtered[j].salience
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	out := make([]aggregatedConcept, len(filtered))
	for i, item := range filtered {
		out[i] = item.agg
	}
	return out
}

func capMisconceptions(rows []misconceptionRow, limit int) []misconceptionRow {
	if len(rows) == 0 {
		return nil
	}
	sorted := append([]misconceptionRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TriggerCount == sorted[j].TriggerCount {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].TriggerCount > sorted[j].TriggerCount
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func toStrengthItems(rows []aggregatedConcept) []StrengthItem {
	out := make([]StrengthItem, len(rows))
	for i, row := range rows {
		out[i] = StrengthItem{
			Concept:      row.Name,
			Mastery:      round2(row.EffectiveMastery),
			Courses:      len(row.CourseIDs),
			AttemptCount: row.AttemptCount,
		}
	}
	return out
}

func toGrowthItems(rows []aggregatedConcept) []GrowthItem {
	out := make([]GrowthItem, len(rows))
	for i, row := range rows {
		m := round2(row.EffectiveMastery)
		out[i] = GrowthItem{
			Concept: row.Name,
			Mastery: &m,
		}
	}
	return out
}

func toMisconceptionGrowthItems(rows []misconceptionRow) []GrowthItem {
	out := make([]GrowthItem, len(rows))
	for i, row := range rows {
		desc := row.Name
		if row.Description != nil && *row.Description != "" {
			desc = *row.Description
		}
		item := GrowthItem{
			Misconception: row.Name,
			Description:   desc,
			TriggerCount:  row.TriggerCount,
		}
		if row.ConceptName != nil && *row.ConceptName != "" {
			item.Concept = *row.ConceptName
		}
		out[i] = item
	}
	return out
}

func toNeedsReviewItems(rows []aggregatedConcept) []NeedsReviewItem {
	out := make([]NeedsReviewItem, len(rows))
	for i, row := range rows {
		out[i] = NeedsReviewItem{
			Concept:      row.Name,
			Mastery:      round2(row.EffectiveMastery),
			LastSeenDays: row.LastSeenDays,
			AttemptCount: row.AttemptCount,
		}
	}
	return out
}

func strengthsGrowthConfidence(signalCount int, summary StrengthsGrowthSummary) float64 {
	if signalCount < strengthsGrowthMinConcepts {
		return 0
	}
	coverage := math.Min(1, float64(signalCount)/10.0)
	items := len(summary.Strengths) + len(summary.Growth) + len(summary.NeedsReview)
	itemFactor := math.Min(1, float64(items)/3.0)
	return round2(math.Max(0.25, coverage*itemFactor))
}

func appendUniqueCourse(ids []uuid.UUID, id uuid.UUID) []uuid.UUID {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}