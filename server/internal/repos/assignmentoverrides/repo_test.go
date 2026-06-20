package assignmentoverrides

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func tp(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestResolve_NoTargets_FallsBackToBase(t *testing.T) {
	base := BaseDates{DueAt: tp("2026-01-10T00:00:00Z")}
	eff := resolve(nil, nil, base)
	if !eff.Visible || eff.MatchedTarget != "" {
		t.Fatalf("want visible implicit everyone, got %+v", eff)
	}
	if eff.DueAt == nil || !eff.DueAt.Equal(*base.DueAt) {
		t.Fatalf("want base due date, got %+v", eff.DueAt)
	}
}

func TestResolve_MostSpecificWins(t *testing.T) {
	sectionID := uuid.New()
	groupID := uuid.New()
	studentID := uuid.New()
	base := BaseDates{DueAt: tp("2026-01-01T00:00:00Z")}

	targets := []Target{
		{TargetType: "everyone", DueAt: tp("2026-01-05T00:00:00Z")},
		{TargetType: "section", TargetID: &sectionID, DueAt: tp("2026-01-10T00:00:00Z")},
		{TargetType: "group", TargetID: &groupID, DueAt: tp("2026-01-15T00:00:00Z")},
		{TargetType: "student", TargetID: &studentID, DueAt: tp("2026-01-20T00:00:00Z")},
	}

	// Student in the section AND the group AND has their own override: student wins.
	stu := &studentContext{EnrollmentID: studentID, SectionID: &sectionID, GroupIDs: []uuid.UUID{groupID}}
	eff := resolve(targets, stu, base)
	if eff.MatchedTarget != "student" || !eff.DueAt.Equal(*tp("2026-01-20T00:00:00Z")) {
		t.Fatalf("want student override to win, got %+v", eff)
	}

	// Different student, same section+group, no personal override: group wins over section.
	other := uuid.New()
	stu2 := &studentContext{EnrollmentID: other, SectionID: &sectionID, GroupIDs: []uuid.UUID{groupID}}
	eff2 := resolve(targets, stu2, base)
	if eff2.MatchedTarget != "group" || !eff2.DueAt.Equal(*tp("2026-01-15T00:00:00Z")) {
		t.Fatalf("want group override to win over section, got %+v", eff2)
	}

	// Student only in the section: section wins over everyone.
	stu3 := &studentContext{EnrollmentID: other, SectionID: &sectionID}
	eff3 := resolve(targets, stu3, base)
	if eff3.MatchedTarget != "section" || !eff3.DueAt.Equal(*tp("2026-01-10T00:00:00Z")) {
		t.Fatalf("want section override to win over everyone, got %+v", eff3)
	}

	// Student with no section/group: everyone applies.
	stu4 := &studentContext{EnrollmentID: other}
	eff4 := resolve(targets, stu4, base)
	if eff4.MatchedTarget != "everyone" || !eff4.DueAt.Equal(*tp("2026-01-05T00:00:00Z")) {
		t.Fatalf("want everyone override to apply, got %+v", eff4)
	}
}

func TestResolve_HidesItemWhenNoTargetMatches(t *testing.T) {
	sectionID := uuid.New()
	targets := []Target{
		{TargetType: "section", TargetID: &sectionID, DueAt: tp("2026-01-10T00:00:00Z")},
	}
	other := uuid.New()
	stu := &studentContext{EnrollmentID: other} // not in the targeted section
	eff := resolve(targets, stu, BaseDates{})
	if eff.Visible {
		t.Fatalf("want item hidden from non-targeted student, got %+v", eff)
	}
}

func TestResolve_PerFieldFallbackToBase(t *testing.T) {
	sectionID := uuid.New()
	base := BaseDates{DueAt: tp("2026-01-01T00:00:00Z"), AvailableFrom: tp("2025-12-01T00:00:00Z"), AvailableUntil: tp("2026-02-01T00:00:00Z")}
	targets := []Target{
		// Section override only sets due_at; availability should fall back to base.
		{TargetType: "section", TargetID: &sectionID, DueAt: tp("2026-01-10T00:00:00Z")},
	}
	stu := &studentContext{SectionID: &sectionID}
	eff := resolve(targets, stu, base)
	if !eff.Visible {
		t.Fatalf("want visible")
	}
	if !eff.DueAt.Equal(*tp("2026-01-10T00:00:00Z")) {
		t.Fatalf("want overridden due date, got %+v", eff.DueAt)
	}
	if !eff.AvailableFrom.Equal(*base.AvailableFrom) || !eff.AvailableUntil.Equal(*base.AvailableUntil) {
		t.Fatalf("want availability to fall back to base, got from=%+v until=%+v", eff.AvailableFrom, eff.AvailableUntil)
	}
}

func TestReplaceForItem_ValidatesTargetShape(t *testing.T) {
	bad := []TargetWrite{{TargetType: "everyone", TargetID: ptrUUID(uuid.New())}}
	if err := validateWrites(bad); err == nil {
		t.Fatalf("want error for everyone target with target id")
	}
	bad2 := []TargetWrite{{TargetType: "section", TargetID: nil}}
	if err := validateWrites(bad2); err == nil {
		t.Fatalf("want error for section target missing target id")
	}
	bad3 := []TargetWrite{{TargetType: "bogus"}}
	if err := validateWrites(bad3); err == nil {
		t.Fatalf("want error for invalid target type")
	}
	id := uuid.New()
	dup := []TargetWrite{{TargetType: "student", TargetID: &id}, {TargetType: "student", TargetID: &id}}
	if err := validateWrites(dup); err == nil {
		t.Fatalf("want error for duplicate target")
	}
	ok := []TargetWrite{{TargetType: "everyone"}, {TargetType: "student", TargetID: &id}}
	if err := validateWrites(ok); err != nil {
		t.Fatalf("want valid writes to pass, got %v", err)
	}
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
