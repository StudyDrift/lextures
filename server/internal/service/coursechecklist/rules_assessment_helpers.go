package coursechecklist

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// assessmentEvidenceColumns is the shared evidence shape for assessment rules (CC.5 §10).
var assessmentEvidenceColumns = []string{"Item", "Type", "Module", "Points", "Issue"}

// gradingEvidenceColumns is the shared evidence shape for grading-group rules.
var gradingEvidenceColumns = []string{"Group", "Weight", "Items"}

func usesWeightedGrading(snap CourseSnapshot) bool {
	if len(snap.AssignmentGroups) == 0 {
		return false
	}
	for _, g := range snap.AssignmentGroups {
		if g.Weight != nil && math.Abs(*g.Weight) > 0.0001 {
			return true
		}
	}
	return false
}

func assessmentItemsFor(snap CourseSnapshot) []AssessmentItemSnap {
	if len(snap.AssessmentItems) > 0 {
		return snap.AssessmentItems
	}
	// Fallback for unit tests that only populate StructureItems / ItemMeta.
	var out []AssessmentItemSnap
	byID := structureByID(snap)
	for _, it := range snap.StructureItems {
		if it.Archived || !isGradableKind(it.Kind) {
			continue
		}
		meta := snap.ItemMeta[it.ID]
		modTitle := ""
		if it.ParentID != nil {
			if m, ok := byID[*it.ParentID]; ok {
				modTitle = m.Title
			}
		}
		out = append(out, AssessmentItemSnap{
			ID:                   it.ID,
			Kind:                 it.Kind,
			Title:                it.Title,
			ParentID:             it.ParentID,
			ModuleTitle:          modTitle,
			SortOrder:            it.SortOrder,
			Published:            it.Published,
			Archived:             it.Archived,
			DueAt:                it.DueAt,
			Points:               meta.PointsWorth,
			AssignmentGroupID:    it.AssignmentGroupID,
			HasBody:              meta.HasBody,
			LateSubmissionPolicy: meta.LateSubmissionPolicy,
		})
	}
	return out
}

func totalCoursePoints(items []AssessmentItemSnap) int {
	total := 0
	for _, it := range items {
		if it.Points != nil && *it.Points > 0 {
			total += *it.Points
		}
	}
	return total
}

func formatPoints(p *int) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *p)
}

func assessmentEvidenceRow(it AssessmentItemSnap, issue, anchor string) EvidenceRow {
	route := itemEditorRoute(it.Kind)
	route = strings.ReplaceAll(route, "{itemId}", it.ID.String())
	return EvidenceRow{
		Label:    it.Title,
		Sublabel: fmt.Sprintf("%s · %s · %s pts · %s", humanKind(it.Kind), it.ModuleTitle, formatPoints(it.Points), issue),
		TargetOverride: &NavTarget{
			Surface:   "web",
			Route:     route,
			Anchor:    anchor,
			EntityKey: it.ID.String(),
		},
	}
}

func groupItemCounts(snap CourseSnapshot) map[uuid.UUID]int {
	counts := map[uuid.UUID]int{}
	for _, it := range assessmentItemsFor(snap) {
		if it.AssignmentGroupID == nil {
			continue
		}
		counts[*it.AssignmentGroupID]++
	}
	return counts
}

func isHighStakes(it AssessmentItemSnap, total int, weighted bool) bool {
	if it.Points == nil || *it.Points <= 0 {
		return false
	}
	if total <= 0 {
		return false
	}
	pct := 100.0 * float64(*it.Points) / float64(total)
	if weighted {
		return pct >= 10.0
	}
	return *it.Points >= 100 || pct >= 10.0
}

func weekKey(t time.Time) string {
	y, w := t.UTC().ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}

func sortAssessmentItems(items []AssessmentItemSnap) []AssessmentItemSnap {
	out := append([]AssessmentItemSnap(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].Title < out[j].Title
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}

func formatWeight(w *float64) string {
	if w == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f%%", *w)
}

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}
