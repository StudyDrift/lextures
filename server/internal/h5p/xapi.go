package h5p

import (
	"encoding/json"
	"strings"
)

// Statement is a minimal xAPI 1.0.3 statement from H5P postMessage payloads.
type Statement struct {
	Verb   VerbRef         `json:"verb"`
	Result *StatementResult `json:"result"`
}

type VerbRef struct {
	ID string `json:"id"`
}

type StatementResult struct {
	Completion *bool        `json:"completion"`
	Success    *bool        `json:"success"`
	Score      *ScoreResult `json:"score"`
}

type ScoreResult struct {
	Raw float64 `json:"raw"`
	Max float64 `json:"max"`
}

// CompletionStatus derives h5p_completions.status from an xAPI statement verb and result.
func CompletionStatus(stmt Statement) (status string, scoreRaw, scoreMax *float64) {
	verb := strings.ToLower(strings.TrimSpace(stmt.Verb.ID))
	switch {
	case strings.HasSuffix(verb, "/completed"):
		status = "completed"
	case strings.HasSuffix(verb, "/passed"):
		status = "passed"
	case strings.HasSuffix(verb, "/failed"):
		status = "failed"
	case strings.HasSuffix(verb, "/attempted"), strings.HasSuffix(verb, "/initialized"), strings.HasSuffix(verb, "/experienced"):
		status = "in_progress"
	default:
		return "", nil, nil
	}
	if stmt.Result != nil && stmt.Result.Score != nil {
		r, m := stmt.Result.Score.Raw, stmt.Result.Score.Max
		scoreRaw, scoreMax = &r, &m
	}
	if stmt.Result != nil && stmt.Result.Success != nil {
		if *stmt.Result.Success && status == "completed" {
			status = "passed"
		}
		if !*stmt.Result.Success && status == "completed" {
			status = "failed"
		}
	}
	return status, scoreRaw, scoreMax
}

// ParseStatement unmarshals a JSON xAPI statement body.
func ParseStatement(raw json.RawMessage) (Statement, error) {
	var stmt Statement
	if len(raw) == 0 {
		return Statement{}, nil
	}
	err := json.Unmarshal(raw, &stmt)
	return stmt, err
}

// DisplayLabel returns a gradebook-friendly label for a completion status.
func DisplayLabel(status string) string {
	switch status {
	case "completed":
		return "Completed"
	case "in_progress":
		return "In progress"
	case "passed":
		return "Passed"
	case "failed":
		return "Failed"
	default:
		return "Not started"
	}
}
