package coursechecklist

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

func ruleOutcomesAssessmentMapping() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesAssessmentMapping,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.assessment-mapping.title",
		TitleDefault: "Map every assessment to an outcome",
		WhyKey:       "coursechecklist.item.outcomes.assessment-mapping.why",
		WhyDefault:   "Alignment is what makes a grade mean something — every rubric asks that each assessment measure a stated outcome.",
		HelpRef:      "course-checklist#outcomes-assessment-mapping",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.1", "NSQ C", "NSQ D", "OSCQR 45"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedOutcomes},
		Evaluate:     evalOutcomesAssessmentMapping,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type", "Module", "Points"}},
	}
}

func evalOutcomesAssessmentMapping(_ context.Context, snap CourseSnapshot) (Finding, error) {
	gradables := GradableItemsFor(snap)
	if len(gradables) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.assessment-mapping.detail.na",
			DetailDefault: "No graded assignments or quizzes to map.",
		}, nil
	}
	mapped := map[uuid.UUID]struct{}{}
	for _, l := range snap.OutcomeLinks {
		mapped[l.ItemID] = struct{}{}
	}
	byID := structureByID(snap)
	var unmapped []EvidenceRow
	mappedCount := 0
	sorted := sortGradableItems(gradables, byID)
	for _, g := range sorted {
		if _, ok := mapped[g.ID]; ok {
			mappedCount++
			continue
		}
		modTitle := ""
		if g.ParentID != nil {
			if m, ok := byID[*g.ParentID]; ok {
				modTitle = m.Title
			}
		}
		pts := "—"
		if g.Points != nil {
			pts = fmt.Sprintf("%d", *g.Points)
		}
		anchor := "assignment.outcomes-mapping"
		route := "/courses/{courseCode}/modules/assignment/{itemId}"
		if g.Kind == "quiz" {
			anchor = "quiz.outcomes"
			route = "/courses/{courseCode}/modules/quiz/{itemId}"
		}
		unmapped = append(unmapped, EvidenceRow{
			Label:    g.Title,
			Sublabel: fmt.Sprintf("%s · %s · %s", humanKind(g.Kind), modTitle, pts),
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface:   "web",
				Route:     route,
				Anchor:    anchor,
				EntityKey: g.ID.String(),
			},
		})
	}
	total := len(sorted)
	if len(unmapped) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.assessment-mapping.detail.done",
			DetailDefault: "Every assessment is mapped to an outcome.",
			Progress:      &Progress{Done: mappedCount, Total: total},
		}, nil
	}
	missing := len(unmapped)
	return Finding{
		Status:        StatusInProgress,
		DetailKey:     "coursechecklist.item.outcomes.assessment-mapping.detail.partial",
		DetailDefault: fmt.Sprintf("%d of %d assessments aren't mapped.", missing, total),
		DetailFields: map[string]any{
			"mapped":        mappedCount,
			"totalGradable": total,
			"unmapped":      missing,
		},
		Progress: &Progress{Done: mappedCount, Total: total},
		Evidence: unmapped,
	}, nil
}

func ruleOutcomesCoverage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesCoverage,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.coverage.title",
		TitleDefault: "Assess every learning outcome",
		WhyKey:       "coursechecklist.item.outcomes.coverage.why",
		WhyDefault:   "Outcomes that are never measured leave a gap in the assessment plan.",
		HelpRef:      "course-checklist#outcomes-coverage",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.1", "NSQ D"},
		DataNeeds:    []DataNeed{DataNeedOutcomes},
		Evaluate:     evalOutcomesCoverage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Outcome"}},
	}
}

func evalOutcomesCoverage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.coverage.detail.na",
			DetailDefault: "No outcomes to check.",
		}, nil
	}
	covered := map[uuid.UUID]struct{}{}
	for _, l := range snap.OutcomeLinks {
		covered[l.OutcomeID] = struct{}{}
	}
	var evidence []EvidenceRow
	for _, o := range sortOutcomes(snap.Outcomes) {
		if _, ok := covered[o.ID]; ok {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:  o.Title,
			Status: StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/settings/outcomes",
				Anchor:  "outcome:" + o.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.coverage.detail.done",
			DetailDefault: "Every outcome is measured by at least one assessment.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.coverage.detail.todo",
		DetailDefault: fmt.Sprintf("%d outcomes have no assessment link.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleOutcomesSummativeCoverage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesSummativeCoverage,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.summative-coverage.title",
		TitleDefault: "Include a summative check per outcome",
		WhyKey:       "coursechecklist.item.outcomes.summative-coverage.why",
		WhyDefault:   "Formative practice alone is not enough — each outcome needs a summative measure.",
		HelpRef:      "course-checklist#outcomes-summative-coverage",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.1"},
		DataNeeds:    []DataNeed{DataNeedOutcomes},
		Evaluate:     evalOutcomesSummativeCoverage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Outcome"}},
	}
}

func evalOutcomesSummativeCoverage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.summative-coverage.detail.na",
			DetailDefault: "No outcomes to check.",
		}, nil
	}
	summative := map[uuid.UUID]struct{}{}
	anyLink := map[uuid.UUID]struct{}{}
	for _, l := range snap.OutcomeLinks {
		anyLink[l.OutcomeID] = struct{}{}
		if strings.EqualFold(l.MeasurementLevel, "summative") || strings.EqualFold(l.MeasurementLevel, "performance") {
			summative[l.OutcomeID] = struct{}{}
		}
	}
	var evidence []EvidenceRow
	for _, o := range sortOutcomes(snap.Outcomes) {
		if _, ok := summative[o.ID]; ok {
			continue
		}
		// Only list outcomes that have some link (formative-only) or none — both need summative.
		evidence = append(evidence, EvidenceRow{
			Label:  o.Title,
			Status: StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/settings/outcomes",
				Anchor:  "outcome:" + o.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.summative-coverage.detail.done",
			DetailDefault: "Every outcome has a summative measure.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.summative-coverage.detail.todo",
		DetailDefault: fmt.Sprintf("%d outcomes lack a summative measure.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleOutcomesMasteryScale() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesMasteryScale,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.mastery-scale.title",
		TitleDefault: "Configure the mastery scale",
		WhyKey:       "coursechecklist.item.outcomes.mastery-scale.why",
		WhyDefault:   "Standards-based grading needs a proficiency scale and mastery threshold.",
		HelpRef:      "course-checklist#outcomes-mastery-scale",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return snap.SbgEnabled
		},
		Evaluate: evalOutcomesMasteryScale,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
			Anchor:  "course.grading.sbg",
		},
	}
}

func evalOutcomesMasteryScale(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if !snap.SbgEnabled {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.mastery-scale.detail.na",
			DetailDefault: "Standards-based grading is off.",
		}, nil
	}
	ok, detail := sbgScaleConfigured(snap.SbgProficiencyScaleJSON)
	if ok {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.mastery-scale.detail.done",
			DetailDefault: "Proficiency scale and mastery threshold are set.",
		}, nil
	}
	if detail == "" {
		detail = "Configure a proficiency scale with a mastery threshold."
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.mastery-scale.detail.todo",
		DetailDefault: detail,
	}, nil
}

func sbgScaleConfigured(raw json.RawMessage) (bool, string) {
	if len(raw) == 0 || string(raw) == "null" {
		return false, "Add a proficiency scale JSON."
	}
	var payload struct {
		Levels []struct {
			Level    *float64 `json:"level"`
			Label    string   `json:"label"`
			MinScore *float64 `json:"minScore"`
		} `json:"levels"`
		MasteryThreshold *float64 `json:"masteryThreshold"`
		MasteryLevel     *float64 `json:"masteryLevel"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, "Proficiency scale JSON is invalid."
	}
	if len(payload.Levels) == 0 {
		return false, "Add proficiency levels to the scale."
	}
	hasThreshold := payload.MasteryThreshold != nil || payload.MasteryLevel != nil
	if !hasThreshold {
		// Accept a level labeled mastery / proficient / meets.
		for _, lv := range payload.Levels {
			lab := strings.ToLower(strings.TrimSpace(lv.Label))
			if strings.Contains(lab, "master") || strings.Contains(lab, "proficien") ||
				strings.Contains(lab, "meets") || strings.Contains(lab, "exceeds") {
				hasThreshold = true
				break
			}
		}
	}
	if !hasThreshold {
		return false, "Set a mastery threshold on the proficiency scale."
	}
	return true, ""
}

func ruleOutcomesStandardsAlignment() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesStandardsAlignment,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.standards-alignment.title",
		TitleDefault: "Align assessments to standards",
		WhyKey:       "coursechecklist.item.outcomes.standards-alignment.why",
		WhyDefault:   "When standards alignment is on, every graded item should map to a standard.",
		HelpRef:      "course-checklist#outcomes-standards-alignment",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedItemMeta, DataNeedStandards},
		Applies: func(snap CourseSnapshot) bool {
			return snap.Features.StandardsAlignmentEnabled || snap.StandardsEnabled
		},
		Evaluate: evalOutcomesStandardsAlignment,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/standards-coverage",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type", "Module", "Points"}},
	}
}

func evalOutcomesStandardsAlignment(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if !snap.Features.StandardsAlignmentEnabled && !snap.StandardsEnabled {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.standards-alignment.detail.na",
			DetailDefault: "Standards alignment is off.",
		}, nil
	}
	if snap.StandardsCount < 1 {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.outcomes.standards-alignment.detail.nostandards",
			DetailDefault: "Attach at least one course standard before aligning items.",
		}, nil
	}
	gradables := GradableItemsFor(snap)
	if len(gradables) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.standards-alignment.detail.nogradable",
			DetailDefault: "No graded items to align.",
		}, nil
	}
	byID := structureByID(snap)
	var unaligned []EvidenceRow
	aligned := 0
	for _, g := range sortGradableItems(gradables, byID) {
		if _, ok := snap.StandardAlignedItemIDs[g.ID]; ok {
			aligned++
			continue
		}
		modTitle := ""
		if g.ParentID != nil {
			if m, ok := byID[*g.ParentID]; ok {
				modTitle = m.Title
			}
		}
		pts := "—"
		if g.Points != nil {
			pts = fmt.Sprintf("%d", *g.Points)
		}
		unaligned = append(unaligned, EvidenceRow{
			Label:    g.Title,
			Sublabel: fmt.Sprintf("%s · %s · %s", humanKind(g.Kind), modTitle, pts),
			Status:   StatusTodo,
		})
	}
	total := len(gradables)
	if len(unaligned) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.standards-alignment.detail.done",
			DetailDefault: "Every graded item has a standards alignment.",
			Progress:      &Progress{Done: aligned, Total: total},
		}, nil
	}
	return Finding{
		Status:        StatusInProgress,
		DetailKey:     "coursechecklist.item.outcomes.standards-alignment.detail.partial",
		DetailDefault: fmt.Sprintf("%d of %d graded items lack a standard alignment.", len(unaligned), total),
		Progress:      &Progress{Done: aligned, Total: total},
		Evidence:      unaligned,
	}, nil
}

func ruleOutcomesSyllabusPublished() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesSyllabusPublished,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.syllabus-published.title",
		TitleDefault: "Publish outcomes in the syllabus",
		WhyKey:       "coursechecklist.item.outcomes.syllabus-published.why",
		WhyDefault:   "Learners should be able to read the course outcomes in the syllabus.",
		HelpRef:      "course-checklist#outcomes-syllabus-published",
		Tier:         TierRecommended,
		Sources:      []string{"QM 2.3", "OSCQR 9"},
		DataNeeds:    []DataNeed{DataNeedOutcomes, DataNeedSyllabus},
		Evaluate:     evalOutcomesSyllabusPublished,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOutcomesSyllabusPublished(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.syllabus-published.detail.na",
			DetailDefault: "No outcomes to publish.",
		}, nil
	}
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOutcomesSyllabusPublished), nil
	}
	text, truncated := SyllabusPlainText(snap)
	lower := strings.ToLower(text)
	// Section key/title referencing outcomes.
	for _, s := range snap.SyllabusSections {
		key := strings.ToLower(s.Key + " " + s.Title)
		if strings.Contains(key, "outcome") || strings.Contains(key, "objective") || strings.Contains(key, "learning goal") {
			if s.HasBody || strings.TrimSpace(text) != "" {
				return Finding{
					Status:        StatusDone,
					DetailKey:     "coursechecklist.item.outcomes.syllabus-published.detail.done",
					DetailDefault: truncatedDetail("Outcomes appear in the syllabus.", truncated),
				}, nil
			}
		}
	}
	// Token overlap ≥ 60% of outcome title tokens.
	matched := 0
	for _, o := range snap.Outcomes {
		tokens := significantTokens(o.Title)
		if len(tokens) == 0 {
			continue
		}
		hits := 0
		for _, tok := range tokens {
			if strings.Contains(lower, tok) {
				hits++
			}
		}
		if float64(hits)/float64(len(tokens)) >= 0.6 {
			matched++
		}
	}
	if len(snap.Outcomes) > 0 && float64(matched)/float64(len(snap.Outcomes)) >= 0.6 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.syllabus-published.detail.done",
			DetailDefault: truncatedDetail("Outcome titles appear in the syllabus.", truncated),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.syllabus-published.detail.todo",
		DetailDefault: truncatedDetail("Add course outcomes to the syllabus so learners can read them.", truncated),
	}, nil
}

func significantTokens(title string) []string {
	parts := strings.Fields(strings.ToLower(title))
	var out []string
	stop := map[string]struct{}{
		"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "of": {}, "to": {}, "in": {}, "for": {}, "on": {},
	}
	for _, p := range parts {
		p = strings.Trim(p, ".,;:()[]\"'")
		if len(p) < 3 {
			continue
		}
		if _, ok := stop[p]; ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func sortOutcomes(outcomes []OutcomeSnap) []OutcomeSnap {
	out := append([]OutcomeSnap(nil), outcomes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].Title < out[j].Title
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}

func sortGradableItems(items []GradableItem, byID map[uuid.UUID]StructureItem) []GradableItem {
	out := append([]GradableItem(nil), items...)
	parentOrder := map[uuid.UUID]int{}
	for _, it := range byID {
		if it.Kind == "module" {
			parentOrder[it.ID] = it.SortOrder
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := 0, 0
		if out[i].ParentID != nil {
			pi = parentOrder[*out[i].ParentID]
		}
		if out[j].ParentID != nil {
			pj = parentOrder[*out[j].ParentID]
		}
		if pi != pj {
			return pi < pj
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Title < out[j].Title
	})
	return out
}
