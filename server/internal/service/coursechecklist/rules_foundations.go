package coursechecklist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func foundationsRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleCourseTitleAndDescription(),
		ruleCourseDates(),
		ruleCourseTimezone(),
		ruleCoursePublished(),
		ruleCourseRelativeSchedule(),
		ruleCourseVisibilityWindow(),
		ruleCourseGradingScheme(),
		ruleCourseHeroImage(),
		ruleCourseHomeLanding(),
		ruleCourseFeaturesReviewed(),
		ruleCourseLanguage(),
	}
}

func ruleCourseTitleAndDescription() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseTitleAndDescription,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.title-and-description.title",
		TitleDefault: "Write a clear title and description",
		WhyKey:       "coursechecklist.item.course.title-and-description.why",
		WhyDefault:   "Learners decide whether a course fits them from the title and short description.",
		HelpRef:      "course-checklist#course-title-and-description",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.1", "OSCQR 2"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseTitleAndDescription,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.description",
		},
	}
}

func evalCourseTitleAndDescription(_ context.Context, snap CourseSnapshot) (Finding, error) {
	titleOK := !isPlaceholderTitle(snap.Title)
	descLen := len([]rune(strings.TrimSpace(snap.Description)))
	descOK := descLen >= 120
	switch {
	case titleOK && descOK:
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.title-and-description.detail.done",
			DetailDefault: "Title and description are set.",
		}, nil
	case titleOK || descOK:
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.course.title-and-description.detail.partial",
			DetailDefault: fmt.Sprintf("Description is %d characters; aim for at least 120.", descLen),
			DetailFields:  map[string]any{"descriptionLength": descLen},
			Progress:      &Progress{Done: boolToInt(titleOK) + boolToInt(descOK), Total: 2},
		}, nil
	default:
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.course.title-and-description.detail.todo",
			DetailDefault: "Add a real course title and a description of at least 120 characters.",
		}, nil
	}
}

func ruleCourseDates() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseDates,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.dates.title",
		TitleDefault: "Set course start and end dates",
		WhyKey:       "coursechecklist.item.course.dates.why",
		WhyDefault:   "Learners need clear term boundaries for planning and pacing.",
		HelpRef:      "course-checklist#course-dates",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.2", "OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return !strings.EqualFold(snap.ScheduleMode, "relative")
		},
		Evaluate: evalCourseDates,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.dates",
		},
	}
}

func evalCourseDates(_ context.Context, snap CourseSnapshot) (Finding, error) {
	hasStart := snap.StartsAt != nil
	hasEnd := snap.EndsAt != nil
	if hasStart && hasEnd {
		if snap.EndsAt.Before(*snap.StartsAt) {
			return Finding{
				Status:        StatusTodo,
				DetailKey:     "coursechecklist.item.course.dates.detail.inverted",
				DetailDefault: "End date is before the start date.",
			}, nil
		}
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.dates.detail.done",
			DetailDefault: "Start and end dates are set.",
		}, nil
	}
	if hasStart || hasEnd {
		done := boolToInt(hasStart) + boolToInt(hasEnd)
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.course.dates.detail.partial",
			DetailDefault: "Set both a start date and an end date.",
			Progress:      &Progress{Done: done, Total: 2},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.dates.detail.todo",
		DetailDefault: "Add a start date and an end date for this course.",
	}, nil
}

func ruleCourseTimezone() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseTimezone,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.timezone.title",
		TitleDefault: "Choose a course timezone",
		WhyKey:       "coursechecklist.item.course.timezone.why",
		WhyDefault:   "Due dates are interpreted in the course timezone, so learners need a consistent zone.",
		HelpRef:      "course-checklist#course-timezone",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseTimezone,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.timezone",
		},
	}
}

func evalCourseTimezone(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if zone, ok := validIANATimezone(snap.CourseTimezone); ok {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.timezone.detail.done",
			DetailDefault: fmt.Sprintf("Due dates use %s.", zone),
			DetailFields:  map[string]any{"timezone": zone},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.timezone.detail.todo",
		DetailDefault: "Set a valid IANA timezone for due dates.",
	}, nil
}

func ruleCoursePublished() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCoursePublished,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.published.title",
		TitleDefault: "Publish the course",
		WhyKey:       "coursechecklist.item.course.published.why",
		WhyDefault:   "Unpublished courses stay hidden from enrolled learners even when the term has started.",
		HelpRef:      "course-checklist#course-published",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCoursePublished,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.published",
		},
	}
}

func evalCoursePublished(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.Published {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.published.detail.done",
			DetailDefault: "Course is published.",
		}, nil
	}
	if snap.StartsAt != nil {
		now := time.Now().UTC()
		start := snap.StartsAt.UTC()
		days := daysUntil(now, start)
		if !start.After(now.Add(7 * 24 * time.Hour)) {
			var detail string
			switch {
			case days < 0:
				detail = fmt.Sprintf("Course is unpublished and the start date was %d days ago.", -days)
			case days == 0:
				detail = "Course is unpublished and starts today."
			default:
				detail = fmt.Sprintf("Course is unpublished and starts in %d days.", days)
			}
			return Finding{
				Status:        StatusInProgress,
				DetailKey:     "coursechecklist.item.course.published.detail.urgent",
				DetailDefault: detail,
				DetailFields:  map[string]any{"daysRemaining": days},
			}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.published.detail.todo",
		DetailDefault: "Publish the course when learners need access.",
	}, nil
}

func ruleCourseRelativeSchedule() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseRelativeSchedule,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.relative-schedule.title",
		TitleDefault: "Set relative due dates for graded work",
		WhyKey:       "coursechecklist.item.course.relative-schedule.why",
		WhyDefault:   "Self-paced courses still need due offsets so learners can pace graded work.",
		HelpRef:      "course-checklist#course-relative-schedule",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure},
		Applies: func(snap CourseSnapshot) bool {
			return strings.EqualFold(snap.ScheduleMode, "relative")
		},
		Evaluate: evalCourseRelativeSchedule,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Kind"}},
	}
}

func evalCourseRelativeSchedule(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var missing []EvidenceRow
	total := 0
	for _, it := range snap.StructureItems {
		if it.Archived || !isGradableKind(it.Kind) {
			continue
		}
		total++
		if it.DueAt == nil {
			missing = append(missing, EvidenceRow{
				Label:    it.Title,
				Sublabel: it.Kind,
				Status:   StatusTodo,
			})
		}
	}
	if total == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.relative-schedule.detail.empty",
			DetailDefault: "No graded items yet.",
		}, nil
	}
	if len(missing) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.relative-schedule.detail.done",
			DetailDefault: "Every graded item has a relative due date.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.relative-schedule.detail.todo",
		DetailDefault: fmt.Sprintf("%d of %d graded items are missing a due offset.", len(missing), total),
		DetailFields:  map[string]any{"missing": len(missing), "total": total},
		Progress:      &Progress{Done: total - len(missing), Total: total},
		Evidence:      missing,
	}, nil
}

func ruleCourseVisibilityWindow() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseVisibilityWindow,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.visibility-window.title",
		TitleDefault: "Check the course visibility window",
		WhyKey:       "coursechecklist.item.course.visibility-window.why",
		WhyDefault:   "A visibility window that closes during the term can hide the course from learners mid-way.",
		HelpRef:      "course-checklist#course-visibility-window",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseVisibilityWindow,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.visibility",
		},
	}
}

func evalCourseVisibilityWindow(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.VisibleFrom == nil && snap.HiddenAt == nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.visibility-window.detail.open",
			DetailDefault: "No visibility window is restricting access.",
		}, nil
	}
	if snap.StartsAt == nil || snap.EndsAt == nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.visibility-window.detail.no-term",
			DetailDefault: "Visibility window is set; term dates are not required for this check.",
		}, nil
	}
	// Window must contain [starts_at, ends_at].
	startsHidden := snap.VisibleFrom != nil && snap.VisibleFrom.After(*snap.StartsAt)
	endsHidden := snap.HiddenAt != nil && snap.HiddenAt.Before(*snap.EndsAt)
	inverted := snap.VisibleFrom != nil && snap.HiddenAt != nil && !snap.HiddenAt.After(*snap.VisibleFrom)
	if startsHidden || endsHidden || inverted {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.course.visibility-window.detail.todo",
			DetailDefault: "Visibility window hides the course during the term.",
		}, nil
	}
	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.course.visibility-window.detail.done",
		DetailDefault: "Visibility window covers the full term.",
	}, nil
}

func ruleCourseGradingScheme() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseGradingScheme,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.grading-scheme.title",
		TitleDefault: "Choose a grading scheme",
		WhyKey:       "coursechecklist.item.course.grading-scheme.why",
		WhyDefault:   "Learners need to know how letter grades or scores will be calculated.",
		HelpRef:      "course-checklist#course-grading-scheme",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.2", "OSCQR 44"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedGrading},
		Evaluate:     evalCourseGradingScheme,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/grading",
		},
	}
}

func evalCourseGradingScheme(_ context.Context, snap CourseSnapshot) (Finding, error) {
	// DONE when a scheme is linked OR grading_scale is not the untouched platform default.
	if snap.GradingSchemeID != nil || !strings.EqualFold(snap.GradingScale, "letter_plus_minus") {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.grading-scheme.detail.done",
			DetailDefault: "Grading scheme is configured.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.grading-scheme.detail.todo",
		DetailDefault: "Select a grading scheme or change the grading scale from the default.",
	}, nil
}

func ruleCourseHeroImage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseHeroImage,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.hero-image.title",
		TitleDefault: "Add a course hero image",
		WhyKey:       "coursechecklist.item.course.hero-image.why",
		WhyDefault:   "A hero image helps learners recognize the course in catalogs and on the home page.",
		HelpRef:      "course-checklist#course-hero-image",
		Tier:         TierRecommended,
		Sources:      []string{"QM 8"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseHeroImage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.hero",
		},
	}
}

func evalCourseHeroImage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.HeroImageURL != nil && strings.TrimSpace(*snap.HeroImageURL) != "" {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.hero-image.detail.done",
			DetailDefault: "Hero image is set. Alt text is checked separately under accessibility.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.hero-image.detail.todo",
		DetailDefault: "Add a hero image so the course is recognizable in catalogs.",
	}, nil
}

func ruleCourseHomeLanding() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseHomeLanding,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.home-landing.title",
		TitleDefault: "Choose what students see first",
		WhyKey:       "coursechecklist.item.course.home-landing.why",
		WhyDefault:   "The home landing decides the first screen learners open — modules, feed, or a page.",
		HelpRef:      "course-checklist#course-home-landing",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 16"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseHomeLanding,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.home-landing",
		},
	}
}

func evalCourseHomeLanding(_ context.Context, snap CourseSnapshot) (Finding, error) {
	landing := strings.TrimSpace(strings.ToLower(snap.CourseHomeLanding))
	if landing != "" && landing != "data" {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.home-landing.detail.done",
			DetailDefault: "Home landing has been chosen.",
		}, nil
	}
	if snap.CourseHomeContentItemID != nil && strings.TrimSpace(*snap.CourseHomeContentItemID) != "" {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.home-landing.detail.content",
			DetailDefault: "Home landing points to a content page.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.home-landing.detail.todo",
		DetailDefault: "Choose what students see first when they open the course.",
	}, nil
}

func ruleCourseFeaturesReviewed() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseFeaturesReviewed,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.features-reviewed.title",
		TitleDefault: "Review course feature switches",
		WhyKey:       "coursechecklist.item.course.features-reviewed.why",
		WhyDefault:   "Feature switches turn tools on or off for learners; a deliberate review prevents silent gaps.",
		HelpRef:      "course-checklist#course-features-reviewed",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalCourseFeaturesReviewed,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/features",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Feature", "State"}},
	}
}

func evalCourseFeaturesReviewed(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.FeaturesReviewedAt != nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.features-reviewed.detail.done",
			DetailDefault: "Course features have been saved at least once.",
		}, nil
	}
	var evidence []EvidenceRow
	type feat struct {
		name string
		on   bool
	}
	for _, f := range []feat{
		{"Discussions", snap.Features.DiscussionsEnabled},
		{"Files", snap.Features.FilesEnabled},
		{"Calendar", snap.Features.CalendarEnabled},
	} {
		if !f.on {
			evidence = append(evidence, EvidenceRow{Label: f.name, Sublabel: "off", Status: StatusTodo})
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.course.features-reviewed.detail.todo",
		DetailDefault: "Open Features and save once so the checklist knows the switches were reviewed.",
		Evidence:      evidence,
	}, nil
}

func ruleCourseLanguage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemCourseLanguage,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.course.language.title",
		TitleDefault: "Set the course language",
		WhyKey:       "coursechecklist.item.course.language.why",
		WhyDefault:   "A declared language helps assistive tech and translation tools treat content correctly.",
		HelpRef:      "course-checklist#course-language",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 3.1.1", "OSCQR 34"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalCourseLanguage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.language",
		},
	}
}

func evalCourseLanguage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if strings.TrimSpace(snap.CatalogLanguage) == "" {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.course.language.detail.todo",
			DetailDefault: "Set the primary course language.",
		}, nil
	}
	text, _ := SyllabusPlainText(snap)
	if languageHeuristicMatch(snap.CatalogLanguage, text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.course.language.detail.done",
			DetailDefault: "Course language is set and matches syllabus text.",
		}, nil
	}
	return Finding{
		Status:        StatusInProgress,
		DetailKey:     "coursechecklist.item.course.language.detail.mismatch",
		DetailDefault: "Course language is set but syllabus text looks like a different language.",
	}, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
