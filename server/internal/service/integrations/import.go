package integrations

import (
	"context"
	"errors"

	"github.com/google/uuid"

	integrationsrepo "github.com/lextures/lextures/server/internal/repos/integrations"
)

// Enroller materializes imported roster members into a Lextures course. It is an
// injected seam so the integrations service stays decoupled from the enrollment
// subsystem (and so the import pipeline is unit-testable). A nil Enroller makes
// Import a non-mutating preview that only computes and records the diff.
type Enroller interface {
	// EnrollMember enrolls one external member into a course, returning whether a
	// new enrollment/invitation was created (false if the member already existed).
	EnrollMember(ctx context.Context, courseID uuid.UUID, member ExternalMember) (created bool, err error)
}

// ImportRequest describes a one-time Google Classroom import.
type ImportRequest struct {
	OrgID            uuid.UUID
	ConnectionID     uuid.UUID
	LexturesCourseID uuid.UUID
	ExternalCourseID string
	// SyncRoster persists a recurring roster-sync link when true.
	SyncRoster        bool
	SyncIntervalHours int16
}

// ImportResult summarizes an import for the audit log and UI.
type ImportResult struct {
	Diff            RosterDiff           `json:"diff"`
	Assignments     []ExternalAssignment `json:"assignments"`
	RecordsImported int                  `json:"recordsImported"`
	RecordsSkipped  int                  `json:"recordsSkipped"`
	AssignmentCount int                  `json:"assignmentCount"`
	LinkID          uuid.UUID            `json:"linkId"`
}

// Preview computes the roster diff and assignment list for an external course
// without mutating enrollments — used by the import wizard's diff step (FR-8).
func (s *Service) Preview(ctx context.Context, orgID uuid.UUID, req ImportRequest) (ImportResult, error) {
	conn, err := integrationsrepo.Get(ctx, s.Pool, orgID, req.ConnectionID)
	if err != nil {
		return ImportResult{}, err
	}
	if conn.Provider != string(ProviderGoogleClassroom) {
		return ImportResult{}, errors.New("integrations: import is only supported for Google Classroom")
	}
	token, err := s.freshAccessToken(ctx, conn)
	if err != nil {
		return ImportResult{}, err
	}
	members, err := s.Classroom.ListMembers(ctx, token, req.ExternalCourseID)
	if err != nil {
		return ImportResult{}, err
	}
	assignments, err := s.Classroom.ListCourseWork(ctx, token, req.ExternalCourseID)
	if err != nil {
		return ImportResult{}, err
	}
	current, err := integrationsrepo.CurrentCourseEmails(ctx, s.Pool, req.LexturesCourseID)
	if err != nil {
		return ImportResult{}, err
	}
	diff := ComputeRosterDiff(members, current)
	return ImportResult{
		Diff:            diff,
		Assignments:     assignments,
		AssignmentCount: len(assignments),
	}, nil
}

// Import performs a one-time import: computes the diff, enrolls added members via
// the injected Enroller, persists the external-course link (for recurring sync),
// and marks the connection synced. The returned result feeds the audit log.
func (s *Service) Import(ctx context.Context, orgID uuid.UUID, enroller Enroller, req ImportRequest) (ImportResult, error) {
	preview, err := s.Preview(ctx, orgID, req)
	if err != nil {
		if markErr := integrationsrepo.MarkSyncError(ctx, s.Pool, req.ConnectionID, err.Error()); markErr != nil {
			return ImportResult{}, errors.Join(err, markErr)
		}
		return ImportResult{}, err
	}
	result := preview
	if enroller != nil {
		for _, m := range preview.Diff.Added {
			created, err := enroller.EnrollMember(ctx, req.LexturesCourseID, m)
			if err != nil {
				result.RecordsSkipped++
				continue
			}
			if created {
				result.RecordsImported++
			} else {
				result.RecordsSkipped++
			}
		}
	} else {
		// Preview-only mode: report the would-be additions as the import count.
		result.RecordsImported = len(preview.Diff.Added)
	}

	link, err := integrationsrepo.UpsertLink(ctx, s.Pool, integrationsrepo.LinkParams{
		LexturesCourseID:  req.LexturesCourseID,
		ConnectionID:      req.ConnectionID,
		ExternalCourseID:  req.ExternalCourseID,
		SyncRoster:        req.SyncRoster,
		SyncIntervalHours: req.SyncIntervalHours,
	})
	if err != nil {
		return ImportResult{}, err
	}
	result.LinkID = link.ID

	now := s.now()
	if err := integrationsrepo.MarkSynced(ctx, s.Pool, req.ConnectionID, now); err != nil {
		return ImportResult{}, err
	}
	if err := integrationsrepo.MarkLinkSynced(ctx, s.Pool, link.ID, now); err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// SyncStatus is the projection returned by the sync-status endpoint (AC-5).
type SyncStatus struct {
	ConnectionID  uuid.UUID  `json:"connectionId"`
	Provider      string     `json:"provider"`
	LastSyncedAt  *string    `json:"lastSyncedAt,omitempty"`
	LastSyncError *string    `json:"lastSyncError,omitempty"`
	Stale         bool       `json:"stale"`
	Links         []LinkView `json:"links"`
}

// LinkView is a redacted external-course-link projection.
type LinkView struct {
	ID                uuid.UUID `json:"id"`
	ExternalCourseID  string    `json:"externalCourseId"`
	SyncRoster        bool      `json:"syncRoster"`
	SyncIntervalHours int16     `json:"syncIntervalHours"`
	LastSyncedAt      *string   `json:"lastSyncedAt,omitempty"`
}

// SyncStatusFor builds the sync-status view for a connection, flagging it stale
// when the last successful sync is older than the staleness threshold (12 h).
func (s *Service) SyncStatusFor(ctx context.Context, orgID, connID uuid.UUID) (SyncStatus, error) {
	conn, err := integrationsrepo.Get(ctx, s.Pool, orgID, connID)
	if err != nil {
		return SyncStatus{}, err
	}
	links, err := integrationsrepo.ListLinksByConnection(ctx, s.Pool, connID)
	if err != nil {
		return SyncStatus{}, err
	}
	const staleAfterHours = 12
	status := SyncStatus{
		ConnectionID:  conn.ID,
		Provider:      conn.Provider,
		LastSyncError: conn.LastSyncError,
		Links:         make([]LinkView, 0, len(links)),
	}
	if conn.LastSyncedAt != nil {
		v := conn.LastSyncedAt.UTC().Format("2006-01-02T15:04:05Z")
		status.LastSyncedAt = &v
		status.Stale = s.now().Sub(*conn.LastSyncedAt).Hours() > staleAfterHours
	} else {
		status.Stale = true
	}
	for _, l := range links {
		lv := LinkView{
			ID:                l.ID,
			ExternalCourseID:  l.ExternalCourseID,
			SyncRoster:        l.SyncRoster,
			SyncIntervalHours: l.SyncIntervalHours,
		}
		if l.LastSyncedAt != nil {
			v := l.LastSyncedAt.UTC().Format("2006-01-02T15:04:05Z")
			lv.LastSyncedAt = &v
		}
		status.Links = append(status.Links, lv)
	}
	return status, nil
}
