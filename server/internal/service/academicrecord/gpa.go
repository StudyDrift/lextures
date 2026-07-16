package academicrecord

import (
	"math"
	"strconv"
	"strings"
)

// ScaleKind selects how quality points are derived from a letter/display grade.
type ScaleKind string

const (
	ScaleFourPoint  ScaleKind = "4.0"
	ScalePercentage ScaleKind = "percentage"
	ScalePassFail   ScaleKind = "pass_fail"
)

// GradePoints maps a display grade to quality points on a 4.0 scale.
// Returns (points, includedInGPA). Pass/fail and non-graded codes are excluded.
func GradePoints(grade string, kind ScaleKind) (points float64, inGPA bool) {
	g := strings.TrimSpace(strings.ToUpper(grade))
	if g == "" || g == "IP" || g == "IN PROGRESS" {
		return 0, false
	}
	switch kind {
	case ScalePassFail:
		return 0, false
	case ScalePercentage:
		// Percentage transcripts store the numeric percent as the grade string.
		if pct, err := strconv.ParseFloat(g, 64); err == nil {
			return clamp((pct/100.0)*4.0, 0, 4), true
		}
		fallthrough
	default:
		switch g {
		case "A+", "A":
			return 4.0, true
		case "A-":
			return 3.7, true
		case "B+":
			return 3.3, true
		case "B":
			return 3.0, true
		case "B-":
			return 2.7, true
		case "C+":
			return 2.3, true
		case "C":
			return 2.0, true
		case "C-":
			return 1.7, true
		case "D+":
			return 1.3, true
		case "D":
			return 1.0, true
		case "D-":
			return 0.7, true
		case "F", "E":
			return 0.0, true
		case "P", "PASS", "S", "CR":
			return 0, false
		case "W", "AU", "I", "NC", "NR", "NG":
			return 0, false
		default:
			return 0, false
		}
	}
}

// CreditsEarnedForGrade returns earned credits for a final grade.
// Failing and non-credit codes earn zero; in-progress earns zero.
func CreditsEarnedForGrade(grade string, attempted float64) float64 {
	g := strings.TrimSpace(strings.ToUpper(grade))
	switch g {
	case "F", "E", "W", "AU", "I", "NC", "NR", "NG", "IP", "IN PROGRESS", "":
		return 0
	default:
		if attempted < 0 {
			return 0
		}
		return attempted
	}
}

// ComputeCumulative calculates GPA and credit totals from course lines.
// Rounding: GPA is rounded to three decimal places (half-up).
func ComputeCumulative(lines []CourseLine, kind ScaleKind) CumulativeBlock {
	var qp, att, earned, gpaCredits float64
	for i := range lines {
		line := &lines[i]
		if line.InProgress {
			att += line.CreditsAttempted
			continue
		}
		att += line.CreditsAttempted
		earned += line.CreditsEarned
		pts, inGPA := GradePoints(line.Grade, kind)
		if inGPA && line.CreditsAttempted > 0 {
			q := pts * line.CreditsAttempted
			line.QualityPoints = &q
			qp += q
			gpaCredits += line.CreditsAttempted
		}
	}
	out := CumulativeBlock{
		CreditsAttempted: round2(att),
		CreditsEarned:    round2(earned),
		QualityPoints:    round3(qp),
	}
	if gpaCredits > 0 {
		gpa := round3(qp / gpaCredits)
		out.GPA = &gpa
	}
	return out
}

// ComputeTermGPA returns term GPA for graded (non-IP) lines in the term.
func ComputeTermGPA(lines []CourseLine, kind ScaleKind) (gpa *float64, termCredits float64) {
	var qp, gpaCredits, earned float64
	for _, line := range lines {
		if line.InProgress {
			continue
		}
		earned += line.CreditsEarned
		pts, inGPA := GradePoints(line.Grade, kind)
		if inGPA && line.CreditsAttempted > 0 {
			qp += pts * line.CreditsAttempted
			gpaCredits += line.CreditsAttempted
		}
	}
	termCredits = round2(earned)
	if gpaCredits > 0 {
		v := round3(qp / gpaCredits)
		return &v, termCredits
	}
	return nil, termCredits
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
