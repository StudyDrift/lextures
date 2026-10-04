package grading

import (
	"bytes"
	"encoding/csv"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/coursegradebook"
)

const (
	// CSVFormatVersion is the metadata-row version written by Export and accepted by Validate.
	CSVFormatVersion = "1"
	csvMetaMarker    = "__lextures__"
	// ImportSessionTTL is how long a validated import may be confirmed.
	ImportSessionTTL = 30 * time.Minute
)

// CSVStudent is one roster row in a gradebook export.
type CSVStudent struct {
	ID    uuid.UUID
	Name  string
	Email string
}

// CSVColumn is one gradable item. ManualHoldBlind is set when a change to the item
// needs the blind manual-hold acknowledgement (plan 3.11 FR-6).
type CSVColumn struct {
	ID              uuid.UUID
	Title           string
	MaxPoints       *int
	ManualHoldBlind bool
}

// CSVBook is the gradebook slice rendered to CSV.
type CSVBook struct {
	Students   []CSVStudent
	Columns    []CSVColumn
	Grades     map[uuid.UUID]map[uuid.UUID]string
	Display    map[uuid.UUID]map[uuid.UUID]string
	Excused    map[uuid.UUID]map[uuid.UUID]bool
	FinalScore map[uuid.UUID]string
	FinalGrade map[uuid.UUID]string
}

// FormulaSafe prefixes cells that spreadsheets would treat as formulas.
func FormulaSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@':
		return "'" + s
	default:
		return s
	}
}

// BuildGradebookCSV writes a UTF-8 BOM CSV: metadata row, header, then one row per student.
// Columns are student_id, student_name, student_email, then score + display per assignment, then final_score, final_grade.
func BuildGradebookCSV(book CSVBook) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)

	nCols := 3 + len(book.Columns)*2 + 2
	meta := make([]string, nCols)
	header := make([]string, nCols)
	meta[0] = csvMetaMarker
	meta[1] = CSVFormatVersion
	header[0] = "student_id"
	header[1] = "student_name"
	header[2] = "student_email"
	for i, col := range book.Columns {
		base := 3 + i*2
		meta[base] = col.ID.String()
		header[base] = col.Title
		header[base+1] = col.Title + " display"
	}
	header[nCols-2] = "final_score"
	header[nCols-1] = "final_grade"
	if err := w.Write(meta); err != nil {
		return nil, err
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, st := range book.Students {
		row := make([]string, nCols)
		row[0] = st.ID.String()
		row[1] = FormulaSafe(st.Name)
		row[2] = FormulaSafe(st.Email)
		grades := book.Grades[st.ID]
		display := book.Display[st.ID]
		for i, col := range book.Columns {
			base := 3 + i*2
			score := ""
			if grades != nil {
				score = strings.TrimSpace(grades[col.ID])
			}
			disp := ""
			if book.Excused[st.ID][col.ID] {
				disp = "EX"
			} else if display != nil {
				disp = display[col.ID]
			}
			row[base] = FormulaSafe(score)
			row[base+1] = FormulaSafe(disp)
		}
		row[nCols-2] = FormulaSafe(book.FinalScore[st.ID])
		row[nCols-1] = FormulaSafe(book.FinalGrade[st.ID])
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StraightFinals fills final_score (points earned) and final_grade (percent) from entered scores.
// Excused cells and columns without a positive max are skipped. Empty when the student has no scored cells.
func StraightFinals(book CSVBook) (score, grade map[uuid.UUID]string) {
	score = make(map[uuid.UUID]string, len(book.Students))
	grade = make(map[uuid.UUID]string, len(book.Students))
	for _, st := range book.Students {
		var earned, max float64
		scored := false
		grades := book.Grades[st.ID]
		excused := book.Excused[st.ID]
		for _, col := range book.Columns {
			if col.MaxPoints == nil || *col.MaxPoints <= 0 {
				continue
			}
			if excused[col.ID] {
				continue
			}
			raw := ""
			if grades != nil {
				raw = grades[col.ID]
			}
			pts, ok := parseScore(raw)
			if !ok {
				continue
			}
			earned += pts
			max += float64(*col.MaxPoints)
			scored = true
		}
		if !scored || max <= 0 {
			continue
		}
		score[st.ID] = formatScore(earned)
		pct := earned / max * 100
		grade[st.ID] = formatScore(pct) + "%"
	}
	return score, grade
}

// ValidateResult is the preview plus the cells a confirm would write.
type ValidateResult struct {
	Preview coursegradebook.GradebookImportValidateResponse
	// Pending is student-id → item-id → new score (empty string clears). Only changed cells.
	Pending    map[string]map[string]string
	RequireAck bool
}

// ValidateGradebookCSV parses an export (or the same shape) and diffs it against book.
// Unknown students and non-numeric scores are errors and make the import not confirmable.
// Scores above max points are warnings and remain confirmable.
func ValidateGradebookCSV(csvText string, book CSVBook) (ValidateResult, error) {
	csvText = strings.TrimPrefix(csvText, "\uFEFF")
	r := csv.NewReader(strings.NewReader(csvText))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return ValidateResult{}, err
	}
	res := ValidateResult{
		Pending: map[string]map[string]string{},
		Preview: coursegradebook.GradebookImportValidateResponse{
			Rows: []coursegradebook.GradebookImportPreviewRow{},
		},
	}
	if len(records) == 0 {
		return res, errEmptyCSV
	}
	start := 0
	var meta []string
	if len(records[0]) > 0 && strings.TrimPrefix(strings.TrimSpace(records[0][0]), "\uFEFF") == csvMetaMarker {
		meta = records[0]
		start = 1
	}
	if start >= len(records) {
		return res, errMissingHeader
	}
	header := records[start]
	data := records[start+1:]

	studentIdx := headerIndex(header, "student_id")
	emailIdx := headerIndex(header, "student_email")
	if studentIdx < 0 && emailIdx < 0 {
		return res, errMissingStudent
	}

	colByID := make(map[uuid.UUID]CSVColumn, len(book.Columns))
	colByTitle := make(map[string]CSVColumn, len(book.Columns))
	for _, c := range book.Columns {
		colByID[c.ID] = c
		colByTitle[strings.ToLower(strings.TrimSpace(c.Title))] = c
	}
	type mapped struct {
		idx int
		col CSVColumn
	}
	var mappedCols []mapped
	seen := map[uuid.UUID]bool{}
	if meta != nil {
		for i := 0; i < len(meta) && i < len(header); i++ {
			id, err := uuid.Parse(strings.TrimSpace(meta[i]))
			if err != nil {
				continue
			}
			col, ok := colByID[id]
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			mappedCols = append(mappedCols, mapped{idx: i, col: col})
		}
	}
	if len(mappedCols) == 0 {
		for i, h := range header {
			name := strings.ToLower(strings.TrimSpace(h))
			if name == "" || strings.HasSuffix(name, " display") {
				continue
			}
			switch name {
			case "student_id", "student_name", "student_email", "final_score", "final_grade", csvMetaMarker:
				continue
			}
			col, ok := colByTitle[name]
			if !ok || seen[col.ID] {
				continue
			}
			seen[col.ID] = true
			mappedCols = append(mappedCols, mapped{idx: i, col: col})
		}
	}
	if len(mappedCols) == 0 {
		return res, errNoColumns
	}

	studentsByID := make(map[uuid.UUID]CSVStudent, len(book.Students))
	studentsByEmail := make(map[string]CSVStudent, len(book.Students))
	for _, st := range book.Students {
		studentsByID[st.ID] = st
		if e := strings.ToLower(strings.TrimSpace(st.Email)); e != "" {
			studentsByEmail[e] = st
		}
	}

	for i, rec := range data {
		if allBlank(rec) {
			continue
		}
		row := coursegradebook.GradebookImportPreviewRow{
			RowIndex: uint(i),
			Cells:    []coursegradebook.GradebookImportCellPreview{},
		}
		st, ok := resolveStudent(rec, studentIdx, emailIdx, studentsByID, studentsByEmail)
		if !ok {
			msg := "Unknown student"
			row.Error = &msg
			res.Preview.Stats.Errors++
			res.Preview.Rows = append(res.Preview.Rows, row)
			continue
		}
		sid := st.ID
		row.StudentID = &sid
		name := st.Name
		row.StudentName = &name
		grades := book.Grades[st.ID]
		for _, mc := range mappedCols {
			raw := ""
			if mc.idx < len(rec) {
				raw = strings.TrimSpace(rec[mc.idx])
			}
			prev := ""
			if grades != nil {
				prev = strings.TrimSpace(grades[mc.col.ID])
			}
			cell := coursegradebook.GradebookImportCellPreview{
				ItemID:   mc.col.ID,
				NewScore: raw,
				State:    "unchanged",
			}
			if prev != "" {
				p := prev
				cell.PreviousScore = &p
			}
			if raw == "" && prev == "" {
				continue
			}
			if raw != "" {
				pts, parsed := parseScore(raw)
				if !parsed {
					cell.State = "error"
					res.Preview.Stats.Errors++
					row.Cells = append(row.Cells, cell)
					continue
				}
				if mc.col.MaxPoints != nil && *mc.col.MaxPoints > 0 && pts > float64(*mc.col.MaxPoints) {
					cell.OutOfRange = true
					res.Preview.Stats.Warnings++
				}
			}
			if scoresEqual(prev, raw) {
				res.Preview.Stats.Unchanged++
				row.Cells = append(row.Cells, cell)
				continue
			}
			if prev == "" {
				cell.State = "added"
				res.Preview.Stats.Added++
			} else if raw == "" {
				cell.State = "deleted"
				res.Preview.Stats.Updated++
			} else {
				cell.State = "updated"
				res.Preview.Stats.Updated++
			}
			if mc.col.ManualHoldBlind {
				res.RequireAck = true
			}
			if res.Pending[st.ID.String()] == nil {
				res.Pending[st.ID.String()] = map[string]string{}
			}
			res.Pending[st.ID.String()][mc.col.ID.String()] = raw
			row.Cells = append(row.Cells, cell)
		}
		if len(row.Cells) > 0 {
			res.Preview.Rows = append(res.Preview.Rows, row)
		}
	}
	res.Preview.RequireBlindManualHoldAck = res.RequireAck
	changed := res.Preview.Stats.Added+res.Preview.Stats.Updated > 0
	res.Preview.Confirmable = res.Preview.Stats.Errors == 0 && changed
	return res, nil
}

var (
	errEmptyCSV       = errString("CSV is empty")
	errMissingHeader  = errString("CSV is missing a header row")
	errMissingStudent = errString("CSV must include student_id or student_email")
	errNoColumns      = errString("CSV does not include any assignment columns from this course")
)

type errString string

func (e errString) Error() string { return string(e) }

func headerIndex(header []string, name string) int {
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF")), name) {
			return i
		}
	}
	return -1
}

func resolveStudent(rec []string, studentIdx, emailIdx int, byID map[uuid.UUID]CSVStudent, byEmail map[string]CSVStudent) (CSVStudent, bool) {
	if studentIdx >= 0 && studentIdx < len(rec) {
		raw := strings.TrimSpace(rec[studentIdx])
		if raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				return CSVStudent{}, false
			}
			st, ok := byID[id]
			return st, ok
		}
	}
	if emailIdx >= 0 && emailIdx < len(rec) {
		email := strings.ToLower(strings.TrimSpace(rec[emailIdx]))
		if email != "" {
			st, ok := byEmail[email]
			return st, ok
		}
	}
	return CSVStudent{}, false
}

func allBlank(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseScore(raw string) (float64, bool) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, false
	}
	// A leading apostrophe is the formula-injection prefix, not a numeric score.
	if strings.HasPrefix(t, "'") {
		return 0, false
	}
	t = strings.ReplaceAll(t, ",", "")
	f, err := strconv.ParseFloat(t, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > 1e9 {
		return 0, false
	}
	return f, true
}

func scoresEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return true
	}
	fa, oka := parseScore(a)
	fb, okb := parseScore(b)
	if oka && okb {
		return math.Abs(fa-fb) < 1e-6
	}
	return false
}

func formatScore(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	for strings.Contains(s, ".") && (strings.HasSuffix(s, "0") || strings.HasSuffix(s, ".")) {
		s = s[:len(s)-1]
	}
	return s
}
