package coursechecklist

import (
	"context"
	"fmt"
	"strings"
)

func orientationRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleOrientationWelcomeMessage(),
		ruleOrientationStartHere(),
		ruleOrientationInstructorContact(),
		ruleOrientationResponseTime(),
		ruleOrientationParticipationExpectations(),
		ruleOrientationNetiquette(),
		ruleOrientationTechRequirements(),
		ruleOrientationSupportResources(),
		ruleOrientationInstructorIntroduction(),
		ruleOrientationLearnerIntroductions(),
	}
}

func ruleOrientationWelcomeMessage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationWelcomeMessage,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.welcome-message.title",
		TitleDefault: "Post a welcome announcement",
		WhyKey:       "coursechecklist.item.orientation.welcome-message.why",
		WhyDefault:   "Students who read a welcome post in week 1 are less likely to disengage.",
		HelpRef:      "course-checklist#orientation-welcome-message",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.1", "OSCQR 1", "NSQ A"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedFeed},
		Applies: func(snap CourseSnapshot) bool {
			return snap.FeedEnabled || snap.Features.FeedEnabled
		},
		Evaluate: evalOrientationWelcomeMessage,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/feed",
			Anchor:  "feed.channel.announcements",
		},
		Action: &ItemAction{
			Kind:       ActionKindDraftWelcome,
			LabelKey:   "coursechecklist.action.draft_welcome",
			Label:      "Draft a welcome announcement",
			Endpoint:   "/api/v1/courses/{courseCode}/feed/draft-welcome",
			RequiresAI: true,
		},
	}
}

func evalOrientationWelcomeMessage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	w := snap.AnnouncementsWelcome
	if w != nil && w.AuthorIsStaff && w.BodyLen >= 200 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.welcome-message.detail.done",
			DetailDefault: "Welcome announcement posted.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.welcome-message.detail.todo",
		DetailDefault: "Students see an empty announcements channel on day one.",
	}, nil
}

func ruleOrientationStartHere() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationStartHere,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.start-here.title",
		TitleDefault: "Add a Start Here page",
		WhyKey:       "coursechecklist.item.orientation.start-here.why",
		WhyDefault:   "A Start Here page orients learners before week-1 content.",
		HelpRef:      "course-checklist#orientation-start-here",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.1", "OSCQR 1"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure},
		Evaluate:     evalOrientationStartHere,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Kind"}},
	}
}

func evalOrientationStartHere(_ context.Context, snap CourseSnapshot) (Finding, error) {
	lx := lexiconForSnap(snap)
	landing := strings.ToLower(strings.TrimSpace(snap.CourseHomeLanding))
	if landing == "content" || (snap.CourseHomeContentItemID != nil && strings.TrimSpace(*snap.CourseHomeContentItemID) != "") {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.start-here.detail.landing",
			DetailDefault: "Course home lands on a content page.",
		}, nil
	}
	_, children, ok := firstModuleItems(snap)
	var evidence []EvidenceRow
	for _, c := range children {
		evidence = append(evidence, EvidenceRow{Label: c.Title, Sublabel: c.Kind, Status: StatusTodo})
		if c.Kind == "content_page" && TitleMatches(lx.StartHereTitles, c.Title) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.start-here.detail.done",
				DetailDefault: "Module 1 includes a Start Here page.",
				Evidence:      evidence,
			}, nil
		}
	}
	if !ok {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.orientation.start-here.detail.todo",
			DetailDefault: "Add a module with a Start Here content page.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.start-here.detail.todo",
		DetailDefault: "Module 1 does not yet include a Start Here page.",
		Evidence:      evidence,
	}, nil
}

func ruleOrientationInstructorContact() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationInstructorContact,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.instructor-contact.title",
		TitleDefault: "Publish instructor contact info",
		WhyKey:       "coursechecklist.item.orientation.instructor-contact.why",
		WhyDefault:   "Learners need a clear way to reach course staff when they get stuck.",
		HelpRef:      "course-checklist#orientation-instructor-contact",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.7", "QM 1.8", "OSCQR 10"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalOrientationInstructorContact,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOrientationInstructorContact(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationInstructorContact), nil
	}
	lx := lexiconForSnap(snap)
	for _, s := range snap.SyllabusSections {
		body := s.Markdown
		headingMatch := TitleMatches(lx.Contact, s.Title)
		bodyMatch := lx.Contact.Match(body)
		if (headingMatch || bodyMatch) && len([]rune(strings.TrimSpace(body))) >= 80 {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.instructor-contact.detail.done",
				DetailDefault: "Contact information is published in the syllabus.",
			}, nil
		}
	}
	text, trunc := SyllabusPlainText(snap)
	if lx.Contact.Match(text) && len([]rune(strings.TrimSpace(text))) >= 80 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.instructor-contact.detail.done",
			DetailDefault: truncatedDetail("Contact information is published in the syllabus.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.instructor-contact.detail.todo",
		DetailDefault: truncatedDetail("Add a syllabus section with contact details (at least 80 characters).", trunc),
	}, nil
}

func ruleOrientationResponseTime() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationResponseTime,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.response-time.title",
		TitleDefault: "State response-time expectations",
		WhyKey:       "coursechecklist.item.orientation.response-time.why",
		WhyDefault:   "Publishing a turnaround commitment sets clear expectations for email and grading.",
		HelpRef:      "course-checklist#orientation-response-time",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.4", "OSCQR 38"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalOrientationResponseTime,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOrientationResponseTime(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationResponseTime), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.ResponseTime.Match(text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.response-time.detail.done",
			DetailDefault: truncatedDetail("Response-time commitment is published.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.response-time.detail.todo",
		DetailDefault: truncatedDetail("State how quickly learners can expect a reply or grade.", trunc),
	}, nil
}

func ruleOrientationParticipationExpectations() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationParticipationExpectations,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.participation-expectations.title",
		TitleDefault: "State participation expectations",
		WhyKey:       "coursechecklist.item.orientation.participation-expectations.why",
		WhyDefault:   "Clear participation frequency helps learners plan weekly effort.",
		HelpRef:      "course-checklist#orientation-participation",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.4", "OSCQR 43"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus, DataNeedStructure},
		Evaluate:     evalOrientationParticipationExpectations,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOrientationParticipationExpectations(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationParticipationExpectations), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.Participation.Match(text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.participation-expectations.detail.done",
			DetailDefault: truncatedDetail("Participation expectations are published.", trunc),
		}, nil
	}
	// Also scan Start Here page titles/bodies via structure titles (meta HasBody as proxy).
	_, children, _ := firstModuleItems(snap)
	for _, c := range children {
		if TitleMatches(lx.StartHereTitles, c.Title) || TitleMatches(lx.Participation, c.Title) {
			if meta, ok := snap.ItemMeta[c.ID]; ok && meta.HasBody {
				return Finding{
					Status:        StatusDone,
					DetailKey:     "coursechecklist.item.orientation.participation-expectations.detail.start-here",
					DetailDefault: "Participation expectations appear on a Start Here page.",
				}, nil
			}
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.participation-expectations.detail.todo",
		DetailDefault: truncatedDetail("Describe how often learners should post or participate.", trunc),
	}, nil
}

func ruleOrientationNetiquette() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationNetiquette,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.netiquette.title",
		TitleDefault: "Publish a community agreement",
		WhyKey:       "coursechecklist.item.orientation.netiquette.why",
		WhyDefault:   "A netiquette or code-of-conduct statement sets expectations for respectful interaction.",
		HelpRef:      "course-checklist#orientation-netiquette",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.3", "OSCQR 43"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Applies: func(snap CourseSnapshot) bool {
			f := snap.Features
			return f.DiscussionsEnabled || f.FeedEnabled || f.GroupSpacesEnabled || f.VisualBoardsEnabled ||
				snap.FeedEnabled
		},
		Evaluate: evalOrientationNetiquette,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOrientationNetiquette(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationNetiquette), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.Netiquette.Match(text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.netiquette.detail.done",
			DetailDefault: truncatedDetail("Community agreement is published.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.netiquette.detail.todo",
		DetailDefault: truncatedDetail("Add a netiquette or code-of-conduct section to the syllabus.", trunc),
	}, nil
}

func ruleOrientationTechRequirements() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationTechRequirements,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.tech-requirements.title",
		TitleDefault: "List technical requirements",
		WhyKey:       "coursechecklist.item.orientation.tech-requirements.why",
		WhyDefault:   "Learners need to know required browsers, software, and technical skills up front.",
		HelpRef:      "course-checklist#orientation-tech-requirements",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.5", "QM 1.6", "OSCQR 11"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalOrientationTechRequirements,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
	}
}

func evalOrientationTechRequirements(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationTechRequirements), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	if lx.TechRequirements.Match(text) {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.tech-requirements.detail.done",
			DetailDefault: truncatedDetail("Technical requirements are published.", trunc),
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.tech-requirements.detail.todo",
		DetailDefault: truncatedDetail("List required technology and technical skills in the syllabus.", trunc),
	}, nil
}

func ruleOrientationSupportResources() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationSupportResources,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.support-resources.title",
		TitleDefault: "Link to support resources",
		WhyKey:       "coursechecklist.item.orientation.support-resources.why",
		WhyDefault:   "Help desk, tutoring, library, and accessibility links give learners places to get help.",
		HelpRef:      "course-checklist#orientation-support-resources",
		Tier:         TierRecommended,
		Sources:      []string{"QM 7.1", "OSCQR 6"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSyllabus},
		Evaluate:     evalOrientationSupportResources,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/syllabus",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Resource"}},
	}
}

func evalOrientationSupportResources(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.SyllabusMalformed {
		return syllabusUnknownFinding(ItemOrientationSupportResources), nil
	}
	lx := lexiconForSnap(snap)
	text, trunc := SyllabusPlainText(snap)
	links := countSupportLinks(text, lx.SupportLinkHints)
	if len(links) >= 2 {
		ev := make([]EvidenceRow, 0, len(links))
		for _, l := range links {
			ev = append(ev, EvidenceRow{Label: l, Status: StatusDone})
		}
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.orientation.support-resources.detail.done",
			DetailDefault: truncatedDetail(fmt.Sprintf("%d support resources detected.", len(links)), trunc),
			Evidence:      ev,
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.support-resources.detail.todo",
		DetailDefault: truncatedDetail("Add at least two support links (help desk, tutoring, library, accessibility).", trunc),
		Evidence: func() []EvidenceRow {
			ev := make([]EvidenceRow, 0, len(links))
			for _, l := range links {
				ev = append(ev, EvidenceRow{Label: l, Status: StatusInProgress})
			}
			return ev
		}(),
	}, nil
}

func ruleOrientationInstructorIntroduction() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationInstructorIntroduction,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.instructor-introduction.title",
		TitleDefault: "Introduce yourself to learners",
		WhyKey:       "coursechecklist.item.orientation.instructor-introduction.why",
		WhyDefault:   "A short instructor introduction builds presence and trust early in the term.",
		HelpRef:      "course-checklist#orientation-instructor-introduction",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.8", "OSCQR 40"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedFeed},
		Evaluate:     evalOrientationInstructorIntroduction,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalOrientationInstructorIntroduction(_ context.Context, snap CourseSnapshot) (Finding, error) {
	lx := lexiconForSnap(snap)
	for _, it := range snap.StructureItems {
		if it.Archived || it.Kind != "content_page" {
			continue
		}
		if TitleMatches(lx.InstructorIntroTitles, it.Title) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.instructor-introduction.detail.done",
				DetailDefault: "Instructor introduction page is present.",
			}, nil
		}
	}
	for _, ch := range snap.FeedChannels {
		if TitleMatches(lx.InstructorIntroTitles, ch.LatestTitle) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.instructor-introduction.detail.feed",
				DetailDefault: "Instructor introduction appears in the feed.",
			}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.instructor-introduction.detail.todo",
		DetailDefault: "Add a short About Your Instructor page or announcement.",
	}, nil
}

func ruleOrientationLearnerIntroductions() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemOrientationLearnerIntroductions,
		Category:     CategoryOrientation,
		TitleKey:     "coursechecklist.item.orientation.learner-introductions.title",
		TitleDefault: "Create an introductions prompt",
		WhyKey:       "coursechecklist.item.orientation.learner-introductions.why",
		WhyDefault:   "An introductions discussion helps learners connect with peers early.",
		HelpRef:      "course-checklist#orientation-learner-introductions",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.9", "OSCQR 41"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedEnrollments, DataNeedFeed},
		Applies: func(snap CourseSnapshot) bool {
			if !snap.Features.DiscussionsEnabled && !snap.FeedEnabled && !snap.Features.FeedEnabled {
				return false
			}
			students := 0
			for _, p := range snap.People {
				if isStudentRole(p.Role) && p.Active && !p.InvitationPending {
					students++
				}
			}
			if n, ok := snap.EnrollmentCounts["student"]; ok && n > students {
				students = n
			}
			// Not enough peers yet — hide until roster grows (or unknown roster → still show).
			if students > 0 && students < 2 {
				return false
			}
			return true
		},
		Evaluate: evalOrientationLearnerIntroductions,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalOrientationLearnerIntroductions(_ context.Context, snap CourseSnapshot) (Finding, error) {
	students := 0
	for _, p := range snap.People {
		if isStudentRole(p.Role) && (p.Active || p.InvitationPending) {
			students++
		}
	}
	if n, ok := snap.EnrollmentCounts["student"]; ok && n > students {
		students = n
	}
	if students > 0 && students < 2 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.orientation.learner-introductions.detail.na",
			DetailDefault: "Does not apply when fewer than two students are enrolled.",
		}, nil
	}
	lx := lexiconForSnap(snap)
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		if TitleMatches(lx.LearnerIntroTitles, it.Title) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.learner-introductions.detail.done",
				DetailDefault: "Introductions prompt is present.",
			}, nil
		}
	}
	for _, ch := range snap.FeedChannels {
		if TitleMatches(lx.LearnerIntroTitles, ch.Name) || TitleMatches(lx.LearnerIntroTitles, ch.LatestTitle) {
			return Finding{
				Status:        StatusDone,
				DetailKey:     "coursechecklist.item.orientation.learner-introductions.detail.feed",
				DetailDefault: "Introductions prompt appears in the feed.",
			}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.orientation.learner-introductions.detail.todo",
		DetailDefault: "Add an introductions discussion or feed prompt for learners.",
	}, nil
}
