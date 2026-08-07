package httpserver

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/repos/course"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestParseAdjustDatesAIResponse(t *testing.T) {
	dateable := []coursestructurerepo.ItemResponse{
		{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Kind: "assignment", Title: "Essay"},
	}
	raw := "```json\n{\"reply\":\"Shifted a week\",\"proposals\":[{\"itemId\":\"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\",\"dueAt\":\"2026-09-15T23:59:00Z\"}]}\n```"
	parsed, err := parseAdjustDatesAIResponse(raw, dateable)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Reply != "Shifted a week" {
		t.Fatalf("reply: %q", parsed.Reply)
	}
	if len(parsed.Proposals) != 1 || parsed.Proposals[0].ItemID == "" {
		t.Fatalf("proposals: %+v", parsed.Proposals)
	}
}

func TestParseAdjustDatesAIResponse_IndexAndPlan(t *testing.T) {
	dateable := []coursestructurerepo.ItemResponse{
		{ID: "id-1", Kind: "assignment", Title: "A"},
		{ID: "id-2", Kind: "quiz", Title: "B"},
		{ID: "id-3", Kind: "content_page", Title: "C"},
	}
	raw := `{"reply":"4 weeks","plan":{"startDate":"2026-08-01","durationDays":28,"applyTo":"undated"},"proposals":[{"i":2,"dueAt":"2026-08-10T23:59:00Z"}]}`
	parsed, err := parseAdjustDatesAIResponse(raw, dateable)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Proposals) != 3 {
		t.Fatalf("expected 3 proposals from plan+override, got %+v", parsed.Proposals)
	}
	byID := map[string]string{}
	for _, p := range parsed.Proposals {
		byID[p.ItemID] = p.DueAt
	}
	if byID["id-2"] != "2026-08-10T23:59:00Z" {
		t.Fatalf("index override for id-2: %s", byID["id-2"])
	}
	if byID["id-1"] == "" || byID["id-3"] == "" {
		t.Fatalf("plan expansion missing items: %+v", byID)
	}
}

func TestSanitizeAdjustDatesAIProposals(t *testing.T) {
	due := time.Date(2026, 1, 10, 23, 59, 0, 0, time.UTC)
	dated := []coursestructurerepo.ItemResponse{
		{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Kind: "assignment", Title: "Essay", DueAt: &due},
	}
	in := []adjustDatesAIProposal{
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-01-17T23:59:00Z"},
		{ItemID: "missing", DueAt: "2026-01-18T23:59:00Z"},
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-01-10T23:59:00Z"}, // same as original, skip
	}
	out := sanitizeAdjustDatesAIProposals(in, dated)
	if len(out) != 1 {
		t.Fatalf("expected 1 proposal, got %+v", out)
	}
	if out[0].DueAt != "2026-01-17T23:59:00Z" {
		t.Fatalf("dueAt: %s", out[0].DueAt)
	}
}

func TestSanitizeAdjustDatesAIProposals_UndatedItems(t *testing.T) {
	dateable := []coursestructurerepo.ItemResponse{
		{ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Kind: "assignment", Title: "Essay", DueAt: nil},
		{ID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", Kind: "quiz", Title: "Quiz 1", DueAt: nil},
	}
	in := []adjustDatesAIProposal{
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-09-08T23:59:00Z"},
		{ItemID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", DueAt: "2026-09-15T23:59:00Z"},
		{ItemID: "missing", DueAt: "2026-09-20T23:59:00Z"},
		{ItemID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", DueAt: "2026-09-09T23:59:00Z"}, // duplicate id, skip
	}
	out := sanitizeAdjustDatesAIProposals(in, dateable)
	if len(out) != 2 {
		t.Fatalf("expected 2 proposals for undated items, got %+v", out)
	}
	if out[0].DueAt != "2026-09-08T23:59:00Z" || out[1].DueAt != "2026-09-15T23:59:00Z" {
		t.Fatalf("dueAts: %+v", out)
	}
}

func TestFilterDateableStructureItems(t *testing.T) {
	due := time.Date(2026, 1, 10, 23, 59, 0, 0, time.UTC)
	items := []coursestructurerepo.ItemResponse{
		{ID: "m1", Kind: "module", Title: "Mod"},
		{ID: "a1", Kind: "assignment", Title: "Essay", DueAt: &due},
		{ID: "p1", Kind: "content_page", Title: "Reading", DueAt: nil},
		{ID: "l1", Kind: "external_link", Title: "Link"},
	}
	dateable := filterDateableStructureItems(items)
	if len(dateable) != 2 {
		t.Fatalf("expected 2 dateable, got %+v", dateable)
	}
	var dated []coursestructurerepo.ItemResponse
	for _, it := range dateable {
		if it.DueAt != nil {
			dated = append(dated, it)
		}
	}
	if len(dated) != 1 || dated[0].ID != "a1" {
		t.Fatalf("expected only a1 dated, got %+v", dated)
	}
}

func TestFormatAdjustDatesAIContext_IncludesUndated(t *testing.T) {
	modID := "mod-1"
	all := []coursestructurerepo.ItemResponse{
		{ID: modID, Kind: "module", Title: "Week 1"},
		{ID: "a1", Kind: "assignment", Title: "Essay", ParentID: &modID, DueAt: nil},
	}
	dateable := filterDateableStructureItems(all)
	c := &course.CoursePublic{Title: "Intro", CourseCode: "INTRO-1", ScheduleMode: "relative"}
	ctx := formatAdjustDatesAIContext(c, all, dateable)
	if !strings.Contains(ctx, "none") {
		t.Fatalf("expected undated dueAt marker in context:\n%s", ctx)
	}
	if !strings.Contains(ctx, "Week 1") {
		t.Fatalf("expected module title in context:\n%s", ctx)
	}
	if !strings.Contains(ctx, "a1") {
		t.Fatalf("expected item id in context:\n%s", ctx)
	}
	if !strings.Contains(ctx, "1 |") {
		t.Fatalf("expected 1-based index in context:\n%s", ctx)
	}
}

func TestResolveDeterministicInitialPlan_FourWeekCourse(t *testing.T) {
	anchor := time.Date(2026, 8, 1, 22, 39, 0, 0, time.UTC)
	c := &course.CoursePublic{
		Title:                    "Intro",
		CourseCode:               "INTRO-1",
		ScheduleMode:             "relative",
		RelativeScheduleAnchorAt: &anchor,
	}
	plan, reply, ok := resolveDeterministicInitialPlan("Set the due dates so that it's a 4 week course", c)
	if !ok || plan == nil {
		t.Fatalf("expected deterministic plan for 4 week instruction")
	}
	if plan.DurationDays != 28 {
		t.Fatalf("durationDays=%d want 28", plan.DurationDays)
	}
	if plan.StartDate != "2026-08-01" {
		t.Fatalf("startDate=%s want 2026-08-01", plan.StartDate)
	}
	if !strings.Contains(strings.ToLower(reply), "week") {
		t.Fatalf("reply should mention weeks: %q", reply)
	}
}

func TestExpandEvenSchedule_39ItemsFourWeeks(t *testing.T) {
	dateable := make([]coursestructurerepo.ItemResponse, 0, 39)
	for i := 0; i < 39; i++ {
		dateable = append(dateable, coursestructurerepo.ItemResponse{
			ID:    fmt.Sprintf("item-%d", i),
			Kind:  "assignment",
			Title: "Item",
		})
	}
	plan := &adjustDatesAIPlan{
		StartDate:    "2026-08-01",
		DurationDays: 28,
		ApplyTo:      "undated",
	}
	out := expandEvenSchedule(dateable, plan, nil)
	if len(out) != 39 {
		t.Fatalf("expected 39 proposals, got %d", len(out))
	}
	first, err := time.Parse(time.RFC3339, out[0].DueAt)
	if err != nil {
		t.Fatal(err)
	}
	last, err := time.Parse(time.RFC3339, out[38].DueAt)
	if err != nil {
		t.Fatal(err)
	}
	if first.Day() != 1 || first.Month() != time.August {
		t.Fatalf("first due: %s", out[0].DueAt)
	}
	// Inclusive 28-day window: Aug 1 .. Aug 28
	if last.Day() != 28 || last.Month() != time.August {
		t.Fatalf("last due: %s", out[38].DueAt)
	}
	if !first.Before(last) {
		t.Fatalf("expected progressive schedule, first=%s last=%s", out[0].DueAt, out[38].DueAt)
	}
}

func TestParseDurationDaysFromInstruction(t *testing.T) {
	cases := []struct {
		in   string
		days int
		ok   bool
	}{
		{"Set the due dates so that it's a 4 week course", 28, true},
		{"make it a 12-week course", 84, true},
		{"spread over 30 days", 30, true},
		{"2 month pace", 60, true},
		{"please be smart about pacing", 0, false},
	}
	for _, tc := range cases {
		days, ok := parseDurationDaysFromInstruction(tc.in)
		if ok != tc.ok || days != tc.days {
			t.Fatalf("%q: got (%d,%v) want (%d,%v)", tc.in, days, ok, tc.days, tc.ok)
		}
	}
}
