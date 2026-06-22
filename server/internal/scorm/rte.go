package scorm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CMIUpdate is a batch of SCORM 1.2 CMI element values from an RTE commit.
type CMIUpdate map[string]string

// RegistrationState holds persisted runtime data for a learner attempt.
type RegistrationState struct {
	CompletionStatus string
	SuccessStatus    string
	ScoreScaled      *float64
	ScoreRaw         *float64
	ScoreMax         *float64
	TotalTimeSeconds int
	SuspendData      string
	Location         string
}

// InitialCMI builds SCORM 1.2 read/write values for LMSInitialize.
func InitialCMI(state RegistrationState, studentID, studentName string, hasResume bool) map[string]string {
	entry := "ab-initio"
	if hasResume {
		entry = "resume"
	}
	out := map[string]string{
		"cmi.core.student_id":   studentID,
		"cmi.core.student_name": studentName,
		"cmi.core.credit":       "credit",
		"cmi.core.lesson_mode":  "normal",
		"cmi.core.entry":        entry,
		"cmi.core.lesson_status": normalizeLessonStatus(state.CompletionStatus),
		"cmi.core.lesson_location": state.Location,
		"cmi.core.suspend_data":    state.SuspendData,
		"cmi.core.total_time":      formatSCORMTime(state.TotalTimeSeconds),
		"cmi.core.session_time":    "0000:00:00.00",
	}
	if state.ScoreRaw != nil {
		out["cmi.core.score.raw"] = formatScore(*state.ScoreRaw)
	}
	if state.ScoreMax != nil {
		out["cmi.core.score.max"] = formatScore(*state.ScoreMax)
	}
	out["cmi.core.score.min"] = "0"
	return out
}

// ApplyCMIUpdate merges commit payload into registration state (SCORM 1.2 subset).
func ApplyCMIUpdate(state *RegistrationState, cmi CMIUpdate) {
	for k, v := range cmi {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		switch key {
		case "cmi.core.lesson_status":
			state.CompletionStatus = val
		case "cmi.core.lesson_location":
			state.Location = val
		case "cmi.core.suspend_data":
			state.SuspendData = val
		case "cmi.core.score.raw":
			if f, ok := parseScore(val); ok {
				state.ScoreRaw = &f
			}
		case "cmi.core.score.max":
			if f, ok := parseScore(val); ok {
				state.ScoreMax = &f
			}
		case "cmi.core.session_time":
			sec := parseSCORMTimeSeconds(val)
			if sec > 0 {
				state.TotalTimeSeconds += sec
			}
		}
	}
	state.SuccessStatus = lessonStatusToSuccess(state.CompletionStatus)
	if state.ScoreRaw != nil && state.ScoreMax != nil && *state.ScoreMax > 0 {
		scaled := *state.ScoreRaw / *state.ScoreMax
		if scaled < 0 {
			scaled = 0
		}
		if scaled > 1 {
			scaled = 1
		}
		state.ScoreScaled = &scaled
	}
}

// GradePoints returns scaled points for gradebook when the attempt has a score.
func GradePoints(state RegistrationState, pointsWorth int) (float64, float64, bool) {
	if state.ScoreRaw == nil {
		return 0, 0, false
	}
	raw := *state.ScoreRaw
	max := 100.0
	if state.ScoreMax != nil && *state.ScoreMax > 0 {
		max = *state.ScoreMax
	}
	if pointsWorth <= 0 {
		pointsWorth = 100
	}
	pts := raw / max * float64(pointsWorth)
	if pts < 0 {
		pts = 0
	}
	if pts > float64(pointsWorth) {
		pts = float64(pointsWorth)
	}
	return pts, float64(pointsWorth), true
}

// IsAttemptComplete returns true when lesson status indicates a terminal state.
func IsAttemptComplete(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "passed" || s == "completed" || s == "failed"
}

func normalizeLessonStatus(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "not attempted"
	}
	return s
}

func lessonStatusToSuccess(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed":
		return "passed"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func formatScore(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "0"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func parseScore(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// parseSCORMTimeSeconds parses SCORM 1.2 time format HHHH:MM:SS.SS to seconds.
func parseSCORMTimeSeconds(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	secParts := strings.Split(parts[2], ".")
	sec, err3 := strconv.Atoi(secParts[0])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	return h*3600 + m*60 + sec
}

func formatSCORMTime(totalSec int) string {
	if totalSec < 0 {
		totalSec = 0
	}
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	return fmt.Sprintf("%04d:%02d:%02d.00", h, m, s)
}
