package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const (
	ferpaAttendanceExportWarning = `WARNING: Attendance export contains FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

	ferpaBehaviorExportWarning = `WARNING: Behavior export contains FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

	defaultAttendanceImportChunk = 50
)

type attendanceSession struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	SessionDate      string  `json:"sessionDate"`
	Status           string  `json:"status"`
	CollectionMethod string  `json:"collectionMethod"`
	SectionID        *string `json:"sectionId"`
}

type attendanceSessionsBody struct {
	Sessions []attendanceSession `json:"sessions"`
}

type attendanceSessionDetail struct {
	attendanceSession
	Records []attendanceRecord `json:"records"`
}

type attendanceRecord struct {
	StudentUserID string `json:"studentUserId"`
	DisplayName   string `json:"displayName"`
	Status        string `json:"status"`
	Source        string `json:"source"`
}

type attendanceImportRow struct {
	StudentRef string
	Date       string
	Period     string
	Status     string
	LineNumber int
}

type attendanceImportSummary struct {
	SessionsCreated int      `json:"sessionsCreated"`
	RecordsSaved    int      `json:"recordsSaved"`
	Updated         int      `json:"updated"`
	Failed          int      `json:"failed"`
	Errors          []string `json:"errors,omitempty"`
}

type attendanceExportRow struct {
	Date        string `json:"date"`
	Period      string `json:"period,omitempty"`
	SessionID   string `json:"sessionId"`
	StudentID   string `json:"studentId"`
	StudentName string `json:"studentName,omitempty"`
	Status      string `json:"status"`
}

type attendanceSummaryRow struct {
	StudentID   string `json:"studentId"`
	StudentName string `json:"studentName,omitempty"`
	Present     int    `json:"present"`
	Absent      int    `json:"absent"`
	Tardy       int    `json:"tardy"`
	Excused     int    `json:"excused"`
	Other       int    `json:"other"`
	Total       int    `json:"total"`
}

type behaviorCategory struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Active bool   `json:"active"`
}

type behaviorCategoriesBody struct {
	Categories []behaviorCategory `json:"categories"`
}

type behaviorAward struct {
	ID           string  `json:"id"`
	StudentID    string  `json:"studentId"`
	CategoryID   string  `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	Points       int     `json:"points"`
	Note         *string `json:"note"`
	AwardedAt    string  `json:"awardedAt"`
}

type behaviorReferral struct {
	ID           string  `json:"id"`
	StudentID    string  `json:"studentId"`
	CategoryID   string  `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	IncidentAt   string  `json:"incidentAt"`
	Description  string  `json:"description,omitempty"`
	Location     *string `json:"location"`
}

type studentBehaviorBody struct {
	StudentID   string             `json:"studentId"`
	TotalPoints int                `json:"totalPoints"`
	Awards      []behaviorAward    `json:"awards"`
	Referrals   []behaviorReferral `json:"referrals"`
}

type behaviorListRow struct {
	StudentID   string `json:"studentId"`
	DisplayName string `json:"displayName,omitempty"`
	TotalPoints int    `json:"totalPoints"`
	AwardCount  int    `json:"awardCount"`
	Referrals   int    `json:"referralCount"`
}

type behaviorExportRow struct {
	StudentID    string `json:"studentId"`
	DisplayName  string `json:"displayName,omitempty"`
	RecordType   string `json:"recordType"`
	CategoryName string `json:"categoryName"`
	Points       int    `json:"points,omitempty"`
	IncidentAt   string `json:"incidentAt,omitempty"`
	Description  string `json:"description,omitempty"`
}

type seatTimeReportBody struct {
	Students []seatTimeStudentRow `json:"students"`
}

type seatTimeStudentRow struct {
	UserID        string  `json:"userId"`
	DisplayName   string  `json:"displayName"`
	TotalMinutes  int     `json:"totalMinutes"`
	ContactHours  float64 `json:"contactHours"`
	CEUEarned     bool    `json:"ceuEarned"`
	RequiredHours float64 `json:"requiredHours"`
}

type hallPassJSON struct {
	ID            string  `json:"id"`
	StudentID     string  `json:"studentId"`
	SectionID     string  `json:"sectionId"`
	Destination   string  `json:"destination"`
	Status        string  `json:"status"`
	EstimatedMins *int    `json:"estimatedMins"`
	RequestedAt   string  `json:"requestedAt"`
	ApprovedAt    *string `json:"approvedAt"`
	ReturnedAt    *string `json:"returnedAt"`
}

func confirmAttendanceExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaAttendanceExportWarning)
}

func confirmBehaviorExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaBehaviorExportWarning)
}

func normalizeAttendanceStatus(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "present", "p":
		return "present", nil
	case "absent", "a":
		return "absent", nil
	case "tardy", "late", "t":
		return "tardy", nil
	case "excused", "e":
		return "excused", nil
	case "not_recorded", "unknown", "":
		return "not_recorded", nil
	default:
		return "", fmt.Errorf("invalid attendance status %q (use present, absent, tardy, or excused)", raw)
	}
}

func attendanceIdempotencyKey(studentRef, date, period string) string {
	return strings.ToLower(strings.TrimSpace(studentRef)) + "|" +
		strings.TrimSpace(date) + "|" +
		strings.TrimSpace(period)
}

func parseAttendanceCSV(raw []byte) ([]attendanceImportRow, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	header := make([]string, len(records[0]))
	for i, col := range records[0] {
		header[i] = strings.ToLower(strings.TrimSpace(col))
	}
	studentIdx, dateIdx, periodIdx, statusIdx, emailIdx := -1, -1, -1, -1, -1
	for i, col := range header {
		switch col {
		case "student", "student_id", "student id", "user_id", "userid":
			studentIdx = i
		case "email", "student_email", "student email":
			emailIdx = i
		case "date", "attendance_date", "attendance date", "session_date":
			dateIdx = i
		case "period", "block":
			periodIdx = i
		case "status", "attendance_status", "attendance status":
			statusIdx = i
		}
	}
	if studentIdx < 0 && emailIdx < 0 {
		return nil, fmt.Errorf("CSV must include a student or email column")
	}
	if dateIdx < 0 {
		return nil, fmt.Errorf("CSV must include a date column")
	}
	if statusIdx < 0 {
		return nil, fmt.Errorf("CSV must include a status column")
	}

	start := 0
	if studentIdx >= 0 || emailIdx >= 0 || dateIdx >= 0 || periodIdx >= 0 || statusIdx >= 0 {
		start = 1
	}

	var rows []attendanceImportRow
	for line := start; line < len(records); line++ {
		rec := records[line]
		if len(rec) == 0 || strings.TrimSpace(strings.Join(rec, "")) == "" {
			continue
		}
		student := ""
		if studentIdx >= 0 && studentIdx < len(rec) {
			student = strings.TrimSpace(rec[studentIdx])
		}
		if student == "" && emailIdx >= 0 && emailIdx < len(rec) {
			student = strings.TrimSpace(rec[emailIdx])
		}
		if student == "" {
			return nil, fmt.Errorf("line %d: missing student identifier", line+1)
		}
		date := ""
		if dateIdx < len(rec) {
			date = strings.TrimSpace(rec[dateIdx])
		}
		if date == "" {
			return nil, fmt.Errorf("line %d: missing date", line+1)
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, fmt.Errorf("line %d: invalid date %q (use YYYY-MM-DD)", line+1, date)
		}
		period := ""
		if periodIdx >= 0 && periodIdx < len(rec) {
			period = strings.TrimSpace(rec[periodIdx])
		}
		statusRaw := ""
		if statusIdx < len(rec) {
			statusRaw = strings.TrimSpace(rec[statusIdx])
		}
		status, err := normalizeAttendanceStatus(statusRaw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line+1, err)
		}
		rows = append(rows, attendanceImportRow{
			StudentRef: student,
			Date:       date,
			Period:     period,
			Status:     status,
			LineNumber: line + 1,
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV contains no data rows")
	}
	return rows, nil
}

func sessionTitleForDate(date, period string) string {
	title := "Attendance — " + date
	if strings.TrimSpace(period) != "" {
		title += " — " + strings.TrimSpace(period)
	}
	return title
}

func sessionMatchesDatePeriod(sess attendanceSession, date, period string) bool {
	if strings.TrimSpace(sess.SessionDate) != strings.TrimSpace(date) {
		return false
	}
	wantTitle := sessionTitleForDate(date, period)
	return strings.EqualFold(strings.TrimSpace(sess.Title), wantTitle)
}

func fetchAttendanceSessions(c *client.Client, course string) ([]attendanceSession, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/attendance/sessions"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing attendance sessions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed attendanceSessionsBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Sessions, nil
}

func fetchAttendanceSession(c *client.Client, course, sessionID string) (attendanceSessionDetail, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/attendance/sessions/" + url.PathEscape(sessionID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return attendanceSessionDetail{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return attendanceSessionDetail{}, fmt.Errorf("getting attendance session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return attendanceSessionDetail{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return attendanceSessionDetail{}, apiErrorBody(resp.StatusCode, body)
	}
	var parsed attendanceSessionDetail
	if err := json.Unmarshal(body, &parsed); err != nil {
		return attendanceSessionDetail{}, fmt.Errorf("decoding response: %w", err)
	}
	return parsed, nil
}

func createAttendanceSession(c *client.Client, course, date, period, sectionID string) (attendanceSession, error) {
	payload := map[string]any{
		"collectionMethod": "roll_call",
		"sessionDate":      date,
		"title":            sessionTitleForDate(date, period),
	}
	if strings.TrimSpace(sectionID) != "" {
		payload["sectionId"] = strings.TrimSpace(sectionID)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return attendanceSession{}, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/attendance/sessions"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return attendanceSession{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return attendanceSession{}, fmt.Errorf("creating attendance session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return attendanceSession{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return attendanceSession{}, apiErrorBody(resp.StatusCode, body)
	}
	var sess attendanceSession
	if err := json.Unmarshal(body, &sess); err != nil {
		return attendanceSession{}, fmt.Errorf("decoding response: %w", err)
	}
	return sess, nil
}

func findOrCreateSessionForDate(c *client.Client, course, date, period, sectionID string) (attendanceSession, bool, error) {
	sessions, err := fetchAttendanceSessions(c, course)
	if err != nil {
		return attendanceSession{}, false, err
	}
	for _, sess := range sessions {
		if sessionMatchesDatePeriod(sess, date, period) {
			return sess, false, nil
		}
	}
	sess, err := createAttendanceSession(c, course, date, period, sectionID)
	if err != nil {
		return attendanceSession{}, false, err
	}
	return sess, true, nil
}

func putAttendanceRecords(c *client.Client, course, sessionID string, records []map[string]string) (int, error) {
	payload := map[string]any{"records": records}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(course) + "/attendance/sessions/" + url.PathEscape(sessionID) + "/records"
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return 0, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return 0, fmt.Errorf("saving attendance records: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Saved int `json:"saved"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return len(records), nil
	}
	return out.Saved, nil
}

func resolveStudentUserID(c *client.Client, ref string) (string, error) {
	id, _, err := resolveUserID(c, ref)
	if err != nil {
		return "", err
	}
	return id, nil
}

func importAttendanceRows(c *client.Client, course string, rows []attendanceImportRow, sectionID string, chunkSize int, progress func(done, total int)) (attendanceImportSummary, error) {
	summary := attendanceImportSummary{}
	if chunkSize < 1 {
		chunkSize = defaultAttendanceImportChunk
	}

	type groupKey struct {
		date   string
		period string
	}
	grouped := map[groupKey][]attendanceImportRow{}
	seen := map[string]struct{}{}
	for _, row := range rows {
		key := attendanceIdempotencyKey(row.StudentRef, row.Date, row.Period)
		if _, ok := seen[key]; ok {
			summary.Updated++
		}
		seen[key] = struct{}{}
		gk := groupKey{date: row.Date, period: row.Period}
		grouped[gk] = append(grouped[gk], row)
	}

	total := len(rows)
	done := 0
	for gk, groupRows := range grouped {
		sess, created, err := findOrCreateSessionForDate(c, course, gk.date, gk.period, sectionID)
		if err != nil {
			summary.Failed += len(groupRows)
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", gk.date, err))
			done += len(groupRows)
			if progress != nil {
				progress(done, total)
			}
			continue
		}
		if created {
			summary.SessionsCreated++
		}

		for i := 0; i < len(groupRows); i += chunkSize {
			end := i + chunkSize
			if end > len(groupRows) {
				end = len(groupRows)
			}
			chunk := groupRows[i:end]
			recs := make([]map[string]string, 0, len(chunk))
			for _, row := range chunk {
				studentID, err := resolveStudentUserID(c, row.StudentRef)
				if err != nil {
					summary.Failed++
					summary.Errors = append(summary.Errors, fmt.Sprintf("line %d: %v", row.LineNumber, err))
					continue
				}
				recs = append(recs, map[string]string{
					"studentUserId": studentID,
					"status":        row.Status,
					"source":        "import",
				})
			}
			if len(recs) == 0 {
				done += len(chunk)
				if progress != nil {
					progress(done, total)
				}
				continue
			}
			saved, err := putAttendanceRecords(c, course, sess.ID, recs)
			if err != nil {
				summary.Failed += len(recs)
				summary.Errors = append(summary.Errors, err.Error())
			} else {
				summary.RecordsSaved += saved
			}
			done += len(chunk)
			if progress != nil {
				progress(done, total)
			}
		}
	}
	return summary, nil
}

func filterSessionsByDateRange(sessions []attendanceSession, from, to string) []attendanceSession {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" && to == "" {
		return sessions
	}
	out := make([]attendanceSession, 0, len(sessions))
	for _, sess := range sessions {
		d := strings.TrimSpace(sess.SessionDate)
		if from != "" && d < from {
			continue
		}
		if to != "" && d > to {
			continue
		}
		out = append(out, sess)
	}
	return out
}

func periodFromSessionTitle(title, date string) string {
	prefix := sessionTitleForDate(date, "")
	title = strings.TrimSpace(title)
	if strings.EqualFold(title, prefix) {
		return ""
	}
	marker := " — "
	if idx := strings.LastIndex(title, marker); idx >= 0 {
		return strings.TrimSpace(title[idx+len(marker):])
	}
	return ""
}

func collectAttendanceExportRows(c *client.Client, course string, sessions []attendanceSession) ([]attendanceExportRow, error) {
	rows := make([]attendanceExportRow, 0)
	for _, sess := range sessions {
		detail, err := fetchAttendanceSession(c, course, sess.ID)
		if err != nil {
			return nil, err
		}
		period := periodFromSessionTitle(sess.Title, sess.SessionDate)
		for _, rec := range detail.Records {
			if rec.Status == "not_recorded" {
				continue
			}
			rows = append(rows, attendanceExportRow{
				Date:        sess.SessionDate,
				Period:      period,
				SessionID:   sess.ID,
				StudentID:   rec.StudentUserID,
				StudentName: rec.DisplayName,
				Status:      rec.Status,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date != rows[j].Date {
			return rows[i].Date < rows[j].Date
		}
		if rows[i].Period != rows[j].Period {
			return rows[i].Period < rows[j].Period
		}
		return rows[i].StudentID < rows[j].StudentID
	})
	return rows, nil
}

func buildAttendanceSummary(rows []attendanceExportRow) []attendanceSummaryRow {
	byStudent := map[string]*attendanceSummaryRow{}
	for _, row := range rows {
		s, ok := byStudent[row.StudentID]
		if !ok {
			s = &attendanceSummaryRow{StudentID: row.StudentID, StudentName: row.StudentName}
			byStudent[row.StudentID] = s
		}
		if s.StudentName == "" {
			s.StudentName = row.StudentName
		}
		switch row.Status {
		case "present":
			s.Present++
		case "absent":
			s.Absent++
		case "tardy":
			s.Tardy++
		case "excused":
			s.Excused++
		default:
			s.Other++
		}
		s.Total++
	}
	out := make([]attendanceSummaryRow, 0, len(byStudent))
	for _, s := range byStudent {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StudentID < out[j].StudentID })
	return out
}

func writeAttendanceExportCSV(w io.Writer, rows []attendanceExportRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"date", "period", "session_id", "student_id", "student_name", "status"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := cw.Write([]string{row.Date, row.Period, row.SessionID, row.StudentID, row.StudentName, row.Status}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func fetchCourseOrgID(c *client.Client, course string) (string, error) {
	co, _, err := fetchCourseDetail(c, course)
	if err != nil {
		return "", err
	}
	if co.OrgID == nil || strings.TrimSpace(*co.OrgID) == "" {
		return "", fmt.Errorf("course %s has no org id", course)
	}
	return strings.TrimSpace(*co.OrgID), nil
}

func fetchBehaviorCategories(c *client.Client, orgID string) ([]behaviorCategory, error) {
	path := "/api/v1/admin/orgs/" + url.PathEscape(orgID) + "/behavior/categories"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing behavior categories: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed behaviorCategoriesBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Categories, nil
}

func resolveBehaviorCategory(categories []behaviorCategory, ref string) (behaviorCategory, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		for _, cat := range categories {
			if cat.Active && cat.Type == "positive" {
				return cat, nil
			}
		}
		for _, cat := range categories {
			if cat.Active {
				return cat, nil
			}
		}
		return behaviorCategory{}, fmt.Errorf("no active behavior categories found; create categories in admin or pass --category")
	}
	for _, cat := range categories {
		if strings.EqualFold(cat.ID, ref) || strings.EqualFold(cat.Name, ref) {
			return cat, nil
		}
	}
	return behaviorCategory{}, fmt.Errorf("behavior category %q not found", ref)
}

func fetchStudentBehavior(c *client.Client, studentID string) (studentBehaviorBody, error) {
	path := "/api/v1/students/" + url.PathEscape(studentID) + "/behavior"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return studentBehaviorBody{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return studentBehaviorBody{}, fmt.Errorf("loading behavior: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return studentBehaviorBody{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return studentBehaviorBody{}, apiErrorBody(resp.StatusCode, body)
	}
	var parsed studentBehaviorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return studentBehaviorBody{}, fmt.Errorf("decoding response: %w", err)
	}
	return parsed, nil
}

func listCourseBehavior(c *client.Client, course string) ([]behaviorListRow, error) {
	enrollments, err := fetchEnrollments(c, course)
	if err != nil {
		return nil, err
	}
	students := filterEnrollments(enrollments, "student", "", "active")
	rows := make([]behaviorListRow, 0, len(students))
	for _, en := range students {
		behavior, err := fetchStudentBehavior(c, en.UserID)
		if err != nil {
			return nil, fmt.Errorf("student %s: %w", en.UserID, err)
		}
		name := ""
		if en.DisplayName != nil {
			name = *en.DisplayName
		}
		rows = append(rows, behaviorListRow{
			StudentID:   en.UserID,
			DisplayName: name,
			TotalPoints: behavior.TotalPoints,
			AwardCount:  len(behavior.Awards),
			Referrals:   len(behavior.Referrals),
		})
	}
	return rows, nil
}

func exportCourseBehavior(c *client.Client, course string) ([]behaviorExportRow, error) {
	enrollments, err := fetchEnrollments(c, course)
	if err != nil {
		return nil, err
	}
	students := filterEnrollments(enrollments, "student", "", "active")
	rows := make([]behaviorExportRow, 0)
	for _, en := range students {
		behavior, err := fetchStudentBehavior(c, en.UserID)
		if err != nil {
			return nil, fmt.Errorf("student %s: %w", en.UserID, err)
		}
		name := ""
		if en.DisplayName != nil {
			name = *en.DisplayName
		}
		for _, award := range behavior.Awards {
			rows = append(rows, behaviorExportRow{
				StudentID:    en.UserID,
				DisplayName:  name,
				RecordType:   "award",
				CategoryName: award.CategoryName,
				Points:       award.Points,
			})
		}
		for _, ref := range behavior.Referrals {
			rows = append(rows, behaviorExportRow{
				StudentID:    en.UserID,
				DisplayName:  name,
				RecordType:   "referral",
				CategoryName: ref.CategoryName,
				IncidentAt:   ref.IncidentAt,
				Description:  ref.Description,
			})
		}
	}
	return rows, nil
}

func writeBehaviorExportCSV(w io.Writer, rows []behaviorExportRow) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"student_id", "display_name", "record_type", "category", "points", "incident_at", "description"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := cw.Write([]string{
			row.StudentID,
			row.DisplayName,
			row.RecordType,
			row.CategoryName,
			fmt.Sprintf("%d", row.Points),
			row.IncidentAt,
			row.Description,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func awardPBISPoints(c *client.Client, studentID, categoryID string, points int, note string) (int, error) {
	award := map[string]any{
		"studentId":  studentID,
		"categoryId": categoryID,
		"points":     points,
	}
	if strings.TrimSpace(note) != "" {
		award["note"] = strings.TrimSpace(note)
	}
	payload := map[string]any{"awards": []map[string]any{award}}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encoding request: %w", err)
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/pbis/awards", bytes.NewReader(raw))
	if err != nil {
		return 0, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return 0, fmt.Errorf("awarding points: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Saved int `json:"saved"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 1, nil
	}
	return out.Saved, nil
}

func fetchSeatTimeReport(c *client.Client, course string) ([]seatTimeStudentRow, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/seat-time-report"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("loading seat-time report: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed seatTimeReportBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Students, nil
}

func fetchMySeatTime(c *client.Client, courseID string) (map[string]any, error) {
	path := "/api/v1/me/seat-time?courseId=" + url.QueryEscape(courseID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("loading seat time: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed, nil
}

func listActiveHallPasses(c *client.Client, sectionID string) ([]hallPassJSON, error) {
	path := "/api/v1/sections/" + url.PathEscape(sectionID) + "/hall-passes/active"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing hall passes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var parsed struct {
		Passes []hallPassJSON `json:"passes"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Passes, nil
}

func issueHallPass(c *client.Client, sectionID, destination string, estimatedMins *int) (hallPassJSON, error) {
	payload := map[string]any{"destination": destination}
	if estimatedMins != nil {
		payload["estimatedMins"] = *estimatedMins
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/sections/" + url.PathEscape(sectionID) + "/hall-passes"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("requesting hall pass: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return hallPassJSON{}, apiErrorBody(resp.StatusCode, body)
	}
	var parsed struct {
		Pass hallPassJSON `json:"pass"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return hallPassJSON{}, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Pass, nil
}

func updateHallPassStatus(c *client.Client, passID, status string) (hallPassJSON, error) {
	payload := map[string]any{"status": status}
	raw, err := json.Marshal(payload)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("encoding request: %w", err)
	}
	path := "/api/v1/hall-passes/" + url.PathEscape(passID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("updating hall pass: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return hallPassJSON{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return hallPassJSON{}, apiErrorBody(resp.StatusCode, body)
	}
	var parsed struct {
		Pass hallPassJSON `json:"pass"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return hallPassJSON{}, fmt.Errorf("decoding response: %w", err)
	}
	return parsed.Pass, nil
}

func resolveSectionForCourse(c *client.Client, course, sectionRef string) (sectionRow, error) {
	sections, err := fetchSections(c, course)
	if err != nil {
		return sectionRow{}, err
	}
	return resolveSectionRef(sections, sectionRef)
}