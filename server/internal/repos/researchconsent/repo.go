// Package researchconsent stores IRB consent studies and the append-only consent
// records ledger (plan 14.15).
package researchconsent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Study status values.
const (
	StatusDraft  = "draft"
	StatusActive = "active"
	StatusClosed = "closed"
)

// Consent decision values.
const (
	DecisionGranted   = "granted"
	DecisionDeclined  = "declined"
	DecisionWithdrawn = "withdrawn"
)

// Study is one consent_studies row.
type Study struct {
	ID             uuid.UUID       `json:"id"`
	OrgID          uuid.UUID       `json:"orgId"`
	ResearcherID   uuid.UUID       `json:"researcherId"`
	Title          string          `json:"title"`
	IRBProtocol    string          `json:"irbProtocol"`
	ConsentText    string          `json:"consentText"`
	DataUseDesc    string          `json:"dataUseDescription"`
	TargetCriteria json.RawMessage `json:"targetCriteria"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// TargetCriteria is the parsed shape of consent_studies.target_criteria.
type TargetCriteria struct {
	CourseIDs []uuid.UUID `json:"courseIds"`
}

// Record is one consent_records row.
type Record struct {
	ID        uuid.UUID `json:"id"`
	StudyID   uuid.UUID `json:"studyId"`
	UserID    uuid.UUID `json:"userId"`
	Decision  string    `json:"decision"`
	IPAddress *string   `json:"ipAddress,omitempty"`
	UserAgent *string   `json:"userAgent,omitempty"`
	HMAC      *string   `json:"hmac,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// StudyConsent pairs a study with the viewer's latest decision (or nil if none).
type StudyConsent struct {
	Study          Study   `json:"study"`
	LatestDecision *string `json:"latestDecision,omitempty"`
	DecidedAt      *string `json:"decidedAt,omitempty"`
}

// CreateStudy inserts a new consent study (status draft by default).
func CreateStudy(ctx context.Context, pool *pgxpool.Pool, s Study) (*Study, error) {
	if len(s.TargetCriteria) == 0 {
		s.TargetCriteria = json.RawMessage(`{}`)
	}
	status := s.Status
	if status == "" {
		status = StatusDraft
	}
	var out Study
	err := pool.QueryRow(ctx, `
INSERT INTO research.consent_studies
    (org_id, researcher_id, title, irb_protocol, consent_text, data_use_desc, target_criteria, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, org_id, researcher_id, title, irb_protocol, consent_text, data_use_desc, target_criteria, status, created_at
`, s.OrgID, s.ResearcherID, s.Title, s.IRBProtocol, s.ConsentText, s.DataUseDesc, s.TargetCriteria, status).Scan(
		&out.ID, &out.OrgID, &out.ResearcherID, &out.Title, &out.IRBProtocol,
		&out.ConsentText, &out.DataUseDesc, &out.TargetCriteria, &out.Status, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetStudy returns a single study by id, or nil if not found.
func GetStudy(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Study, error) {
	var s Study
	err := pool.QueryRow(ctx, `
SELECT id, org_id, researcher_id, title, irb_protocol, consent_text, data_use_desc, target_criteria, status, created_at
FROM research.consent_studies
WHERE id = $1
`, id).Scan(
		&s.ID, &s.OrgID, &s.ResearcherID, &s.Title, &s.IRBProtocol,
		&s.ConsentText, &s.DataUseDesc, &s.TargetCriteria, &s.Status, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListStudiesForOrg returns all studies for an org, newest first.
func ListStudiesForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Study, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, researcher_id, title, irb_protocol, consent_text, data_use_desc, target_criteria, status, created_at
FROM research.consent_studies
WHERE org_id = $1
ORDER BY created_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStudies(rows)
}

func scanStudies(rows pgx.Rows) ([]Study, error) {
	var out []Study
	for rows.Next() {
		var s Study
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.ResearcherID, &s.Title, &s.IRBProtocol,
			&s.ConsentText, &s.DataUseDesc, &s.TargetCriteria, &s.Status, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateStudy applies non-nil fields to a study and returns the updated row.
func UpdateStudy(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	title, irbProtocol, consentText, dataUseDesc, status *string,
	targetCriteria json.RawMessage,
) (*Study, error) {
	var s Study
	err := pool.QueryRow(ctx, `
UPDATE research.consent_studies
SET
    title = COALESCE($2, title),
    irb_protocol = COALESCE($3, irb_protocol),
    consent_text = COALESCE($4, consent_text),
    data_use_desc = COALESCE($5, data_use_desc),
    status = COALESCE($6, status),
    target_criteria = COALESCE($7, target_criteria)
WHERE id = $1
RETURNING id, org_id, researcher_id, title, irb_protocol, consent_text, data_use_desc, target_criteria, status, created_at
`, id, title, irbProtocol, consentText, dataUseDesc, status, targetCriteria).Scan(
		&s.ID, &s.OrgID, &s.ResearcherID, &s.Title, &s.IRBProtocol,
		&s.ConsentText, &s.DataUseDesc, &s.TargetCriteria, &s.Status, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// InsertRecord appends a consent decision to the immutable ledger.
func InsertRecord(ctx context.Context, pool *pgxpool.Pool, r Record) (*Record, error) {
	var out Record
	err := pool.QueryRow(ctx, `
INSERT INTO research.consent_records (study_id, user_id, decision, ip_address, user_agent, hmac)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, study_id, user_id, decision, host(ip_address), user_agent, hmac, created_at
`, r.StudyID, r.UserID, r.Decision, r.IPAddress, r.UserAgent, r.HMAC).Scan(
		&out.ID, &out.StudyID, &out.UserID, &out.Decision, &out.IPAddress, &out.UserAgent, &out.HMAC, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// LatestDecision returns the user's most recent decision for a study, or nil.
func LatestDecision(ctx context.Context, pool *pgxpool.Pool, studyID, userID uuid.UUID) (*Record, error) {
	var r Record
	err := pool.QueryRow(ctx, `
SELECT id, study_id, user_id, decision, host(ip_address), user_agent, hmac, created_at
FROM research.consent_records
WHERE study_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT 1
`, studyID, userID).Scan(
		&r.ID, &r.StudyID, &r.UserID, &r.Decision, &r.IPAddress, &r.UserAgent, &r.HMAC, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRecordsForStudy returns the full audit trail for a study, newest first
// (privacy-officer / audit view).
func ListRecordsForStudy(ctx context.Context, pool *pgxpool.Pool, studyID uuid.UUID) ([]Record, error) {
	rows, err := pool.Query(ctx, `
SELECT id, study_id, user_id, decision, host(ip_address), user_agent, hmac, created_at
FROM research.consent_records
WHERE study_id = $1
ORDER BY created_at DESC
`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(
			&r.ID, &r.StudyID, &r.UserID, &r.Decision, &r.IPAddress, &r.UserAgent, &r.HMAC, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListHistoryForUser returns every consent decision a user has made, newest first.
func ListHistoryForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Record, error) {
	rows, err := pool.Query(ctx, `
SELECT id, study_id, user_id, decision, host(ip_address), user_agent, hmac, created_at
FROM research.consent_records
WHERE user_id = $1
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(
			&r.ID, &r.StudyID, &r.UserID, &r.Decision, &r.IPAddress, &r.UserAgent, &r.HMAC, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ConsentRate counts distinct users by their latest decision for a study.
type ConsentRate struct {
	Granted   int `json:"granted"`
	Declined  int `json:"declined"`
	Withdrawn int `json:"withdrawn"`
}

// GetConsentRate aggregates the latest decision per user for a study.
func GetConsentRate(ctx context.Context, pool *pgxpool.Pool, studyID uuid.UUID) (ConsentRate, error) {
	var cr ConsentRate
	rows, err := pool.Query(ctx, `
SELECT decision, COUNT(*)::int
FROM (
    SELECT DISTINCT ON (user_id) user_id, decision
    FROM research.consent_records
    WHERE study_id = $1
    ORDER BY user_id, created_at DESC
) latest
GROUP BY decision
`, studyID)
	if err != nil {
		return cr, err
	}
	defer rows.Close()
	for rows.Next() {
		var decision string
		var n int
		if err := rows.Scan(&decision, &n); err != nil {
			return cr, err
		}
		switch decision {
		case DecisionGranted:
			cr.Granted = n
		case DecisionDeclined:
			cr.Declined = n
		case DecisionWithdrawn:
			cr.Withdrawn = n
		}
	}
	return cr, rows.Err()
}

// Participant is one consenting student in an export.
type Participant struct {
	UserID      uuid.UUID `json:"userId"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"displayName,omitempty"`
	ConsentedAt string    `json:"consentedAt"`
}

// ExportConsenting returns the participants whose latest decision for a study is
// 'granted' (FR-5 export gate). Withdrawal or decline excludes the user.
func ExportConsenting(ctx context.Context, pool *pgxpool.Pool, studyID uuid.UUID) ([]Participant, error) {
	rows, err := pool.Query(ctx, `
SELECT latest.user_id, u.email, u.display_name, latest.created_at
FROM (
    SELECT DISTINCT ON (user_id) user_id, decision, created_at
    FROM research.consent_records
    WHERE study_id = $1
    ORDER BY user_id, created_at DESC
) latest
INNER JOIN "user".users u ON u.id = latest.user_id
WHERE latest.decision = 'granted'
ORDER BY u.email
`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Participant
	for rows.Next() {
		var p Participant
		var display *string
		var consentedAt time.Time
		if err := rows.Scan(&p.UserID, &p.Email, &display, &consentedAt); err != nil {
			return nil, err
		}
		p.DisplayName = display
		p.ConsentedAt = consentedAt.UTC().Format(time.RFC3339)
		out = append(out, p)
	}
	return out, rows.Err()
}

// PendingStudiesForUser returns active studies that target the user and for
// which the user has not yet recorded a decision.
func PendingStudiesForUser(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID) ([]Study, error) {
	rows, err := pool.Query(ctx, `
SELECT s.id, s.org_id, s.researcher_id, s.title, s.irb_protocol, s.consent_text,
       s.data_use_desc, s.target_criteria, s.status, s.created_at
FROM research.consent_studies s
WHERE s.org_id = $1
  AND s.status = 'active'
  AND NOT EXISTS (
      SELECT 1 FROM research.consent_records r
      WHERE r.study_id = s.id AND r.user_id = $2
  )
  AND (
      -- No course targeting → targets everyone in the org.
      COALESCE(jsonb_array_length(s.target_criteria -> 'courseIds'), 0) = 0
      OR EXISTS (
          SELECT 1
          FROM course.course_enrollments e
          WHERE e.user_id = $2
            AND e.role = 'student'
            AND e.active = TRUE
            AND e.course_id IN (
                SELECT (value #>> '{}')::uuid
                FROM jsonb_array_elements(s.target_criteria -> 'courseIds')
            )
      )
  )
ORDER BY s.created_at DESC
`, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStudies(rows)
}

// UserTargetedBy reports whether a study currently targets the given user.
func UserTargetedBy(ctx context.Context, pool *pgxpool.Pool, s *Study, userID uuid.UUID) (bool, error) {
	var tc TargetCriteria
	if len(s.TargetCriteria) > 0 {
		_ = json.Unmarshal(s.TargetCriteria, &tc)
	}
	if len(tc.CourseIDs) == 0 {
		return true, nil
	}
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments e
    WHERE e.user_id = $1 AND e.role = 'student' AND e.active = TRUE
      AND e.course_id = ANY($2)
)
`, userID, tc.CourseIDs).Scan(&exists)
	return exists, err
}
