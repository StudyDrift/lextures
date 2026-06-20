package assignmentoverrides

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func mustTime(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestResolveFromRows_NoTargets_VisibleWithNoOverride(t *testing.T) {
	visible, eff := ResolveFromRows(nil, uuid.New(), nil, nil)
	if !visible {
		t.Fatal("expected an item with no override rows to be visible (backward compatibility)")
	}
	if eff.DueAt != nil || eff.AvailableFrom != nil || eff.AvailableUntil != nil {
		t.Fatal("expected no field overrides when there are no target rows")
	}
}

func TestResolveFromRows_StudentBeatsGroupBeatsSectionBeatsEveryone(t *testing.T) {
	enrollmentID := uuid.New()
	groupID := uuid.New()
	sectionID := uuid.New()

	everyoneDue := mustTime("2026-01-01T00:00:00Z")
	sectionDue := mustTime("2026-01-05T00:00:00Z")
	groupDue := mustTime("2026-01-10T00:00:00Z")
	studentDue := mustTime("2026-01-15T00:00:00Z")

	rows := []OverrideRow{
		{TargetType: TargetEveryone, DueAt: everyoneDue, CreatedAt: time.Unix(1, 0)},
		{TargetType: TargetSection, TargetID: &sectionID, DueAt: sectionDue, CreatedAt: time.Unix(2, 0)},
		{TargetType: TargetGroup, TargetID: &groupID, DueAt: groupDue, CreatedAt: time.Unix(3, 0)},
		{TargetType: TargetStudent, TargetID: &enrollmentID, DueAt: studentDue, CreatedAt: time.Unix(4, 0)},
	}

	// Student-specific override wins over everything else.
	visible, eff := ResolveFromRows(rows, enrollmentID, &sectionID, []uuid.UUID{groupID})
	if !visible || eff.DueAt == nil || !eff.DueAt.Equal(*studentDue) {
		t.Fatalf("expected student override to win, got visible=%v eff=%+v", visible, eff)
	}

	// A different student in the same section/group falls back to the group override.
	other := uuid.New()
	visible, eff = ResolveFromRows(rows, other, &sectionID, []uuid.UUID{groupID})
	if !visible || eff.DueAt == nil || !eff.DueAt.Equal(*groupDue) {
		t.Fatalf("expected group override to win for non-targeted student, got visible=%v eff=%+v", visible, eff)
	}

	// A student in the section but no targeted group falls back to the section override.
	visible, eff = ResolveFromRows(rows, other, &sectionID, nil)
	if !visible || eff.DueAt == nil || !eff.DueAt.Equal(*sectionDue) {
		t.Fatalf("expected section override to win, got visible=%v eff=%+v", visible, eff)
	}

	// A student in neither section nor group falls back to the everyone override.
	visible, eff = ResolveFromRows(rows, other, nil, nil)
	if !visible || eff.DueAt == nil || !eff.DueAt.Equal(*everyoneDue) {
		t.Fatalf("expected everyone override to win, got visible=%v eff=%+v", visible, eff)
	}
}

func TestResolveFromRows_TargetedItemHidesNonMembers(t *testing.T) {
	sectionID := uuid.New()
	rows := []OverrideRow{
		{TargetType: TargetSection, TargetID: &sectionID, DueAt: mustTime("2026-01-05T00:00:00Z")},
	}
	visible, _ := ResolveFromRows(rows, uuid.New(), nil, nil)
	if visible {
		t.Fatal("expected a student outside the only target to be hidden")
	}
}

func TestResolveFromRows_QuizFieldsCarryThrough(t *testing.T) {
	enrollmentID := uuid.New()
	extra := int32(2)
	mult := 1.5
	rows := []OverrideRow{
		{TargetType: TargetStudent, TargetID: &enrollmentID, ExtraAttempts: &extra, TimeMultiplier: &mult},
	}
	visible, eff := ResolveFromRows(rows, enrollmentID, nil, nil)
	if !visible {
		t.Fatal("expected visible")
	}
	if eff.ExtraAttempts == nil || *eff.ExtraAttempts != 2 {
		t.Fatalf("expected extra attempts 2, got %+v", eff.ExtraAttempts)
	}
	if eff.TimeMultiplier == nil || *eff.TimeMultiplier != 1.5 {
		t.Fatalf("expected time multiplier 1.5, got %+v", eff.TimeMultiplier)
	}
}

func TestReplaceForItem_RejectsInvalidTargetType(t *testing.T) {
	writes := []OverrideWrite{{TargetType: "bogus"}}
	if err := ReplaceForItem(nil, nil, uuid.New(), writes, uuid.New()); err == nil { //nolint:staticcheck // validation runs before any pool/context use
		t.Fatal("expected an error for an invalid target type")
	}
}

func TestReplaceForItem_RejectsEveryoneWithTargetID(t *testing.T) {
	id := uuid.New()
	writes := []OverrideWrite{{TargetType: TargetEveryone, TargetID: &id}}
	if err := ReplaceForItem(nil, nil, uuid.New(), writes, uuid.New()); err == nil { //nolint:staticcheck
		t.Fatal("expected an error when an everyone target carries a target id")
	}
}

func TestReplaceForItem_RejectsNonEveryoneWithoutTargetID(t *testing.T) {
	writes := []OverrideWrite{{TargetType: TargetSection}}
	if err := ReplaceForItem(nil, nil, uuid.New(), writes, uuid.New()); err == nil { //nolint:staticcheck
		t.Fatal("expected an error when a section target is missing a target id")
	}
}
