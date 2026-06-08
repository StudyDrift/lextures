// Package incompletegrades implements the HE Incomplete grade workflow (plan 14.4).
package incompletegrades

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	modelenrollment "github.com/lextures/lextures/server/internal/models/enrollment"
	repo "github.com/lextures/lextures/server/internal/repos/incompletegrades"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

var (
	ErrNotFound         = errors.New("incomplete grade record not found")
	ErrAlreadyExists    = errors.New("enrollment already has an incomplete grade record")
	ErrNotOpen          = errors.New("incomplete grade record is not open")
	ErrInvalidDeadline  = errors.New("extension deadline must be in the future")
	ErrInvalidGrade     = errors.New("resolved grade is required")
	ErrOutstandingEmpty = errors.New("at least one outstanding assignment is required")
)

// Service coordinates incomplete grade grants, extensions, and resolutions.
type Service struct {
	Pool *pgxpool.Pool
	Notify *notifications.Service
}

// GrantParams is input for granting an Incomplete.
type GrantParams struct {
	CourseID           uuid.UUID
	EnrollmentID       uuid.UUID
	ActorID            uuid.UUID
	ExtensionDeadline  time.Time
	OutstandingItemIDs []uuid.UUID
	Notes              *string
}

// Grant creates an incomplete record and transitions enrollment to incomplete.
func (s *Service) Grant(ctx context.Context, p GrantParams) (*repo.Record, error) {
	if s.Pool == nil {
		return nil, fmt.Errorf("incompletegrades: pool required")
	}
	deadline := dateOnly(p.ExtensionDeadline)
	today := dateOnly(time.Now().UTC())
	if !deadline.After(today) {
		return nil, ErrInvalidDeadline
	}
	if len(p.OutstandingItemIDs) == 0 {
		return nil, ErrOutstandingEmpty
	}

	existing, err := repo.GetByEnrollmentID(ctx, s.Pool, p.EnrollmentID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == repo.StatusOpen {
		return nil, ErrAlreadyExists
	}

	enroll, err := enrollment.GetStateByID(ctx, s.Pool, p.CourseID, p.EnrollmentID)
	if err != nil {
		return nil, err
	}
	if enroll == nil {
		return nil, ErrNotFound
	}
	if enroll.Role != "student" {
		return nil, fmt.Errorf("incomplete grades apply to student enrollments only")
	}

	reason := "Incomplete grade granted"
	actor := p.ActorID
	_, err = enrollment.TransitionState(
		ctx, s.Pool, p.EnrollmentID, p.CourseID, &actor,
		modelenrollment.StateIncomplete, &reason, "manual",
		modelenrollment.DeadlineContext{OverrideDeadlines: true},
	)
	if err != nil {
		return nil, err
	}

	var rec *repo.Record
	if existing != nil {
		rec, err = repo.Reopen(ctx, s.Pool, existing.ID, repo.InsertParams{
			EnrollmentID:       p.EnrollmentID,
			GrantedBy:          p.ActorID,
			ExtensionDeadline:  deadline,
			OutstandingItemIDs: p.OutstandingItemIDs,
			Notes:              p.Notes,
		})
	} else {
		rec, err = repo.Insert(ctx, s.Pool, repo.InsertParams{
			EnrollmentID:       p.EnrollmentID,
			GrantedBy:          p.ActorID,
			ExtensionDeadline:  deadline,
			OutstandingItemIDs: p.OutstandingItemIDs,
			Notes:              p.Notes,
		})
	}
	if err != nil {
		return nil, err
	}

	after, _ := json.Marshal(map[string]any{
		"extensionDeadline":  deadline.Format("2006-01-02"),
		"outstandingItemIds": p.OutstandingItemIDs,
		"status":             repo.StatusOpen,
	})
	targetType := "enrollment"
	_, _ = adminaudit.Record(ctx, s.Pool, adminaudit.RecordParams{
		EventType:   adminaudit.EventIncompleteGranted,
		ActorID:     p.ActorID,
		TargetType:  &targetType,
		TargetID:    &p.EnrollmentID,
		AfterValue:  after,
	})

	if s.Notify != nil {
		s.Notify.NotifyIncompleteGranted(ctx, enroll.UserID, p.CourseID, rec.ExtensionDeadline)
	}
	return rec, nil
}

// ExtendDeadline updates the extension date on an open record.
func (s *Service) ExtendDeadline(ctx context.Context, courseID, enrollmentID, actorID uuid.UUID, deadline time.Time) (*repo.Record, error) {
	deadline = dateOnly(deadline)
	today := dateOnly(time.Now().UTC())
	if !deadline.After(today) {
		return nil, ErrInvalidDeadline
	}
	rec, err := repo.GetByEnrollmentID(ctx, s.Pool, enrollmentID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.Status != repo.StatusOpen {
		return nil, ErrNotOpen
	}
	enroll, err := enrollment.GetStateByID(ctx, s.Pool, courseID, enrollmentID)
	if err != nil {
		return nil, err
	}
	if enroll == nil {
		return nil, ErrNotFound
	}
	return repo.UpdateExtensionDeadline(ctx, s.Pool, rec.ID, deadline)
}

// Resolve records the final grade and transitions enrollment back to active.
func (s *Service) Resolve(ctx context.Context, courseID, enrollmentID, actorID uuid.UUID, grade string) (*repo.Record, error) {
	grade = strings.TrimSpace(grade)
	if grade == "" {
		return nil, ErrInvalidGrade
	}
	rec, err := repo.GetByEnrollmentID(ctx, s.Pool, enrollmentID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.Status != repo.StatusOpen {
		return nil, ErrNotOpen
	}
	enroll, err := enrollment.GetStateByID(ctx, s.Pool, courseID, enrollmentID)
	if err != nil {
		return nil, err
	}
	if enroll == nil {
		return nil, ErrNotFound
	}

	updated, err := repo.Resolve(ctx, s.Pool, rec.ID, actorID, grade)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrNotOpen
	}

	reason := fmt.Sprintf("Incomplete resolved: %s", grade)
	actor := actorID
	_, err = enrollment.TransitionState(
		ctx, s.Pool, enrollmentID, courseID, &actor,
		modelenrollment.StateActive, &reason, "manual",
		modelenrollment.DeadlineContext{OverrideDeadlines: true},
	)
	if err != nil {
		return nil, err
	}

	after, _ := json.Marshal(map[string]any{
		"resolvedGrade": grade,
		"status":        repo.StatusResolved,
	})
	targetType := "enrollment"
	_, _ = adminaudit.Record(ctx, s.Pool, adminaudit.RecordParams{
		EventType:   adminaudit.EventIncompleteResolved,
		ActorID:     actorID,
		TargetType:  &targetType,
		TargetID:    &enrollmentID,
		AfterValue:  after,
	})
	return updated, nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
