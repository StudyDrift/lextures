package coursechecklist

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func outcomesRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleOutcomesDefined(),
		ruleOutcomesMeasurable(),
		ruleOutcomesDescribed(),
		ruleOutcomesModuleAlignment(),
		ruleOutcomesAssessmentMapping(),
		ruleOutcomesCoverage(),
		ruleOutcomesSummativeCoverage(),
		ruleOutcomesMasteryScale(),
		ruleOutcomesStandardsAlignment(),
		ruleOutcomesSyllabusPublished(),
	}
}

func ruleOutcomesDefined() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesDefined,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.defined.title",
		TitleDefault: "Define at least three learning outcomes",
		WhyKey:       "coursechecklist.item.outcomes.defined.why",
		WhyDefault:   "Stated outcomes tell learners what success looks like before they start.",
		HelpRef:      "course-checklist#outcomes-defined",
		Tier:         TierRecommended,
		Sources:      []string{"QM 2.1", "NSQ C", "OSCQR 9"},
		DataNeeds:    []DataNeed{DataNeedOutcomes},
		Evaluate:     evalOutcomesDefined,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
	}
}

func evalOutcomesDefined(_ context.Context, snap CourseSnapshot) (Finding, error) {
	n := len(snap.Outcomes)
	const need = 3
	switch {
	case n >= need:
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.defined.detail.done",
			DetailDefault: fmt.Sprintf("%d learning outcomes are defined.", n),
			Progress:      &Progress{Done: n, Total: need},
		}, nil
	case n >= 1:
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.outcomes.defined.detail.partial",
			DetailDefault: fmt.Sprintf("%d of %d learning outcomes defined.", n, need),
			Progress:      &Progress{Done: n, Total: need},
		}, nil
	default:
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.outcomes.defined.detail.todo",
			DetailDefault: "Add at least three course learning outcomes.",
			Progress:      &Progress{Done: 0, Total: need},
		}, nil
	}
}

func ruleOutcomesMeasurable() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesMeasurable,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.measurable.title",
		TitleDefault: "Start outcomes with measurable verbs",
		WhyKey:       "coursechecklist.item.outcomes.measurable.why",
		WhyDefault:   "Measurable verbs make it clear how learners will demonstrate the outcome.",
		HelpRef:      "course-checklist#outcomes-measurable",
		Tier:         TierRecommended,
		Sources:      []string{"QM 2.1", "QM 2.4"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedOutcomes},
		Evaluate:     evalOutcomesMeasurable,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Outcome", "Suggestion"}},
	}
}

func evalOutcomesMeasurable(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.measurable.detail.na",
			DetailDefault: "No outcomes to check.",
		}, nil
	}
	lx := BloomLexiconFor(snapLocale(snap))
	if lx == nil {
		return Finding{
			Status:        StatusUnknown,
			DetailKey:     "coursechecklist.item.outcomes.measurable.detail.unknown",
			DetailDefault: "Measurable-verb check is unavailable for this course language.",
		}, nil
	}
	var evidence []EvidenceRow
	for _, o := range sortOutcomes(snap.Outcomes) {
		_, flagged, sug := measurableOutcomeVerb(o.Title, lx)
		if !flagged {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    o.Title,
			Sublabel: "try: " + sug,
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/settings/outcomes",
				Anchor:  "course.outcomes.item", EntityKey: o.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.measurable.detail.done",
			DetailDefault: "Outcome titles start with measurable verbs.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.measurable.detail.todo",
		DetailDefault: fmt.Sprintf("%d outcomes use a non-measurable opener.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleOutcomesDescribed() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesDescribed,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.described.title",
		TitleDefault: "Add a description to every outcome",
		WhyKey:       "coursechecklist.item.outcomes.described.why",
		WhyDefault:   "A short description clarifies scope beyond the title verb.",
		HelpRef:      "course-checklist#outcomes-described",
		Tier:         TierRecommended,
		Sources:      []string{"QM 2.3"},
		DataNeeds:    []DataNeed{DataNeedOutcomes},
		Evaluate:     evalOutcomesDescribed,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Outcome"}},
	}
}

func evalOutcomesDescribed(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.described.detail.na",
			DetailDefault: "No outcomes to check.",
		}, nil
	}
	var evidence []EvidenceRow
	for _, o := range sortOutcomes(snap.Outcomes) {
		if strings.TrimSpace(o.Description) != "" {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:  o.Title,
			Status: StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/settings/outcomes",
				Anchor:  "course.outcomes.item", EntityKey: o.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.described.detail.done",
			DetailDefault: "Every outcome has a description.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.described.detail.todo",
		DetailDefault: fmt.Sprintf("%d outcomes are missing a description.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleOutcomesModuleAlignment() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOutcomesModuleAlignment,
		Category:     CategoryOutcomes,
		TitleKey:     "coursechecklist.item.outcomes.module-alignment.title",
		TitleDefault: "Map each module to an outcome",
		WhyKey:       "coursechecklist.item.outcomes.module-alignment.why",
		WhyDefault:   "Every module should teach toward at least one stated outcome.",
		HelpRef:      "course-checklist#outcomes-module-alignment",
		Tier:         TierRecommended,
		Sources:      []string{"QM 2.2", "NSQ C"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedOutcomes},
		Evaluate:     evalOutcomesModuleAlignment,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/outcomes",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module"}},
	}
}

func evalOutcomesModuleAlignment(_ context.Context, snap CourseSnapshot) (Finding, error) {
	mods := listModules(snap)
	if len(mods) == 0 || len(snap.Outcomes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.outcomes.module-alignment.detail.na",
			DetailDefault: "Need modules and outcomes to check alignment.",
		}, nil
	}
	linkedItems := map[uuid.UUID]struct{}{}
	for _, l := range snap.OutcomeLinks {
		linkedItems[l.ItemID] = struct{}{}
	}
	children := childrenByParent(snap)
	var evidence []EvidenceRow
	for _, m := range mods {
		has := false
		for _, c := range children[m.ID] {
			if _, ok := linkedItems[c.ID]; ok {
				has = true
				break
			}
		}
		if !has {
			evidence = append(evidence, EvidenceRow{
				Label:  m.Title,
				Status: StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web",
					Route:   "/courses/{courseCode}/modules",
					Anchor:  "modules.module", EntityKey: m.ID.String(),
				},
			})
		}
	}
	evidence = sortEvidenceByLabel(evidence)
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.outcomes.module-alignment.detail.done",
			DetailDefault: "Every module has at least one outcome-mapped item.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.outcomes.module-alignment.detail.todo",
		DetailDefault: fmt.Sprintf("%d modules teach toward nothing.", len(evidence)),
		Evidence:      evidence,
	}, nil
}
