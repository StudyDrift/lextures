// Package managedlearners implements parent-created managed learner accounts (Homeschool #640).
package managedlearners

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/auth"
	appcrypto "github.com/lextures/lextures/server/internal/crypto"
	impersonationrepo "github.com/lextures/lextures/server/internal/repos/impersonation"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/coppa"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	// MaxActiveManagedPerParent caps active managed learners per parent account.
	MaxActiveManagedPerParent = 8
	// UnusablePasswordHash prevents credential login (same pattern as system users).
	UnusablePasswordHash = "!"
	relationshipDefault  = "parent"
	managedLearnerTTL    = 8 * time.Hour
)

var (
	ErrForbidden       = errors.New("managedlearners: forbidden")
	ErrNotFound        = errors.New("managedlearners: not found")
	ErrInvalidInput    = errors.New("managedlearners: invalid input")
	ErrCapExceeded     = errors.New("managedlearners: managed learner cap exceeded")
	ErrFeatureDisabled = errors.New("managedlearners: feature disabled")
	ErrStoreDown       = errors.New("managedlearners: token store unavailable")
	ErrActorBlocked    = errors.New("managedlearners: actor cannot manage dependents")
)

// Dependent is a managed learner summary for parent UI (synthetic email redacted).
type Dependent struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"displayName"`
	GradeLevel   *string   `json:"gradeLevel,omitempty"`
	Relationship string    `json:"relationship"`
	LinkID       string    `json:"linkId"`
	Status       string    `json:"status"`
	CoppaMinor   bool      `json:"coppaMinor"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CreateParams is input for creating a managed learner.
type CreateParams struct {
	ActorID      uuid.UUID
	DisplayName  string
	GradeLevel   *string
	Relationship string
	Under13      bool
}

// PatchParams updates mutable fields on a managed learner.
type PatchParams struct {
	ActorID     uuid.UUID
	DependentID uuid.UUID
	DisplayName *string
	GradeLevel  *string // empty string clears
}

// SessionStartResult is returned when Learn-as begins.
type SessionStartResult struct {
	Token     string
	ExpiresAt time.Time
	Target    DependentSummary
}

// DependentSummary is the learn-as target for clients.
type DependentSummary struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"displayName,omitempty"`
}

func normalizeDisplayName(s string) (string, error) {
	name := strings.TrimSpace(s)
	if name == "" {
		return "", fmt.Errorf("%w: displayName required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(name) > 120 {
		return "", fmt.Errorf("%w: displayName too long", ErrInvalidInput)
	}
	return name, nil
}

func normalizeRelationship(s string) string {
	r := strings.ToLower(strings.TrimSpace(s))
	switch r {
	case "parent", "guardian", "other":
		return r
	case "":
		return relationshipDefault
	default:
		return relationshipDefault
	}
}

func normalizeGradeLevel(p *string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	g := strings.TrimSpace(*p)
	if g == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(g) > 32 {
		return nil, fmt.Errorf("%w: gradeLevel too long", ErrInvalidInput)
	}
	return &g, nil
}

// assertCanManageDependents ensures the actor may create/manage managed learners.
func assertCanManageDependents(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID) (*user.Row, uuid.UUID, error) {
	row, err := user.FindByID(ctx, pool, actorID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if row == nil {
		return nil, uuid.Nil, ErrForbidden
	}
	if row.LoginBlocked || row.DeactivatedAt != nil {
		return nil, uuid.Nil, ErrActorBlocked
	}
	switch row.AccountType {
	case user.AccountTypeManaged, user.AccountTypeSystem:
		return nil, uuid.Nil, ErrActorBlocked
	}
	orgID, err := organization.OrgIDForUser(ctx, pool, actorID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return row, orgID, nil
}

func countActiveManaged(ctx context.Context, pool *pgxpool.Pool, parentID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
WHERE l.parent_user_id = $1
  AND l.status = 'active'
  AND u.account_type = 'managed'
  AND u.deactivated_at IS NULL
`, parentID).Scan(&n)
	return n, err
}

// List returns active managed dependents for the parent (email redacted).
func List(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID) ([]Dependent, error) {
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT u.id::text, COALESCE(u.display_name, ''), u.grade_level, l.relationship, l.id::text, l.status,
       u.coppa_minor, l.linked_at
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
WHERE l.parent_user_id = $1 AND l.org_id = $2
  AND l.status IN ('active', 'pending')
  AND u.account_type = 'managed'
ORDER BY u.display_name NULLS LAST, l.linked_at
`, actorID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Dependent, 0)
	for rows.Next() {
		var d Dependent
		var gl *string
		if err := rows.Scan(&d.ID, &d.DisplayName, &gl, &d.Relationship, &d.LinkID, &d.Status, &d.CoppaMinor, &d.CreatedAt); err != nil {
			return nil, err
		}
		d.GradeLevel = gl
		out = append(out, d)
	}
	return out, rows.Err()
}

// Create provisions a managed learner + active parent_student_links row.
func Create(ctx context.Context, pool *pgxpool.Pool, p CreateParams) (*Dependent, error) {
	actor, orgID, err := assertCanManageDependents(ctx, pool, p.ActorID)
	if err != nil {
		return nil, err
	}
	name, err := normalizeDisplayName(p.DisplayName)
	if err != nil {
		return nil, err
	}
	gl, err := normalizeGradeLevel(p.GradeLevel)
	if err != nil {
		return nil, err
	}
	rel := normalizeRelationship(p.Relationship)

	n, err := countActiveManaged(ctx, pool, p.ActorID)
	if err != nil {
		return nil, err
	}
	if n >= MaxActiveManagedPerParent {
		return nil, ErrCapExceeded
	}

	childID := uuid.New()
	email := user.ManagedEmail(childID)

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
INSERT INTO "user".users (
  id, email, password_hash, display_name, account_type, org_id, login_blocked, grade_level
) VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7)
`, childID, email, UnusablePasswordHash, name, user.AccountTypeManaged, orgID, gl)
	if err != nil {
		return nil, fmt.Errorf("managedlearners: insert user: %w", err)
	}

	var linkID uuid.UUID
	var linkedAt time.Time
	err = tx.QueryRow(ctx, `
INSERT INTO "user".parent_student_links (org_id, parent_user_id, student_user_id, relationship, status, linked_by)
VALUES ($1, $2, $3, $4, 'active', $2)
RETURNING id, linked_at
`, orgID, p.ActorID, childID, rel).Scan(&linkID, &linkedAt)
	if err != nil {
		return nil, fmt.Errorf("managedlearners: insert link: %w", err)
	}

	coppaMinor := false
	if p.Under13 {
		coppaMinor = true
		encParent, err := appcrypto.EncryptString(strings.ToLower(strings.TrimSpace(actor.Email)))
		if err != nil {
			return nil, fmt.Errorf("managedlearners: encrypt parent email: %w", err)
		}
		_, err = tx.Exec(ctx, `
UPDATE "user".users
SET coppa_minor = TRUE,
    coppa_consent_status = 'approved',
    parent_email = $2
WHERE id = $1
`, childID, encParent)
		if err != nil {
			return nil, fmt.Errorf("managedlearners: flag coppa: %w", err)
		}
		_, err = tx.Exec(ctx, `
INSERT INTO compliance.coppa_consents
  (org_id, student_id, parent_email, consent_method, consented_at, ai_features_enabled)
VALUES ($1, $2, $3, $4, NOW(), FALSE)
`, orgID, childID, strings.ToLower(strings.TrimSpace(actor.Email)), string(coppa.ConsentMethodDirect))
		if err != nil {
			return nil, fmt.Errorf("managedlearners: coppa consent: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("managed_learner_created")
	return &Dependent{
		ID:           childID.String(),
		DisplayName:  name,
		GradeLevel:   gl,
		Relationship: rel,
		LinkID:       linkID.String(),
		Status:       "active",
		CoppaMinor:   coppaMinor,
		CreatedAt:    linkedAt.UTC(),
	}, nil
}

// Patch renames or updates grade for a managed dependent owned by the actor.
func Patch(ctx context.Context, pool *pgxpool.Pool, p PatchParams) (*Dependent, error) {
	_, orgID, err := assertCanManageDependents(ctx, pool, p.ActorID)
	if err != nil {
		return nil, err
	}
	ok, err := ownsManaged(ctx, pool, orgID, p.ActorID, p.DependentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	sets := make([]string, 0, 2)
	args := []any{p.DependentID}
	argN := 2
	if p.DisplayName != nil {
		name, err := normalizeDisplayName(*p.DisplayName)
		if err != nil {
			return nil, err
		}
		sets = append(sets, fmt.Sprintf("display_name = $%d", argN))
		args = append(args, name)
		argN++
	}
	if p.GradeLevel != nil {
		gl, err := normalizeGradeLevel(p.GradeLevel)
		if err != nil {
			return nil, err
		}
		sets = append(sets, fmt.Sprintf("grade_level = $%d", argN))
		args = append(args, gl)
		argN++
	}
	if len(sets) == 0 {
		return getDependent(ctx, pool, orgID, p.ActorID, p.DependentID)
	}
	q := `UPDATE "user".users SET ` + strings.Join(sets, ", ") + ` WHERE id = $1 AND account_type = 'managed'`
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return getDependent(ctx, pool, orgID, p.ActorID, p.DependentID)
}

// Deactivate soft-deactivates the managed user and revokes the parent link.
func Deactivate(ctx context.Context, pool *pgxpool.Pool, actorID, dependentID uuid.UUID) error {
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return err
	}
	ok, err := ownsManaged(ctx, pool, orgID, actorID, dependentID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
UPDATE "user".users
SET deactivated_at = COALESCE(deactivated_at, NOW()), login_blocked = TRUE
WHERE id = $1 AND account_type = 'managed'
`, dependentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
UPDATE "user".parent_student_links
SET status = 'revoked'
WHERE parent_user_id = $1 AND student_user_id = $2 AND org_id = $3 AND status <> 'revoked'
`, actorID, dependentID, orgID)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	telemetry.RecordBusinessEvent("managed_learner_deactivated")
	return nil
}

func ownsManaged(ctx context.Context, pool *pgxpool.Pool, orgID, parentID, childID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM "user".parent_student_links l
  INNER JOIN "user".users u ON u.id = l.student_user_id
  WHERE l.org_id = $1 AND l.parent_user_id = $2 AND l.student_user_id = $3
    AND l.status = 'active'
    AND u.account_type = 'managed'
    AND u.deactivated_at IS NULL
)
`, orgID, parentID, childID).Scan(&exists)
	return exists, err
}

func getDependent(ctx context.Context, pool *pgxpool.Pool, orgID, parentID, childID uuid.UUID) (*Dependent, error) {
	var d Dependent
	var gl *string
	err := pool.QueryRow(ctx, `
SELECT u.id::text, COALESCE(u.display_name, ''), u.grade_level, l.relationship, l.id::text, l.status,
       u.coppa_minor, l.linked_at
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
WHERE l.parent_user_id = $1 AND l.org_id = $2 AND l.student_user_id = $3
  AND u.account_type = 'managed'
`, parentID, orgID, childID).Scan(&d.ID, &d.DisplayName, &gl, &d.Relationship, &d.LinkID, &d.Status, &d.CoppaMinor, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.GradeLevel = gl
	return &d, nil
}

// CanEnrollManaged reports whether actor may enroll childID as a managed learner (active link).
func CanEnrollManaged(ctx context.Context, pool *pgxpool.Pool, actorID, childID, courseOrgID uuid.UUID) (bool, error) {
	row, err := user.FindByID(ctx, pool, childID)
	if err != nil || row == nil {
		return false, err
	}
	if row.AccountType != user.AccountTypeManaged || row.DeactivatedAt != nil {
		return false, nil
	}
	childOrg, err := organization.OrgIDForUser(ctx, pool, childID)
	if err != nil || childOrg != courseOrgID {
		return false, err
	}
	ln, err := parentlinks.ActiveLinkBetween(ctx, pool, courseOrgID, actorID, childID)
	if err != nil {
		return false, err
	}
	return ln != nil && ln.Status == "active", nil
}

// StartSession issues a Learn-as managed_learner JWT for an active managed link.
func StartSession(
	ctx context.Context,
	pool *pgxpool.Pool,
	signer *auth.JWTSigner,
	actorID, dependentID uuid.UUID,
) (SessionStartResult, error) {
	if pool == nil || signer == nil {
		return SessionStartResult{}, ErrStoreDown
	}
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return SessionStartResult{}, err
	}
	ok, err := ownsManaged(ctx, pool, orgID, actorID, dependentID)
	if err != nil {
		return SessionStartResult{}, err
	}
	if !ok {
		return SessionStartResult{}, ErrNotFound
	}
	row, err := user.FindByID(ctx, pool, dependentID)
	if err != nil || row == nil {
		return SessionStartResult{}, ErrNotFound
	}
	orgSlug, err := organization.OrgSlugForUser(ctx, pool, dependentID)
	if err != nil {
		return SessionStartResult{}, err
	}
	tok, sess, err := signer.SignManagedLearner(
		actorID.String(), dependentID.String(), row.Email, orgID.String(), orgSlug,
	)
	if err != nil {
		return SessionStartResult{}, fmt.Errorf("managedlearners: sign: %w", err)
	}
	if err := impersonationrepo.Insert(ctx, pool, sess.JTI, actorID, dependentID, sess.ExpiresAt); err != nil {
		return SessionStartResult{}, ErrStoreDown
	}
	telemetry.RecordBusinessEvent("managed_learner_session_start")
	return SessionStartResult{
		Token:     tok,
		ExpiresAt: sess.ExpiresAt,
		Target: DependentSummary{
			ID:          row.ID,
			DisplayName: row.DisplayName,
		},
	}, nil
}

// EndSessionParams ends a Learn-as session.
type EndSessionParams struct {
	JTI          string
	ActorID      uuid.UUID
	TargetUserID uuid.UUID
}

// EndSession revokes the Learn-as token.
func EndSession(ctx context.Context, pool *pgxpool.Pool, p EndSessionParams) error {
	if pool == nil {
		return ErrStoreDown
	}
	if err := impersonationrepo.Revoke(ctx, pool, p.JTI, time.Now().UTC()); err != nil {
		return ErrStoreDown
	}
	telemetry.RecordBusinessEvent("managed_learner_session_end")
	return nil
}

// TTL exported for tests / JWT signer alignment.
func SessionTTL() time.Duration { return managedLearnerTTL }
