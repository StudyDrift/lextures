// Package introcourse provisions and exposes the canonical "Welcome to Lextures" course (IC01).
package introcourse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
)

// Course is the provisioned intro course identity.
type Course struct {
	ID         uuid.UUID `json:"courseId"`
	CourseCode string    `json:"courseCode"`
	ShortCode  string    `json:"shortCode"`
	Created    bool      `json:"created"`
}

// Service is the single source of truth for intro course existence and identity.
type Service struct {
	Pool *pgxpool.Pool

	cacheMu sync.RWMutex
	cached  *uuid.UUID
}

// New returns a Service bound to pool.
func New(pool *pgxpool.Pool) *Service {
	return &Service{Pool: pool}
}

// Enabled reports whether intro course provisioning and discovery are active.
func Enabled(cfg config.Config) bool {
	return cfg.IntroCourseEnabled
}

// CourseID returns the cached course id when present.
func (s *Service) CourseID(ctx context.Context) (uuid.UUID, bool, error) {
	if s == nil || s.Pool == nil {
		return uuid.UUID{}, false, nil
	}
	if id := s.getCached(); id != nil {
		return *id, true, nil
	}
	id, err := icrepo.LookupIDByShortCode(ctx, s.Pool, ShortCode)
	if err != nil {
		return uuid.UUID{}, false, err
	}
	if id == nil {
		setCoursePresent(false)
		return uuid.UUID{}, false, nil
	}
	s.setCached(id)
	setCoursePresent(true)
	return *id, true, nil
}

// EnsureProvisioned idempotently creates or reconciles the canonical intro course.
func (s *Service) EnsureProvisioned(ctx context.Context, cfg config.Config) (Course, error) {
	if s == nil || s.Pool == nil {
		return Course{}, fmt.Errorf("intro course: database unavailable")
	}
	started := time.Now()

	existing, err := icrepo.LookupIDByShortCode(ctx, s.Pool, ShortCode)
	if err != nil {
		recordProvision("error", started)
		return Course{}, err
	}
	if !Enabled(cfg) && existing == nil {
		setCoursePresent(false)
		recordProvision("skipped_disabled", started)
		return Course{}, nil
	}

	out, created, err := s.provisionLocked(ctx, cfg, existing)
	if err != nil {
		recordProvision("error", started)
		return Course{}, err
	}
	s.setCached(&out.ID)
	setCoursePresent(true)
	if created {
		recordProvision("created", started)
	} else {
		recordProvision("reconciled", started)
	}
	slog.Info("intro course provisioned",
		"course_id", out.ID,
		"course_code", out.CourseCode,
		"created", created,
	)
	return Course{
		ID:         out.ID,
		CourseCode: out.CourseCode,
		ShortCode:  ShortCode,
		Created:    created,
	}, nil
}

func (s *Service) provisionLocked(ctx context.Context, cfg config.Config, knownExisting *uuid.UUID) (Course, bool, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Course{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('intro_course_provision'))`); err != nil {
		return Course{}, false, err
	}

	if err := icrepo.EnsureSystemInstructor(ctx, tx, SystemUserID); err != nil {
		return Course{}, false, err
	}
	orgID, err := icrepo.DefaultOrgID(ctx, tx)
	if err != nil {
		return Course{}, false, err
	}

	existing := knownExisting
	if existing == nil {
		existing, err = icrepo.LookupIDByShortCodeTx(ctx, tx, ShortCode)
		if err != nil {
			return Course{}, false, err
		}
	}

	now := time.Now().UTC()
	created := false
	var courseID uuid.UUID
	if existing == nil {
		courseID, err = icrepo.CreateCourse(ctx, tx, orgID, SystemUserID, now)
		if err != nil {
			return Course{}, false, err
		}
		created = true
	} else {
		courseID = *existing
		if err := icrepo.ReconcileCourse(ctx, tx, courseID, now); err != nil {
			return Course{}, false, err
		}
	}

	if err := icrepo.EnsureTeacherEnrollment(ctx, tx, courseID, SystemUserID, CourseCode); err != nil {
		return Course{}, false, err
	}
	if err := icrepo.EnsureAssignmentGroups(ctx, tx, courseID); err != nil {
		return Course{}, false, err
	}
	if err := gradingagentrepo.SeedDefaultTemplates(ctx, tx, courseID, SystemUserID); err != nil {
		return Course{}, false, err
	}
	if err := EnsureHeroBanner(ctx, tx, courseID, cfg); err != nil {
		return Course{}, false, err
	}

	if _, err := SyncContent(ctx, tx, courseID, cfg); err != nil {
		return Course{}, false, fmt.Errorf("intro course content sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Course{}, false, err
	}

	return Course{
		ID:         courseID,
		CourseCode: CourseCode,
		ShortCode:  ShortCode,
	}, created, nil
}

// SyncContentForCourse re-syncs curriculum fixtures into an existing intro course (IC03 / IC08).
func (s *Service) SyncContentForCourse(ctx context.Context, cfg config.Config, courseID uuid.UUID) (ContentSyncReport, error) {
	if s == nil || s.Pool == nil {
		return ContentSyncReport{}, fmt.Errorf("intro course: database unavailable")
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ContentSyncReport{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	report, err := SyncContent(ctx, tx, courseID, cfg)
	if err != nil {
		return report, err
	}
	if err := tx.Commit(ctx); err != nil {
		return report, err
	}
	_ = RecordSyncStatus(ctx, s.Pool, report, nil)
	return report, nil
}

func (s *Service) getCached() *uuid.UUID {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if s.cached == nil {
		return nil
	}
	id := *s.cached
	return &id
}

func (s *Service) setCached(id *uuid.UUID) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if id == nil {
		s.cached = nil
		return
	}
	c := *id
	s.cached = &c
}

// InvalidateCache clears the in-process short_code → id memo (tests and post-provision hooks).
func (s *Service) InvalidateCache() {
	s.setCached(nil)
}

// ErrDisabled is returned when provisioning is requested while the flag is off and no course exists.
var ErrDisabled = errors.New("intro course disabled")