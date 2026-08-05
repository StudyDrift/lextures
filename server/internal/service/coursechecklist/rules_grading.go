package coursechecklist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/gradingdisplay"
)

func gradingRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleGradingGroupWeights(),
		ruleGradingEmptyGroups(),
		ruleGradingDropRules(),
		ruleGradingSchemeCoverage(),
		ruleGradingPostingPolicy(),
		ruleGradingLatePolicyConfigured(),
	}
}

func ruleGradingGroupWeights() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingGroupWeights,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.group-weights.title",
		TitleDefault: "Make weights total 100%",
		WhyKey:       "coursechecklist.item.grading.group-weights.why",
		WhyDefault:   "Weights that do not sum to 100% silently produce the wrong final grade.",
		HelpRef:      "course-checklist#grading-group-weights",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.2", "OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedGrading},
		Applies:      usesWeightedGrading,
		Evaluate:     evalGradingGroupWeights,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
			Anchor:  "course.grading.groups",
		},
	}
}

func evalGradingGroupWeights(_ context.Context, snap CourseSnapshot) (Finding, error) {
	sum := assignmentGroupWeightSum(snap.AssignmentGroups)
	if almostEqual(sum, 100.0, 0.01) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.group-weights.detail.done",
			DetailDefault: "Weights add up to 100%.",
			DetailFields:  map[string]any{"sum": sum},
		}, nil
	}
	// Format without trailing noise for common cases.
	sumStr := fmt.Sprintf("%.0f", sum)
	if !almostEqual(sum, float64(int(sum+0.5)), 0.01) {
		sumStr = fmt.Sprintf("%.2f", sum)
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.grading.group-weights.detail.todo",
		DetailDefault: fmt.Sprintf("Weights add up to %s%%, not 100%%", sumStr),
		DetailFields:  map[string]any{"sum": sum},
	}, nil
}

func ruleGradingEmptyGroups() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingEmptyGroups,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.empty-groups.title",
		TitleDefault: "Fill or remove empty weighted groups",
		WhyKey:       "coursechecklist.item.grading.empty-groups.why",
		WhyDefault:   "A weighted group with no items silently redistributes the grade.",
		HelpRef:      "course-checklist#grading-empty-groups",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedGrading, DataNeedAssessmentItems, DataNeedStructure},
		Applies:      usesWeightedGrading,
		Evaluate:     evalGradingEmptyGroups,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
			Anchor:  "course.grading.groups",
		},
		EvidenceShape: &EvidenceShape{Columns: gradingEvidenceColumns},
	}
}

func evalGradingEmptyGroups(_ context.Context, snap CourseSnapshot) (Finding, error) {
	counts := groupItemCounts(snap)
	var evidence []EvidenceRow
	for _, g := range snap.AssignmentGroups {
		if g.Weight == nil || *g.Weight <= 0 {
			continue
		}
		n := counts[g.ID]
		if n == 0 {
			evidence = append(evidence, EvidenceRow{
				Label:    g.Name,
				Sublabel: fmt.Sprintf("%s · 0 items", formatWeight(g.Weight)),
			})
		}
	}
	sortEvidenceByLabel(evidence)
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.empty-groups.detail.done",
			DetailDefault: "Every weighted group has at least one item.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.grading.empty-groups.detail.todo",
		DetailDefault: fmt.Sprintf("%d weighted groups have no items.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleGradingDropRules() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingDropRules,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.drop-rules.title",
		TitleDefault: "Fix impossible drop rules",
		WhyKey:       "coursechecklist.item.grading.drop-rules.why",
		WhyDefault:   "Dropping more items than a group contains cannot be applied.",
		HelpRef:      "course-checklist#grading-drop-rules",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedGrading, DataNeedAssessmentItems, DataNeedStructure},
		Evaluate:     evalGradingDropRules,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
			Anchor:  "course.grading.groups",
		},
		EvidenceShape: &EvidenceShape{Columns: gradingEvidenceColumns},
	}
}

func evalGradingDropRules(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.AssignmentGroups) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.grading.drop-rules.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	counts := groupItemCounts(snap)
	var evidence []EvidenceRow
	for _, g := range snap.AssignmentGroups {
		drops := g.DropLowest + g.DropHighest
		if drops <= 0 {
			continue
		}
		n := counts[g.ID]
		if drops >= n {
			evidence = append(evidence, EvidenceRow{
				Label: g.Name,
				Sublabel: fmt.Sprintf("drop %d of %d items (lowest %d, highest %d)",
					drops, n, g.DropLowest, g.DropHighest),
			})
		}
	}
	sortEvidenceByLabel(evidence)
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.drop-rules.detail.done",
			DetailDefault: "Drop rules fit within each group's item count.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.grading.drop-rules.detail.todo",
		DetailDefault: fmt.Sprintf("%d groups have impossible drop rules.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleGradingSchemeCoverage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingSchemeCoverage,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.scheme-coverage.title",
		TitleDefault: "Close gaps in your grading scale",
		WhyKey:       "coursechecklist.item.grading.scheme-coverage.why",
		WhyDefault:   "Letter bands should cover 0–100 with no gaps or overlaps.",
		HelpRef:      "course-checklist#grading-scheme-coverage",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedGrading, DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return snap.GradingSchemeID != nil || strings.EqualFold(snap.GradingScale, "letter") ||
				strings.EqualFold(snap.GradingScale, "gpa")
		},
		Evaluate: evalGradingSchemeCoverage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
		},
	}
}

func evalGradingSchemeCoverage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	scale := snap.GradingSchemeScaleJSON
	if len(scale) == 0 {
		// No custom scheme bands — treat platform defaults as covering 0–100.
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.scheme-coverage.detail.done-default",
			DetailDefault: "Using the default grading scale.",
		}, nil
	}
	kind, ok := gradingdisplay.ParseKind(snap.GradingScale)
	if !ok {
		kind = gradingdisplay.Letter
	}
	raw := json.RawMessage(scale)
	parsed, err := gradingdisplay.ParseScale(kind, &raw)
	if err != nil {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.grading.scheme-coverage.detail.todo",
			DetailDefault: "Grading scale has gaps or overlaps: " + err.Error(),
			DetailFields:  map[string]any{"error": err.Error()},
		}, nil
	}
	if kind == gradingdisplay.Letter || kind == gradingdisplay.Gpa {
		if len(parsed.LetterTiers) == 0 {
			return Finding{
				Status:        StatusTodo,
				DetailKey:     "coursechecklist.item.grading.scheme-coverage.detail.todo-empty",
				DetailDefault: "Grading scale has no letter bands.",
			}, nil
		}
	}
	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.grading.scheme-coverage.detail.done",
		DetailDefault: "Grading scale covers 0–100 with no gaps.",
	}, nil
}

func ruleGradingPostingPolicy() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingPostingPolicy,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.posting-policy.title",
		TitleDefault: "Choose when grades are posted",
		WhyKey:       "coursechecklist.item.grading.posting-policy.why",
		WhyDefault:   "Learners should know whether grades appear automatically or after review.",
		HelpRef:      "course-checklist#grading-posting-policy",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.5", "OSCQR 38"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems, DataNeedSyllabus},
		Evaluate:     evalGradingPostingPolicy,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
		},
	}
}

func evalGradingPostingPolicy(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := assessmentItemsFor(snap)
	assignments := 0
	manual := 0
	for _, it := range items {
		if it.Kind != "assignment" {
			continue
		}
		assignments++
		if strings.EqualFold(it.PostingPolicy, "manual") {
			manual++
		}
	}
	if assignments == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.grading.posting-policy.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	if manual == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.posting-policy.detail.done-auto",
			DetailDefault: "Grades post automatically.",
		}, nil
	}
	// Manual posting: syllabus should state when grades are released.
	text, _ := SyllabusPlainText(snap)
	if containsAny(strings.ToLower(text),
		[]string{"grade release", "grades are released", "grades will be posted", "posting grades", "when grades"}) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.posting-policy.detail.done-manual",
			DetailDefault: "Manual posting is set and the syllabus explains grade release.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.grading.posting-policy.detail.todo",
		DetailDefault: fmt.Sprintf("%d assignments use manual posting; add when grades are released to the syllabus.", manual),
		DetailFields:  map[string]any{"manualCount": manual},
	}, nil
}

func ruleGradingLatePolicyConfigured() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemGradingLatePolicyConfigured,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.grading.late-policy-configured.title",
		TitleDefault: "Configure late-work handling",
		WhyKey:       "coursechecklist.item.grading.late-policy-configured.why",
		WhyDefault:   "Every graded item should say whether late work is allowed, penalized, or blocked.",
		HelpRef:      "course-checklist#grading-late-policy-configured",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems, DataNeedItemMeta},
		Evaluate:     evalGradingLatePolicyConfigured,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.grading",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalGradingLatePolicyConfigured(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := sortAssessmentItems(assessmentItemsFor(snap))
	if len(items) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.grading.late-policy-configured.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	var evidence []EvidenceRow
	for _, it := range items {
		p := strings.TrimSpace(strings.ToLower(it.LateSubmissionPolicy))
		if p == "" || (p != "allow" && p != "penalty" && p != "block") {
			evidence = append(evidence, assessmentEvidenceRow(it, "Late policy not set", "assignment.grading"))
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.grading.late-policy-configured.detail.done",
			DetailDefault: "Late-work policy is set on every graded item.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.grading.late-policy-configured.detail.todo",
		DetailDefault: fmt.Sprintf("%d items need a late-work policy.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func containsAny(text string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}
