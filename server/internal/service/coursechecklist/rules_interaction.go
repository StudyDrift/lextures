package coursechecklist

import (
	"context"
	"fmt"
	"strings"
)

func interactionRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleIntegrityHighStakesSettings(),
		ruleIntegrityAIPolicyAlignment(),
		ruleAccommodationsHonored(),
		ruleAccommodationsReviewed(),
		ruleInteractionDiscussionExists(),
		ruleInteractionInstructorPresencePlan(),
		ruleInteractionOfficeHours(),
		ruleInteractionGroupsConfigured(),
	}
}

func ruleIntegrityHighStakesSettings() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemIntegrityHighStakesSettings,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.integrity.high-stakes-settings.title",
		TitleDefault: "Review integrity settings on major assessments",
		WhyKey:       "coursechecklist.item.integrity.high-stakes-settings.why",
		WhyDefault:   "Items worth 20% or more deserve an intentional integrity review.",
		HelpRef:      "course-checklist#integrity-high-stakes-settings",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.x", "OSCQR 45"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems, DataNeedCourse},
		Evaluate:     evalIntegrityHighStakesSettings,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalIntegrityHighStakesSettings(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := assessmentItemsFor(snap)
	total := totalCoursePoints(items)
	if total <= 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.integrity.high-stakes-settings.detail.na",
			DetailDefault: "Does not apply when no item crosses the 20% threshold.",
		}, nil
	}
	hasMajor := false
	for _, it := range items {
		if it.Points == nil || *it.Points <= 0 {
			continue
		}
		if 100.0*float64(*it.Points)/float64(total) >= 20.0 {
			hasMajor = true
			break
		}
	}
	if !hasMajor {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.integrity.high-stakes-settings.detail.na",
			DetailDefault: "Does not apply when no item crosses the 20% threshold.",
		}, nil
	}
	if snap.IntegritySettingsReviewedAt != nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.integrity.high-stakes-settings.detail.done",
			DetailDefault: "Integrity settings have been reviewed.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.integrity.high-stakes-settings.detail.todo",
		DetailDefault: "Review integrity settings on assessments worth 20% or more.",
	}, nil
}

func ruleIntegrityAIPolicyAlignment() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemIntegrityAIPolicyAlignment,
		Category:     CategoryAssessment,
		TitleKey:     "coursechecklist.item.integrity.ai-policy-alignment.title",
		TitleDefault: "Say how AI may be used",
		WhyKey:       "coursechecklist.item.integrity.ai-policy-alignment.why",
		WhyDefault:   "When AI features are on for learners, the syllabus should say how AI may be used.",
		HelpRef:      "course-checklist#integrity-ai-policy-alignment",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.3"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Applies: func(snap CourseSnapshot) bool {
			return snap.Features.AiTutorEnabled || snap.Features.AdaptiveContentEnabled ||
				snap.Features.ModulesAiAssistantEnabled || snap.Features.ContentToolsEnabled
		},
		Evaluate: evalIntegrityAIPolicyAlignment,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalIntegrityAIPolicyAlignment(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemIntegrityAIPolicyAlignment), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx != nil && lx.AcademicIntegrity != nil && lx.AcademicIntegrity.Match(text) &&
		containsAny(strings.ToLower(text), []string{"ai", "artificial intelligence", "generative", "chatgpt", "llm"}) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.integrity.ai-policy-alignment.detail.done",
			DetailDefault: truncatedDetail("Syllabus covers AI use alongside academic integrity.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.integrity.ai-policy-alignment.detail.todo",
		DetailDefault: truncatedDetail("Add how learners may use AI to the academic-integrity section.", trunc),
	}, nil
}

func ruleAccommodationsHonored() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAccommodationsHonored,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.accommodations.honored.title",
		TitleDefault: "Check timed work against accommodations",
		WhyKey:       "coursechecklist.item.accommodations.honored.why",
		WhyDefault:   "Timed quizzes may conflict with extended-time accommodations the platform cannot auto-apply.",
		HelpRef:      "course-checklist#accommodations-honored",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 48", "ADA/§504"},
		DataNeeds:    []DataNeed{DataNeedAccommodations, DataNeedAssessmentItems},
		Applies: func(snap CourseSnapshot) bool {
			return snap.AccommodationCount >= 1
		},
		Evaluate: evalAccommodationsHonored,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/accessibility",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type", "Accommodation", "Issue"}},
	}
}

func evalAccommodationsHonored(_ context.Context, snap CourseSnapshot) (Finding, error) {
	extended := 0
	for _, t := range snap.AccommodationTypeCounts {
		if t.Type == "extended_time" {
			extended = t.Count
		}
	}
	if extended == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.accommodations.honored.detail.done-no-extended",
			DetailDefault: "No extended-time accommodations conflict with timed quizzes.",
		}, nil
	}
	var timedQuizzes int
	var evidence []EvidenceRow
	for _, it := range sortAssessmentItems(assessmentItemsFor(snap)) {
		if it.Kind != "quiz" || it.TimeLimitMinutes == nil || *it.TimeLimitMinutes <= 0 {
			continue
		}
		timedQuizzes++
		// Privacy: counts and accommodation type only — never user IDs or names (FR-21 / AC-10).
		evidence = append(evidence, EvidenceRow{
			Label:    it.Title,
			Sublabel: fmt.Sprintf("Quiz · extended_time · hard time limit (%d min)", *it.TimeLimitMinutes),
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/settings/accessibility",
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.accommodations.honored.detail.done",
			DetailDefault: "No timed quizzes conflict with extended-time accommodations.",
		}, nil
	}
	noun := "timed quiz"
	if len(evidence) != 1 {
		noun = "timed quizzes"
	}
	return Finding{
		Status:    StatusTodo,
		DetailKey: "coursechecklist.item.accommodations.honored.detail.todo",
		DetailDefault: fmt.Sprintf(
			"%d %s may conflict with an extended-time accommodation",
			len(evidence), noun,
		),
		DetailFields: map[string]any{
			"timedQuizCount":    len(evidence),
			"extendedTimeCount": extended,
			"accommodationType": "extended_time",
		},
		Evidence: evidence,
	}, nil
}

func ruleAccommodationsReviewed() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemAccommodationsReviewed,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.accommodations.reviewed.title",
		TitleDefault: "Review new accommodations",
		WhyKey:       "coursechecklist.item.accommodations.reviewed.why",
		WhyDefault:   "Open the accommodations surface after new accommodations are added.",
		HelpRef:      "course-checklist#accommodations-reviewed",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 48"},
		DataNeeds:    []DataNeed{DataNeedAccommodations, DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return snap.AccommodationCount >= 1
		},
		Evaluate: evalAccommodationsReviewed,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/accessibility",
		},
	}
}

func evalAccommodationsReviewed(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.AccommodationsReviewedAt != nil {
		if snap.LatestAccommodationAt == nil || !snap.LatestAccommodationAt.After(*snap.AccommodationsReviewedAt) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.accommodations.reviewed.detail.done",
				DetailDefault: "Accommodations have been reviewed since the latest addition.",
			}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.accommodations.reviewed.detail.todo",
		DetailDefault: "Review accommodations for this course.",
	}, nil
}

func ruleInteractionDiscussionExists() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemInteractionDiscussionExists,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.interaction.discussion-exists.title",
		TitleDefault: "Add a discussion or collaborative activity",
		WhyKey:       "coursechecklist.item.interaction.discussion-exists.why",
		WhyDefault:   "Courses with multiple learners benefit from a structured discussion prompt.",
		HelpRef:      "course-checklist#interaction-discussion-exists",
		Tier:         TierRecommended,
		Sources:      []string{"QM 5.2", "OSCQR 39", "OSCQR 42", "NSQ C"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedEnrollments, DataNeedDiscussions},
		Applies: func(snap CourseSnapshot) bool {
			students := snap.EnrollmentCounts["student"]
			if students == 0 {
				for _, p := range snap.People {
					if p.Active && isStudentRole(p.Role) {
						students++
					}
				}
			}
			if students < 2 {
				return false
			}
			return snap.Features.DiscussionsEnabled || snap.Features.FeedEnabled ||
				snap.Features.GroupSpacesEnabled || snap.Features.VisualBoardsEnabled
		},
		Evaluate: evalInteractionDiscussionExists,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/discussions",
		},
	}
}

func evalInteractionDiscussionExists(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.DiscussionForumCount >= 1 || snap.DiscussionPromptCount >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.interaction.discussion-exists.detail.done",
			DetailDefault: "A discussion or collaborative activity is set up.",
			DetailFields: map[string]any{
				"forums":  snap.DiscussionForumCount,
				"threads": snap.DiscussionPromptCount,
			},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.interaction.discussion-exists.detail.todo",
		DetailDefault: "Add a structured discussion prompt, forum, or graded discussion.",
	}, nil
}

func ruleInteractionInstructorPresencePlan() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemInteractionInstructorPresencePlan,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.interaction.instructor-presence-plan.title",
		TitleDefault: "Plan your announcements",
		WhyKey:       "coursechecklist.item.interaction.instructor-presence-plan.why",
		WhyDefault:   "A steady announcement cadence signals instructor presence.",
		HelpRef:      "course-checklist#interaction-instructor-presence-plan",
		Tier:         TierRecommended,
		Sources:      []string{"QM 5.3", "OSCQR 40"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedAnnouncementCadence, DataNeedFeed},
		Evaluate:     evalInteractionInstructorPresencePlan,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/feed",
		},
	}
}

func evalInteractionInstructorPresencePlan(_ context.Context, snap CourseSnapshot) (Finding, error) {
	weeks := courseDurationWeeks(snap)
	if weeks <= 0 {
		weeks = 1
	}
	needed := (weeks + 1) / 2 // ≥ 1 per 2 weeks
	if needed < 1 {
		needed = 1
	}
	have := len(snap.AnnouncementTimes)
	if have >= needed {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.interaction.instructor-presence-plan.detail.done",
			DetailDefault: fmt.Sprintf("%d announcements for a %d-week course (need %d).", have, weeks, needed),
			DetailFields:  map[string]any{"have": have, "needed": needed, "weeks": weeks},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.interaction.instructor-presence-plan.detail.todo",
		DetailDefault: fmt.Sprintf("%d of %d planned announcements for a %d-week course.", have, needed, weeks),
		DetailFields:  map[string]any{"have": have, "needed": needed, "weeks": weeks},
		Progress:      &Progress{Done: have, Total: needed},
	}, nil
}

func courseDurationWeeks(snap CourseSnapshot) int {
	if snap.StartsAt == nil || snap.EndsAt == nil {
		return 0
	}
	d := snap.EndsAt.Sub(*snap.StartsAt)
	if d <= 0 {
		return 0
	}
	weeks := int(d.Hours() / (24 * 7))
	if weeks < 1 {
		weeks = 1
	}
	return weeks
}

func ruleInteractionOfficeHours() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemInteractionOfficeHours,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.interaction.office-hours.title",
		TitleDefault: "Publish office hours",
		WhyKey:       "coursechecklist.item.interaction.office-hours.why",
		WhyDefault:   "When office hours are enabled, publish at least one upcoming slot.",
		HelpRef:      "course-checklist#interaction-office-hours",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.7", "OSCQR 40"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedOfficeHours},
		Applies: func(snap CourseSnapshot) bool {
			return snap.Features.OfficeHoursEnabled
		},
		Evaluate: evalInteractionOfficeHours,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/office-hours",
		},
	}
}

func evalInteractionOfficeHours(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.FutureOfficeHourSlots >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.interaction.office-hours.detail.done",
			DetailDefault: fmt.Sprintf("%d upcoming office-hour slots.", snap.FutureOfficeHourSlots),
			DetailFields:  map[string]any{"slots": snap.FutureOfficeHourSlots},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.interaction.office-hours.detail.todo",
		DetailDefault: "Publish at least one upcoming office-hour slot.",
	}, nil
}

func ruleInteractionGroupsConfigured() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemInteractionGroupsConfigured,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.interaction.groups-configured.title",
		TitleDefault: "Put every student in a group",
		WhyKey:       "coursechecklist.item.interaction.groups-configured.why",
		WhyDefault:   "When groups are enabled, every student should belong to a group.",
		HelpRef:      "course-checklist#interaction-groups-configured",
		Tier:         TierRecommended,
		Sources:      []string{"QM 5.2"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedEnrollments, DataNeedEnrollmentGroups},
		Applies: func(snap CourseSnapshot) bool {
			students := studentCount(snap)
			if students < 2 {
				return false
			}
			return snap.EnrollmentGroupsEnabled || snap.Features.GroupSpacesEnabled || snap.EnrollmentGroupSetCount > 0
		},
		Evaluate: evalInteractionGroupsConfigured,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/groups",
		},
	}
}

func evalInteractionGroupsConfigured(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.EnrollmentGroupSetCount < 1 {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.interaction.groups-configured.detail.todo-sets",
			DetailDefault: "Create at least one group set.",
		}, nil
	}
	if snap.UnassignedStudentCount > 0 {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.interaction.groups-configured.detail.todo",
			// Count only — never names (FR-26).
			DetailDefault: fmt.Sprintf("%d students are not in a group.", snap.UnassignedStudentCount),
			DetailFields:  map[string]any{"unassignedCount": snap.UnassignedStudentCount},
		}, nil
	}
	return Finding{
		Status:        StatusDone,
		DetailKey:     "coursechecklist.item.interaction.groups-configured.detail.done",
		DetailDefault: "Every student belongs to a group.",
	}, nil
}

func studentCount(snap CourseSnapshot) int {
	if n, ok := snap.EnrollmentCounts["student"]; ok && n > 0 {
		return n
	}
	n := 0
	for _, p := range snap.People {
		if p.Active && isStudentRole(p.Role) {
			n++
		}
	}
	return n
}
