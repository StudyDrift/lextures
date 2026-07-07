package httpserver

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/attendance"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

type parentGradeItemOut struct {
	ItemID      string   `json:"itemId"`
	Title       string   `json:"title"`
	Category    *string  `json:"category,omitempty"`
	Score       string   `json:"score"`
	Percentage  *float64 `json:"percentage,omitempty"`
	Status      string   `json:"status"`
	DueAt       *string  `json:"dueAt,omitempty"`
	PostedAt    *string  `json:"postedAt,omitempty"`
}

type parentCourseGradesOut struct {
	CourseCode   string               `json:"courseCode"`
	Title        string               `json:"title"`
	TeacherEmail *string              `json:"teacherEmail,omitempty"`
	TeacherName  *string              `json:"teacherName,omitempty"`
	Grades       map[string]string    `json:"grades"`
	Items        []parentGradeItemOut `json:"items"`
}

type parentAttendanceDayOut struct {
	Date      string  `json:"date"`
	Code      string  `json:"code"`
	CodeLabel string  `json:"codeLabel"`
	Category  string  `json:"category"`
	Period    *string `json:"period,omitempty"`
}

type parentAttendanceSummaryOut struct {
	TermStart  string                   `json:"termStart"`
	Present    int                      `json:"present"`
	Absent     int                      `json:"absent"`
	Tardy      int                      `json:"tardy"`
	RecentDays []parentAttendanceDayOut `json:"recentDays"`
}

func parentCourseTitle(c course.CoursePublic) string {
	if strings.TrimSpace(c.Title) != "" {
		return c.Title
	}
	return c.CourseCode
}

func parentPrimaryCourseTeacher(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (email, name *string) {
	row := pool.QueryRow(ctx, `
SELECT u.email, u.display_name
FROM course.course_sections cs
JOIN "user".users u ON u.id = cs.instructor_user_id
WHERE cs.course_id = $1 AND cs.instructor_user_id IS NOT NULL
ORDER BY cs.created_at ASC NULLS LAST
LIMIT 1
`, courseID)
	var em string
	var dn *string
	if err := row.Scan(&em, &dn); err != nil {
		return nil, nil
	}
	em = strings.TrimSpace(em)
	if em != "" {
		email = &em
	}
	if dn != nil && strings.TrimSpace(*dn) != "" {
		n := strings.TrimSpace(*dn)
		name = &n
	}
	return email, name
}

func parentGradeStatus(excused bool, posted *time.Time) string {
	if excused {
		return "excused"
	}
	if posted != nil {
		return "posted"
	}
	return "graded"
}

func parentScorePercentage(score string, pointsPossible *int) *float64 {
	if pointsPossible == nil || *pointsPossible <= 0 {
		return nil
	}
	pts, err := strconv.ParseFloat(strings.TrimSpace(score), 64)
	if err != nil || math.IsNaN(pts) || math.IsInf(pts, 0) {
		return nil
	}
	pct := pts / float64(*pointsPossible) * 100
	if pct < 0 {
		return nil
	}
	rounded := math.Round(pct*10) / 10
	return &rounded
}

func parentGradeItemTitle(meta *coursestructure.ItemResponse) string {
	if meta != nil && strings.TrimSpace(meta.Title) != "" {
		return strings.TrimSpace(meta.Title)
	}
	return "Graded assignment"
}

func (d Deps) buildParentCourseGrades(ctx context.Context, studentID uuid.UUID, c course.CoursePublic) (*parentCourseGradesOut, error) {
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, c.CourseCode)
	if err != nil || cid == nil {
		return nil, err
	}
	gmap, _, postedAtMap, excusedMap, err := coursegrades.ListForCourse(ctx, d.Pool, *cid)
	if err != nil {
		return nil, err
	}
	sid := studentID.String()
	row := gmap[sid]
	if row == nil {
		row = map[string]string{}
	}
	structItems, err := coursestructure.ListForCourseWithEnrichment(ctx, d.Pool, *cid, false)
	if err != nil {
		return nil, err
	}
	itemMeta := make(map[string]coursestructure.ItemResponse, len(structItems))
	for _, it := range structItems {
		itemMeta[it.ID] = it
	}
	groupNames := map[string]string{}
	if groups, err := coursegrading.ListAssignmentGroups(ctx, d.Pool, *cid); err == nil {
		for _, g := range groups {
			groupNames[g.ID.String()] = g.Name
		}
	}
	teacherEmail, teacherName := parentPrimaryCourseTeacher(ctx, d.Pool, *cid)
	items := make([]parentGradeItemOut, 0, len(row))
	for itemID, score := range row {
		if strings.TrimSpace(score) == "" {
			continue
		}
		meta := itemMeta[itemID]
		var metaPtr *coursestructure.ItemResponse
		if meta.ID != "" {
			metaPtr = &meta
		}
		var category *string
		if meta.AssignmentGroupID != nil {
			if name, ok := groupNames[*meta.AssignmentGroupID]; ok && strings.TrimSpace(name) != "" {
				n := strings.TrimSpace(name)
				category = &n
			}
		}
		var dueAt *string
		if meta.DueAt != nil {
			s := meta.DueAt.UTC().Format(time.RFC3339Nano)
			dueAt = &s
		}
		excused := false
		if excusedMap[sid] != nil {
			excused = excusedMap[sid][itemID]
		}
		var posted *time.Time
		if postedAtMap[sid] != nil {
			posted = postedAtMap[sid][itemID]
		}
		var postedOut *string
		if posted != nil {
			s := posted.UTC().Format(time.RFC3339Nano)
			postedOut = &s
		}
		items = append(items, parentGradeItemOut{
			ItemID:     itemID,
			Title:      parentGradeItemTitle(metaPtr),
			Category:   category,
			Score:      score,
			Percentage: parentScorePercentage(score, meta.PointsPossible),
			Status:     parentGradeStatus(excused, posted),
			DueAt:      dueAt,
			PostedAt:   postedOut,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Title != items[j].Title {
			return items[i].Title < items[j].Title
		}
		return items[i].ItemID < items[j].ItemID
	})
	return &parentCourseGradesOut{
		CourseCode:   c.CourseCode,
		Title:        parentCourseTitle(c),
		TeacherEmail: teacherEmail,
		TeacherName:  teacherName,
		Grades:       row,
		Items:        items,
	}, nil
}

func parentAttendanceCategory(rec attendance.Record) string {
	if strings.TrimSpace(rec.Category) != "" {
		return strings.TrimSpace(rec.Category)
	}
	if strings.TrimSpace(rec.Code) != "" {
		return strings.TrimSpace(rec.Code)
	}
	return ""
}

func parentAttendanceCounts(records []attendance.Record) (present, absent, tardy int) {
	for _, rec := range records {
		cat := strings.ToLower(parentAttendanceCategory(rec))
		switch {
		case strings.Contains(cat, "absent") || cat == "a":
			absent++
		case strings.Contains(cat, "tardy") || cat == "t":
			tardy++
		default:
			present++
		}
	}
	return present, absent, tardy
}

func parentAttendanceSummary(records []attendance.Record, recentLimit int) parentAttendanceSummaryOut {
	if recentLimit <= 0 {
		recentLimit = 14
	}
	termStart := time.Now().UTC().AddDate(0, -3, 0).Format("2006-01-02")
	filtered := make([]attendance.Record, 0, len(records))
	for _, rec := range records {
		if rec.Date.Format("2006-01-02") >= termStart {
			filtered = append(filtered, rec)
		}
	}
	present, absent, tardy := parentAttendanceCounts(filtered)
	recent := make([]parentAttendanceDayOut, 0, recentLimit)
	for i, rec := range records {
		if i >= recentLimit {
			break
		}
		code := strings.TrimSpace(rec.Code)
		label := code
		if strings.TrimSpace(rec.CodeLabel) != "" {
			label = strings.TrimSpace(rec.CodeLabel)
		}
		day := parentAttendanceDayOut{
			Date:      rec.Date.Format("2006-01-02"),
			Code:      code,
			CodeLabel: label,
			Category:  parentAttendanceCategory(rec),
		}
		if rec.Period != nil {
			day.Period = rec.Period
		}
		recent = append(recent, day)
	}
	return parentAttendanceSummaryOut{
		TermStart:  termStart,
		Present:    present,
		Absent:     absent,
		Tardy:      tardy,
		RecentDays: recent,
	}
}
