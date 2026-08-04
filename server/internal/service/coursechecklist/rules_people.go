package coursechecklist

import (
	"context"
	"fmt"
)

func peopleRules() []ItemDescriptor {
	return []ItemDescriptor{
		rulePeopleStudentsEnrolled(),
		rulePeopleStaffRoles(),
		rulePeopleSections(),
		rulePeopleStaleInvitations(),
		rulePeopleGuardianLinks(),
	}
}

func rulePeopleStudentsEnrolled() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemPeopleStudentsEnrolled,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.people.students-enrolled.title",
		TitleDefault: "Enroll at least one student",
		WhyKey:       "coursechecklist.item.people.students-enrolled.why",
		WhyDefault:   "A course without active learners is not ready for day one.",
		HelpRef:      "course-checklist#people-students-enrolled",
		Tier:         TierRecommended,
		Sources:      []string{"Product", "OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedEnrollments},
		Evaluate:     evalPeopleStudentsEnrolled,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/enrollments",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Invitee", "Days pending"}},
	}
}

func evalPeopleStudentsEnrolled(_ context.Context, snap CourseSnapshot) (Finding, error) {
	activeStudents := 0
	for _, p := range snap.People {
		if isStudentRole(p.Role) && p.Active && !p.InvitationPending {
			activeStudents++
		}
	}
	if activeStudents >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.people.students-enrolled.detail.done",
			DetailDefault: fmt.Sprintf("%d active student(s) enrolled.", activeStudents),
			DetailFields:  map[string]any{"count": activeStudents},
		}, nil
	}
	pending := snap.PendingInvitations
	if len(pending) == 0 {
		for _, p := range snap.People {
			if isStudentRole(p.Role) && p.InvitationPending {
				days := 0
				if p.EnrolledAt != nil {
					for _, inv := range snap.PendingInvitations {
						if inv.UserID == p.UserID {
							days = inv.DaysPending
							break
						}
					}
				}
				pending = append(pending, PendingInviteSnap{
					DisplayName: p.DisplayName,
					UserID:      p.UserID,
					DaysPending: days,
				})
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
					Anchor:  "resend",
				},
			})
		}
		return Finding{
			Status:        StatusInProgress,
			DetailKey:     "coursechecklist.item.people.students-enrolled.detail.pending",
			DetailDefault: fmt.Sprintf("%d pending invitation(s); no active students yet.", len(pending)),
			Evidence:      ev,
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.people.students-enrolled.detail.todo",
		DetailDefault: "Enroll or invite at least one student.",
	}, nil
}

func rulePeopleStaffRoles() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemPeopleStaffRoles,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.people.staff-roles.title",
		TitleDefault: "Add a co-teacher or TA",
		WhyKey:       "coursechecklist.item.people.staff-roles.why",
		WhyDefault:   "A second staff member provides coverage when the creator is unavailable.",
		HelpRef:      "course-checklist#people-staff-roles",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.8"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedEnrollments},
		Applies: func(snap CourseSnapshot) bool {
			return !snap.HomeschoolMode
		},
		Evaluate: evalPeopleStaffRoles,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/enrollments",
		},
	}
}

func evalPeopleStaffRoles(_ context.Context, snap CourseSnapshot) (Finding, error) {
	staff := 0
	beyondCreator := 0
	for _, p := range snap.People {
		if !isStaffRole(p.Role) || (!p.Active && !p.InvitationPending) {
			continue
		}
		staff++
		if snap.CreatorUserID != nil && p.UserID != *snap.CreatorUserID {
			beyondCreator++
		}
	}
	// With creator id: DONE when ≥1 staff beyond creator.
	// Without creator id: use staff count ≥ 2 as proxy for "beyond creator"; 1 staff = todo.
	done := false
	if snap.CreatorUserID != nil {
		done = beyondCreator >= 1
	} else {
		done = staff >= 2
	}
	if done {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.people.staff-roles.detail.done",
			DetailDefault: "Course has staff beyond the creator.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.people.staff-roles.detail.todo",
		DetailDefault: "Add at least one staff member beyond the course creator, or dismiss if not needed.",
	}, nil
}

func rulePeopleSections() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemPeopleSections,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.people.sections.title",
		TitleDefault: "Assign students to sections",
		WhyKey:       "coursechecklist.item.people.sections.why",
		WhyDefault:   "When sections are enabled, every student should belong to a section for roster accuracy.",
		HelpRef:      "course-checklist#people-sections",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedSections, DataNeedEnrollments},
		Applies: func(snap CourseSnapshot) bool {
			return snap.SectionsEnabled
		},
		Evaluate: evalPeopleSections,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/sections",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Student", "ID"}},
	}
}

func evalPeopleSections(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(snap.Sections) == 0 {
		return Finding{
			Status:        StatusTodo,
			DetailKey:     "coursechecklist.item.people.sections.detail.no-sections",
			DetailDefault: "Create at least one section, then assign students.",
		}, nil
	}
	var unsectioned []EvidenceRow
	for _, p := range snap.People {
		if !isStudentRole(p.Role) {
			continue
		}
		if p.SectionID == nil {
			unsectioned = append(unsectioned, EvidenceRow{
				Label:    p.DisplayName,
				Sublabel: p.UserID.String(),
				Status:   StatusTodo,
			})
		}
	}
	if len(unsectioned) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.people.sections.detail.done",
			DetailDefault: "Every student belongs to a section.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.people.sections.detail.todo",
		DetailDefault: fmt.Sprintf("%d student(s) are not assigned to a section.", len(unsectioned)),
		Evidence:      unsectioned,
	}, nil
}

func rulePeopleStaleInvitations() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemPeopleStaleInvitations,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.people.stale-invitations.title",
		TitleDefault: "Follow up on stale invitations",
		WhyKey:       "coursechecklist.item.people.stale-invitations.why",
		WhyDefault:   "Invitations older than two weeks usually need a resend or cleanup.",
		HelpRef:      "course-checklist#people-stale-invitations",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedEnrollments},
		Evaluate:     evalPeopleStaleInvitations,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/enrollments",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Invitee", "Days pending"}},
	}
}

func evalPeopleStaleInvitations(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var stale []EvidenceRow
	for _, inv := range snap.PendingInvitations {
		if inv.DaysPending > 14 {
			stale = append(stale, EvidenceRow{
				Label:    inv.DisplayName,
				Sublabel: fmt.Sprintf("%d days", inv.DaysPending),
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web",
					Route:   "/courses/{courseCode}/enrollments",
					Anchor:  "resend",
				},
			})
		}
	}
	if len(stale) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.people.stale-invitations.detail.done",
			DetailDefault: "No invitations older than 14 days.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.people.stale-invitations.detail.todo",
		DetailDefault: fmt.Sprintf("%d invitation(s) older than 14 days.", len(stale)),
		Evidence:      stale,
	}, nil
}

func rulePeopleGuardianLinks() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemPeopleGuardianLinks,
		Category:     CategoryFoundations,
		TitleKey:     "coursechecklist.item.people.guardian-links.title",
		TitleDefault: "Link a guardian for each student",
		WhyKey:       "coursechecklist.item.people.guardian-links.why",
		WhyDefault:   "In K-12 orgs with the parent portal, every student needs a linked guardian.",
		HelpRef:      "course-checklist#people-guardian-links",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedEnrollments},
		Applies: func(snap CourseSnapshot) bool {
			if snap.HomeschoolMode {
				return false
			}
			return snap.OrgIsK12 && snap.ParentPortalEnabled
		},
		Evaluate: evalPeopleGuardianLinks,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/enrollments",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Student", "ID"}},
	}
}

func evalPeopleGuardianLinks(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var missing []EvidenceRow
	for _, p := range snap.People {
		if !isStudentRole(p.Role) || !p.Active || p.InvitationPending {
			continue
		}
		if !p.HasGuardianLink {
			missing = append(missing, EvidenceRow{
				Label:    p.DisplayName,
				Sublabel: p.UserID.String(),
				Status:   StatusTodo,
			})
		}
	}
	if len(missing) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.people.guardian-links.detail.done",
			DetailDefault: "Every active student has a linked guardian.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.people.guardian-links.detail.todo",
		DetailDefault: fmt.Sprintf("%d student(s) have no linked guardian.", len(missing)),
		Evidence:      missing,
	}, nil
}
