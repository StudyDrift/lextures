package coursechecklist

import "fmt"

// familyItemOrder is the short setup list for Homeschool, K–12, parent-created,
// and managed-learner courses (#654). IDs stay on the institutional registry so
// dismiss, restore, and recheck keep working.
var familyItemOrder = []ItemID{
	ItemStructureModulesExist,
	ItemPeopleStudentsEnrolled,
	ItemAssessmentGradableItems,
	ItemLaunchStudentPreview,
	ItemStructurePacingSignal,
}

func familyChecklistAudience(snap CourseSnapshot) bool {
	return snap.HomeschoolMode || snap.OrgIsK12 || snap.CreatorIsParent || snap.HasManagedLearner
}

func isFamilyChecklistItem(id ItemID) bool {
	for _, want := range familyItemOrder {
		if id == want {
			return true
		}
	}
	return false
}

// projectFamilyFindings replaces an institutional evaluation with the family
// setup list. A full pass (more findings than the family list) returns exactly
// those five items. A single-item recheck rewrites a family item in place and
// marks any other requested item not applicable.
func projectFamilyFindings(snap CourseSnapshot, findings []ItemResult) []ItemResult {
	if len(findings) > len(familyItemOrder) {
		byID := make(map[ItemID]ItemResult, len(findings))
		for _, fr := range findings {
			byID[fr.ID] = fr
		}
		out := make([]ItemResult, 0, len(familyItemOrder))
		for _, id := range familyItemOrder {
			out = append(out, familyItemResult(snap, id, byID[id]))
		}
		return out
	}
	out := make([]ItemResult, 0, len(findings))
	for _, fr := range findings {
		if !isFamilyChecklistItem(fr.ID) {
			fr.Finding = Finding{
				Status:        StatusNotApplicable,
				DetailKey:     fmt.Sprintf("coursechecklist.item.%s.detail.na", fr.ID),
				DetailDefault: "Does not apply to this course.",
			}
			fr.Sources = []string{}
			out = append(out, fr)
			continue
		}
		out = append(out, familyItemResult(snap, fr.ID, fr))
	}
	return out
}

func familyItemResult(snap CourseSnapshot, id ItemID, base ItemResult) ItemResult {
	out := base
	if out.ID == "" {
		if it := MustDefault().Get(id); it != nil {
			out = ItemResult{
				ID:       it.ID,
				TitleKey: it.TitleKey,
				WhyKey:   it.WhyKey,
				HelpRef:  it.HelpRef,
				Target:   it.Target,
			}
		}
		out.ID = id
	}
	out.Category = CategoryFamily
	out.Tier = TierEssential
	out.Sources = []string{"Product"}
	out.Action = nil
	out.EvidenceShape = nil
	switch id {
	case ItemStructureModulesExist:
		out.TitleDefault = "Set up a learning plan"
		out.WhyDefault = "A short list of lessons gives your family a path through the course."
		out.Target = NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"}
		out.Finding = evalFamilyLearningPlan(snap)
	case ItemPeopleStudentsEnrolled:
		out.TitleDefault = "Enroll your kids"
		out.WhyDefault = "Add the learners who will take this course."
		out.Target = NavTarget{Surface: "web", Route: "/courses/{courseCode}/enrollments"}
		out.Finding = evalFamilyEnroll(snap)
	case ItemAssessmentGradableItems:
		out.TitleDefault = "Add a first lesson or quiz"
		out.WhyDefault = "One lesson or quiz is enough to start."
		out.Target = NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"}
		out.Finding = evalFamilyFirstActivity(snap)
	case ItemLaunchStudentPreview:
		out.TitleDefault = "Check progress"
		out.WhyDefault = "See how your kids are doing once they have something to work on."
		out.Target = NavTarget{Surface: "web", Route: "/courses/{courseCode}/reports"}
		out.Finding = evalFamilyCheckProgress(snap)
	case ItemStructurePacingSignal:
		out.TitleDefault = "Set a pacing rhythm"
		out.WhyDefault = "Dates or a due date keep the week from drifting."
		out.Target = NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/general",
			Anchor:  "course.general.dates",
		}
		out.Finding = evalFamilyPacing(snap)
	default:
		out.Finding = Finding{
			Status:        StatusNotApplicable,
			DetailDefault: "Does not apply to this course.",
		}
	}
	return out
}

func evalFamilyLearningPlan(snap CourseSnapshot) Finding {
	n := len(listModules(snap))
	if n >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailDefault: fmt.Sprintf("%d lesson group(s) are in the plan.", n),
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Add a module for the lessons you want to teach.",
	}
}

func evalFamilyEnroll(snap CourseSnapshot) Finding {
	active := activeLearnerCount(snap)
	if active >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailDefault: fmt.Sprintf("%d learner(s) enrolled.", active),
		}
	}
	pending := snap.PendingInvitations
	if len(pending) == 0 {
		for _, p := range snap.People {
			if isStudentRole(p.Role) && p.InvitationPending {
				pending = append(pending, PendingInviteSnap{DisplayName: p.DisplayName, UserID: p.UserID})
			}
		}
	}
	if len(pending) > 0 {
		ev := make([]EvidenceRow, 0, len(pending))
		for _, inv := range pending {
			ev = append(ev, EvidenceRow{
				Label:    inv.DisplayName,
				Sublabel: fmt.Sprintf("%d days", inv.DaysPending),
				Status:   StatusInProgress,
				TargetOverride: &NavTarget{
					Surface: "web",
					Route:   "/courses/{courseCode}/enrollments",
					Anchor:  "enrollments.invitations",
				},
			})
		}
		return Finding{
			Status:        StatusInProgress,
			DetailDefault: fmt.Sprintf("%d invitation(s) are still waiting.", len(pending)),
			Evidence:      ev,
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Enroll or invite at least one learner.",
	}
}

func evalFamilyFirstActivity(snap CourseSnapshot) Finding {
	n := familyActivityCount(snap)
	if n >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailDefault: fmt.Sprintf("%d lesson or quiz item(s) are ready.", n),
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Add a lesson page, assignment, or quiz.",
	}
}

func evalFamilyCheckProgress(snap CourseSnapshot) Finding {
	learners := activeLearnerCount(snap)
	activities := familyActivityCount(snap)
	switch {
	case learners >= 1 && activities >= 1:
		return Finding{
			Status:        StatusDone,
			DetailDefault: "Open progress to see how your learners are doing.",
		}
	case learners >= 1 || activities >= 1:
		return Finding{
			Status:        StatusInProgress,
			DetailDefault: "Add both a learner and a lesson, then open progress.",
			Progress:      &Progress{Done: 1, Total: 2},
		}
	default:
		return Finding{
			Status:        StatusTodo,
			DetailDefault: "Enroll a learner and add a lesson, then open progress.",
		}
	}
}

func evalFamilyPacing(snap CourseSnapshot) Finding {
	if familyPacingSet(snap) {
		return Finding{
			Status:        StatusDone,
			DetailDefault: "The course has dates or a due date.",
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Set a start and end date, or add a due date on a lesson.",
	}
}

func activeLearnerCount(snap CourseSnapshot) int {
	n := 0
	for _, p := range snap.People {
		if isStudentRole(p.Role) && p.Active && !p.InvitationPending {
			n++
		}
	}
	return n
}

func familyActivityCount(snap CourseSnapshot) int {
	n := 0
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		switch it.Kind {
		case "content_page", "quiz", "assignment":
			n++
		}
	}
	return n
}

func familyPacingSet(snap CourseSnapshot) bool {
	if snap.StartsAt != nil && snap.EndsAt != nil && !snap.EndsAt.Before(*snap.StartsAt) {
		return true
	}
	for _, it := range snap.StructureItems {
		if it.Archived || it.DueAt == nil {
			continue
		}
		return true
	}
	return false
}
