package httpserver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/repos/course"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// filterDateableStructureItems returns assignments, quizzes, and content pages
// that can receive a due date, including items that are currently undated.
func filterDateableStructureItems(items []coursestructurerepo.ItemResponse) []coursestructurerepo.ItemResponse {
	out := make([]coursestructurerepo.ItemResponse, 0)
	for _, it := range items {
		switch it.Kind {
		case "assignment", "quiz", "content_page":
			out = append(out, it)
		}
	}
	return out
}

func formatAdjustDatesAIContext(c *course.CoursePublic, allItems, dateable []coursestructurerepo.ItemResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Course title: %s\nCourse code: %s\nSchedule mode: %s\n", c.Title, c.CourseCode, c.ScheduleMode)
	if c.RelativeScheduleAnchorAt != nil {
		fmt.Fprintf(&b, "Relative schedule anchor: %s\n", c.RelativeScheduleAnchorAt.UTC().Format(time.RFC3339))
	}
	if c.StartsAt != nil {
		fmt.Fprintf(&b, "Course startsAt: %s\n", c.StartsAt.UTC().Format(time.RFC3339))
	}
	if c.EndsAt != nil {
		fmt.Fprintf(&b, "Course endsAt: %s\n", c.EndsAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "Today (UTC): %s\n", time.Now().UTC().Format(time.RFC3339))

	// Module titles help the model keep progressive order by outline.
	moduleTitleByID := map[string]string{}
	for _, it := range allItems {
		if it.Kind == "module" {
			moduleTitleByID[it.ID] = it.Title
		}
	}

	datedCount := 0
	for _, it := range dateable {
		if it.DueAt != nil {
			datedCount++
		}
	}
	// Numbered list (1-based "i") keeps model output compact and avoids UUID hallucination.
	fmt.Fprintf(&b, "\nDateable items in outline order (%d total, %d already dated) (i | kind | module | title | dueAt | itemId):\n",
		len(dateable), datedCount)
	for i, it := range dateable {
		due := "none"
		if it.DueAt != nil {
			due = it.DueAt.UTC().Format(time.RFC3339)
		}
		mod := ""
		if it.ParentID != nil {
			mod = moduleTitleByID[*it.ParentID]
		}
		fmt.Fprintf(&b, "%d | %s | %q | %q | %s | %s\n", i+1, it.Kind, mod, it.Title, due, it.ID)
	}
	return b.String()
}

func parseAdjustDatesAIResponse(raw string, dateable []coursestructurerepo.ItemResponse) (adjustDatesAIResponse, error) {
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "```json"); idx != -1 {
		clean = clean[idx+7:]
		if endIdx := strings.Index(clean, "```"); endIdx != -1 {
			clean = clean[:endIdx]
		}
	} else if idx := strings.Index(clean, "```"); idx != -1 {
		clean = clean[idx+3:]
		if endIdx := strings.Index(clean, "```"); endIdx != -1 {
			clean = clean[:endIdx]
		}
	}
	clean = strings.TrimSpace(clean)
	if !strings.HasPrefix(clean, "{") {
		if start := strings.Index(clean, "{"); start != -1 {
			if end := strings.LastIndex(clean, "}"); end > start {
				clean = clean[start : end+1]
			}
		}
	}
	var rawParsed adjustDatesAIRaw
	if err := json.Unmarshal([]byte(clean), &rawParsed); err != nil {
		return adjustDatesAIResponse{}, err
	}
	out := adjustDatesAIResponse{
		Reply: strings.TrimSpace(rawParsed.Reply),
		Plan:  rawParsed.Plan,
	}
	if out.Reply == "" {
		out.Reply = "Here are the proposed due date changes."
	}

	// Map index-based and itemId-based proposals onto real item IDs.
	explicit := make([]adjustDatesAIProposal, 0, len(rawParsed.Proposals))
	for _, p := range rawParsed.Proposals {
		due := strings.TrimSpace(p.DueAt)
		if due == "" {
			continue
		}
		id := strings.TrimSpace(p.ItemID)
		if p.I != nil {
			idx := *p.I
			if idx >= 1 && idx <= len(dateable) {
				id = dateable[idx-1].ID
			}
		}
		if id == "" {
			continue
		}
		explicit = append(explicit, adjustDatesAIProposal{ItemID: id, DueAt: due})
	}

	// Expand compact plan first, then let explicit proposals override.
	var planned []adjustDatesAIProposal
	if rawParsed.Plan != nil {
		planned = expandEvenSchedule(dateable, normalizePlan(*rawParsed.Plan), nil)
	}
	out.Proposals = mergePlanAndProposals(planned, explicit)
	return out, nil
}

// mergePlanAndProposals prefers later explicit proposals for the same itemId.
func mergePlanAndProposals(plan, explicit []adjustDatesAIProposal) []adjustDatesAIProposal {
	if len(plan) == 0 {
		return explicit
	}
	if len(explicit) == 0 {
		return plan
	}
	byID := map[string]string{}
	order := make([]string, 0, len(plan)+len(explicit))
	for _, p := range plan {
		if _, ok := byID[p.ItemID]; !ok {
			order = append(order, p.ItemID)
		}
		byID[p.ItemID] = p.DueAt
	}
	for _, p := range explicit {
		if _, ok := byID[p.ItemID]; !ok {
			order = append(order, p.ItemID)
		}
		byID[p.ItemID] = p.DueAt
	}
	out := make([]adjustDatesAIProposal, 0, len(order))
	for _, id := range order {
		out = append(out, adjustDatesAIProposal{ItemID: id, DueAt: byID[id]})
	}
	return out
}

var (
	reWeeks  = regexp.MustCompile(`(?i)\b(\d{1,3})\s*-\s*week\b|\b(\d{1,3})\s*weeks?\b`)
	reMonths = regexp.MustCompile(`(?i)\b(\d{1,2})\s*months?\b`)
	reDays   = regexp.MustCompile(`(?i)\b(\d{1,3})\s*days?\b`)
	reTerm   = regexp.MustCompile(`(?i)\b(over|across|for|during)\s+(the\s+)?(term|semester|course\s+term)\b|\bterm\s+bounds?\b`)
)

// resolveDeterministicInitialPlan builds an even-spacing plan for undated courses
// when the instruction is empty or clearly expresses a duration / term span.
// Returns ok=false when the request needs a full model call.
func resolveDeterministicInitialPlan(instruction string, c *course.CoursePublic) (*adjustDatesAIPlan, string, bool) {
	instr := strings.TrimSpace(instruction)
	start := scheduleStartUTC(c)
	applyTo := "undated"

	// Explicit duration in natural language.
	if days, ok := parseDurationDaysFromInstruction(instr); ok && days > 0 {
		plan := &adjustDatesAIPlan{
			StartDate:    start.Format("2006-01-02"),
			DurationDays: days,
			ApplyTo:      applyTo,
		}
		reply := fmt.Sprintf("Scheduled all undated items evenly over %d days (end-of-day deadlines), in outline order.", days)
		if days%7 == 0 {
			weeks := days / 7
			reply = fmt.Sprintf("Scheduled all undated items evenly over %d weeks (end-of-day deadlines), in outline order.", weeks)
		}
		return plan, reply, true
	}

	// "over the term" with course bounds.
	if instr == "" || reTerm.MatchString(instr) {
		if c != nil && c.StartsAt != nil && c.EndsAt != nil && c.EndsAt.After(*c.StartsAt) {
			plan := &adjustDatesAIPlan{
				StartDate: c.StartsAt.UTC().Format("2006-01-02"),
				EndDate:   c.EndsAt.UTC().Format("2006-01-02"),
				ApplyTo:   applyTo,
			}
			return plan, "Scheduled all undated items evenly across the course term (end-of-day deadlines), in outline order.", true
		}
	}

	// Empty guidance with no term bounds: default to 4 weeks from start/anchor/today.
	if instr == "" {
		plan := &adjustDatesAIPlan{
			StartDate:    start.Format("2006-01-02"),
			DurationDays: 28,
			ApplyTo:      applyTo,
		}
		return plan, "Scheduled all undated items evenly over 4 weeks from the course start/anchor (end-of-day deadlines), in outline order.", true
	}

	return nil, "", false
}

func parseDurationDaysFromInstruction(instruction string) (int, bool) {
	if instruction == "" {
		return 0, false
	}
	if m := reWeeks.FindStringSubmatch(instruction); m != nil {
		n := firstNonEmptyInt(m[1], m[2])
		if n > 0 && n <= 104 {
			return n * 7, true
		}
	}
	if m := reMonths.FindStringSubmatch(instruction); m != nil {
		n := firstNonEmptyInt(m[1])
		if n > 0 && n <= 24 {
			return n * 30, true
		}
	}
	if m := reDays.FindStringSubmatch(instruction); m != nil {
		n := firstNonEmptyInt(m[1])
		if n > 0 && n <= 400 {
			return n, true
		}
	}
	return 0, false
}

func firstNonEmptyInt(parts ...string) int {
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil {
			return n
		}
	}
	return 0
}

func scheduleStartUTC(c *course.CoursePublic) time.Time {
	now := time.Now().UTC()
	if c != nil {
		if c.StartsAt != nil {
			return endOfDayUTC(*c.StartsAt)
		}
		if c.RelativeScheduleAnchorAt != nil {
			return endOfDayUTC(*c.RelativeScheduleAnchorAt)
		}
	}
	return endOfDayUTC(now)
}

func endOfDayUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 23, 59, 0, 0, time.UTC)
}

func normalizePlan(p adjustDatesAIPlan) *adjustDatesAIPlan {
	cp := p
	if strings.TrimSpace(cp.ApplyTo) == "" {
		cp.ApplyTo = "undated"
	}
	return &cp
}

// expandEvenSchedule spaces selected dateable items evenly between plan start and end.
// alreadyProposed item IDs are skipped (explicit overrides win).
func expandEvenSchedule(dateable []coursestructurerepo.ItemResponse, plan *adjustDatesAIPlan, alreadyProposed map[string]bool) []adjustDatesAIProposal {
	if plan == nil || len(dateable) == 0 {
		return nil
	}
	start, end, ok := resolvePlanWindow(plan)
	if !ok {
		return nil
	}
	applyAll := strings.EqualFold(strings.TrimSpace(plan.ApplyTo), "all")

	targets := make([]coursestructurerepo.ItemResponse, 0, len(dateable))
	for _, it := range dateable {
		if alreadyProposed != nil && alreadyProposed[it.ID] {
			continue
		}
		if !applyAll && it.DueAt != nil {
			continue
		}
		targets = append(targets, it)
	}
	if len(targets) == 0 {
		return nil
	}

	out := make([]adjustDatesAIProposal, 0, len(targets))
	span := end.Sub(start)
	if span < 0 {
		span = 0
	}
	n := len(targets)
	for i, it := range targets {
		var due time.Time
		if n == 1 {
			due = start
		} else {
			// Inclusive even spacing: first at start, last at end.
			due = start.Add(time.Duration(float64(span) * float64(i) / float64(n-1)))
		}
		due = endOfDayUTC(due)
		out = append(out, adjustDatesAIProposal{
			ItemID: it.ID,
			DueAt:  due.Format(time.RFC3339),
		})
	}
	return out
}

func resolvePlanWindow(plan *adjustDatesAIPlan) (start, end time.Time, ok bool) {
	if plan == nil {
		return time.Time{}, time.Time{}, false
	}
	start, ok = parsePlanDate(plan.StartDate)
	if !ok {
		start = endOfDayUTC(time.Now().UTC())
	}
	if ed, eok := parsePlanDate(plan.EndDate); eok {
		end = ed
	} else if plan.DurationDays > 0 {
		// Duration covers the inclusive span: 7 days => start .. start+6d.
		days := plan.DurationDays
		if days < 1 {
			days = 1
		}
		end = start.AddDate(0, 0, days-1)
	} else {
		end = start.AddDate(0, 0, 27) // default 4 weeks inclusive
	}
	if end.Before(start) {
		start, end = end, start
	}
	return endOfDayUTC(start), endOfDayUTC(end), true
}

func parsePlanDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

// sanitizeAdjustDatesAIProposals keeps proposals for known dateable items only.
// Items with no current due date accept any valid timestamp; items with a due date
// skip proposals that match the existing value.
func sanitizeAdjustDatesAIProposals(in []adjustDatesAIProposal, dateable []coursestructurerepo.ItemResponse) []adjustDatesAIProposal {
	known := map[string]*time.Time{}
	for _, it := range dateable {
		var due *time.Time
		if it.DueAt != nil {
			t := it.DueAt.UTC()
			due = &t
		}
		known[it.ID] = due
	}
	var out []adjustDatesAIProposal
	seen := map[string]bool{}
	for _, p := range in {
		id := strings.TrimSpace(p.ItemID)
		if id == "" || seen[id] {
			continue
		}
		orig, ok := known[id]
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(p.DueAt))
		if err != nil {
			// try RFC3339Nano / common variants
			t, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(p.DueAt))
			if err != nil {
				continue
			}
		}
		t = t.UTC()
		if orig != nil && t.Equal(*orig) {
			continue
		}
		seen[id] = true
		out = append(out, adjustDatesAIProposal{ItemID: id, DueAt: t.Format(time.RFC3339)})
	}
	return out
}
