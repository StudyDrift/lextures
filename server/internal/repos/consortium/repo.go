// Package consortium stores consortium sharing agreements and cross-org enrollment helpers (plan 14.18).
package consortium

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusPending    = "pending"
	StatusActive     = "active"
	StatusTerminated = "terminated"
)

// Agreement is one tenant.consortium_agreements row.
type Agreement struct {
	ID         uuid.UUID  `json:"id"`
	HostOrgID  uuid.UUID  `json:"hostOrgId"`
	GuestOrgID uuid.UUID  `json:"guestOrgId"`
	Status     string     `json:"status"`
	SignedAt   *time.Time `json:"signedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	HostOrgName  string   `json:"hostOrgName,omitempty"`
	GuestOrgName string   `json:"guestOrgName,omitempty"`
}

// SharedCourse is a consortium-shareable course visible to a guest institution.
type SharedCourse struct {
	ID          uuid.UUID `json:"id"`
	CourseCode  string    `json:"courseCode"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	HostOrgID   uuid.UUID `json:"hostOrgId"`
	HostOrgName string    `json:"hostOrgName"`
}

// EnrollmentReportRow is one row in the consortium enrollment report.
type EnrollmentReportRow struct {
	HostOrgID    uuid.UUID `json:"hostOrgId"`
	HostOrgName  string    `json:"hostOrgName"`
	GuestOrgID   uuid.UUID `json:"guestOrgId"`
	GuestOrgName string    `json:"guestOrgName"`
	CourseCode   string    `json:"courseCode"`
	CourseTitle  string    `json:"courseTitle"`
	Headcount    int       `json:"headcount"`
}

// CreateAgreement inserts a new agreement (default status pending).
func CreateAgreement(ctx context.Context, pool *pgxpool.Pool, hostOrgID, guestOrgID uuid.UUID, status string) (*Agreement, error) {
	if hostOrgID == guestOrgID {
		return nil, errors.New("host and guest org must differ")
	}
	if status == "" {
		status = StatusPending
	}
	var out Agreement
	var signedAt *time.Time
	if status == StatusActive {
		now := time.Now().UTC()
		signedAt = &now
	}
	err := pool.QueryRow(ctx, `
INSERT INTO tenant.consortium_agreements (host_org_id, guest_org_id, status, signed_at)
VALUES ($1, $2, $3, $4)
RETURNING id, host_org_id, guest_org_id, status, signed_at, expires_at, created_at
`, hostOrgID, guestOrgID, status, signedAt).Scan(
		&out.ID, &out.HostOrgID, &out.GuestOrgID, &out.Status, &out.SignedAt, &out.ExpiresAt, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAgreement returns an agreement by id, or nil if not found.
func GetAgreement(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Agreement, error) {
	var a Agreement
	err := pool.QueryRow(ctx, `
SELECT ca.id, ca.host_org_id, ca.guest_org_id, ca.status, ca.signed_at, ca.expires_at, ca.created_at,
       ho.name, go.name
FROM tenant.consortium_agreements ca
INNER JOIN tenant.organizations ho ON ho.id = ca.host_org_id
INNER JOIN tenant.organizations go ON go.id = ca.guest_org_id
WHERE ca.id = $1
`, id).Scan(
		&a.ID, &a.HostOrgID, &a.GuestOrgID, &a.Status, &a.SignedAt, &a.ExpiresAt, &a.CreatedAt,
		&a.HostOrgName, &a.GuestOrgName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAgreementsForOrg returns agreements where org is host or guest.
func ListAgreementsForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Agreement, error) {
	rows, err := pool.Query(ctx, `
SELECT ca.id, ca.host_org_id, ca.guest_org_id, ca.status, ca.signed_at, ca.expires_at, ca.created_at,
       ho.name, go.name
FROM tenant.consortium_agreements ca
INNER JOIN tenant.organizations ho ON ho.id = ca.host_org_id
INNER JOIN tenant.organizations go ON go.id = ca.guest_org_id
WHERE ca.host_org_id = $1 OR ca.guest_org_id = $1
ORDER BY ca.created_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Agreement
	for rows.Next() {
		var a Agreement
		if err := rows.Scan(
			&a.ID, &a.HostOrgID, &a.GuestOrgID, &a.Status, &a.SignedAt, &a.ExpiresAt, &a.CreatedAt,
			&a.HostOrgName, &a.GuestOrgName,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAgreementStatus sets status and signed_at when activating.
func UpdateAgreementStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string) (*Agreement, error) {
	var out Agreement
	err := pool.QueryRow(ctx, `
UPDATE tenant.consortium_agreements
SET status = $2,
    signed_at = CASE WHEN $2 = 'active' THEN COALESCE(signed_at, NOW()) ELSE signed_at END
WHERE id = $1
RETURNING id, host_org_id, guest_org_id, status, signed_at, expires_at, created_at
`, id, status).Scan(
		&out.ID, &out.HostOrgID, &out.GuestOrgID, &out.Status, &out.SignedAt, &out.ExpiresAt, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ActiveAgreementExists is true when host and guest have a non-expired active agreement.
func ActiveAgreementExists(ctx context.Context, pool *pgxpool.Pool, hostOrgID, guestOrgID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM tenant.consortium_agreements ca
  WHERE ca.host_org_id = $1 AND ca.guest_org_id = $2
    AND ca.status = 'active'
    AND (ca.expires_at IS NULL OR ca.expires_at > NOW())
)
`, hostOrgID, guestOrgID).Scan(&ok)
	return ok, err
}

// ListShareableCoursesForGuest returns published consortium-shareable courses from partner hosts.
func ListShareableCoursesForGuest(ctx context.Context, pool *pgxpool.Pool, guestOrgID uuid.UUID) ([]SharedCourse, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.course_code, c.title, COALESCE(c.description, ''), c.org_id, ho.name
FROM course.courses c
INNER JOIN tenant.organizations ho ON ho.id = c.org_id
INNER JOIN tenant.consortium_agreements ca
  ON ca.host_org_id = c.org_id AND ca.guest_org_id = $1
WHERE ca.status = 'active'
  AND (ca.expires_at IS NULL OR ca.expires_at > NOW())
  AND c.consortium_shareable = true
  AND c.published = true
  AND c.archived = false
ORDER BY ho.name ASC, c.title ASC
`, guestOrgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SharedCourse
	for rows.Next() {
		var sc SharedCourse
		if err := rows.Scan(&sc.ID, &sc.CourseCode, &sc.Title, &sc.Description, &sc.HostOrgID, &sc.HostOrgName); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// CountPassbackEnrollments counts enrollments eligible for SIS passback for an org.
// Host org passback: home_org_id IS NULL. Guest org passback: home_org_id = orgID.
func CountPassbackEnrollments(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, guestPassback bool) (int, error) {
	var count int
	var err error
	if guestPassback {
		err = pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE ce.home_org_id = $1
  AND ce.role = 'student'
  AND ce.active
`, orgID).Scan(&count)
	} else {
		err = pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
WHERE c.org_id = $1
  AND ce.home_org_id IS NULL
  AND ce.role = 'student'
  AND ce.active
`, orgID).Scan(&count)
	}
	return count, err
}

// EnrollmentReport returns cross-institution headcounts for an agreement.
func EnrollmentReport(ctx context.Context, pool *pgxpool.Pool, agreementID uuid.UUID) ([]EnrollmentReportRow, error) {
	rows, err := pool.Query(ctx, `
SELECT ca.host_org_id, ho.name, ca.guest_org_id, go.name,
       c.course_code, c.title, COUNT(ce.id)::int
FROM tenant.consortium_agreements ca
INNER JOIN tenant.organizations ho ON ho.id = ca.host_org_id
INNER JOIN tenant.organizations go ON go.id = ca.guest_org_id
INNER JOIN course.courses c ON c.org_id = ca.host_org_id
INNER JOIN course.course_enrollments ce ON ce.course_id = c.id AND ce.home_org_id = ca.guest_org_id
WHERE ca.id = $1
  AND ce.active
  AND ce.role = 'student'
GROUP BY ca.host_org_id, ho.name, ca.guest_org_id, go.name, c.course_code, c.title
ORDER BY c.course_code ASC
`, agreementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EnrollmentReportRow
	for rows.Next() {
		var r EnrollmentReportRow
		if err := rows.Scan(
			&r.HostOrgID, &r.HostOrgName, &r.GuestOrgID, &r.GuestOrgName,
			&r.CourseCode, &r.CourseTitle, &r.Headcount,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
