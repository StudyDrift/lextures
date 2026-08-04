package coursechecklist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func assessmentRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleAssessmentGradableItems(),
		ruleAssessmentDueDates(),
		ruleAssessmentPoints(),
		ruleAssessmentDatesWithinTerm(),
		ruleAssessmentAvailabilityWindows(),
		ruleAssessmentSpread(),
	}
}

func ruleAssessmentGradableItems() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentGradableItems,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.gradable-items.title",
		TitleDefault: "Add something to grade",
		WhyKey:       "coursechecklist.item.assessment.gradable-items.why",
		WhyDefault:   "Learners need at least one graded activity so the gradebook has meaning.",
		HelpRef:      "course-checklist#assessment-gradable-items",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.1", "OSCQR 45"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedAssessmentItems},
		Evaluate:     evalAssessmentGradableItems,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalAssessmentGradableItems(_ context.Context, snap CourseSnapshot) (Finding, error) {
	// Non-graded mode: SBG-only courses with no traditional items stay N/A when SBG is on
	// and there are no assignment groups / points items expected.
	if snap.SbgEnabled && len(snap.AssignmentGroups) == 0 && len(assessmentItemsFor(snap)) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.gradable-items.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	items := assessmentItemsFor(snap)
	if len(items) >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.gradable-items.detail.done",
			DetailDefault: fmt.Sprintf("%d graded items are ready.", len(items)),
			DetailFields:  map[string]any{"count": len(items)},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.gradable-items.detail.todo",
		DetailDefault: "Add at least one assignment or quiz to grade.",
	}, nil
}

func ruleAssessmentDueDates() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentDueDates,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.due-dates.title",
		TitleDefault: "Give every assessment a due date",
		WhyKey:       "coursechecklist.item.assessment.due-dates.why",
		WhyDefault:   "Due dates keep the gradebook predictable and help learners plan.",
		HelpRef:      "course-checklist#assessment-due-dates",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.2", "OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedAssessmentItems, DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return !strings.EqualFold(snap.ScheduleMode, "relative")
		},
		Evaluate: evalAssessmentDueDates,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.scheduling",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalAssessmentDueDates(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := sortAssessmentItems(assessmentItemsFor(snap))
	if len(items) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.due-dates.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	var evidence []EvidenceRow
	for _, it := range items {
		if it.DueAt == nil {
			evidence = append(evidence, assessmentEvidenceRow(it, "No due date", "assignment.scheduling"))
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.due-dates.detail.done",
			DetailDefault: "Every assessment has a due date.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.due-dates.detail.todo",
		DetailDefault: fmt.Sprintf("%d assessments have no due date.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleAssessmentPoints() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentPoints,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.points.title",
		TitleDefault: "Give every assessment points",
		WhyKey:       "coursechecklist.item.assessment.points.why",
		WhyDefault:   "Zero-point items in a weighted group silently distort the gradebook.",
		HelpRef:      "course-checklist#assessment-points",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44", "OSCQR 46"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedAssessmentItems, DataNeedGrading},
		Evaluate:     evalAssessmentPoints,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.grading",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalAssessmentPoints(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := sortAssessmentItems(assessmentItemsFor(snap))
	weightedIDs := map[string]bool{}
	for _, g := range snap.AssignmentGroups {
		if g.Weight != nil && *g.Weight > 0 {
			weightedIDs[g.ID.String()] = true
		}
	}
	var evidence []EvidenceRow
	for _, it := range items {
		inWeighted := it.AssignmentGroupID != nil && weightedIDs[it.AssignmentGroupID.String()]
		if !inWeighted && !usesWeightedGrading(snap) {
			// Unweighted: flag null/zero points on any gradable item.
			if it.Points == nil || *it.Points == 0 {
				evidence = append(evidence, assessmentEvidenceRow(it, "Missing or zero points", "assignment.grading"))
			}
			continue
		}
		if inWeighted && (it.Points == nil || *it.Points == 0) {
			evidence = append(evidence, assessmentEvidenceRow(it, "Missing or zero points in weighted group", "assignment.grading"))
		}
	}
	if len(items) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.points.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.points.detail.done",
			DetailDefault: "Graded items have points set.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.points.detail.todo",
		DetailDefault: fmt.Sprintf("%d items need points.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleAssessmentDatesWithinTerm() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentDatesWithinTerm,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.dates-within-term.title",
		TitleDefault: "Move due dates inside the term",
		WhyKey:       "coursechecklist.item.assessment.dates-within-term.why",
		WhyDefault:   "Due dates outside the term are a common copy-forward defect.",
		HelpRef:      "course-checklist#assessment-dates-within-term",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedAssessmentItems, DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return snap.StartsAt != nil && snap.EndsAt != nil && !strings.EqualFold(snap.ScheduleMode, "relative")
		},
		Evaluate: evalAssessmentDatesWithinTerm,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.scheduling",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalAssessmentDatesWithinTerm(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.StartsAt == nil || snap.EndsAt == nil {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.dates-within-term.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	start, end := *snap.StartsAt, *snap.EndsAt
	var evidence []EvidenceRow
	for _, it := range sortAssessmentItems(assessmentItemsFor(snap)) {
		if it.DueAt == nil {
			continue
		}
		d := it.DueAt.UTC()
		if d.Before(start.UTC()) || d.After(end.UTC()) {
			evidence = append(evidence, assessmentEvidenceRow(it, "Due date outside the term", "assignment.scheduling"))
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.dates-within-term.detail.done",
			DetailDefault: "All due dates fall inside the term.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.dates-within-term.detail.todo",
		DetailDefault: fmt.Sprintf("%d due dates fall outside the term.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleAssessmentAvailabilityWindows() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentAvailabilityWindows,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.availability-windows.title",
		TitleDefault: "Fix contradictory availability windows",
		WhyKey:       "coursechecklist.item.assessment.availability-windows.why",
		WhyDefault:   "Available-from after due, or available-until before due, locks learners out.",
		HelpRef:      "course-checklist#assessment-availability-windows",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems},
		Evaluate:     evalAssessmentAvailabilityWindows,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.scheduling",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalAssessmentAvailabilityWindows(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var evidence []EvidenceRow
	for _, it := range sortAssessmentItems(assessmentItemsFor(snap)) {
		if it.DueAt == nil {
			continue
		}
		due := it.DueAt.UTC()
		issue := ""
		if it.AvailableFrom != nil && it.AvailableFrom.UTC().After(due) {
			issue = "Available from is after the due date"
		}
		if it.AvailableUntil != nil && it.AvailableUntil.UTC().Before(due) {
			if issue != "" {
				issue += "; "
			}
			issue += "Available until is before the due date"
		}
		if issue != "" {
			evidence = append(evidence, assessmentEvidenceRow(it, issue, "assignment.scheduling"))
		}
	}
	if len(assessmentItemsFor(snap)) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.availability-windows.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.availability-windows.detail.done",
			DetailDefault: "Availability windows are consistent with due dates.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.availability-windows.detail.todo",
		DetailDefault: fmt.Sprintf("%d items have contradictory availability windows.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleAssessmentSpread() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAssessmentSpread,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.assessment.spread.title",
		TitleDefault: "Spread assessments across the term",
		WhyKey:       "coursechecklist.item.assessment.spread.why",
		WhyDefault:   "Stacking most points in one week overwhelms learners.",
		HelpRef:      "course-checklist#assessment-spread",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.4", "OSCQR 47"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems, DataNeedCourse},
		Evaluate:     evalAssessmentSpread,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalAssessmentSpread(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := assessmentItemsFor(snap)
	total := totalCoursePoints(items)
	if total <= 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.spread.detail.na",
			DetailDefault: "Does not apply when the course has no points.",
		}, nil
	}
	byWeek := map[string]int{}
	var keys []string
	for _, it := range items {
		if it.DueAt == nil || it.Points == nil || *it.Points <= 0 {
			continue
		}
		k := weekKey(*it.DueAt)
		if _, ok := byWeek[k]; !ok {
			keys = append(keys, k)
		}
		byWeek[k] += *it.Points
	}
	if len(byWeek) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.assessment.spread.detail.na-dates",
			DetailDefault: "Does not apply until assessments have due dates.",
		}, nil
	}
	// Cap pathological buckets (NFR: 520 weeks).
	if len(byWeek) > 520 {
		return Finding{
			Status:        StatusUnknown,
			DetailKey:     "coursechecklist.item.assessment.spread.detail.unknown",
			DetailDefault: "Too many distinct due-date weeks to evaluate.",
		}, nil
	}

	var overloaded []string
	var maxPct float64
	var maxWeek string
	for _, k := range keys {
		pct := 100.0 * float64(byWeek[k]) / float64(total)
		if pct > maxPct {
			maxPct = pct
			maxWeek = k
		}
		if pct > 40.0 {
			overloaded = append(overloaded, fmt.Sprintf("%s (%.0f%%)", k, pct))
		}
	}

	finalWeekPct := 0.0
	if snap.EndsAt != nil {
		fk := weekKey(*snap.EndsAt)
		finalWeekPct = 100.0 * float64(byWeek[fk]) / float64(total)
	} else {
		// Last due week by calendar.
		var last time.Time
		var lastKey string
		for _, it := range items {
			if it.DueAt == nil {
				continue
			}
			if last.IsZero() || it.DueAt.After(last) {
				last = *it.DueAt
				lastKey = weekKey(last)
			}
		}
		if lastKey != "" {
			finalWeekPct = 100.0 * float64(byWeek[lastKey]) / float64(total)
		}
	}

	if len(overloaded) == 0 && finalWeekPct < 50.0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.assessment.spread.detail.done",
			DetailDefault: fmt.Sprintf("Heaviest week holds %.0f%% of points.", maxPct),
			DetailFields:  map[string]any{"maxWeekPct": maxPct, "maxWeek": maxWeek, "finalWeekPct": finalWeekPct},
		}, nil
	}
	detail := fmt.Sprintf("Heaviest week holds %.0f%% of points", maxPct)
	if finalWeekPct >= 50.0 {
		detail += fmt.Sprintf("; final week holds %.0f%%", finalWeekPct)
	}
	detail += " (limits: 40% any week, 50% final week)."
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.assessment.spread.detail.todo",
		DetailDefault: detail,
		DetailFields: map[string]any{
			"maxWeekPct":    maxPct,
			"maxWeek":       maxWeek,
			"finalWeekPct":  finalWeekPct,
			"overloaded":    overloaded,
			"totalPoints":   total,
		},
	}, nil
}
