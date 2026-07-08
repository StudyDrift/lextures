package introcourse

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
)

// AdminStatus is the platform-admin operational snapshot (IC08 FR-1, FR-7).
type AdminStatus struct {
	Enabled                bool               `json:"enabled"`
	CoursePresent          bool               `json:"coursePresent"`
	CourseID               *uuid.UUID         `json:"courseId,omitempty"`
	CourseCode             string             `json:"courseCode"`
	ContentVersion         int                `json:"contentVersion"`
	ModuleCount            int                `json:"moduleCount"`
	LastSyncedAt           *time.Time         `json:"lastSyncedAt,omitempty"`
	LastSyncResult         *string            `json:"lastSyncResult,omitempty"`
	LastValidatedAt        *time.Time         `json:"lastValidatedAt,omitempty"`
	LastValidationResult   *string            `json:"lastValidationResult,omitempty"`
	AvailableLocales       []string           `json:"availableLocales"`
	LocaleCoverage         map[string]float64 `json:"localeCoverage"`
	Backfill               BackfillStatus     `json:"backfill"`
}

// LoadAdminStatus returns the intro course admin panel snapshot.
func LoadAdminStatus(ctx context.Context, pool *pgxpool.Pool, svc *Service, cfg config.Config) (AdminStatus, error) {
	out := AdminStatus{
		Enabled:          Enabled(cfg),
		CourseCode:       CourseCode,
		ContentVersion:   ContentVersion,
		LocaleCoverage:   map[string]float64{},
		AvailableLocales: []string{"en"},
	}
	if pool == nil || svc == nil {
		return out, nil
	}

	locales, err := ListContentLocales()
	if err == nil && len(locales) > 0 {
		out.AvailableLocales = locales
	}
	for _, loc := range out.AvailableLocales {
		cov, cerr := LocaleCoverage(loc)
		if cerr == nil {
			out.LocaleCoverage[loc] = cov
		}
	}

	st, err := icrepo.LoadStatus(ctx, pool)
	if err != nil {
		return out, err
	}
	if st.ContentVersion > 0 {
		out.ContentVersion = st.ContentVersion
	}
	out.LastSyncedAt = st.LastSyncedAt
	out.LastSyncResult = st.LastSyncResult
	out.LastValidatedAt = st.LastValidatedAt
	out.LastValidationResult = st.LastValidationResult

	backfill, err := svc.BackfillStatus(ctx, cfg)
	if err != nil {
		return out, err
	}
	out.Backfill = backfill

	courseID, found, err := svc.CourseID(ctx)
	if err != nil {
		return out, err
	}
	if !found || courseID == uuid.Nil {
		return out, nil
	}
	out.CoursePresent = true
	id := courseID
	out.CourseID = &id

	modCount, err := icrepo.CountModules(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	out.ModuleCount = modCount
	return out, nil
}

// RunValidation validates all locale fixtures and records the result.
func RunValidation(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	now := time.Now().UTC()
	if err := ValidateAllLocales(); err != nil {
		result := err.Error()
		if pool != nil {
			_ = icrepo.RecordValidationResult(ctx, pool, now, result)
		}
		return result, err
	}
	result := "ok"
	if pool != nil {
		if err := icrepo.RecordValidationResult(ctx, pool, now, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

// RecordSyncStatus persists sync outcome to the singleton status row.
func RecordSyncStatus(ctx context.Context, pool *pgxpool.Pool, report ContentSyncReport, syncErr error) error {
	if pool == nil {
		return nil
	}
	now := time.Now().UTC()
	result := "success"
	if syncErr != nil {
		result = fmt.Sprintf("error: %v", syncErr)
	} else if report.Skipped {
		result = "noop"
	}
	return icrepo.RecordSyncResult(ctx, pool, report.ContentVersion, now, result)
}