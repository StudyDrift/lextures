package homeschoolrecords

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
)

// LogRow is one line on the daily hours log (auto and/or manual).
type LogRow struct {
	ID        *string   `json:"id,omitempty"` // manual entry id when present
	Date      string    `json:"date"`
	Subject   string    `json:"subject"`
	CourseID  *string   `json:"courseId,omitempty"`
	Hours     float64   `json:"hours"`
	Notes     string    `json:"notes,omitempty"`
	Source    string    `json:"source"` // activity | manual | offline
	Editable  bool      `json:"editable"`
}

// SubjectTotal is hours summed by subject.
type SubjectTotal struct {
	Subject string  `json:"subject"`
	Hours   float64 `json:"hours"`
}

// HoursLog is the merged daily log plus totals.
type HoursLog struct {
	From          string         `json:"from"`
	To            string         `json:"to"`
	SchoolYear    string         `json:"schoolYear,omitempty"`
	Entries       []LogRow       `json:"entries"`
	TotalsBySubject []SubjectTotal `json:"totalsBySubject"`
	TotalHours    float64        `json:"totalHours"`
}

// BuildHoursLog merges auto activity hours with parent-entered entries.
func BuildHoursLog(from, to time.Time, schoolYear string, auto []homeschoolrecords.AutoHoursRow, manual []homeschoolrecords.HoursEntry) HoursLog {
	entries := make([]LogRow, 0, len(auto)+len(manual))
	for _, a := range auto {
		hours := roundHours(a.Hours)
		if hours <= 0 {
			continue
		}
		row := LogRow{
			Date:     a.EntryDate.UTC().Format("2006-01-02"),
			Subject:  strings.TrimSpace(a.Subject),
			Hours:    hours,
			Source:   "activity",
			Editable: false,
		}
		if a.CourseID != nil {
			s := a.CourseID.String()
			row.CourseID = &s
		}
		if row.Subject == "" {
			row.Subject = "General"
		}
		entries = append(entries, row)
	}
	for _, m := range manual {
		id := m.ID.String()
		row := LogRow{
			ID:       &id,
			Date:     m.EntryDate.UTC().Format("2006-01-02"),
			Subject:  m.Subject,
			Hours:    roundHours(m.Hours),
			Notes:    m.Notes,
			Source:   m.Source,
			Editable: true,
		}
		if m.CourseID != nil {
			s := m.CourseID.String()
			row.CourseID = &s
		}
		if row.Subject == "" {
			row.Subject = "General"
		}
		entries = append(entries, row)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Date != entries[j].Date {
			return entries[i].Date > entries[j].Date
		}
		if entries[i].Subject != entries[j].Subject {
			return entries[i].Subject < entries[j].Subject
		}
		return entries[i].Source < entries[j].Source
	})

	totalsMap := map[string]float64{}
	var total float64
	for _, e := range entries {
		totalsMap[e.Subject] += e.Hours
		total += e.Hours
	}
	subjects := make([]string, 0, len(totalsMap))
	for s := range totalsMap {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects)
	totals := make([]SubjectTotal, 0, len(subjects))
	for _, s := range subjects {
		totals = append(totals, SubjectTotal{Subject: s, Hours: roundHours(totalsMap[s])})
	}

	return HoursLog{
		From:            from.UTC().Format("2006-01-02"),
		To:              to.UTC().Format("2006-01-02"),
		SchoolYear:      schoolYear,
		Entries:         entries,
		TotalsBySubject: totals,
		TotalHours:      roundHours(total),
	}
}

func roundHours(h float64) float64 {
	// hundredths of an hour
	return float64(int(h*100+0.5)) / 100
}

// ParseOptionalCourseID parses a course id string.
func ParseOptionalCourseID(s string) (*uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
