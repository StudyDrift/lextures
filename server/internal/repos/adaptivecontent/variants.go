package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VariantRow is a course.content_variants row (AC.1 / AC.3 / AC.5).
type VariantRow struct {
	ID               uuid.UUID
	UnitID           uuid.UUID
	ProfileSignature string
	AxesApplied      []string
	VariantMarkdown  string
	Model            string
	FidelityScore    *float64
	SafetyFlags      []byte // JSON array
	Status           string
	ApprovedBy       *uuid.UUID
	CreatedAt        time.Time
	// AC.3
	PromptVersion    string
	ContentVersion   int32
	PromptTokens     int32
	CompletionTokens int32
	A11yFlags        []byte // JSON array
	// AC.5 review metadata
	HumanEdited    bool
	ReviewedBy     *uuid.UUID
	ReviewedAt     *time.Time
	ReviewNote     *string
	VariantVersion int32
}

// KeyTermRow is a course.adaptive_content_key_terms row.
type KeyTermRow struct {
	ID         uuid.UUID
	UnitID     uuid.UUID
	Term       string
	MustAppear bool
	CreatedAt  time.Time
}

// UnitVariantCoverage is per-unit variant counts for the authoring workspace (AC.5).
type UnitVariantCoverage struct {
	UnitID         uuid.UUID
	Total          int64
	Approved       int64
	PendingReview  int64
	Rejected       int64
	AutoServed     int64
	Superseded     int64
}

// GetVariantBySignature returns the cached variant for (unit, signature), or nil.
// Only returns rows whose content_version matches expectedContentVersion when expected > 0.
func GetVariantBySignature(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID uuid.UUID,
	signature string,
	expectedContentVersion int32,
) (*VariantRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, profile_signature, axes_applied, variant_markdown, model,
       fidelity_score, safety_flags, status, approved_by, created_at,
       prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
       human_edited, reviewed_by, reviewed_at, review_note, variant_version
FROM course.content_variants
WHERE unit_id = $1 AND profile_signature = $2
`, unitID, signature)
	v, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expectedContentVersion > 0 && v.ContentVersion != expectedContentVersion {
		return nil, nil
	}
	// Superseded rows are not cache hits for serving/generation reuse.
	if v.Status == "superseded" {
		return nil, nil
	}
	return &v, nil
}

// GetVariantByID returns a variant by id, scoped to a course via its unit, or nil.
func GetVariantByID(ctx context.Context, pool *pgxpool.Pool, courseID, variantID uuid.UUID) (*VariantRow, error) {
	row := pool.QueryRow(ctx, `
SELECT v.id, v.unit_id, v.profile_signature, v.axes_applied, v.variant_markdown, v.model,
       v.fidelity_score, v.safety_flags, v.status, v.approved_by, v.created_at,
       v.prompt_version, v.content_version, v.prompt_tokens, v.completion_tokens, v.a11y_flags,
       v.human_edited, v.reviewed_by, v.reviewed_at, v.review_note, v.variant_version
FROM course.content_variants v
JOIN course.adaptive_content_units u ON u.id = v.unit_id
WHERE v.id = $1 AND u.course_id = $2
`, variantID, courseID)
	v, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVariants returns all variants for a unit (any status), newest first.
func ListVariants(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]VariantRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, unit_id, profile_signature, axes_applied, variant_markdown, model,
       fidelity_score, safety_flags, status, approved_by, created_at,
       prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
       human_edited, reviewed_by, reviewed_at, review_note, variant_version
FROM course.content_variants
WHERE unit_id = $1
ORDER BY created_at DESC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VariantRow
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if out == nil {
		out = []VariantRow{}
	}
	return out, rows.Err()
}

// ListVariantsByStatus returns variants for a unit filtered by status, newest first.
// When status is empty, returns all statuses (same as ListVariants).
func ListVariantsByStatus(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, status string) ([]VariantRow, error) {
	if status == "" {
		return ListVariants(ctx, pool, unitID)
	}
	rows, err := pool.Query(ctx, `
SELECT id, unit_id, profile_signature, axes_applied, variant_markdown, model,
       fidelity_score, safety_flags, status, approved_by, created_at,
       prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
       human_edited, reviewed_by, reviewed_at, review_note, variant_version
FROM course.content_variants
WHERE unit_id = $1 AND status = $2
ORDER BY created_at DESC
`, unitID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VariantRow
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if out == nil {
		out = []VariantRow{}
	}
	return out, rows.Err()
}

// ListPendingReviewForCourse returns pending_review variants across all units in a course.
// limit/offset support pagination (limit <= 0 ⇒ default 50; max 200).
func ListPendingReviewForCourse(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	limit, offset int,
) ([]VariantRow, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM course.content_variants v
JOIN course.adaptive_content_units u ON u.id = v.unit_id
WHERE u.course_id = $1 AND v.status = 'pending_review'
`, courseID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, `
SELECT v.id, v.unit_id, v.profile_signature, v.axes_applied, v.variant_markdown, v.model,
       v.fidelity_score, v.safety_flags, v.status, v.approved_by, v.created_at,
       v.prompt_version, v.content_version, v.prompt_tokens, v.completion_tokens, v.a11y_flags,
       v.human_edited, v.reviewed_by, v.reviewed_at, v.review_note, v.variant_version
FROM course.content_variants v
JOIN course.adaptive_content_units u ON u.id = v.unit_id
WHERE u.course_id = $1 AND v.status = 'pending_review'
ORDER BY v.created_at ASC
LIMIT $2 OFFSET $3
`, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []VariantRow
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	if out == nil {
		out = []VariantRow{}
	}
	return out, total, rows.Err()
}

// CountVariantCoverageByCourse returns variant status counts keyed by unit id for a course.
func CountVariantCoverageByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[uuid.UUID]UnitVariantCoverage, error) {
	rows, err := pool.Query(ctx, `
SELECT v.unit_id,
       COUNT(*)::bigint AS total,
       COUNT(*) FILTER (WHERE v.status = 'approved')::bigint AS approved,
       COUNT(*) FILTER (WHERE v.status = 'pending_review')::bigint AS pending_review,
       COUNT(*) FILTER (WHERE v.status = 'rejected')::bigint AS rejected,
       COUNT(*) FILTER (WHERE v.status = 'auto_served')::bigint AS auto_served,
       COUNT(*) FILTER (WHERE v.status = 'superseded')::bigint AS superseded
FROM course.content_variants v
JOIN course.adaptive_content_units u ON u.id = v.unit_id
WHERE u.course_id = $1
GROUP BY v.unit_id
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]UnitVariantCoverage)
	for rows.Next() {
		var c UnitVariantCoverage
		if err := rows.Scan(&c.UnitID, &c.Total, &c.Approved, &c.PendingReview, &c.Rejected, &c.AutoServed, &c.Superseded); err != nil {
			return nil, err
		}
		out[c.UnitID] = c
	}
	return out, rows.Err()
}

// UpsertVariant inserts or replaces the unique (unit_id, profile_signature) row.
// Regeneration resets human_edited / review metadata and sets variant_version to 1.
func UpsertVariant(ctx context.Context, pool *pgxpool.Pool, v VariantRow) (*VariantRow, error) {
	if v.AxesApplied == nil {
		v.AxesApplied = []string{}
	}
	if v.SafetyFlags == nil {
		v.SafetyFlags = []byte("[]")
	}
	if v.A11yFlags == nil {
		v.A11yFlags = []byte("[]")
	}
	if v.PromptVersion == "" {
		v.PromptVersion = "v1"
	}
	if v.ContentVersion <= 0 {
		v.ContentVersion = 1
	}
	if v.Status == "" {
		v.Status = "draft"
	}
	if v.VariantVersion <= 0 {
		v.VariantVersion = 1
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_variants (
  unit_id, profile_signature, axes_applied, variant_markdown, model,
  fidelity_score, safety_flags, status, approved_by,
  prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
  human_edited, reviewed_by, reviewed_at, review_note, variant_version
) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,$12,$13,$14::jsonb,
          COALESCE($15, FALSE), $16, $17, $18, $19)
ON CONFLICT (unit_id, profile_signature) DO UPDATE SET
  axes_applied = EXCLUDED.axes_applied,
  variant_markdown = EXCLUDED.variant_markdown,
  model = EXCLUDED.model,
  fidelity_score = EXCLUDED.fidelity_score,
  safety_flags = EXCLUDED.safety_flags,
  status = EXCLUDED.status,
  approved_by = EXCLUDED.approved_by,
  prompt_version = EXCLUDED.prompt_version,
  content_version = EXCLUDED.content_version,
  prompt_tokens = EXCLUDED.prompt_tokens,
  completion_tokens = EXCLUDED.completion_tokens,
  a11y_flags = EXCLUDED.a11y_flags,
  human_edited = FALSE,
  reviewed_by = NULL,
  reviewed_at = NULL,
  review_note = NULL,
  variant_version = 1,
  created_at = NOW()
RETURNING id, unit_id, profile_signature, axes_applied, variant_markdown, model,
          fidelity_score, safety_flags, status, approved_by, created_at,
          prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
          human_edited, reviewed_by, reviewed_at, review_note, variant_version
`, v.UnitID, v.ProfileSignature, v.AxesApplied, v.VariantMarkdown, v.Model,
		v.FidelityScore, string(v.SafetyFlags), v.Status, v.ApprovedBy,
		v.PromptVersion, v.ContentVersion, v.PromptTokens, v.CompletionTokens, string(v.A11yFlags),
		v.HumanEdited, v.ReviewedBy, v.ReviewedAt, v.ReviewNote, v.VariantVersion,
	)
	out, err := scanVariant(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReviewDecision is the set of fields applied by approve / reject / edit-and-approve.
type ReviewDecision struct {
	// NewStatus must be approved, rejected, or auto_served (not used for revoke).
	NewStatus string
	// ExpectedVersion is the optimistic concurrency token (0 = skip check).
	ExpectedVersion int32
	// Markdown when non-nil replaces variant_markdown and marks human_edited.
	Markdown *string
	// ReviewNote optional reject reason / approve note.
	ReviewNote *string
	// Actor is the reviewer user id.
	Actor uuid.UUID
	// HumanEdited forces the human_edited flag (true for edit-and-approve).
	HumanEdited bool
	// OverrideGate when true allows approving a gate-failed variant (soft flags only;
	// must_appear key-term hard failures are still blocked by the service layer).
	OverrideGate bool
}

// ErrVariantVersionConflict is returned when expectedVariantVersion does not match.
var ErrVariantVersionConflict = errors.New("variant version conflict")

// ApplyReviewDecision updates a variant for approve/reject/edit with optimistic concurrency.
// Returns nil, ErrVariantVersionConflict when the version does not match.
// Returns nil, nil when the variant id is not found.
func ApplyReviewDecision(
	ctx context.Context,
	pool *pgxpool.Pool,
	variantID uuid.UUID,
	dec ReviewDecision,
) (*VariantRow, error) {
	// Load current row for version check and optional markdown merge.
	var cur VariantRow
	err := pool.QueryRow(ctx, `
SELECT id, unit_id, profile_signature, axes_applied, variant_markdown, model,
       fidelity_score, safety_flags, status, approved_by, created_at,
       prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
       human_edited, reviewed_by, reviewed_at, review_note, variant_version
FROM course.content_variants
WHERE id = $1
`, variantID).Scan(
		&cur.ID, &cur.UnitID, &cur.ProfileSignature, &cur.AxesApplied, &cur.VariantMarkdown, &cur.Model,
		&cur.FidelityScore, &cur.SafetyFlags, &cur.Status, &cur.ApprovedBy, &cur.CreatedAt,
		&cur.PromptVersion, &cur.ContentVersion, &cur.PromptTokens, &cur.CompletionTokens, &cur.A11yFlags,
		&cur.HumanEdited, &cur.ReviewedBy, &cur.ReviewedAt, &cur.ReviewNote, &cur.VariantVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dec.ExpectedVersion > 0 && cur.VariantVersion != dec.ExpectedVersion {
		return nil, ErrVariantVersionConflict
	}

	md := cur.VariantMarkdown
	humanEdited := cur.HumanEdited || dec.HumanEdited
	if dec.Markdown != nil {
		md = *dec.Markdown
		humanEdited = true
	}

	var approvedBy *uuid.UUID
	if dec.NewStatus == "approved" || dec.NewStatus == "auto_served" {
		approvedBy = &dec.Actor
	}

	row := pool.QueryRow(ctx, `
UPDATE course.content_variants
SET status = $2,
    variant_markdown = $3,
    human_edited = $4,
    reviewed_by = $5,
    reviewed_at = NOW(),
    review_note = $6,
    approved_by = $7,
    variant_version = variant_version + 1
WHERE id = $1
  AND ($8::int = 0 OR variant_version = $8)
RETURNING id, unit_id, profile_signature, axes_applied, variant_markdown, model,
          fidelity_score, safety_flags, status, approved_by, created_at,
          prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
          human_edited, reviewed_by, reviewed_at, review_note, variant_version
`, variantID, dec.NewStatus, md, humanEdited, dec.Actor, dec.ReviewNote, approvedBy, dec.ExpectedVersion)
	out, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// Race: version changed between select and update.
		return nil, ErrVariantVersionConflict
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeVariant marks an approved/auto_served variant as superseded (optimistic concurrency).
func RevokeVariant(
	ctx context.Context,
	pool *pgxpool.Pool,
	variantID uuid.UUID,
	actor uuid.UUID,
	expectedVersion int32,
	note *string,
) (*VariantRow, error) {
	row := pool.QueryRow(ctx, `
UPDATE course.content_variants
SET status = 'superseded',
    reviewed_by = $2,
    reviewed_at = NOW(),
    review_note = COALESCE($3, review_note),
    variant_version = variant_version + 1
WHERE id = $1
  AND status IN ('approved', 'auto_served')
  AND ($4::int = 0 OR variant_version = $4)
RETURNING id, unit_id, profile_signature, axes_applied, variant_markdown, model,
          fidelity_score, safety_flags, status, approved_by, created_at,
          prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
          human_edited, reviewed_by, reviewed_at, review_note, variant_version
`, variantID, actor, note, expectedVersion)
	out, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// Distinguish not-found vs version conflict by re-checking existence.
		var exists bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM course.content_variants WHERE id = $1)`, variantID).Scan(&exists)
		if !exists {
			return nil, nil
		}
		return nil, ErrVariantVersionConflict
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SupersedeVariantsForUnit marks non-superseded variants for a unit as superseded
// when their content_version is older than currentVersion.
func SupersedeVariantsForUnit(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, currentVersion int32) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.content_variants
SET status = 'superseded'
WHERE unit_id = $1
  AND content_version < $2
  AND status <> 'superseded'
`, unitID, currentVersion)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// BumpUnitContentVersion increments content_version and returns the new value.
// Also supersedes outdated variants.
func BumpUnitContentVersion(ctx context.Context, pool *pgxpool.Pool, courseID, unitID uuid.UUID) (int32, error) {
	var next int32
	err := pool.QueryRow(ctx, `
UPDATE course.adaptive_content_units
SET content_version = content_version + 1, updated_at = NOW()
WHERE course_id = $1 AND id = $2
RETURNING content_version
`, courseID, unitID).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if _, err := SupersedeVariantsForUnit(ctx, pool, unitID, next); err != nil {
		return next, err
	}
	return next, nil
}

// BumpUnitsForBaseContentItem bumps content_version for every unit whose base_content_item_id matches.
// Returns the number of units bumped and their IDs (for AC.4 regen enqueue).
func BumpUnitsForBaseContentItem(ctx context.Context, pool *pgxpool.Pool, courseID, baseItemID uuid.UUID) (int64, error) {
	_, n, err := BumpUnitsForBaseContentItemWithIDs(ctx, pool, courseID, baseItemID)
	return n, err
}

// BumpUnitsForBaseContentItemWithIDs bumps versions and returns the affected unit rows.
func BumpUnitsForBaseContentItemWithIDs(ctx context.Context, pool *pgxpool.Pool, courseID, baseItemID uuid.UUID) ([]UnitRow, int64, error) {
	rows, err := pool.Query(ctx, `
SELECT id FROM course.adaptive_content_units
WHERE course_id = $1 AND base_content_item_id = $2
`, courseID, baseItemID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var unitIDs []uuid.UUID
	for rows.Next() {
		var unitID uuid.UUID
		if err := rows.Scan(&unitID); err != nil {
			return nil, 0, err
		}
		unitIDs = append(unitIDs, unitID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var out []UnitRow
	var n int64
	for _, unitID := range unitIDs {
		if _, err := BumpUnitContentVersion(ctx, pool, courseID, unitID); err != nil {
			return out, n, err
		}
		u, err := GetUnit(ctx, pool, courseID, unitID)
		if err != nil {
			return out, n, err
		}
		if u != nil {
			out = append(out, *u)
		}
		n++
	}
	return out, n, nil
}

// ListKeyTerms returns key terms for a unit.
func ListKeyTerms(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]KeyTermRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, unit_id, term, must_appear, created_at
FROM course.adaptive_content_key_terms
WHERE unit_id = $1
ORDER BY created_at ASC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KeyTermRow
	for rows.Next() {
		var r KeyTermRow
		if err := rows.Scan(&r.ID, &r.UnitID, &r.Term, &r.MustAppear, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []KeyTermRow{}
	}
	return out, rows.Err()
}

// MustAppearKeyTerms returns only terms with must_appear = true.
func MustAppearKeyTerms(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]string, error) {
	terms, err := ListKeyTerms(ctx, pool, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		if t.MustAppear {
			out = append(out, t.Term)
		}
	}
	return out, nil
}

// ReplaceKeyTerms replaces all key terms for a unit with the given list.
func ReplaceKeyTerms(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, terms []KeyTermRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM course.adaptive_content_key_terms WHERE unit_id = $1`, unitID); err != nil {
		return err
	}
	for _, t := range terms {
		term := t.Term
		if term == "" {
			continue
		}
		must := t.MustAppear
		if _, err := tx.Exec(ctx, `
INSERT INTO course.adaptive_content_key_terms (unit_id, term, must_appear)
VALUES ($1, $2, $3)
`, unitID, term, must); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// InsertKeyTerm adds one key term.
func InsertKeyTerm(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, term string, mustAppear bool) (*KeyTermRow, error) {
	var r KeyTermRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_key_terms (unit_id, term, must_appear)
VALUES ($1, $2, $3)
RETURNING id, unit_id, term, must_appear, created_at
`, unitID, term, mustAppear).Scan(&r.ID, &r.UnitID, &r.Term, &r.MustAppear, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteKeyTerm removes a key term by id within a unit.
func DeleteKeyTerm(ctx context.Context, pool *pgxpool.Pool, unitID, termID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.adaptive_content_key_terms WHERE unit_id = $1 AND id = $2
`, unitID, termID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FlagsJSON marshals string flags to JSON bytes for storage.
func FlagsJSON(flags []string) []byte {
	if flags == nil {
		return []byte("[]")
	}
	b, err := json.Marshal(flags)
	if err != nil {
		return []byte("[]")
	}
	return b
}

// ParseFlagsJSON unmarshals a JSON string array.
func ParseFlagsJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func scanVariant(row scannable) (VariantRow, error) {
	var v VariantRow
	err := row.Scan(
		&v.ID, &v.UnitID, &v.ProfileSignature, &v.AxesApplied, &v.VariantMarkdown, &v.Model,
		&v.FidelityScore, &v.SafetyFlags, &v.Status, &v.ApprovedBy, &v.CreatedAt,
		&v.PromptVersion, &v.ContentVersion, &v.PromptTokens, &v.CompletionTokens, &v.A11yFlags,
		&v.HumanEdited, &v.ReviewedBy, &v.ReviewedAt, &v.ReviewNote, &v.VariantVersion,
	)
	if err != nil {
		return VariantRow{}, err
	}
	if v.AxesApplied == nil {
		v.AxesApplied = []string{}
	}
	if v.SafetyFlags == nil {
		v.SafetyFlags = []byte("[]")
	}
	if v.A11yFlags == nil {
		v.A11yFlags = []byte("[]")
	}
	if v.PromptVersion == "" {
		v.PromptVersion = "v1"
	}
	if v.VariantVersion <= 0 {
		v.VariantVersion = 1
	}
	return v, nil
}
