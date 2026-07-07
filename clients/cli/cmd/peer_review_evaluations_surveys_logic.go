package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

var respondentIdentityKeys = []string{
	"userId", "user_id",
	"respondentId", "respondent_id",
	"studentId", "student_id",
	"submittedBy", "submitted_by",
	"submittedByDisplayName", "submitted_by_display_name",
	"email", "displayName", "display_name",
	"enrollmentId", "enrollment_id",
}

// surveyAnonymityMode returns the survey anonymity mode from a survey payload.
func surveyAnonymityMode(survey map[string]any) string {
	return strings.ToLower(strings.TrimSpace(stringField(survey, "anonymityMode")))
}

// surveyIsAnonymous reports whether exports must omit respondent identity.
func surveyIsAnonymous(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "anonymous", "pseudo_anonymous":
		return true
	default:
		return false
	}
}

// stripRespondentIdentity removes identity fields from export payloads when anonymity applies.
func stripRespondentIdentity(data map[string]any) {
	for _, key := range respondentIdentityKeys {
		delete(data, key)
	}
	if responses, ok := data["responses"].([]any); ok {
		for _, row := range responses {
			if m, ok := row.(map[string]any); ok {
				stripRespondentIdentity(m)
			}
		}
	}
	if rows, ok := data["rows"].([]any); ok {
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				stripRespondentIdentity(m)
			}
		}
	}
}

// prepareSurveyResultsExport builds an export-safe survey results document.
func prepareSurveyResultsExport(survey map[string]any, results map[string]any) (map[string]any, error) {
	mode := surveyAnonymityMode(survey)
	out := map[string]any{
		"surveyId":      stringField(survey, "id"),
		"title":         stringField(survey, "title"),
		"anonymityMode": mode,
		"responseCount": results["responseCount"],
		"questions":     results["questions"],
	}
	if surveyIsAnonymous(mode) {
		stripRespondentIdentity(out)
	}
	return out, nil
}

// surveyResultsToCSV converts aggregated survey results to a flat CSV.
func surveyResultsToCSV(exportDoc map[string]any) ([]byte, error) {
	questions, ok := exportDoc["questions"].([]any)
	if !ok {
		questions = []any{}
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"survey_id", "title", "anonymity_mode", "question_id", "subtype", "response_count", "mean", "distribution"}); err != nil {
		return nil, err
	}
	surveyID := stringField(exportDoc, "surveyId")
	title := stringField(exportDoc, "title")
	mode := stringField(exportDoc, "anonymityMode")
	for _, q := range questions {
		m, ok := q.(map[string]any)
		if !ok {
			continue
		}
		dist := ""
		if raw, ok := m["distribution"]; ok && raw != nil {
			b, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			dist = string(b)
		}
		mean := ""
		if v, ok := m["mean"].(float64); ok {
			mean = fmt.Sprintf("%.4f", v)
		}
		count := ""
		switch v := m["responseCount"].(type) {
		case float64:
			count = fmt.Sprintf("%.0f", v)
		case int64:
			count = fmt.Sprintf("%d", v)
		case int:
			count = fmt.Sprintf("%d", v)
		}
		if err := w.Write([]string{
			surveyID,
			title,
			mode,
			stringField(m, "questionId"),
			stringField(m, "subtype"),
			count,
			mean,
			dist,
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

// evaluationResultsToCSV converts aggregate evaluation results to CSV rows.
func evaluationResultsToCSV(results map[string]any) ([]byte, error) {
	questions, ok := results["questions"].([]any)
	if !ok {
		questions = []any{}
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{
		"window_id", "opens_at", "closes_at", "response_count", "enrolled_count",
		"completion_pct", "meets_threshold", "question_index", "question_type", "question_text",
		"average", "distribution", "open_texts",
	}); err != nil {
		return nil, err
	}
	windowID := stringField(results, "windowId")
	opensAt := stringField(results, "opensAt")
	closesAt := stringField(results, "closesAt")
	responseCount := fmt.Sprintf("%v", results["responseCount"])
	enrolledCount := fmt.Sprintf("%v", results["enrolledCount"])
	completionPct := fmt.Sprintf("%v", results["completionPct"])
	meetsThreshold := fmt.Sprintf("%v", results["meetsThreshold"])
	for _, q := range questions {
		m, ok := q.(map[string]any)
		if !ok {
			continue
		}
		avg := ""
		if v, ok := m["average"].(float64); ok {
			avg = fmt.Sprintf("%.4f", v)
		}
		dist := ""
		if raw, ok := m["distribution"]; ok && raw != nil {
			b, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			dist = string(b)
		}
		openTexts := ""
		if raw, ok := m["openTexts"].([]any); ok && len(raw) > 0 {
			parts := make([]string, 0, len(raw))
			for _, item := range raw {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			openTexts = strings.Join(parts, " | ")
		}
		if err := w.Write([]string{
			windowID,
			opensAt,
			closesAt,
			responseCount,
			enrolledCount,
			completionPct,
			meetsThreshold,
			fmt.Sprintf("%v", m["index"]),
			stringField(m, "type"),
			stringField(m, "text"),
			avg,
			dist,
			openTexts,
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

// peerReviewPerReviewer validates the --per allocation flag.
func peerReviewPerReviewer(per int) error {
	if per < 1 || per > 20 {
		return fmt.Errorf("--per must be between 1 and 20")
	}
	return nil
}