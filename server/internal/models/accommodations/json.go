package accommodations

import (
	"database/sql"
	"strings"
	"time"
)

type AccommodationSummaryPublic struct {
	HasAccommodation bool     `json:"hasAccommodation"`
	Flags            []string `json:"flags"`
}

type UserSearchHit struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
	FirstName   *string `json:"firstName,omitempty"`
	LastName    *string `json:"lastName,omitempty"`
	Sid         *string `json:"sid,omitempty"`
}

type UserSearchResponse struct {
	Users []UserSearchHit `json:"users"`
}

type StudentAccommodation struct {
	ID                     string  `json:"id"`
	UserID                 string  `json:"userId"`
	CourseID               *string `json:"courseId,omitempty"`
	CourseCode             *string `json:"courseCode,omitempty"`
	TimeMultiplier         float64 `json:"timeMultiplier"`
	ExtraAttempts          int32   `json:"extraAttempts"`
	HintsAlwaysEnabled     bool    `json:"hintsAlwaysEnabled"`
	ReducedDistraction     bool    `json:"reducedDistractionMode"`
	SpeechToTextEnabled    bool    `json:"speechToTextEnabled"`
	TTSEnabled             bool    `json:"ttsEnabled"`
	DyslexiaDisplayEnabled bool    `json:"dyslexiaDisplayEnabled"`
	HighContrastEnabled    bool    `json:"highContrastEnabled"`
	ReducedMotionEnabled   bool    `json:"reducedMotionEnabled"`
	SeparateSetting        bool    `json:"separateSetting"`
	AlternativeFormat      *string `json:"alternativeFormat,omitempty"`
	EffectiveFrom          *string `json:"effectiveFrom,omitempty"`
	EffectiveUntil         *string `json:"effectiveUntil,omitempty"`
	CreatedBy              string  `json:"createdBy"`
	UpdatedBy              *string `json:"updatedBy,omitempty"`
	CreatedAt              string  `json:"createdAt"`
	UpdatedAt              string  `json:"updatedAt"`
}

type CreateRequest struct {
	CourseCode             *string  `json:"courseCode"`
	TimeMultiplier         *float64 `json:"timeMultiplier"`
	ExtraAttempts          *int32   `json:"extraAttempts"`
	HintsAlwaysEnabled     *bool    `json:"hintsAlwaysEnabled"`
	ReducedDistraction     *bool    `json:"reducedDistractionMode"`
	SpeechToText           *bool    `json:"speechToTextEnabled"`
	TTS                    *bool    `json:"ttsEnabled"`
	DyslexiaDisplay        *bool    `json:"dyslexiaDisplayEnabled"`
	HighContrast           *bool    `json:"highContrastEnabled"`
	ReducedMotion          *bool    `json:"reducedMotionEnabled"`
	SeparateSetting        *bool    `json:"separateSetting"`
	AlternativeFormat      *string  `json:"alternativeFormat"`
	EffectiveFrom          *string  `json:"effectiveFrom"`
	EffectiveUntil         *string  `json:"effectiveUntil"`
}

type UpdateRequest struct {
	TimeMultiplier         float64 `json:"timeMultiplier"`
	ExtraAttempts          int32   `json:"extraAttempts"`
	HintsAlwaysEnabled     bool    `json:"hintsAlwaysEnabled"`
	ReducedDistraction     bool    `json:"reducedDistractionMode"`
	SpeechToTextEnabled    bool    `json:"speechToTextEnabled"`
	TTSEnabled             bool    `json:"ttsEnabled"`
	DyslexiaDisplayEnabled bool    `json:"dyslexiaDisplayEnabled"`
	HighContrastEnabled    bool    `json:"highContrastEnabled"`
	ReducedMotionEnabled   bool    `json:"reducedMotionEnabled"`
	SeparateSetting        bool    `json:"separateSetting"`
	AlternativeFormat      *string `json:"alternativeFormat"`
	EffectiveFrom          *string `json:"effectiveFrom"`
	EffectiveUntil         *string `json:"effectiveUntil"`
}

func YYYYMMDDFromNull(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	s := nt.Time.UTC().Format("2006-01-02")
	return &s
}

type MyResponse struct {
	Accommodations []MyEntry `json:"accommodations"`
}

type MyEntry struct {
	CourseCode             *string `json:"courseCode,omitempty"`
	HasExtendedTime        bool    `json:"hasExtendedTime"`
	HasExtraAttempts       bool    `json:"hasExtraAttempts"`
	HintsAlwaysAvailable   bool    `json:"hintsAlwaysAvailable"`
	ReducedDistraction     bool    `json:"reducedDistractionRecommended"`
	SpeechToTextEnabled    bool    `json:"speechToTextEnabled"`
	TTSEnabled             bool    `json:"ttsEnabled"`
	DyslexiaDisplayEnabled bool    `json:"dyslexiaDisplayEnabled"`
	HighContrastEnabled    bool    `json:"highContrastEnabled"`
	ReducedMotionEnabled   bool    `json:"reducedMotionEnabled"`
	SeparateSetting        bool    `json:"separateSetting"`
	EffectiveFrom          *string `json:"effectiveFrom,omitempty"`
	EffectiveUntil         *string `json:"effectiveUntil,omitempty"`
}

func ParseDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &utc, nil
}
