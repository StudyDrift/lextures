package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	stprog "github.com/lextures/lextures/server/internal/repos/studentprogress"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/coursefinalgrade"
)

// Parent home (#772): one card per child with due this week, completed this week,
// current course grades, and time spent this week.

type parentHomeWorkItem struct {
	CourseCode  string  `json:"courseCode"`
	CourseTitle string  `json:"courseTitle"`
	ItemID      string  `json:"itemId"`
	Kind        string  `json:"kind"`
	Title       string  `json:"title"`
	DueAt       *string `json:"dueAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
	Overdue     bool    `json:"overdue"`
}

type parentHomeCourse struct {
	CourseCode   string   `json:"courseCode"`
	Title        string   `json:"title"`
	EnrollmentID *string  `json:"enrollmentId,omitempty"`
	GradePercent *float64 `json:"gradePercent,omitempty"`
}

type parentHomeChild struct {
	StudentUserID     string               `json:"studentUserId"`
	DisplayName       string               `json:"displayName"`
	Courses           []parentHomeCourse   `json:"courses"`
	DueThisWeek       []parentHomeWorkItem `json:"dueThisWeek"`
	CompletedThisWeek []parentHomeWorkItem `json:"completedThisWeek"`
	TimeSpentSeconds  int                  `json:"timeSpentSeconds"`
}

// parentHomeWeek returns the Monday-start week containing now in loc, as UTC instants.
func parentHomeWeek(now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	offset := (int(local.Weekday()) + 6) % 7 // Monday = 0
	start := time.Date(local.Year(), local.Month(), local.Day()-offset, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 7)
	return start.UTC(), end.UTC()
}

// classifyParentHomeWork splits a child's work into due-this-week (incomplete items due in the
// week, plus any overdue incomplete item, flagged) and completed-this-week.
func classifyParentHomeWork(rows []stprog.WorkItemRow, now, weekStart, weekEnd time.Time) (due, done []stprog.WorkItemRow, overdue map[uuid.UUID]bool) {
	overdue = map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.CompletedAt != nil {
			if !r.CompletedAt.Before(weekStart) && r.CompletedAt.Before(weekEnd) {
				done = append(done, r)
			}
			continue
		}
		if r.DueAt == nil {
			continue
		}
		isOverdue := r.DueAt.Before(now)
		inWeek := !r.DueAt.Before(weekStart) && r.DueAt.Before(weekEnd)
		if isOverdue || inWeek {
			due = append(due, r)
			if isOverdue {
				overdue[r.ItemID] = true
			}
		}
	}
	return due, done, overdue
}

func parentHomeLocation(ctx context.Context, d Deps, viewer uuid.UUID, queryTZ string) *time.Location {
	if tz, err := user.GetTimezone(ctx, d.Pool, viewer); err == nil && tz != nil {
		if loc, err := time.LoadLocation(*tz); err == nil {
			return loc
		}
	}
	if q := strings.TrimSpace(queryTZ); q != "" {
		if loc, err := time.LoadLocation(q); err == nil {
			return loc
		}
	}
	return time.UTC
}

func fmtRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func (d Deps) buildParentHomeChild(ctx context.Context, ln parentlinks.Link, now, weekStart, weekEnd time.Time) (parentHomeChild, error) {
	name := ln.StudentEmail
	if ln.StudentDisplay != nil && strings.TrimSpace(*ln.StudentDisplay) != "" {
		name = strings.TrimSpace(*ln.StudentDisplay)
	}
	child := parentHomeChild{
		StudentUserID:     ln.StudentUserID.String(),
		DisplayName:       name,
		Courses:           []parentHomeCourse{},
		DueThisWeek:       []parentHomeWorkItem{},
		CompletedThisWeek: []parentHomeWorkItem{},
	}
	courses, err := course.ListForEnrolledUser(ctx, d.Pool, ln.StudentUserID, nil)
	if err != nil {
		return child, err
	}
	for _, c := range courses {
		if c.Archived {
			continue
		}
		cid, err := uuid.Parse(c.ID)
		if err != nil {
			continue
		}
		title := c.Title
		if title == "" {
			title = c.CourseCode
		}
		hc := parentHomeCourse{CourseCode: c.CourseCode, Title: title}
		eid, _ := enrollment.GetStudentEnrollmentID(ctx, d.Pool, cid, ln.StudentUserID)
		if eid != nil {
			s := eid.String()
			hc.EnrollmentID = &s
		}
		// Same live calculation as the gradebook FINAL column and the progress page.
		if pct, err := coursefinalgrade.PercentForStudent(ctx, d.Pool, cid, ln.StudentUserID, now); err == nil {
			hc.GradePercent = pct
		} else {
			slog.Warn("parent_home.grade_failed", "err", err)
		}
		child.Courses = append(child.Courses, hc)

		rows, err := stprog.ListWorkItems(ctx, d.Pool, cid, ln.StudentUserID)
		if err != nil {
			return child, err
		}
		if eid != nil && len(rows) > 0 {
			ids := make([]uuid.UUID, 0, len(rows))
			bases := make(map[uuid.UUID]assignmentoverrides.BaseDates, len(rows))
			for _, r := range rows {
				ids = append(ids, r.ItemID)
				bases[r.ItemID] = assignmentoverrides.BaseDates{DueAt: r.DueAt}
			}
			if eff, err := assignmentoverrides.EffectiveForStudentBatch(ctx, d.Pool, *eid, ids, bases); err == nil {
				kept := rows[:0]
				for _, r := range rows {
					if e, ok := eff[r.ItemID]; ok {
						if !e.Visible {
							continue
						}
						r.DueAt = e.DueAt
					}
					kept = append(kept, r)
				}
				rows = kept
			}
		}
		due, done, overdue := classifyParentHomeWork(rows, now, weekStart, weekEnd)
		for _, r := range due {
			child.DueThisWeek = append(child.DueThisWeek, parentHomeWorkItem{
				CourseCode: c.CourseCode, CourseTitle: title, ItemID: r.ItemID.String(), Kind: r.Kind,
				Title: r.Title, DueAt: fmtRFC3339(r.DueAt), Overdue: overdue[r.ItemID],
			})
		}
		for _, r := range done {
			child.CompletedThisWeek = append(child.CompletedThisWeek, parentHomeWorkItem{
				CourseCode: c.CourseCode, CourseTitle: title, ItemID: r.ItemID.String(), Kind: r.Kind,
				Title: r.Title, DueAt: fmtRFC3339(r.DueAt), CompletedAt: fmtRFC3339(r.CompletedAt),
			})
		}
	}
	sort.SliceStable(child.DueThisWeek, func(i, j int) bool {
		a, b := child.DueThisWeek[i].DueAt, child.DueThisWeek[j].DueAt
		return a != nil && b != nil && *a < *b
	})
	sort.SliceStable(child.CompletedThisWeek, func(i, j int) bool {
		a, b := child.CompletedThisWeek[i].CompletedAt, child.CompletedThisWeek[j].CompletedAt
		return a != nil && b != nil && *a > *b
	})
	secs, err := stprog.TimeSpentSeconds(ctx, d.Pool, ln.StudentUserID, weekStart, weekEnd)
	if err != nil {
		slog.Warn("parent_home.time_spent_failed", "err", err)
	}
	child.TimeSpentSeconds = secs
	return child, nil
}

// handleParentHome is GET /api/v1/parent/home
func (d Deps) handleParentHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		links, err := parentlinks.ListChildrenForParent(ctx, d.Pool, parentID, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list linked students.")
			return
		}
		now := time.Now().UTC()
		loc := parentHomeLocation(ctx, d, parentID, r.URL.Query().Get("tz"))
		weekStart, weekEnd := parentHomeWeek(now, loc)
		out := make([]parentHomeChild, 0, len(links))
		for _, ln := range links {
			child, err := d.buildParentHomeChild(ctx, ln, now, weekStart, weekEnd)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load parent home.")
				return
			}
			out = append(out, child)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"children":  out,
			"weekStart": weekStart.Format(time.RFC3339),
			"weekEnd":   weekEnd.Format(time.RFC3339),
			"timezone":  loc.String(),
		})
	}
}
