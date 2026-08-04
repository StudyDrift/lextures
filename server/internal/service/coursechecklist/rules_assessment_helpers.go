package coursechecklist

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/coursechecklist/linkhealth"
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

const LazyLinkHealth LazyLoaderID = "link_health"

// LinkHealthLazy is the lazy payload for links.external-health.
type LinkHealthLazy struct {
	Pending  bool
	Capped   bool
	Rows     []linkhealth.Row
	CheckedAt *time.Time
}

func linkAndLaunchRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleLinksExternalHealth(),
		ruleLaunchStudentPreview(),
		ruleLaunchNoDraftsAfterStart(),
		ruleLaunchCalendarSanity(),
		ruleLaunchBackupExport(),
	}
}

func ruleLinksExternalHealth() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemLinksExternalHealth,
		Category:     CategoryLaunch,
		TitleKey:     "coursechecklist.item.links.external-health.title",
		TitleDefault: "Check external links",
		WhyKey:       "coursechecklist.item.links.external-health.why",
		WhyDefault:   "Dead external links frustrate learners; this check runs in the background with a crawl budget.",
		HelpRef:      "course-checklist#links-external-health",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 37"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		LazyNeeds:    []LazyLoaderID{LazyLinkHealth},
		Evaluate:     evalLinksExternalHealth,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"URL", "Status", "Page"}},
	}
}

func evalLinksExternalHealth(_ context.Context, snap CourseSnapshot) (Finding, error) {
	raw, ok := snap.Lazy[LazyLinkHealth]
	if !ok {
		return Finding{
			Status:        StatusUnknown,
			DetailKey:     "coursechecklist.item.links.external-health.detail.checking",
			DetailDefault: "Checking links…",
		}, nil
	}
	payload, ok := raw.(LinkHealthLazy)
	if !ok {
		return Finding{Status: StatusUnknown, DetailDefault: "Checking links…"}, nil
	}
	if payload.Pending {
		return Finding{
			Status:        StatusUnknown,
			DetailKey:     "coursechecklist.item.links.external-health.detail.checking",
			DetailDefault: "Checking links…",
		}, nil
	}
	doc := contentDocFor(snap)
	pageByURL := map[string]string{}
	var urls []string
	for _, p := range doc.Pages {
		for _, link := range p.Links {
			href := strings.TrimSpace(link.Href)
			if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
				continue
			}
			n := linkhealth.NormalizeURL(href)
			urls = append(urls, n)
			if _, ok := pageByURL[n]; !ok {
				pageByURL[n] = pageLabel(p)
			}
		}
	}
	if len(urls) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "No external links to check."}, nil
	}
	var evidence []EvidenceRow
	dead := 0
	errors := 0
	for _, row := range payload.Rows {
		switch row.Result {
		case linkhealth.ResultDead:
			dead++
			code := ""
			if row.StatusCode != nil {
				code = fmt.Sprintf("HTTP %d", *row.StatusCode)
			}
			evidence = append(evidence, EvidenceRow{
				Label:    truncateRunes(row.URL, 80),
				Sublabel: strings.TrimSpace(code + " · " + pageByURL[row.URL]),
				Status:   StatusTodo,
			})
		case linkhealth.ResultError:
			errors++
		}
	}
	detailExtra := ""
	if payload.Capped {
		detailExtra = fmt.Sprintf(" Cap of %d URLs applied.", linkhealth.MaxURLsPerCourse)
	}
	if dead == 0 && errors > 0 && len(evidence) == 0 {
		// Network hiccups → unknown, never todo (NFR reliability).
		return Finding{
			Status:        StatusUnknown,
			DetailDefault: "Some link checks could not complete." + detailExtra,
		}, nil
	}
	if dead == 0 {
		return Finding{
			Status:        StatusDone,
			DetailDefault: "External links resolved on the last check." + detailExtra,
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		Evidence:      evidence,
		DetailDefault: fmt.Sprintf("%d dead external links found.%s", dead, detailExtra),
	}, nil
}

func ruleLaunchStudentPreview() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemLaunchStudentPreview,
		Category:     CategoryLaunch,
		TitleKey:     "coursechecklist.item.launch.student-preview.title",
		TitleDefault: "Preview the course as a student",
		WhyKey:       "coursechecklist.item.launch.student-preview.why",
		WhyDefault:   "Use View as: Student after structural changes so you see what learners see.",
		HelpRef:      "course-checklist#launch-student-preview",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 16", "QM 8.x"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure},
		Evaluate:     evalLaunchStudentPreview,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}"},
	}
}

func evalLaunchStudentPreview(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.StudentPreviewAt == nil {
		return Finding{
			Status:        StatusTodo,
			DetailDefault: "Staff have not previewed this course as a student yet.",
		}, nil
	}
	if snap.StructureChangedAt != nil && snap.StudentPreviewAt.Before(*snap.StructureChangedAt) {
		return Finding{
			Status:        StatusTodo,
			DetailDefault: "Structure changed since the last student preview — preview again.",
		}, nil
	}
	return Finding{
		Status:        StatusDone,
		DetailDefault: "Student preview is current since the last structural change.",
	}, nil
}

func ruleLaunchNoDraftsAfterStart() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemLaunchNoDraftsAfterStart,
		Category:     CategoryLaunch,
		TitleKey:     "coursechecklist.item.launch.no-drafts-after-start.title",
		TitleDefault: "Publish near-term drafts after start",
		WhyKey:       "coursechecklist.item.launch.no-drafts-after-start.why",
		WhyDefault:   "Once the course has started, gradable items due in the next 14 days should be published.",
		HelpRef:      "course-checklist#launch-no-drafts-after-start",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedAssessmentItems},
		Applies: func(snap CourseSnapshot) bool {
			if snap.StartsAt == nil {
				return false
			}
			return !time.Now().UTC().Before(snap.StartsAt.UTC())
		},
		Evaluate: evalLaunchNoDraftsAfterStart,
		Target:   NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalLaunchNoDraftsAfterStart(_ context.Context, snap CourseSnapshot) (Finding, error) {
	now := time.Now().UTC()
	horizon := now.Add(14 * 24 * time.Hour)
	var evidence []EvidenceRow
	for _, a := range assessmentItemsFor(snap) {
		if a.Archived || a.Published || a.DueAt == nil {
			continue
		}
		due := a.DueAt.UTC()
		if due.Before(now) || due.After(horizon) {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    a.Title,
			Sublabel: fmt.Sprintf("%s · unpublished · due %s", a.Kind, due.Format("2006-01-02")),
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   contentPageRoute(a.Kind, a.ID),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Near-term gradable items are published."}, nil
	}
	return Finding{
		Status: StatusTodo, Evidence: evidence,
		DetailDefault: fmt.Sprintf("%d unpublished items are due within 14 days.", len(evidence)),
	}, nil
}

func ruleLaunchCalendarSanity() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemLaunchCalendarSanity,
		Category:     CategoryLaunch,
		TitleKey:     "coursechecklist.item.launch.calendar-sanity.title",
		TitleDefault: "Move due dates off non-instructional days",
		WhyKey:       "coursechecklist.item.launch.calendar-sanity.why",
		WhyDefault:   "Due dates on institutional holidays or blackout days confuse learners.",
		HelpRef:      "course-checklist#launch-calendar-sanity",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedAssessmentItems, DataNeedAcademicCalendar},
		Applies: func(snap CourseSnapshot) bool {
			return len(snap.BlackoutDates) > 0
		},
		Evaluate: evalLaunchCalendarSanity,
		Target:   NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Due date", "Issue"}},
	}
}

func evalLaunchCalendarSanity(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.BlackoutDates) == 0 {
		return Finding{Status: StatusNotApplicable, DetailDefault: "No institutional academic calendar is published."}, nil
	}
	blackout := map[string]struct{}{}
	for _, d := range snap.BlackoutDates {
		blackout[d.UTC().Format("2006-01-02")] = struct{}{}
	}
	var evidence []EvidenceRow
	for _, a := range assessmentItemsFor(snap) {
		if a.DueAt == nil {
			continue
		}
		key := a.DueAt.UTC().Format("2006-01-02")
		if _, ok := blackout[key]; ok {
			evidence = append(evidence, EvidenceRow{
				Label: a.Title, Sublabel: key + " · non-instructional day", Status: StatusTodo,
				TargetOverride: &NavTarget{Surface: "web", Route: contentPageRoute(a.Kind, a.ID)},
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "No due dates fall on non-instructional days."}, nil
	}
	return Finding{Status: StatusTodo, Evidence: evidence, DetailDefault: fmt.Sprintf("%d due dates fall on non-instructional days.", len(evidence))}, nil
}

func ruleLaunchBackupExport() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemLaunchBackupExport,
		Category:     CategoryLaunch,
		TitleKey:     "coursechecklist.item.launch.backup-export.title",
		TitleDefault: "Export a fresh course backup",
		WhyKey:       "coursechecklist.item.launch.backup-export.why",
		WhyDefault:   "Keep an export newer than the last structural change (or wait until the course is a week old).",
		HelpRef:      "course-checklist#launch-backup-export",
		Tier:         TierRecommended,
		Sources:      []string{"operational"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure},
		Evaluate:     evalLaunchBackupExport,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/settings/import-export"},
	}
}

func evalLaunchBackupExport(_ context.Context, snap CourseSnapshot) (Finding, error) {
	age := time.Since(snap.CreatedAt)
	if age < 7*24*time.Hour {
		return Finding{
			Status:        StatusDone,
			DetailDefault: "Course is younger than 7 days — backup not required yet.",
		}, nil
	}
	if snap.LastExportAt == nil {
		return Finding{Status: StatusTodo, DetailDefault: "No course export has been taken yet."}, nil
	}
	if snap.StructureChangedAt != nil && snap.LastExportAt.Before(*snap.StructureChangedAt) {
		return Finding{Status: StatusTodo, DetailDefault: "Structure changed since the last export — export again."}, nil
	}
	return Finding{Status: StatusDone, DetailDefault: "A course export is current since the last structural change."}, nil
}
