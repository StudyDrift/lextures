package learnerprofile

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	adaptBoostInterest   = 0.15
	adaptBoostGrowth     = 0.12
	adaptBoostNeedsReview = 0.20
	modalityBiasCap      = 0.25
)

var videoMarkdownPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<video\b`),
	regexp.MustCompile(`(?i)youtube\.com|youtu\.be`),
	regexp.MustCompile(`(?i)vimeo\.com`),
	regexp.MustCompile(`(?i)!\[[^\]]*\]\([^)]+\.(mp4|webm|mov|m4v)`),
}

// RecommendationItem is the minimal recommendation shape consumers adapt.
type RecommendationItem struct {
	ItemID   string
	ItemType string
	Title    string
	Surface  string
	Reason   string
	Score    float64
	Rationale *ProfileRationale `json:"profileRationale,omitempty"`
}

// ReviewQueueItem is the minimal review-queue shape consumers adapt.
type ReviewQueueItem struct {
	StateID      string
	QuestionID   string
	CourseID     string
	CourseCode   string
	CourseTitle  string
	NextReviewAt string
	Stem         string
	QuestionType string
	PriorityBoost float64
	Rationale    *ProfileRationale `json:"profileRationale,omitempty"`
}

// ModalityAlternate is an equivalent content item in another format.
type ModalityAlternate struct {
	ItemID   uuid.UUID
	Title    string
	Modality string
}

// ModalityPreference is the profile-driven modality selection for one content item.
type ModalityPreference struct {
	PreferredItemID *uuid.UUID
	Alternates      []ModalityAlternate
	Rationale       *ProfileRationale
}

// ApplyRecommendations re-ranks recommendations using profile interests and growth areas.
func ApplyRecommendations(ctx AdaptiveContext, items []RecommendationItem) []RecommendationItem {
	if !ctx.Usable(true) || len(items) == 0 {
		recordAdaptation("recommendations", "suppressed")
		return items
	}
	out := make([]RecommendationItem, len(items))
	copy(out, items)
	applied := false
	for i := range out {
		title := out[i].Title
		if topic := MatchTopic(title, ctx.Interests); topic != "" {
			out[i].Score += adaptBoostInterest
			r := RationaleForInterest(topic)
			out[i].Rationale = &r
			applied = true
			continue
		}
		if concept := MatchConcept(title, ctx.GrowthAreas); concept != "" {
			out[i].Score += adaptBoostGrowth
			r := RationaleForGrowth(concept)
			out[i].Rationale = &r
			applied = true
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	if applied {
		recordAdaptation("recommendations", "applied")
	} else {
		recordAdaptation("recommendations", "suppressed")
	}
	return out
}

// ApplyReviewQueue re-orders review items using needs-review concepts and peak study windows.
func ApplyReviewQueue(ctx AdaptiveContext, items []ReviewQueueItem, now time.Time) []ReviewQueueItem {
	if !ctx.Usable(true) || len(items) == 0 {
		recordAdaptation("review", "suppressed")
		return items
	}
	out := make([]ReviewQueueItem, len(items))
	copy(out, items)
	applied := false
	inPeak := inPeakStudyWindow(ctx, now)
	var peakRationale *ProfileRationale
	if len(ctx.PeakWindows) > 0 {
		r := RationaleForPeakWindow(ctx.PeakWindows[0], inPeak)
		peakRationale = &r
	}
	for i := range out {
		if concept := MatchConcept(out[i].Stem, ctx.NeedsReview); concept != "" {
			out[i].PriorityBoost += adaptBoostNeedsReview
			r := RationaleForNeedsReview(concept)
			out[i].Rationale = &r
			applied = true
		} else if peakRationale != nil && out[i].Rationale == nil {
			r := *peakRationale
			out[i].Rationale = &r
			if inPeak {
				out[i].PriorityBoost += 0.05
			}
			applied = true
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PriorityBoost != out[j].PriorityBoost {
			return out[i].PriorityBoost > out[j].PriorityBoost
		}
		return out[i].NextReviewAt < out[j].NextReviewAt
	})
	if applied {
		recordAdaptation("review", "applied")
	} else {
		recordAdaptation("review", "suppressed")
	}
	return out
}

func inPeakStudyWindow(ctx AdaptiveContext, now time.Time) bool {
	if len(ctx.PeakWindows) == 0 {
		return true
	}
	w := ctx.PeakWindows[0]
	if !dowMatches(now.Weekday(), w.Dow) {
		return false
	}
	return hourBucketMatches(now.Hour(), w.HourBucket)
}

func dowMatches(weekday time.Weekday, dow string) bool {
	switch strings.ToLower(strings.TrimSpace(dow)) {
	case "weekday":
		return weekday >= time.Monday && weekday <= time.Friday
	case "weekend":
		return weekday == time.Saturday || weekday == time.Sunday
	case "mon", "monday":
		return weekday == time.Monday
	case "tue", "tuesday":
		return weekday == time.Tuesday
	case "wed", "wednesday":
		return weekday == time.Wednesday
	case "thu", "thursday":
		return weekday == time.Thursday
	case "fri", "friday":
		return weekday == time.Friday
	case "sat", "saturday":
		return weekday == time.Saturday
	case "sun", "sunday":
		return weekday == time.Sunday
	default:
		return true
	}
}

func hourBucketMatches(hour int, bucket string) bool {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return true
	}
	parts := strings.Split(bucket, "-")
	if len(parts) != 2 {
		return true
	}
	low, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	high, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	if low >= high {
		return true
	}
	return hour >= low && hour < high
}

// ResolveModalityPreference picks a preferred alternate content item when siblings exist.
func ResolveModalityPreference(
	ctx context.Context,
	pool *pgxpool.Pool,
	adaptive AdaptiveContext,
	courseID, itemID uuid.UUID,
) (ModalityPreference, error) {
	var pref ModalityPreference
	if !adaptive.Usable(true) || adaptive.PreferredModality == "" || pool == nil {
		recordAdaptation("modality", "suppressed")
		return pref, nil
	}
	alternates, currentModality, err := listModalityAlternates(ctx, pool, courseID, itemID)
	if err != nil {
		return pref, err
	}
	if len(alternates) == 0 {
		recordAdaptation("modality", "suppressed")
		return pref, nil
	}
	pref.Alternates = alternates
	preferred := adaptive.PreferredModality
	currentScore := modalityScore(currentModality, adaptive.ModalityAffinity)
	preferredScore := modalityScore(preferred, adaptive.ModalityAffinity)
	if preferredScore-currentScore > modalityBiasCap {
		// Cap modality bias to avoid over-narrowing exposure (LP09 risk mitigation).
		preferredScore = currentScore + modalityBiasCap
	}
	if currentModality != preferred && preferredScore > currentScore {
		for _, alt := range alternates {
			if alt.Modality != preferred {
				continue
			}
			id := alt.ItemID
			pref.PreferredItemID = &id
			r := RationaleForModality(preferred)
			pref.Rationale = &r
			recordAdaptation("modality", "applied")
			return pref, nil
		}
	}
	if currentModality == preferred {
		r := RationaleForModality(preferred)
		pref.Rationale = &r
		recordAdaptation("modality", "applied")
		return pref, nil
	}
	recordAdaptation("modality", "suppressed")
	return pref, nil
}

func modalityScore(modality string, affinity map[string]float64) float64 {
	if affinity == nil {
		return 0
	}
	if s, ok := affinity[modality]; ok {
		return s
	}
	return 0
}

func listModalityAlternates(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) ([]ModalityAlternate, string, error) {
	var parentID *uuid.UUID
	var title string
	err := pool.QueryRow(ctx, `
SELECT parent_id, title
FROM course.course_structure_items
WHERE id = $1 AND course_id = $2 AND kind = 'content_page' AND archived = false
`, itemID, courseID).Scan(&parentID, &title)
	if err != nil {
		return nil, "", err
	}
	if parentID == nil {
		return nil, "", nil
	}
	baseKey := normalizeTitleKey(title)
	rows, err := pool.Query(ctx, `
SELECT si.id, si.title, COALESCE(mcp.markdown, '')
FROM course.course_structure_items si
LEFT JOIN course.module_content_pages mcp ON mcp.structure_item_id = si.id
WHERE si.course_id = $1
  AND si.parent_id = $2
  AND si.kind = 'content_page'
  AND si.archived = false
  AND si.published = true
`, courseID, *parentID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var currentModality string
	var alternates []ModalityAlternate
	for rows.Next() {
		var id uuid.UUID
		var rowTitle, markdown string
		if err := rows.Scan(&id, &rowTitle, &markdown); err != nil {
			return nil, "", err
		}
		mod := classifyContentModality(markdown)
		if id == itemID {
			currentModality = mod
			continue
		}
		if normalizeTitleKey(rowTitle) != baseKey && !titlesAreEquivalent(baseKey, normalizeTitleKey(rowTitle)) {
			continue
		}
		alternates = append(alternates, ModalityAlternate{
			ItemID:   id,
			Title:    rowTitle,
			Modality: mod,
		})
	}
	return alternates, currentModality, rows.Err()
}

func normalizeTitleKey(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	for _, suffix := range []string{" (video)", " (reading)", " - video", " - reading", " video", " reading"} {
		t = strings.TrimSuffix(t, suffix)
	}
	return strings.TrimSpace(t)
}

func titlesAreEquivalent(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

func classifyContentModality(markdown string) string {
	for _, re := range videoMarkdownPatterns {
		if re.MatchString(markdown) {
			return "video"
		}
	}
	return "reading"
}

// TutorScaffoldingPrompt returns PII-safe system-prompt augmentation for help-seeking style (LP09 FR-5).
func TutorScaffoldingPrompt(style string) string {
	switch strings.TrimSpace(style) {
	case "early-reliance":
		return "\n- Scaffolding: This learner tends to seek hints early. When they seem stuck, offer a brief nudge or guiding question before giving a full explanation or hint."
	case "independent":
		return "\n- Scaffolding: This learner tends to work independently. Give them space to reason; offer hints only after they have had time to try."
	case "balanced":
		return "\n- Scaffolding: This learner has a balanced help-seeking style. Offer a light nudge when stuck, then escalate to hints if needed."
	default:
		return ""
	}
}