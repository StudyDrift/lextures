// Package catalogsync implements the HE course catalog sync worker (plan 14.2).
package catalogsync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repoCatalog "github.com/lextures/lextures/server/internal/repos/catalog"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
	repoTerms "github.com/lextures/lextures/server/internal/repos/terms"
	serviceSIS "github.com/lextures/lextures/server/internal/service/sis"
)

// SyncResult is returned by RunSync.
type SyncResult struct {
	LogID          uuid.UUID
	Status         string
	SectionsSynced int
	ShellsCreated  int
	ShellsUpdated  int
	Errors         []repoCatalog.SyncError
}

// RunSync pulls catalog data from the SIS adapter, upserts sections, and creates LMS shells.
func RunSync(ctx context.Context, pool *pgxpool.Pool, conn repoSIS.Connection) (*SyncResult, error) {
	connID := conn.ID
	log, err := repoCatalog.CreateSyncLog(ctx, pool, conn.OrgID, &connID)
	if err != nil {
		return nil, err
	}

	slog.Info("catalog sync started", "connection_id", conn.ID, "vendor", conn.Vendor, "log_id", log.ID)

	sectionsSynced, shellsCreated, shellsUpdated, errs := runCatalogPull(ctx, pool, conn)

	status := repoCatalog.SyncStatusSuccess
	if len(errs) > 0 && sectionsSynced == 0 {
		status = repoCatalog.SyncStatusFailed
	} else if len(errs) > 0 {
		status = repoCatalog.SyncStatusPartial
	}

	if err := repoCatalog.FinishSyncLog(ctx, pool, log.ID, status, sectionsSynced, shellsCreated, shellsUpdated, errs); err != nil {
		slog.Warn("catalog sync: failed to finish log", "log_id", log.ID, "err", err)
	}

	slog.Info("catalog sync finished", "connection_id", conn.ID, "status", status,
		"sections_synced", sectionsSynced, "shells_created", shellsCreated)

	return &SyncResult{
		LogID:          log.ID,
		Status:         status,
		SectionsSynced: sectionsSynced,
		ShellsCreated:  shellsCreated,
		ShellsUpdated:  shellsUpdated,
		Errors:         errs,
	}, nil
}

// SweepScheduled runs catalog sync for HE SIS connections due for nightly pull.
func SweepScheduled(ctx context.Context, pool *pgxpool.Pool) {
	conns, err := repoSIS.ListActiveConnections(ctx, pool)
	if err != nil {
		slog.Warn("catalog sweep: list connections failed", "err", err)
		return
	}
	now := time.Now().UTC()
	for _, c := range conns {
		if !serviceSIS.IsHEVendor(c.Vendor) {
			continue
		}
		if c.LastSyncAt != nil && now.Sub(*c.LastSyncAt) < 20*time.Hour {
			continue
		}
		conn := c
		go func() {
			if _, err := RunSync(ctx, pool, conn); err != nil {
				slog.Warn("catalog sweep: sync failed", "connection_id", conn.ID, "err", err)
			}
		}()
	}
}

func runCatalogPull(ctx context.Context, pool *pgxpool.Pool, conn repoSIS.Connection) (int, int, int, []repoCatalog.SyncError) {
	adapter := serviceSIS.AdapterFor(conn.Vendor)
	if adapter == nil {
		return 0, 0, 0, []repoCatalog.SyncError{{Message: "no HE adapter for vendor: " + conn.Vendor}}
	}

	termID, termName, err := activeTermForOrg(ctx, pool, conn.OrgID)
	if err != nil {
		return 0, 0, 0, []repoCatalog.SyncError{{Message: "active term lookup: " + err.Error()}}
	}
	if termID == nil {
		return 0, 0, 0, []repoCatalog.SyncError{{Message: "no active term for org"}}
	}

	sections, regErrs, err := serviceSIS.SyncCatalog(ctx, adapter, serviceSIS.ConnectionConfig{
		Vendor:          conn.Vendor,
		BaseURL:         conn.BaseURL,
		ClientIDRef:     conn.ClientIDRef,
		ClientSecretRef: conn.ClientSecretRef,
	}, *termID, termName)
	if err != nil {
		return 0, 0, 0, []repoCatalog.SyncError{{Message: err.Error()}}
	}

	var syncErrs []repoCatalog.SyncError
	for _, e := range regErrs {
		syncErrs = append(syncErrs, repoCatalog.SyncError{RecordID: e.RecordID, Message: e.Message})
	}

	sectionsSynced := 0
	shellsCreated := 0
	shellsUpdated := 0

	creatorID, _ := repoCatalog.FindOrgBootstrapUser(ctx, pool, conn.OrgID)

	for _, sec := range sections {
		in := repoCatalog.UpsertSectionInput{
			TermID:         sec.TermID,
			SISCourseID:    sec.SISCourseID,
			SISSectionID:   sec.SISSectionID,
			CRN:            sec.CRN,
			Subject:        sec.Subject,
			CourseNumber:   sec.CourseNumber,
			SectionNumber:  sec.SectionNumber,
			Title:          sec.Title,
			Credits:        sec.Credits,
			MeetingPattern: sec.MeetingPattern,
			Room:           sec.Room,
			Department:     sec.Department,
			Prerequisites:  sec.Prerequisites,
			InstructorName: sec.InstructorName,
			Status:         sec.Status,
		}
		if in.TermID == uuid.Nil {
			in.TermID = *termID
		}
		row, err := repoCatalog.UpsertSection(ctx, pool, conn.OrgID, in)
		if err != nil {
			syncErrs = append(syncErrs, repoCatalog.SyncError{RecordID: sec.SISSectionID, Message: err.Error()})
			continue
		}
		sectionsSynced++

		if row.LMSCourseID == nil && row.Status == repoCatalog.StatusActive && creatorID != nil {
			title := shellTitle(row, termName)
			course, err := repoCourse.CreateCourse(ctx, pool, *creatorID, title, "", "traditional", nil, &row.TermID, nil)
			if err != nil {
				syncErrs = append(syncErrs, repoCatalog.SyncError{RecordID: sec.SISSectionID, Message: "shell create: " + err.Error()})
				continue
			}
			courseID, parseErr := uuid.Parse(course.ID)
			if parseErr != nil {
				syncErrs = append(syncErrs, repoCatalog.SyncError{RecordID: sec.SISSectionID, Message: "shell link: invalid course id"})
				continue
			}
			if err := repoCatalog.LinkLMSShell(ctx, pool, row.ID, courseID); err != nil {
				syncErrs = append(syncErrs, repoCatalog.SyncError{RecordID: sec.SISSectionID, Message: "shell link: " + err.Error()})
				continue
			}
			shellsCreated++
		} else if row.LMSCourseID != nil {
			shellsUpdated++
		}
	}

	return sectionsSynced, shellsCreated, shellsUpdated, syncErrs
}

func activeTermForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*uuid.UUID, string, error) {
	terms, err := repoTerms.ListByOrg(ctx, pool, orgID)
	if err != nil {
		return nil, "", err
	}
	for _, t := range terms {
		if t.Status == "active" {
			id, err := uuid.Parse(t.ID)
			if err != nil {
				continue
			}
			return &id, t.Name, nil
		}
	}
	if len(terms) > 0 {
		id, err := uuid.Parse(terms[0].ID)
		if err == nil {
			return &id, terms[0].Name, nil
		}
	}
	return nil, "", nil
}

func shellTitle(s *repoCatalog.Section, termName string) string {
	parts := []string{s.Subject, s.CourseNumber}
	if s.SectionNumber != nil && strings.TrimSpace(*s.SectionNumber) != "" {
		parts = append(parts, "—", strings.TrimSpace(*s.SectionNumber))
	}
	if termName != "" {
		parts = append(parts, "—", termName)
	}
	if len(parts) >= 2 {
		return strings.Join(parts, " ")
	}
	return s.Title
}

// SeedDemoRegistration inserts a demo registration for e2e when user has no schedule data.
func SeedDemoRegistration(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID) error {
	sections, err := repoCatalog.ListSections(ctx, pool, orgID, repoCatalog.ListFilter{Limit: 1})
	if err != nil || len(sections) == 0 {
		return fmt.Errorf("no catalog sections to seed")
	}
	sec := sections[0]
	prereq := make([]repoCatalog.PrereqStatus, 0, len(sec.Prerequisites))
	for _, p := range sec.Prerequisites {
		prereq = append(prereq, repoCatalog.PrereqStatus{Code: p.Code, Status: "met"})
	}
	return repoCatalog.UpsertRegistration(ctx, pool, orgID, userID, sec.ID, repoCatalog.RegRegistered, prereq)
}
