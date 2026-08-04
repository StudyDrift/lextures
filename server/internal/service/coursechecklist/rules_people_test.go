package coursechecklist

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

func TestPeopleCopyContract(t *testing.T) {
	for _, it := range peopleRules() {
		if utf8.RuneCountInString(it.TitleDefault) > 60 {
			t.Errorf("%s title too long", it.ID)
		}
		low := strings.ToLower(it.TitleDefault + " " + it.WhyDefault)
		for _, ban := range []string{"failed", "should have", "!"} {
			if strings.Contains(low, ban) {
				t.Errorf("%s banned %q", it.ID, ban)
			}
		}
	}
}

func TestPeopleSectionsAC8(t *testing.T) {
	// AC-8
	rule := findRule(t, ItemPeopleSections)
	people := make([]PersonSnap, 0, 12)
	for i := 0; i < 12; i++ {
		people = append(people, PersonSnap{
			UserID: uuid.New(), DisplayName: "S", Role: "student", Active: true,
		})
	}
	f, err := rule.Evaluate(context.Background(), CourseSnapshot{
		SectionsEnabled: true,
		Sections:        []SectionSnap{{SectionCode: "A", Name: "A"}},
		People:          people,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusTodo || len(f.Evidence) != 12 {
		t.Fatalf("status=%s evidence=%d", f.Status, len(f.Evidence))
	}
	res := Evaluate(context.Background(), CourseSnapshot{SectionsEnabled: false}, EvaluateOptions{
		Only: []ItemID{ItemPeopleSections},
	})
	if res.Findings[0].Finding.Status != StatusNotApplicable {
		t.Fatalf("disabled=%s", res.Findings[0].Finding.Status)
	}
}

func TestPeopleStudentsEnrolledAC9(t *testing.T) {
	// AC-9
	now := time.Now().UTC()
	invites := make([]PendingInviteSnap, 0, 3)
	for i := 0; i < 3; i++ {
		invites = append(invites, PendingInviteSnap{
			DisplayName: "Invitee", UserID: uuid.New(), CreatedAt: now.Add(-48 * time.Hour), DaysPending: 2,
		})
	}
	f, err := findRule(t, ItemPeopleStudentsEnrolled).Evaluate(context.Background(), CourseSnapshot{
		PendingInvitations: invites,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != StatusInProgress || len(f.Evidence) != 3 {
		t.Fatalf("status=%s evidence=%d", f.Status, len(f.Evidence))
	}
	if f.Evidence[0].TargetOverride == nil {
		t.Fatal("expected resend target")
	}
}

func TestPeopleStaffAndGuardianAC12(t *testing.T) {
	// AC-12
	res := Evaluate(context.Background(), CourseSnapshot{HomeschoolMode: true}, EvaluateOptions{
		Only: []ItemID{ItemPeopleStaffRoles, ItemPeopleGuardianLinks},
	})
	if len(res.Findings) != 2 {
		t.Fatalf("findings=%d", len(res.Findings))
	}
	for _, fr := range res.Findings {
		if fr.Finding.Status != StatusNotApplicable {
			t.Fatalf("%s=%s", fr.ID, fr.Finding.Status)
		}
	}
}

func TestPeopleStaffBeyondCreator(t *testing.T) {
	creator := uuid.New()
	other := uuid.New()
	f, _ := findRule(t, ItemPeopleStaffRoles).Evaluate(context.Background(), CourseSnapshot{
		CreatorUserID: &creator,
		People: []PersonSnap{
			{UserID: creator, Role: "teacher", Active: true, DisplayName: "C"},
			{UserID: other, Role: "ta", Active: true, DisplayName: "T"},
		},
	})
	if f.Status != StatusDone {
		t.Fatalf("with beyond creator=%s", f.Status)
	}
	f, _ = findRule(t, ItemPeopleStaffRoles).Evaluate(context.Background(), CourseSnapshot{
		People: []PersonSnap{{UserID: creator, Role: "teacher", Active: true, DisplayName: "C"}},
	})
	if f.Status != StatusTodo {
		t.Fatalf("single staff without creator id=%s", f.Status)
	}
}
