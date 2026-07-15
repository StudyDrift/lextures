// Package badges persists competency micro-badge definitions and awards (plan B1).
package badges

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AwardSource distinguishes manual staff awards from automatic mastery awards.
type AwardSource string

const (
	AwardSourceManual AwardSource = "manual"
	AwardSourceAuto   AwardSource = "auto"
)

// Definition is one row in badges.badge_definitions.
type Definition struct {
	ID                uuid.UUID
	CourseID          uuid.UUID
	OutcomeID         *uuid.UUID
	SubOutcomeID      *uuid.UUID
	Slug              string
	Name              string
	Description       string
	CriteriaNarrative string
	ImageKey          *string
	Tags              []string
	AlignmentJSON     json.RawMessage
	AutoAward         bool
	CreatedBy         uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// AwardedBadge is one row in badges.awarded_badges.
type AwardedBadge struct {
	ID             uuid.UUID
	DefinitionID   uuid.UUID
	RecipientID    uuid.UUID
	AwardedBy      *uuid.UUID
	AwardSource    AwardSource
	EvidenceJSON   json.RawMessage
	CredentialJSON json.RawMessage
	Proof          json.RawMessage
	ShareSlug      string
	IsPublic       bool
	Revoked        bool
	RevokedReason  *string
	RevokedAt      *time.Time
	IssuedAt       time.Time
}

// BadgeProfile is one row in user.user_badge_profiles.
type BadgeProfile struct {
	UserID               uuid.UUID
	Handle               *string
	PagePublic           bool
	SearchIndexable      bool
	DisplayNameOverride  *string
	HideRealName         bool
	HandleChangedAt      *time.Time
	HandleChangeCount30d int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// PublicAward is a public list/detail projection (minimal PII).
type PublicAward struct {
	AwardedID         uuid.UUID
	DefinitionID      uuid.UUID
	Slug              string
	Name              string
	Description       string
	CriteriaNarrative string
	ImageKey          *string
	Tags              []string
	ShareSlug         string
	IssuedAt          time.Time
	CourseID          uuid.UUID
	CourseTitle       string
}

// CreateDefinitionInput is the payload for inserting a badge definition.
type CreateDefinitionInput struct {
	CourseID          uuid.UUID
	OutcomeID         *uuid.UUID
	SubOutcomeID      *uuid.UUID
	Slug              string
	Name              string
	Description       string
	CriteriaNarrative string
	ImageKey          *string
	Tags              []string
	AlignmentJSON     json.RawMessage
	AutoAward         bool
	CreatedBy         uuid.UUID
}

// UpdateDefinitionInput is a partial update for a definition.
type UpdateDefinitionInput struct {
	Name              *string
	Slug              *string
	Description       *string
	CriteriaNarrative *string
	ImageKey          *string
	ClearImageKey     bool
	Tags              *[]string
	AlignmentJSON     json.RawMessage
	AutoAward         *bool
	OutcomeID         *uuid.UUID
	ClearOutcomeID    bool
	SubOutcomeID      *uuid.UUID
	ClearSubOutcomeID bool
}

const definitionCols = `
id, course_id, outcome_id, sub_outcome_id, slug, name, description, criteria_narrative,
image_key, tags, alignment_json, auto_award, created_by, created_at, updated_at`

func scanDefinition(row pgx.Row) (*Definition, error) {
	var d Definition
	var tags []string
	err := row.Scan(
		&d.ID, &d.CourseID, &d.OutcomeID, &d.SubOutcomeID, &d.Slug, &d.Name, &d.Description, &d.CriteriaNarrative,
		&d.ImageKey, &tags, &d.AlignmentJSON, &d.AutoAward, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	d.Tags = tags
	return &d, nil
}

const awardCols = `
id, definition_id, recipient_id, awarded_by, award_source, evidence_json, credential_json, proof,
share_slug, is_public, revoked, revoked_reason, revoked_at, issued_at`

func scanAward(row pgx.Row) (*AwardedBadge, error) {
	var a AwardedBadge
	var source string
	err := row.Scan(
		&a.ID, &a.DefinitionID, &a.RecipientID, &a.AwardedBy, &source, &a.EvidenceJSON, &a.CredentialJSON, &a.Proof,
		&a.ShareSlug, &a.IsPublic, &a.Revoked, &a.RevokedReason, &a.RevokedAt, &a.IssuedAt,
	)
	if err != nil {
		return nil, err
	}
	a.AwardSource = AwardSource(source)
	return &a, nil
}

// CreateDefinition inserts a badge definition.
func CreateDefinition(ctx context.Context, pool *pgxpool.Pool, in CreateDefinitionInput) (*Definition, error) {
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	row := pool.QueryRow(ctx, `
INSERT INTO badges.badge_definitions (
    course_id, outcome_id, sub_outcome_id, slug, name, description, criteria_narrative,
    image_key, tags, alignment_json, auto_award, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING `+definitionCols, in.CourseID, in.OutcomeID, in.SubOutcomeID, in.Slug, in.Name, in.Description,
		in.CriteriaNarrative, in.ImageKey, tags, nullableJSON(in.AlignmentJSON), in.AutoAward, in.CreatedBy)
	return scanDefinition(row)
}

// GetDefinitionByID loads a definition by id.
func GetDefinitionByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Definition, error) {
	d, err := scanDefinition(pool.QueryRow(ctx, `SELECT `+definitionCols+` FROM badges.badge_definitions WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

// ListDefinitionsByCourse lists definitions for a course.
func ListDefinitionsByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]Definition, error) {
	rows, err := pool.Query(ctx, `
SELECT `+definitionCols+`
FROM badges.badge_definitions
WHERE course_id = $1
ORDER BY created_at DESC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// ListAutoAwardDefinitionsForOutcome returns auto-award definitions for an outcome in a course.
func ListAutoAwardDefinitionsForOutcome(ctx context.Context, pool *pgxpool.Pool, courseID, outcomeID uuid.UUID) ([]Definition, error) {
	rows, err := pool.Query(ctx, `
SELECT `+definitionCols+`
FROM badges.badge_definitions
WHERE course_id = $1 AND outcome_id = $2 AND auto_award = TRUE
`, courseID, outcomeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		d, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// UpdateDefinition applies a partial update.
func UpdateDefinition(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, in UpdateDefinitionInput) (*Definition, error) {
	existing, err := GetDefinitionByID(ctx, pool, id)
	if err != nil || existing == nil {
		return existing, err
	}
	name := existing.Name
	if in.Name != nil {
		name = *in.Name
	}
	slug := existing.Slug
	if in.Slug != nil {
		slug = *in.Slug
	}
	desc := existing.Description
	if in.Description != nil {
		desc = *in.Description
	}
	criteria := existing.CriteriaNarrative
	if in.CriteriaNarrative != nil {
		criteria = *in.CriteriaNarrative
	}
	imageKey := existing.ImageKey
	if in.ClearImageKey {
		imageKey = nil
	} else if in.ImageKey != nil {
		imageKey = in.ImageKey
	}
	tags := existing.Tags
	if in.Tags != nil {
		tags = *in.Tags
	}
	alignment := existing.AlignmentJSON
	if in.AlignmentJSON != nil {
		alignment = in.AlignmentJSON
	}
	autoAward := existing.AutoAward
	if in.AutoAward != nil {
		autoAward = *in.AutoAward
	}
	outcomeID := existing.OutcomeID
	if in.ClearOutcomeID {
		outcomeID = nil
	} else if in.OutcomeID != nil {
		outcomeID = in.OutcomeID
	}
	subOutcomeID := existing.SubOutcomeID
	if in.ClearSubOutcomeID {
		subOutcomeID = nil
	} else if in.SubOutcomeID != nil {
		subOutcomeID = in.SubOutcomeID
	}

	d, err := scanDefinition(pool.QueryRow(ctx, `
UPDATE badges.badge_definitions SET
    name = $2, slug = $3, description = $4, criteria_narrative = $5,
    image_key = $6, tags = $7, alignment_json = $8, auto_award = $9,
    outcome_id = $10, sub_outcome_id = $11, updated_at = NOW()
WHERE id = $1
RETURNING `+definitionCols, id, name, slug, desc, criteria, imageKey, tags, nullableJSON(alignment), autoAward, outcomeID, subOutcomeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

// DeleteDefinition removes a definition (cascades awards).
func DeleteDefinition(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM badges.badge_definitions WHERE id = $1`, id)
	return err
}

// SlugExistsInCourse reports whether a slug is taken in the course (optionally excluding an id).
func SlugExistsInCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM badges.badge_definitions
  WHERE course_id = $1 AND lower(slug) = lower($2) AND id <> $3
)`, courseID, slug, *excludeID).Scan(&exists)
		return exists, err
	}
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM badges.badge_definitions
  WHERE course_id = $1 AND lower(slug) = lower($2)
)`, courseID, slug).Scan(&exists)
	return exists, err
}

// CreateAwardInput is the payload for inserting an award (idempotent unique on definition+recipient).
type CreateAwardInput struct {
	DefinitionID   uuid.UUID
	RecipientID    uuid.UUID
	AwardedBy      *uuid.UUID
	AwardSource    AwardSource
	EvidenceJSON   json.RawMessage
	CredentialJSON json.RawMessage
	Proof          json.RawMessage
	ShareSlug      string
	IsPublic       bool
	IssuedAt       time.Time
}

// CreateAward inserts an award. On unique conflict returns the existing row with created=false.
func CreateAward(ctx context.Context, pool *pgxpool.Pool, in CreateAwardInput) (*AwardedBadge, bool, error) {
	source := string(in.AwardSource)
	if source == "" {
		source = string(AwardSourceManual)
	}
	issued := in.IssuedAt
	if issued.IsZero() {
		issued = time.Now().UTC()
	}
	a, err := scanAward(pool.QueryRow(ctx, `
INSERT INTO badges.awarded_badges (
    definition_id, recipient_id, awarded_by, award_source, evidence_json,
    credential_json, proof, share_slug, is_public, issued_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (definition_id, recipient_id) DO NOTHING
RETURNING `+awardCols, in.DefinitionID, in.RecipientID, in.AwardedBy, source, nullableJSON(in.EvidenceJSON),
		in.CredentialJSON, in.Proof, in.ShareSlug, in.IsPublic, issued))
	if err == nil {
		return a, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	existing, err := GetAwardByDefinitionAndRecipient(ctx, pool, in.DefinitionID, in.RecipientID)
	return existing, false, err
}

// GetAwardByID loads an award by id.
func GetAwardByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*AwardedBadge, error) {
	a, err := scanAward(pool.QueryRow(ctx, `SELECT `+awardCols+` FROM badges.awarded_badges WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// GetAwardByShareSlug loads an award by public share slug.
func GetAwardByShareSlug(ctx context.Context, pool *pgxpool.Pool, shareSlug string) (*AwardedBadge, error) {
	a, err := scanAward(pool.QueryRow(ctx, `SELECT `+awardCols+` FROM badges.awarded_badges WHERE share_slug = $1`, shareSlug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// GetAwardByDefinitionAndRecipient returns the award for idempotency checks.
func GetAwardByDefinitionAndRecipient(ctx context.Context, pool *pgxpool.Pool, defID, recipientID uuid.UUID) (*AwardedBadge, error) {
	a, err := scanAward(pool.QueryRow(ctx, `
SELECT `+awardCols+`
FROM badges.awarded_badges
WHERE definition_id = $1 AND recipient_id = $2
`, defID, recipientID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// ListAwardsByRecipient returns all awards for a learner.
func ListAwardsByRecipient(ctx context.Context, pool *pgxpool.Pool, recipientID uuid.UUID) ([]AwardedBadge, error) {
	rows, err := pool.Query(ctx, `
SELECT `+awardCols+`
FROM badges.awarded_badges
WHERE recipient_id = $1
ORDER BY issued_at DESC
`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AwardedBadge
	for rows.Next() {
		a, err := scanAward(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// ListAwardsByDefinition returns awards for a definition.
func ListAwardsByDefinition(ctx context.Context, pool *pgxpool.Pool, defID uuid.UUID) ([]AwardedBadge, error) {
	rows, err := pool.Query(ctx, `
SELECT `+awardCols+`
FROM badges.awarded_badges
WHERE definition_id = $1
ORDER BY issued_at DESC
`, defID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AwardedBadge
	for rows.Next() {
		a, err := scanAward(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// SetAwardPublic toggles per-badge public visibility.
func SetAwardPublic(ctx context.Context, pool *pgxpool.Pool, awardID uuid.UUID, isPublic bool) (*AwardedBadge, error) {
	a, err := scanAward(pool.QueryRow(ctx, `
UPDATE badges.awarded_badges SET is_public = $2 WHERE id = $1
RETURNING `+awardCols, awardID, isPublic))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// RevokeAward marks an award revoked.
func RevokeAward(ctx context.Context, pool *pgxpool.Pool, awardID uuid.UUID, reason string) (*AwardedBadge, error) {
	a, err := scanAward(pool.QueryRow(ctx, `
UPDATE badges.awarded_badges SET
    revoked = TRUE,
    revoked_reason = $2,
    revoked_at = NOW(),
    is_public = FALSE
WHERE id = $1
RETURNING `+awardCols, awardID, strings.TrimSpace(reason)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// GetProfile loads a badge profile for a user (nil if none).
func GetProfile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*BadgeProfile, error) {
	var p BadgeProfile
	err := pool.QueryRow(ctx, `
SELECT user_id, handle, page_public, search_indexable, display_name_override, hide_real_name,
       handle_changed_at, handle_change_count_30d, created_at, updated_at
FROM "user".user_badge_profiles
WHERE user_id = $1
`, userID).Scan(
		&p.UserID, &p.Handle, &p.PagePublic, &p.SearchIndexable, &p.DisplayNameOverride, &p.HideRealName,
		&p.HandleChangedAt, &p.HandleChangeCount30d, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// EnsureProfile creates a profile with a default opaque handle if missing.
func EnsureProfile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, defaultHandle string) (*BadgeProfile, error) {
	existing, err := GetProfile(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	handle := strings.TrimSpace(defaultHandle)
	if handle == "" {
		h, err := NewOpaqueHandle()
		if err != nil {
			return nil, err
		}
		handle = h
	}
	_, err = pool.Exec(ctx, `
INSERT INTO "user".user_badge_profiles (user_id, handle)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING
`, userID, handle)
	if err != nil {
		return nil, err
	}
	return GetProfile(ctx, pool, userID)
}

// UpdateProfileInput is a partial profile update.
type UpdateProfileInput struct {
	Handle              *string
	PagePublic          *bool
	SearchIndexable     *bool
	DisplayNameOverride *string
	ClearDisplayName    bool
	HideRealName        *bool
	// HandleChangeCount30d and HandleChangedAt are set by the service when handle changes.
	HandleChangeCount30d *int
	HandleChangedAt      *time.Time
}

// UpdateProfile applies a partial update; creates the row if missing.
func UpdateProfile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, in UpdateProfileInput) (*BadgeProfile, error) {
	if _, err := EnsureProfile(ctx, pool, userID, ""); err != nil {
		return nil, err
	}
	existing, err := GetProfile(ctx, pool, userID)
	if err != nil || existing == nil {
		return existing, err
	}

	handle := existing.Handle
	if in.Handle != nil {
		h := strings.TrimSpace(*in.Handle)
		if h == "" {
			handle = nil
		} else {
			handle = &h
		}
	}
	pagePublic := existing.PagePublic
	if in.PagePublic != nil {
		pagePublic = *in.PagePublic
	}
	searchIndexable := existing.SearchIndexable
	if in.SearchIndexable != nil {
		searchIndexable = *in.SearchIndexable
	}
	displayOverride := existing.DisplayNameOverride
	if in.ClearDisplayName {
		displayOverride = nil
	} else if in.DisplayNameOverride != nil {
		displayOverride = in.DisplayNameOverride
	}
	hideReal := existing.HideRealName
	if in.HideRealName != nil {
		hideReal = *in.HideRealName
	}
	changeCount := existing.HandleChangeCount30d
	if in.HandleChangeCount30d != nil {
		changeCount = *in.HandleChangeCount30d
	}
	changedAt := existing.HandleChangedAt
	if in.HandleChangedAt != nil {
		changedAt = in.HandleChangedAt
	}

	_, err = pool.Exec(ctx, `
UPDATE "user".user_badge_profiles SET
    handle = $2,
    page_public = $3,
    search_indexable = $4,
    display_name_override = $5,
    hide_real_name = $6,
    handle_change_count_30d = $7,
    handle_changed_at = $8,
    updated_at = NOW()
WHERE user_id = $1
`, userID, handle, pagePublic, searchIndexable, displayOverride, hideReal, changeCount, changedAt)
	if err != nil {
		return nil, err
	}
	return GetProfile(ctx, pool, userID)
}

// RecordHandleHistory stores an old handle for redirects.
func RecordHandleHistory(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, oldHandle string) error {
	h := strings.ToLower(strings.TrimSpace(oldHandle))
	if h == "" {
		return nil
	}
	_, err := pool.Exec(ctx, `
INSERT INTO "user".user_badge_handle_history (old_handle_lower, user_id)
VALUES ($1, $2)
ON CONFLICT (old_handle_lower) DO UPDATE SET user_id = EXCLUDED.user_id, released_at = NOW()
`, h, userID)
	return err
}

// ResolveHandle looks up current handle or redirect history.
// Returns (userID, currentHandle, redirected, notFound error path).
func ResolveHandle(ctx context.Context, pool *pgxpool.Pool, handle string) (userID uuid.UUID, currentHandle string, redirected bool, err error) {
	h := strings.ToLower(strings.TrimSpace(handle))
	if h == "" {
		return uuid.Nil, "", false, nil
	}
	var uid uuid.UUID
	var cur *string
	err = pool.QueryRow(ctx, `
SELECT user_id, handle FROM "user".user_badge_profiles WHERE handle_lower = $1
`, h).Scan(&uid, &cur)
	if err == nil {
		if cur != nil {
			return uid, *cur, false, nil
		}
		return uid, h, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, err
	}

	err = pool.QueryRow(ctx, `
SELECT h.user_id, p.handle
FROM "user".user_badge_handle_history h
JOIN "user".user_badge_profiles p ON p.user_id = h.user_id
WHERE h.old_handle_lower = $1
  AND h.released_at > NOW() - INTERVAL '90 days'
`, h).Scan(&uid, &cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, err
	}
	if cur == nil {
		return uid, "", true, nil
	}
	return uid, *cur, true, nil
}

// IsHandleTaken reports whether a handle is used by a different user or reserved.
func IsHandleTaken(ctx context.Context, pool *pgxpool.Pool, handle string, excludeUserID *uuid.UUID) (bool, error) {
	h := strings.ToLower(strings.TrimSpace(handle))
	var reserved bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM badges.reserved_handles WHERE handle_lower = $1)`, h).Scan(&reserved); err != nil {
		return false, err
	}
	if reserved {
		return true, nil
	}
	var taken bool
	if excludeUserID != nil {
		err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM "user".user_badge_profiles WHERE handle_lower = $1 AND user_id <> $2
)`, h, *excludeUserID).Scan(&taken)
		return taken, err
	}
	err := pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM "user".user_badge_profiles WHERE handle_lower = $1)
`, h).Scan(&taken)
	return taken, err
}

// IsReservedHandle checks the reserved_handles seed table.
func IsReservedHandle(ctx context.Context, pool *pgxpool.Pool, handle string) (bool, error) {
	var reserved bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM badges.reserved_handles WHERE handle_lower = $1)
`, strings.ToLower(strings.TrimSpace(handle))).Scan(&reserved)
	return reserved, err
}

// ListPublicAwardsForUser returns public non-revoked awards for a backpack page.
func ListPublicAwardsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]PublicAward, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id, d.id, d.slug, d.name, d.description, d.criteria_narrative, d.image_key, d.tags,
       a.share_slug, a.issued_at, d.course_id, COALESCE(c.title, '')
FROM badges.awarded_badges a
JOIN badges.badge_definitions d ON d.id = a.definition_id
LEFT JOIN course.courses c ON c.id = d.course_id
WHERE a.recipient_id = $1 AND a.is_public = TRUE AND a.revoked = FALSE
ORDER BY a.issued_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublicAward
	for rows.Next() {
		var p PublicAward
		var tags []string
		if err := rows.Scan(
			&p.AwardedID, &p.DefinitionID, &p.Slug, &p.Name, &p.Description, &p.CriteriaNarrative, &p.ImageKey, &tags,
			&p.ShareSlug, &p.IssuedAt, &p.CourseID, &p.CourseTitle,
		); err != nil {
			return nil, err
		}
		if tags == nil {
			tags = []string{}
		}
		p.Tags = tags
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPublicAwardByHandleAndSlug returns a single public award.
func GetPublicAwardByHandleAndSlug(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, badgeSlug string) (*PublicAward, *AwardedBadge, error) {
	var p PublicAward
	var tags []string
	var a AwardedBadge
	var source string
	err := pool.QueryRow(ctx, `
SELECT a.id, d.id, d.slug, d.name, d.description, d.criteria_narrative, d.image_key, d.tags,
       a.share_slug, a.issued_at, d.course_id, COALESCE(c.title, ''),
       a.id, a.definition_id, a.recipient_id, a.awarded_by, a.award_source, a.evidence_json,
       a.credential_json, a.proof, a.share_slug, a.is_public, a.revoked, a.revoked_reason, a.revoked_at, a.issued_at
FROM badges.awarded_badges a
JOIN badges.badge_definitions d ON d.id = a.definition_id
LEFT JOIN course.courses c ON c.id = d.course_id
WHERE a.recipient_id = $1 AND lower(d.slug) = lower($2) AND a.is_public = TRUE AND a.revoked = FALSE
`, userID, badgeSlug).Scan(
		&p.AwardedID, &p.DefinitionID, &p.Slug, &p.Name, &p.Description, &p.CriteriaNarrative, &p.ImageKey, &tags,
		&p.ShareSlug, &p.IssuedAt, &p.CourseID, &p.CourseTitle,
		&a.ID, &a.DefinitionID, &a.RecipientID, &a.AwardedBy, &source, &a.EvidenceJSON,
		&a.CredentialJSON, &a.Proof, &a.ShareSlug, &a.IsPublic, &a.Revoked, &a.RevokedReason, &a.RevokedAt, &a.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if tags == nil {
		tags = []string{}
	}
	p.Tags = tags
	a.AwardSource = AwardSource(source)
	return &p, &a, nil
}

// UserDisplayName loads the user's display name (or email local-part fallback).
func UserDisplayName(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var displayName *string
	var email string
	var isMinor bool
	err := pool.QueryRow(ctx, `
SELECT display_name, email, COALESCE(is_minor, FALSE)
FROM "user".users WHERE id = $1
`, userID).Scan(&displayName, &email, &isMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName), nil
	}
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at], nil
	}
	return email, nil
}

// UserIsMinor returns the COPPA minor flag for a user.
func UserIsMinor(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var isMinor bool
	err := pool.QueryRow(ctx, `SELECT COALESCE(is_minor, FALSE) FROM "user".users WHERE id = $1`, userID).Scan(&isMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return isMinor, err
}

// HasActiveGuardianConsent reports whether a minor has active parental consent.
func HasActiveGuardianConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var ok bool
	// Consent table may not exist in all environments; treat missing as no consent via soft check.
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM compliance.coppa_consents
  WHERE student_id = $1 AND consented_at IS NOT NULL AND revoked_at IS NULL
)
`, userID).Scan(&ok)
	if err != nil {
		// Table may not exist or schema differs — fail closed for public toggles.
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

// CourseTitleByID loads a course title.
func CourseTitleByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var title string
	err := pool.QueryRow(ctx, `SELECT title FROM course.courses WHERE id = $1`, courseID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return title, err
}

// CourseCodeByID loads course_code for authz checks.
func CourseCodeByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var code string
	err := pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return code, err
}

// CourseIDByCode loads course id from course_code.
func CourseIDByCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM course.courses WHERE course_code = $1`, courseCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	return id, err
}

// OutcomeBelongsToCourse checks outcome ownership.
func OutcomeBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, outcomeID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_learning_outcomes WHERE id = $1 AND course_id = $2
)`, outcomeID, courseID).Scan(&ok)
	return ok, err
}

// MasteryReached reports whether a student has met the outcomes-report mastery threshold
// for the given outcome. Uses analytics.outcomes_report_student when available.
func MasteryReached(ctx context.Context, pool *pgxpool.Pool, courseID, studentID, outcomeID uuid.UUID) (bool, error) {
	_ = courseID
	var met bool
	var assessed bool
	err := pool.QueryRow(ctx, `
SELECT COALESCE(met, FALSE), COALESCE(assessed, FALSE)
FROM analytics.outcomes_report_student
WHERE outcome_id = $1 AND user_id = $2
`, outcomeID, studentID).Scan(&met, &assessed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		// Soft-fail if analytics tables are missing.
		if strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}
	return assessed && met, nil
}

// NewShareSlug mints an opaque share token (≥22 hex chars).
func NewShareSlug() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// NewOpaqueHandle mints a default unguessable handle (≥22 chars, charset-safe).
func NewOpaqueHandle() (string, error) {
	b := make([]byte, 14)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// hex is [a-f0-9], length 28; prefix with 'u' so it always matches handle regex (starts/ends alnum).
	return "u" + hex.EncodeToString(b) + "x", nil
}

// IncrementPageView bumps the PII-free daily counter.
func IncrementPageView(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID, awardID *uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO badges.badge_page_views (handle_owner_id, awarded_badge_id, viewed_on, view_count)
VALUES ($1, $2, CURRENT_DATE, 1)
ON CONFLICT (handle_owner_id, awarded_badge_id, viewed_on)
DO UPDATE SET view_count = badges.badge_page_views.view_count + 1
`, ownerID, awardID)
	return err
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
