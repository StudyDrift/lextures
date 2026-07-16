package transcriptorder

import (
	"fmt"
	"strings"
)

// HoldType classifies a transcript hold.
type HoldType string

const (
	HoldFinancial     HoldType = "financial"
	HoldDisciplinary  HoldType = "disciplinary"
	HoldRegistrar     HoldType = "registrar"
	HoldLibrary       HoldType = "library"
	HoldOther         HoldType = "other"
)

// ParseHoldType validates a hold type string.
func ParseHoldType(raw string) (HoldType, error) {
	t := HoldType(strings.ToLower(strings.TrimSpace(raw)))
	switch t {
	case HoldFinancial, HoldDisciplinary, HoldRegistrar, HoldLibrary, HoldOther:
		return t, nil
	default:
		return "", fmt.Errorf("invalid hold type %q", raw)
	}
}

// DefaultStudentMessage returns sanitized resolution guidance for a hold type.
func DefaultStudentMessage(t HoldType) string {
	switch t {
	case HoldFinancial:
		return "There is a financial hold on your account. Contact the bursar's office to resolve it before your transcript can be released."
	case HoldDisciplinary:
		return "There is a disciplinary hold on your account. Contact student affairs for next steps."
	case HoldRegistrar:
		return "There is a registrar hold on your transcript. Contact the registrar's office for assistance."
	case HoldLibrary:
		return "There is a library hold on your account. Contact the library to clear outstanding items."
	case HoldOther:
		return "There is a hold on your transcript order. Contact your institution for details."
	default:
		return "There is a hold on your transcript order. Contact your institution for details."
	}
}

// StudentFacingMessage prefers an explicit sanitized message; falls back to type defaults.
// Never returns the internal reason field.
func StudentFacingMessage(t HoldType, studentMessage *string) string {
	if studentMessage != nil {
		msg := strings.TrimSpace(*studentMessage)
		if msg != "" {
			return msg
		}
	}
	return DefaultStudentMessage(t)
}
