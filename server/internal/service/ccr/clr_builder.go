package ccr

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	repo "github.com/lextures/lextures/server/internal/repos/ccr"
)

const clrContextURL = "https://purl.imsglobal.org/spec/clr/v2p0/context.json"

// BuildCLRInput describes a CLR v2.0 document build.
type BuildCLRInput struct {
	DocumentID      uuid.UUID
	InstitutionName string
	StudentName     string
	StudentDID      string
	IssuerDID       string
	IssuedAt        time.Time
	Achievements    []repo.Achievement
}

// BuildCLR assembles an IMS CLR Standard v2.0 JSON document.
func BuildCLR(in BuildCLRInput) map[string]any {
	assertions := make([]map[string]any, 0, len(in.Achievements))
	for _, a := range in.Achievements {
		assertions = append(assertions, map[string]any{
			"id":   fmt.Sprintf("urn:uuid:%s", a.ID),
			"type": "Assertion",
			"achievement": map[string]any{
				"id":                fmt.Sprintf("urn:uuid:%s-achievement", a.ID),
				"type":              "Achievement",
				"achievementType":   string(a.AchievementType),
				"name":              a.Title,
				"description":       stringOrNil(a.Description),
				"criteria":          map[string]any{"narrative": achievementCriteria(a)},
				"humanCode":         humanCodeForType(a.AchievementType),
				"fieldOfStudy":      outcomeTagsOrNil(a.OutcomeTags),
			},
			"issuedOn": a.IssuedAt.UTC().Format(time.RFC3339),
			"result": []map[string]any{
				{
					"type":  "Result",
					"name":  "Completed",
					"value": "Achieved",
				},
			},
			"evidence": evidenceOrNil(a.EvidenceURL),
		})
	}
	return map[string]any{
		"@context":  []string{clrContextURL},
		"id":        fmt.Sprintf("urn:uuid:%s", in.DocumentID),
		"type":      "Clr",
		"name":      "Comprehensive Learner Record",
		"issuedOn":  in.IssuedAt.UTC().Format(time.RFC3339),
		"publisher": map[string]any{"id": in.IssuerDID, "name": in.InstitutionName},
		"learner":   map[string]any{"id": in.StudentDID, "name": in.StudentName},
		"assertions": assertions,
	}
}

func stringOrNil(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func outcomeTagsOrNil(tags []string) any {
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func evidenceOrNil(url *string) any {
	if url == nil || *url == "" {
		return nil
	}
	return []map[string]any{{"type": "Evidence", "name": "Evidence", "url": *url}}
}

func achievementCriteria(a repo.Achievement) string {
	switch a.AchievementType {
	case repo.TypeCourseCompletion:
		return "Completed course with a posted final grade."
	case repo.TypeBadge:
		return "Earned institutional Open Badge."
	case repo.TypeCertificate:
		return "Awarded completion certificate."
	case repo.TypePortfolio:
		return "Demonstrated portfolio milestone."
	case repo.TypeExtracurricular:
		return "Verified extracurricular achievement."
	default:
		var unreachable repo.AchievementType = a.AchievementType
		_ = unreachable
		return "Verified achievement."
	}
}

func humanCodeForType(t repo.AchievementType) string {
	switch t {
	case repo.TypeCourseCompletion:
		return "CourseCompletion"
	case repo.TypeBadge:
		return "Badge"
	case repo.TypeCertificate:
		return "Certificate"
	case repo.TypePortfolio:
		return "Portfolio"
	case repo.TypeExtracurricular:
		return "Extracurricular"
	default:
		var unreachable repo.AchievementType = t
		_ = unreachable
		return "Achievement"
	}
}
