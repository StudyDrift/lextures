// Package accessibility is application logic for the accessibility services intake
// workflow (plan 14.16): it maps coordinator-managed accommodation profiles onto the
// 2.11 student_accommodations override engine and renders instructor notification letters.
package accessibility

import (
	"encoding/json"

	stac "github.com/lextures/lextures/server/internal/repos/studentaccommodations"
)

// Accommodation type values (mirror the accessibility.accommodation_type enum).
const (
	TypeExtendedTime15  = "extended_time_1_5x"
	TypeExtendedTime2   = "extended_time_2x"
	TypeSeparateTesting = "separate_testing"
	TypeAlternateFormat = "alternate_format"
	TypeScreenReader    = "screen_reader"
	TypeSpeechToText    = "speech_to_text"
	TypeReducedDistract = "reduced_distraction"
	TypeOther           = "other"
)

var validTypes = map[string]struct{}{
	TypeExtendedTime15:  {},
	TypeExtendedTime2:   {},
	TypeSeparateTesting: {},
	TypeAlternateFormat: {},
	TypeScreenReader:    {},
	TypeSpeechToText:    {},
	TypeReducedDistract: {},
	TypeOther:           {},
}

// ValidType reports whether t is a recognized accommodation type.
func ValidType(t string) bool {
	_, ok := validTypes[t]
	return ok
}

// ValidTypes reports whether every entry is recognized and the slice is non-empty.
func ValidTypes(types []string) bool {
	if len(types) == 0 {
		return false
	}
	for _, t := range types {
		if !ValidType(t) {
			return false
		}
	}
	return true
}

// customParams is the parsed shape of accommodation_profiles.custom_params.
type customParams struct {
	TimeMultiplier  float64 `json:"timeMultiplier"`
	AlternateFormat string  `json:"alternateFormat"`
}

// BuildWrite translates an accommodation type list plus custom params into a 2.11
// student_accommodations override payload (FR-3). The resulting row is the operational
// projection of the profile; no disability information is carried across.
func BuildWrite(types []string, rawParams json.RawMessage) stac.AccommodationWrite {
	var p customParams
	if len(rawParams) > 0 {
		_ = json.Unmarshal(rawParams, &p)
	}
	w := stac.AccommodationWrite{TimeMultiplier: 1.0}
	for _, t := range types {
		switch t {
		case TypeExtendedTime15:
			w.TimeMultiplier = maxF(w.TimeMultiplier, 1.5)
		case TypeExtendedTime2:
			w.TimeMultiplier = maxF(w.TimeMultiplier, 2.0)
		case TypeSeparateTesting:
			w.SeparateSetting = true
		case TypeAlternateFormat:
			fmtStr := p.AlternateFormat
			if fmtStr == "" {
				fmtStr = "alternate"
			}
			w.AlternativeFormat = &fmtStr
		case TypeScreenReader:
			w.TTSEnabled = true
		case TypeSpeechToText:
			w.SpeechToTextEnabled = true
		case TypeReducedDistract:
			w.ReducedDistraction = true
		case TypeOther:
			// No operational projection; recorded on the profile for coordinator reference.
		}
	}
	// An explicit coordinator-set multiplier overrides the preset (Risk: wrong multiplier
	// is mitigated by an explicit confirmation in the UI, but the value still wins here).
	if p.TimeMultiplier >= 1.0 {
		w.TimeMultiplier = p.TimeMultiplier
	}
	return w
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
