package ccr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
)

// AggregatedAchievement is a normalized achievement for CLR building.
type AggregatedAchievement struct {
	ID          string
	Type        ccrrepo.AchievementType
	Title       string
	Description string
	IssuedAt    time.Time
	EvidenceURL string
	OutcomeTags []string
}

// AggregateAchievements merges stored achievements and derived course completions.
func AggregateAchievements(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]AggregatedAchievement, error) {
	stored, err := ccrrepo.ListAchievements(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	completions, err := ccrrepo.ListCourseCompletions(ctx, pool, userID)
	if err != nil {
		return nil, err
	}

	out := make([]AggregatedAchievement, 0, len(stored)+len(completions))
	seenCourse := make(map[uuid.UUID]struct{}, len(completions))

	for _, c := range completions {
		seenCourse[c.CourseID] = struct{}{}
		desc := fmt.Sprintf("Final grade: %s", c.FinalGrade)
		out = append(out, AggregatedAchievement{
			ID:          "course:" + c.CourseID.String(),
			Type:        ccrrepo.TypeCourseCompletion,
			Title:       c.CourseTitle,
			Description: desc,
			IssuedAt:    c.IssuedAt,
		})
	}

	for _, a := range stored {
		if a.AchievementType == ccrrepo.TypeCourseCompletion && a.SourceID != nil {
			if _, ok := seenCourse[*a.SourceID]; ok {
				continue
			}
		}
		desc := ""
		if a.Description != nil {
			desc = *a.Description
		}
		evidence := ""
		if a.EvidenceURL != nil {
			evidence = *a.EvidenceURL
		}
		out = append(out, AggregatedAchievement{
			ID:          a.ID.String(),
			Type:        a.AchievementType,
			Title:       a.Title,
			Description: desc,
			IssuedAt:    a.IssuedAt,
			EvidenceURL: evidence,
			OutcomeTags: append([]string(nil), a.OutcomeTags...),
		})
	}
	return out, nil
}

// BuildCLRSubject constructs an IMS CLR v2.0 credentialSubject payload.
func BuildCLRSubject(learnerID, learnerName, publisherDID, publisherName string, achievements []AggregatedAchievement, issuedAt time.Time) map[string]any {
	assertions := make([]map[string]any, 0, len(achievements))
	for _, a := range achievements {
		achievement := map[string]any{
			"id":   "urn:uuid:" + a.ID,
			"type": "Achievement",
			"name": a.Title,
		}
		if a.Description != "" {
			achievement["description"] = a.Description
		}
		if len(a.OutcomeTags) > 0 {
			achievement["alignment"] = outcomeAlignments(a.OutcomeTags)
		}
		assertion := map[string]any{
			"id":          "urn:uuid:assertion:" + a.ID,
			"type":        "Assertion",
			"achievement": achievement,
			"issuedOn":    a.IssuedAt.UTC().Format(time.RFC3339),
		}
		if a.EvidenceURL != "" {
			assertion["evidence"] = []map[string]any{
				{
					"type": "Evidence",
					"url":  a.EvidenceURL,
				},
			}
		}
		assertions = append(assertions, assertion)
	}

	return map[string]any{
		"id":   "urn:uuid:clr:" + learnerID,
		"type": "ClrCredential",
		"name": "Comprehensive Learner Record",
		"learner": map[string]any{
			"id":   learnerID,
			"name": learnerName,
		},
		"publisher": map[string]any{
			"id":   publisherDID,
			"name": publisherName,
		},
		"issuedOn":   issuedAt.UTC().Format(time.RFC3339),
		"assertions": assertions,
	}
}

func outcomeAlignments(tags []string) []map[string]any {
	out := make([]map[string]any, 0, len(tags))
	for _, tag := range tags {
		out = append(out, map[string]any{
			"type":        "Alignment",
			"targetName":  tag,
			"targetFramework": "Suskie HE Assessment Framework",
		})
	}
	return out
}

// MarshalCLRJSON serializes the CLR subject for storage.
func MarshalCLRJSON(subject map[string]any) (json.RawMessage, error) {
	b, err := json.Marshal(subject)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
