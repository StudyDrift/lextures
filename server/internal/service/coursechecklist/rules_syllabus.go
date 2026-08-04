package coursechecklist

import (
	"context"
	"fmt"
	"strings"
)

func syllabusRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleSyllabusExists(),
		ruleSyllabusGradingPolicy(),
		ruleSyllabusLatePolicy(),
		ruleSyllabusAcademicIntegrity(),
		ruleSyllabusAccessibilityStatement(),
		ruleSyllabusAcceptanceDecision(),
		ruleSyllabusPrintable(),
	}
}

func ruleSyllabusExists() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusExists,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.exists.title",
		TitleDefault: "Publish a course syllabus",
		WhyKey:       "coursechecklist.item.syllabus.exists.why",
		WhyDefault:   "A published syllabus is the first place learners look for policies and expectations.",
		HelpRef:      "course-checklist#syllabus-exists",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 3", "QM 1.2"},
		DataNeeds:    []DataNeed{DataNeedSyllabus},
		Evaluate:     evalSyllabusExists,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalSyllabusExists(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusExists), nil
	}
	text, trunc := SyllabusPlainText(snap)
	chars := len([]rune(strings.TrimSpace(text)))
	sections := len(snap.SyllabusSections)
	if sections >= 2 && chars >= 600 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.syllabus.exists.detail.done",
			DetailDefault: truncatedDetail("Syllabus has enough sections and content.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.syllabus.exists.detail.todo",
		DetailDefault: truncatedDetail(fmt.Sprintf("Syllabus has %d sections and %d characters; aim for ≥2 sections and ≥600 characters.", sections, chars), trunc),
		DetailFields:  map[string]any{"sections": sections, "characters": chars},
	}, nil
}

func ruleSyllabusGradingPolicy() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusGradingPolicy,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.grading-policy.title",
		TitleDefault: "Explain how grades are calculated",
		WhyKey:       "coursechecklist.item.syllabus.grading-policy.why",
		WhyDefault:   "Learners need to know how weights, points, or scales produce a final grade.",
		HelpRef:      "course-checklist#syllabus-grading-policy",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.2", "OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedSyllabus, DataNeedGrading},
		Evaluate:     evalSyllabusGradingPolicy,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalSyllabusGradingPolicy(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusGradingPolicy), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if !lx.GradingPolicy.Match(text) {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.syllabus.grading-policy.detail.todo",
			DetailDefault: truncatedDetail("Describe how grades are computed (weights, points, or scale).", trunc),
		}, nil
	}
	weighted := strings.Contains(strings.ToLower(text), "weight")
	sum := assignmentGroupWeightSum(snap.AssignmentGroups)
	if weighted && sum == 0 && len(snap.AssignmentGroups) > 0 {
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.syllabus.grading-policy.detail.weights",
			DetailDefault: truncatedDetail("Syllabus mentions weights, but assignment groups sum to 0.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.syllabus.grading-policy.detail.done",
		DetailDefault: truncatedDetail("Grading policy is published.", trunc),
	}, nil
}

func ruleSyllabusLatePolicy() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusLatePolicy,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.late-policy.title",
		TitleDefault: "Publish a late-work policy",
		WhyKey:       "coursechecklist.item.syllabus.late-policy.why",
		WhyDefault:   "A late-work policy reduces confusion when deadlines are missed.",
		HelpRef:      "course-checklist#syllabus-late-policy",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.2", "OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedSyllabus, DataNeedStructure, DataNeedItemMeta},
		Evaluate:     evalSyllabusLatePolicy,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Policy"}},
	}
}

func evalSyllabusLatePolicy(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusLatePolicy), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	hasPolicy := lx.LatePolicyPresent.Match(text)
	noLate := lx.LatePolicyNoLate.Match(text)
	if !hasPolicy && !noLate {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.syllabus.late-policy.detail.todo",
			DetailDefault: truncatedDetail("State a late-work policy in the syllabus.", trunc),
		}, nil
	}

	type itemPol struct {
		title, policy string
	}
	var items []itemPol
	allDefaultAllow := true
	var contradictions []EvidenceRow
	for _, it := range snap.StructureItems {
		if it.Archived || !isGradableKind(it.Kind) {
			continue
		}
		meta, ok := snap.ItemMeta[it.ID]
		policy := ""
		if ok {
			policy = meta.LateSubmissionPolicy
		}
		if policy == "" {
			policy = "allow"
		}
		items = append(items, itemPol{title: it.Title, policy: policy})
		if policy != "allow" {
			allDefaultAllow = false
		}
		if noLate && policy == "allow" {
			contradictions = append(contradictions, EvidenceRow{
				Label: it.Title, Sublabel: policy, Status: StatusTodo,
			})
		}
	}

	if noLate && len(contradictions) > 0 {
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.syllabus.late-policy.detail.contradiction",
			DetailDefault: truncatedDetail(fmt.Sprintf("Syllabus says no late work, but %d items still allow late submissions.", len(contradictions)), trunc),
			Evidence:      contradictions,
			Progress:      &Progress{Done: len(items) - len(contradictions), Total: len(items)},
		}, nil
	}

	if len(items) > 0 && allDefaultAllow && !noLate {
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.syllabus.late-policy.detail.defaults",
			DetailDefault: truncatedDetail("Syllabus states a late policy, but all graded items still use the default allow setting.", trunc),
		}, nil
	}

	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.syllabus.late-policy.detail.done",
		DetailDefault: truncatedDetail("Late-work policy is published and item settings are consistent.", trunc),
	}, nil
}

func ruleSyllabusAcademicIntegrity() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusAcademicIntegrity,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.academic-integrity.title",
		TitleDefault: "State academic integrity expectations",
		WhyKey:       "coursechecklist.item.syllabus.academic-integrity.why",
		WhyDefault:   "An integrity statement sets expectations for honesty, plagiarism, and AI use.",
		HelpRef:      "course-checklist#syllabus-academic-integrity",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.3", "OSCQR 5"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalSyllabusAcademicIntegrity,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalSyllabusAcademicIntegrity(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusAcademicIntegrity), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.AcademicIntegrity.Match(text) {
		detail := "Academic integrity section is present."
		if snap.Features.AiTutorEnabled || snap.Features.ModulesAiAssistantEnabled {
			detail += " Consider stating an AI-use policy as well."
		}
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.syllabus.academic-integrity.detail.done",
			DetailDefault: truncatedDetail(detail, trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.syllabus.academic-integrity.detail.todo",
		DetailDefault: truncatedDetail("Add an academic integrity section to the syllabus.", trunc),
	}, nil
}

func ruleSyllabusAccessibilityStatement() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusAccessibilityStatement,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.accessibility-statement.title",
		TitleDefault: "Add an accessibility statement",
		WhyKey:       "coursechecklist.item.syllabus.accessibility-statement.why",
		WhyDefault:   "An accessibility or accommodations statement tells learners how to request support.",
		HelpRef:      "course-checklist#syllabus-accessibility",
		Tier:         TierRecommended,
		Sources:      []string{"QM 8.1", "OSCQR 5"},
		DataNeeds:    []DataNeed{DataNeedSyllabus},
		Evaluate:     evalSyllabusAccessibilityStatement,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalSyllabusAccessibilityStatement(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusAccessibilityStatement), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.Accessibility.Match(text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.syllabus.accessibility-statement.detail.done",
			DetailDefault: truncatedDetail("Accessibility statement is present.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.syllabus.accessibility-statement.detail.todo",
		DetailDefault: truncatedDetail("Add an accessibility or accommodations section.", trunc),
	}, nil
}

func ruleSyllabusAcceptanceDecision() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusAcceptanceDecision,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.acceptance-decision.title",
		TitleDefault: "Decide on syllabus acceptance",
		WhyKey:       "coursechecklist.item.syllabus.acceptance-decision.why",
		WhyDefault:   "Choosing whether learners must accept the syllabus makes the policy intentional.",
		HelpRef:      "course-checklist#syllabus-acceptance-decision",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedSyllabus},
		Evaluate:     evalSyllabusAcceptanceDecision,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalSyllabusAcceptanceDecision(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusAcceptanceDecision), nil
	}
	if snap.AcceptanceDecidedAt != nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.syllabus.acceptance-decision.detail.done",
			DetailDefault: "Syllabus acceptance requirement has been set.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.syllabus.acceptance-decision.detail.todo",
		DetailDefault: "Choose whether learners must accept the syllabus.",
	}, nil
}

func ruleSyllabusPrintable() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemSyllabusPrintable,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.syllabus.printable.title",
		TitleDefault: "Keep the syllabus printable",
		WhyKey:       "coursechecklist.item.syllabus.printable.why",
		WhyDefault:   "Learners often print or export the syllabus; unsupported embeds break that path.",
		HelpRef:      "course-checklist#syllabus-printable",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 4"},
		DataNeeds:    []DataNeed{DataNeedSyllabus},
		Evaluate:     evalSyllabusPrintable,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Embed"}},
	}
}

func evalSyllabusPrintable(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemSyllabusPrintable), nil
	}
	text, trunc := SyllabusPlainText(snap)
	offenders := hasPrintBreakingEmbeds(text)
	if len(offenders) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.syllabus.printable.detail.done",
			DetailDefault: truncatedDetail("Syllabus has no print-breaking embeds.", trunc),
		}, nil
	}
	ev := make([]EvidenceRow, 0, len(offenders))
	for _, o := range offenders {
		ev = append(ev, EvidenceRow{Label: o, Status: StatusTodo})
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.syllabus.printable.detail.todo",
		DetailDefault: truncatedDetail("Remove or replace embeds that break the print view.", trunc),
		Evidence:      ev,
	}, nil
}
