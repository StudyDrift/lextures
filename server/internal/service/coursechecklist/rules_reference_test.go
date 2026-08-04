package coursechecklist

import (
	"context"
	"fmt"
)

// Reference helpers kept for engine unit tests that build custom registries.
// They are NOT registered in BuildBuiltinRegistry (CC.3). course.sections is retired.

func referenceCourseDates() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseDates,
		Category:     CategoryReference,
		TitleKey:     "coursechecklist.item.course.dates.title",
		TitleDefault: "Set course start and end dates",
		WhyKey:       "coursechecklist.item.course.dates.why",
		WhyDefault:   "Learners need clear term boundaries for planning and pacing.",
		HelpRef:      "course-checklist#course-dates",
		Tier:         TierEssential,
		Sources:      []string{"OSCQR 1", "QM 1.2"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Applies:      nil, // always applies
		Evaluate:     evalCourseDatesReference,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.dates",
		},
	}
}

func evalCourseDatesReference(_ context.Context, snap CourseSnapshot) (Finding, error) {
	hasStart := snap.StartsAt != nil
	hasEnd := snap.EndsAt != nil
	switch {
	case hasStart && hasEnd:
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.dates.detail.done",
			DetailDefault: "Start and end dates are set.",
		}, nil
	case hasStart || hasEnd:
		done := 0
		if hasStart {
			done = 1
		}
		if hasEnd {
			done++
		}
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.course.dates.detail.partial",
			DetailDefault: "Set both a start date and an end date.",
			Progress:      &Progress{Done: done, Total: 2},
		}, nil
	default:
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.course.dates.detail.todo",
			DetailDefault: "Add a start date and an end date for this course.",
		}, nil
	}
}

func referenceCourseSections() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseSections,
		Category:     CategoryReference,
		TitleKey:     "coursechecklist.item.course.sections.title",
		TitleDefault: "Create at least one section",
		WhyKey:       "coursechecklist.item.course.sections.why",
		WhyDefault:   "Sectioned courses need a section before students can be rostered correctly.",
		HelpRef:      "course-checklist#course-sections",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSections},
		Applies: func(snap CourseSnapshot) bool {
			return snap.SectionsEnabled
		},
		Evaluate: evalCourseSections,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/sections",
			Anchor:  "course.sections.list",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Section", "Code"}},
	}
}

func evalCourseSections(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Sections) == 0 {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.course.sections.detail.todo",
			DetailDefault: "Create a section so enrollments can be assigned.",
		}, nil
	}
	evidence := make([]EvidenceRow, 0, len(snap.Sections))
	for _, s := range snap.Sections {
		name := s.Name
		if name == "" {
			name = s.SectionCode
		}
		evidence = append(evidence, EvidenceRow{
			Label:    name,
			Sublabel: s.SectionCode,
			Status:   StatusDone,
		})
	}
	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.course.sections.detail.done",
		DetailDefault: fmt.Sprintf("%d section(s) configured.", len(snap.Sections)),
		DetailFields:  map[string]any{"count": len(snap.Sections)},
		Evidence:      evidence,
	}, nil
}
