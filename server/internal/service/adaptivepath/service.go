package adaptivepath

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// Service provides adaptive path evaluation and validation helpers.
type Service struct {
	Name string
}

func New() Service {
	return Service{Name: "adaptivepath"}
}

// Health returns a stable service heartbeat string for wiring/tests.
func (s Service) Health(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	return s.Name + ":ok", nil
}

// AdaptivePathsGloballyEnabled reads the platform kill-switch ADAPTIVE_PATHS_ENABLED (default off).
func AdaptivePathsGloballyEnabled() bool {
	v := strings.TrimSpace(os.Getenv("ADAPTIVE_PATHS_ENABLED"))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func AdaptivePathsActiveForCourse(globalOn, courseFlag bool) bool {
	return globalOn && courseFlag
}

var (
	ErrInvalidRuleType   = errors.New("ruleType must be skip_if_mastered, required_if_not_mastered, unlock_after, or remediation_insert.")
	ErrEmptyConceptIDs   = errors.New("conceptIds must be non-empty.")
	ErrUnknownConcepts   = errors.New("One or more conceptIds are unknown or not usable in this course.")
	ErrHostNotInCourse   = errors.New("structureItemId is not part of this course.")
	ErrTargetNotInCourse = errors.New("targetItemId is not part of this course.")
	ErrTargetRequired    = errors.New("targetItemId is required for this rule type.")
	ErrBadThreshold      = errors.New("threshold must be between 0 and 1.")
)

func ValidateRuleType(rt string) error {
	switch rt {
	case "skip_if_mastered", "required_if_not_mastered", "unlock_after", "remediation_insert":
		return nil
	default:
		return ErrInvalidRuleType
	}
}

func ValidateThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return ErrBadThreshold
	}
	return nil
}

// ValidateConceptsForCourse ensures concept ids belong to the course (or are tagged to course questions).
func ValidateConceptsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, conceptIDs []uuid.UUID) error {
	if len(conceptIDs) == 0 {
		return ErrEmptyConceptIDs
	}
	rows, err := pool.Query(ctx, `
SELECT c.id
FROM course.concepts c
WHERE c.id = ANY($1)
  AND (
    c.course_id = $2
    OR EXISTS (
      SELECT 1
      FROM course.concept_question_tags t
      INNER JOIN course.questions q ON q.id = t.question_id
      WHERE t.concept_id = c.id AND q.course_id = $2
    )
  )
`, conceptIDs, courseID)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		found++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Deduped comparison: unique request ids must all match unique found rows.
	unique := make(map[uuid.UUID]struct{}, len(conceptIDs))
	for _, id := range conceptIDs {
		unique[id] = struct{}{}
	}
	if found != len(unique) {
		return ErrUnknownConcepts
	}
	return nil
}

// ValidateRuleTargetsInCourse ensures host and optional target structure items belong to the course.
func ValidateRuleTargetsInCourse(ctx context.Context, pool *pgxpool.Pool, courseID, hostItemID uuid.UUID, targetItemID *uuid.UUID) error {
	host, err := coursestructure.GetItemRow(ctx, pool, courseID, hostItemID)
	if err != nil {
		return err
	}
	if host == nil {
		return ErrHostNotInCourse
	}
	if targetItemID == nil {
		return nil
	}
	target, err := coursestructure.GetItemRow(ctx, pool, courseID, *targetItemID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrTargetNotInCourse
	}
	return nil
}

// RequireTargetForRuleType returns ErrTargetRequired when the rule type needs a target item.
func RequireTargetForRuleType(ruleType string, targetItemID *uuid.UUID) error {
	switch ruleType {
	case "required_if_not_mastered", "unlock_after", "remediation_insert":
		if targetItemID == nil {
			return ErrTargetRequired
		}
	}
	return nil
}
