package predict_reveal

import (
	"fmt"
	"math"
)

// NormalizeConfidence maps a raw scale value to 0..1 and a facet bucket label.
// For ScaleNone, raw is ignored and bucket is "none".
func NormalizeConfidence(scale ConfidenceScale, raw float64) (normalized float64, bucket string, err error) {
	switch scale {
	case ScaleNone:
		return 0, "none", nil
	case ScaleThree:
		n := int(math.Round(raw))
		if n < 1 || n > 3 {
			return 0, "", fmt.Errorf("confidence must be 1, 2, or 3 for three-point scale")
		}
		buckets := []string{"guessing", "fairly_sure", "certain"}
		return float64(n-1) / 2.0, buckets[n-1], nil
	case ScaleFive:
		n := int(math.Round(raw))
		if n < 1 || n > 5 {
			return 0, "", fmt.Errorf("confidence must be 1–5 for five-point scale")
		}
		return float64(n-1) / 4.0, fmt.Sprintf("%d", n), nil
	case ScalePercent:
		if raw < 0 || raw > 100 {
			return 0, "", fmt.Errorf("confidence must be 0–100 for percent scale")
		}
		norm := raw / 100.0
		switch {
		case raw < 20:
			bucket = "0_20"
		case raw < 40:
			bucket = "20_40"
		case raw < 60:
			bucket = "40_60"
		case raw < 80:
			bucket = "60_80"
		default:
			bucket = "80_100"
		}
		return norm, bucket, nil
	default:
		return 0, "", fmt.Errorf("unknown confidence scale")
	}
}

// CalibrationCell is one cell in the confidence × correctness matrix.
type CalibrationCell struct {
	ConfidenceBucket string `json:"confidenceBucket"`
	Correct          bool   `json:"correct"`
	Count            int    `json:"count"`
	Highlight        bool   `json:"highlight"` // confidently wrong
}

// BuildCalibrationMatrix counts (bucket, correct) pairs and highlights confidently-wrong.
func BuildCalibrationMatrix(rows []struct {
	Bucket  string
	Correct bool
}) []CalibrationCell {
	type key struct {
		b string
		c bool
	}
	counts := map[key]int{}
	order := []string{}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Bucket == "" {
			continue
		}
		counts[key{r.Bucket, r.Correct}]++
		if !seen[r.Bucket] {
			seen[r.Bucket] = true
			order = append(order, r.Bucket)
		}
	}
	out := make([]CalibrationCell, 0, len(order)*2)
	for _, b := range order {
		for _, correct := range []bool{true, false} {
			n := counts[key{b, correct}]
			if n == 0 {
				continue
			}
			cell := CalibrationCell{
				ConfidenceBucket: b,
				Correct:          correct,
				Count:            n,
			}
			// Confidently wrong: high-confidence incorrect predictions.
			if !correct && isHighConfidenceBucket(b) {
				cell.Highlight = true
			}
			out = append(out, cell)
		}
	}
	return out
}

func isHighConfidenceBucket(b string) bool {
	switch b {
	case "certain", "5", "80_100", "4":
		return true
	default:
		return false
	}
}
