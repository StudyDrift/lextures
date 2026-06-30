package csvimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
)

// RowError is a validation error for one CSV cell.
type RowError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ParsedRow is a validated user row ready for import.
type ParsedRow struct {
	RowNumber    int
	Email        string
	FirstName    string
	LastName     string
	Role         string
	ExternalID   string
	CustomFields map[string]string
}

// ParseResult holds parsed rows and validation errors.
type ParseResult struct {
	Rows   []ParsedRow
	Errors []RowError
}

// MergeStrategy controls create/update/deactivate behaviour.
type MergeStrategy string

const (
	MergeCreateOnly MergeStrategy = "create_only"
	MergeUpsert     MergeStrategy = "upsert"
	MergeSync       MergeStrategy = "sync"
)

// ParseMergeStrategy normalizes a merge strategy value.
func ParseMergeStrategy(s string) (MergeStrategy, error) {
	switch MergeStrategy(s) {
	case MergeCreateOnly, MergeUpsert, MergeSync:
		return MergeStrategy(s), nil
	default:
		return "", fmt.Errorf("unknown merge strategy %q", s)
	}
}

// ParseCSV reads and validates a user import CSV.
func ParseCSV(r io.Reader, profile Profile) (*ParseResult, error) {
	return ParseCSVWithExtraColumns(r, profile, nil)
}

// ParseCSVWithExtraColumns accepts additional logical column keys (e.g. custom field keys).
func ParseCSVWithExtraColumns(r io.Reader, profile Profile, extraColumns []string) (*ParseResult, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	cr := csv.NewReader(bytes.NewReader(raw))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("empty csv file")
		}
		return nil, fmt.Errorf("csv header: %w", err)
	}
	idx := buildHeaderIndex(header)
	colMap := profile.ColumnMap()
	for _, key := range extraColumns {
		if key == "" {
			continue
		}
		if _, exists := colMap[key]; !exists {
			colMap[key] = []string{key}
		}
	}

	res := &ParseResult{}
	rowNum := 1 // header is row 1
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv row %d: %w", rowNum+1, err)
		}
		rowNum++
		if rowEmpty(rec) {
			continue
		}
		parsed, errs := parseRecord(rowNum, rec, idx, colMap, extraColumns)
		if len(errs) > 0 {
			res.Errors = append(res.Errors, errs...)
			continue
		}
		res.Rows = append(res.Rows, parsed)
	}
	return res, nil
}

func buildHeaderIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, ok := m[key]; !ok {
			m[key] = i
		}
	}
	return m
}

func rowEmpty(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func getCol(rec []string, idx map[string]int, names ...string) string {
	for _, name := range names {
		if i, ok := idx[strings.ToLower(name)]; ok && i >= 0 && i < len(rec) {
			return sanitizeField(strings.TrimSpace(rec[i]))
		}
	}
	return ""
}

func parseRecord(rowNum int, rec []string, idx map[string]int, colMap map[string][]string, extraColumns []string) (ParsedRow, []RowError) {
	var errs []RowError
	email := getCol(rec, idx, colMap["email"]...)
	if email == "" {
		errs = append(errs, RowError{Row: rowNum, Column: "email", Message: "email is required", Code: "required"})
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs = append(errs, RowError{Row: rowNum, Column: "email", Message: "invalid email address", Code: "invalid_email"})
	}
	first := getCol(rec, idx, colMap["first_name"]...)
	last := getCol(rec, idx, colMap["last_name"]...)
	roleRaw := strings.ToLower(getCol(rec, idx, colMap["role"]...))
	role, roleErr := normalizeRole(roleRaw)
	if roleErr != "" {
		errs = append(errs, RowError{Row: rowNum, Column: "role", Message: roleErr, Code: "invalid_role"})
	}
	extID := getCol(rec, idx, colMap["external_id"]...)
	if len(errs) > 0 {
		return ParsedRow{}, errs
	}
	custom := make(map[string]string)
	for _, key := range extraColumns {
		if val := getCol(rec, idx, key); val != "" {
			custom[key] = val
		}
	}
	return ParsedRow{
		RowNumber:    rowNum,
		Email:        strings.ToLower(email),
		FirstName:    first,
		LastName:     last,
		Role:         role,
		ExternalID:   extID,
		CustomFields: custom,
	}, nil
}

func normalizeRole(raw string) (string, string) {
	switch strings.TrimSpace(raw) {
	case "teacher", "instructor", "staff":
		return "teacher", ""
	case "student", "learner":
		return "student", ""
	case "admin", "administrator", "org_admin":
		return "admin", ""
	case "":
		return "", "role is required"
	default:
		return "", fmt.Sprintf("unsupported role %q (use teacher, student, or admin)", raw)
	}
}

// sanitizeField strips CSV formula-injection prefixes (plan 18.2 risk mitigation).
func sanitizeField(s string) string {
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

// MapRoleToAppRole returns the app role name for a normalized CSV role.
func MapRoleToAppRole(role string) string {
	switch role {
	case "teacher":
		return "Teacher"
	case "admin":
		return "Teacher"
	default:
		return "Student"
	}
}

// IsOrgAdminRole reports whether the CSV role should receive org_admin grant.
func IsOrgAdminRole(role string) bool {
	return role == "admin"
}
