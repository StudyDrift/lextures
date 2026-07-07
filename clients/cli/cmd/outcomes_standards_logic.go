package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// frameworkStandardRow is one importable SBG standard (native JSON framework format).
type frameworkStandardRow struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	DomainCode  string `json:"domainCode"`
	DomainName  string `json:"domainName"`
	GradeLevel  string `json:"gradeLevel"`
}

type frameworkDomain struct {
	Code       string                 `json:"code"`
	Name       string                 `json:"name"`
	GradeLevel string                 `json:"gradeLevel"`
	Standards  []frameworkStandardRow `json:"standards"`
}

type frameworkImportDoc struct {
	Standards []frameworkStandardRow `json:"standards"`
	Domains   []frameworkDomain      `json:"domains"`
}

// outcomeAlignRow is one bulk outcome-to-item alignment.
type outcomeAlignRow struct {
	OutcomeID        string  `json:"outcomeId"`
	StructureItemID  string  `json:"structureItemId"`
	TargetKind       string  `json:"targetKind"`
	QuizQuestionID   *string `json:"quizQuestionId,omitempty"`
	MeasurementLevel *string `json:"measurementLevel,omitempty"`
	IntensityLevel   *string `json:"intensityLevel,omitempty"`
	SubOutcomeID     *string `json:"subOutcomeId,omitempty"`
}

type outcomeAlignDoc struct {
	Links []outcomeAlignRow `json:"links"`
}

// frameworkImportToCSV converts native JSON framework docs to the SBG CSV schema
// expected by POST /api/v1/admin/orgs/{orgId}/sbg/standards/import.
func frameworkImportToCSV(data []byte) ([]byte, error) {
	var doc frameworkImportDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid framework JSON: %w", err)
	}
	rows := make([]frameworkStandardRow, 0, len(doc.Standards))
	rows = append(rows, doc.Standards...)
	for _, d := range doc.Domains {
		for _, s := range d.Standards {
			row := s
			if strings.TrimSpace(row.DomainCode) == "" {
				row.DomainCode = d.Code
			}
			if strings.TrimSpace(row.DomainName) == "" {
				row.DomainName = d.Name
			}
			if strings.TrimSpace(row.GradeLevel) == "" {
				row.GradeLevel = d.GradeLevel
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("framework file has no standards")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"code", "description", "domain_code", "domain_name", "grade_level"}); err != nil {
		return nil, err
	}
	for i, row := range rows {
		code := strings.TrimSpace(row.Code)
		desc := strings.TrimSpace(row.Description)
		domainCode := strings.TrimSpace(row.DomainCode)
		domainName := strings.TrimSpace(row.DomainName)
		if code == "" || desc == "" || domainCode == "" || domainName == "" {
			return nil, fmt.Errorf("standards[%d]: code, description, domainCode, and domainName are required", i)
		}
		if err := w.Write([]string{
			code,
			desc,
			domainCode,
			domainName,
			strings.TrimSpace(row.GradeLevel),
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseOutcomeAlignFile reads JSON or CSV alignments for bulk outcome linking.
func parseOutcomeAlignFile(data []byte) ([]outcomeAlignRow, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("alignment file is empty")
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		var doc outcomeAlignDoc
		if err := json.Unmarshal(trimmed, &doc); err != nil {
			return nil, fmt.Errorf("invalid alignment JSON: %w", err)
		}
		if len(doc.Links) == 0 {
			return nil, fmt.Errorf("alignment JSON has no links")
		}
		return doc.Links, nil
	}
	return parseOutcomeAlignCSV(trimmed)
}

func parseOutcomeAlignCSV(data []byte) ([]outcomeAlignRow, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading alignment CSV header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := []string{"outcome_id", "structure_item_id", "target_kind"}
	for _, req := range required {
		if _, ok := col[req]; !ok {
			return nil, fmt.Errorf("alignment CSV missing required column %q", req)
		}
	}
	get := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	var rows []outcomeAlignRow
	for line := 2; ; line++ {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("alignment CSV line %d: %w", line, err)
		}
		outcomeID := get(rec, "outcome_id")
		structureItemID := get(rec, "structure_item_id")
		targetKind := get(rec, "target_kind")
		if outcomeID == "" || structureItemID == "" || targetKind == "" {
			return nil, fmt.Errorf("alignment CSV line %d: outcome_id, structure_item_id, and target_kind are required", line)
		}
		row := outcomeAlignRow{
			OutcomeID:       outcomeID,
			StructureItemID: structureItemID,
			TargetKind:      targetKind,
		}
		if q := get(rec, "quiz_question_id"); q != "" {
			row.QuizQuestionID = &q
		}
		if m := get(rec, "measurement_level"); m != "" {
			row.MeasurementLevel = &m
		}
		if i := get(rec, "intensity_level"); i != "" {
			row.IntensityLevel = &i
		}
		if s := get(rec, "sub_outcome_id"); s != "" {
			row.SubOutcomeID = &s
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("alignment CSV has no data rows")
	}
	return rows, nil
}

type reportCardRecord map[string]any

// reportCardsToCSV serializes report card records for export.
func reportCardsToCSV(cards []reportCardRecord) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{
		"card_id", "student_id", "grading_period", "status",
		"final_grade_pct", "letter_grade", "comment",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, card := range cards {
		if err := w.Write([]string{
			stringField(card, "id"),
			stringField(card, "studentId"),
			stringField(card, "gradingPeriod"),
			stringField(card, "status"),
			numberField(card, "finalGradePct"),
			stringField(card, "letterGrade"),
			stringField(card, "comment"),
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func numberField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case float64:
		return fmt.Sprintf("%g", t)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}