package marketplacecourses

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketplacecourses"
)

// Course is a provisioned official marketplace course identity.
type Course struct {
	ID          uuid.UUID         `json:"courseId"`
	CourseCode  string            `json:"courseCode"`
	CatalogSlug string            `json:"catalogSlug"`
	ShortCode   string            `json:"shortCode"`
	Created     bool              `json:"created"`
	Report      ContentSyncReport `json:"report"`
}

// Service provisions official marketplace courses from embedded content.
type Service struct {
	Pool *pgxpool.Pool
}

// New returns a Service bound to pool.
func New(pool *pgxpool.Pool) *Service {
	return &Service{Pool: pool}
}

// EnsureProvisioned idempotently creates or reconciles one official marketplace course by slug.
func (s *Service) EnsureProvisioned(ctx context.Context, cfg config.Config, courseSlug string) (Course, error) {
	if s == nil || s.Pool == nil {
		return Course{}, fmt.Errorf("marketplace courses: database unavailable")
	}
	dir, err := ResolveCourseDir(courseSlug)
	if err != nil {
		return Course{}, err
	}
	spec, err := LoadCourseSpec(dir)
	if err != nil {
		return Course{}, err
	}
	if err := ValidateCourseSpec(spec); err != nil {
		return Course{}, err
	}

	started := time.Now()
	slug := spec.Manifest.CatalogSlug
	out, created, report, err := s.provisionLocked(ctx, cfg, spec)
	if err != nil {
		recordProvision(slug, "error", started)
		return Course{}, err
	}
	if created {
		recordProvision(slug, "created", started)
	} else if report.Skipped {
		recordProvision(slug, "noop", started)
	} else {
		recordProvision(slug, "reconciled", started)
	}
	slog.Info("marketplace course provisioned",
		"slug", slug,
		"course_id", out.ID,
		"course_code", out.CourseCode,
		"created", created,
		"content_skipped", report.Skipped,
		"modules", report.Modules,
		"pages", report.Pages,
		"quizzes", report.Quizzes,
		"updated", report.Updated,
		"unchanged", report.Unchanged,
	)
	out.Created = created
	out.Report = report
	return out, nil
}

// EnsureAllProvisioned provisions every embedded marketplace course.
func (s *Service) EnsureAllProvisioned(ctx context.Context, cfg config.Config) ([]Course, error) {
	return s.ensureProvisionedDirs(ctx, cfg, nil)
}

// EnsureDeployProvisioned provisions official catalog courses for API startup / deploy.
// Skips harness-smoke (CI-only fixture).
func (s *Service) EnsureDeployProvisioned(ctx context.Context, cfg config.Config) ([]Course, error) {
	return s.ensureProvisionedDirs(ctx, cfg, IsDeployCourse)
}

func (s *Service) ensureProvisionedDirs(ctx context.Context, cfg config.Config, include func(string) bool) ([]Course, error) {
	slugs, err := ListCourseSlugs()
	if err != nil {
		return nil, err
	}
	var out []Course
	for _, dir := range slugs {
		if include != nil && !include(dir) {
			continue
		}
		c, err := s.EnsureProvisioned(ctx, cfg, dir)
		if err != nil {
			return out, fmt.Errorf("%s: %w", dir, err)
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Service) provisionLocked(ctx context.Context, cfg config.Config, spec *CourseSpec) (Course, bool, ContentSyncReport, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockKey := "marketplace_course_provision:" + spec.Manifest.CatalogSlug
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	if err := mcrepo.EnsureSystemPublisher(ctx, tx); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}
	orgID, err := mcrepo.DefaultOrgID(ctx, tx)
	if err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	existing, err := mcrepo.LookupBySlugTx(ctx, tx, spec.Manifest.CatalogSlug)
	if err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	now := time.Now().UTC()
	created := false
	var courseID uuid.UUID
	if existing == nil {
		courseID, err = mcrepo.CreateCourse(ctx, tx, mcrepo.CreateCourseParams{
			CourseCode:        spec.Manifest.Code,
			ShortCode:         spec.Manifest.ShortCode,
			Title:             spec.Manifest.Title,
			Description:       spec.Manifest.Summary,
			CatalogSlug:       spec.Manifest.CatalogSlug,
			CatalogCategory:   spec.Manifest.CatalogCategory,
			DifficultyLevel:   spec.Manifest.DifficultyLevel,
			CatalogLanguage:   spec.Manifest.CatalogLanguage,
			PriceCents:        spec.Manifest.PriceCents,
			IsPublic:          spec.Manifest.IsPublic,
			MarketplaceListed: spec.Manifest.MarketplaceListed,
			OrgID:             orgID,
			CreatedBy:         SystemPublisherID,
			Now:               now,
		})
		if err != nil {
			return Course{}, false, ContentSyncReport{}, err
		}
		created = true
	} else {
		courseID = existing.CourseID
		if err := mcrepo.ReconcileCourse(ctx, tx, mcrepo.ReconcileCourseParams{
			CourseID:          courseID,
			Title:             spec.Manifest.Title,
			Description:       spec.Manifest.Summary,
			CatalogSlug:       spec.Manifest.CatalogSlug,
			CatalogCategory:   spec.Manifest.CatalogCategory,
			DifficultyLevel:   spec.Manifest.DifficultyLevel,
			CatalogLanguage:   spec.Manifest.CatalogLanguage,
			PriceCents:        spec.Manifest.PriceCents,
			IsPublic:          spec.Manifest.IsPublic,
			MarketplaceListed: spec.Manifest.MarketplaceListed,
			Now:               now,
		}); err != nil {
			return Course{}, false, ContentSyncReport{}, err
		}
	}

	// Ledger must exist before content items (FK on course_slug).
	if err := mcrepo.UpsertLedger(ctx, tx, spec.Manifest.CatalogSlug, courseID, spec.Manifest.ContentVersion); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	if err := mcrepo.EnsureTeacherEnrollment(ctx, tx, courseID, SystemPublisherID, spec.Manifest.Code); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}
	if err := mcrepo.EnsureAssignmentGroups(ctx, tx, courseID); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}
	if err := mcrepo.SyncLearningOutcomes(ctx, tx, courseID, spec.Manifest.Outcomes); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}
	if err := EnsureHeroBanner(ctx, tx, courseID, spec.Manifest.Code, spec.Manifest.CatalogSlug, courseFilesRoot(cfg)); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	report, err := SyncContent(ctx, tx, courseID, spec)
	if err != nil {
		return Course{}, false, ContentSyncReport{}, fmt.Errorf("content sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Course{}, false, ContentSyncReport{}, err
	}

	return Course{
		ID:          courseID,
		CourseCode:  spec.Manifest.Code,
		CatalogSlug: spec.Manifest.CatalogSlug,
		ShortCode:   spec.Manifest.ShortCode,
	}, created, report, nil
}

func courseFilesRoot(cfg config.Config) string {
	if root := strings.TrimSpace(cfg.CourseFilesRoot); root != "" {
		return root
	}
	return "data/course-files"
}
