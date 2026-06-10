// Package portfolios provides data access for student-owned ePortfolios and
// capstone artifact collections (plan 14.12): portfolios, artifacts, rubric
// evaluations, public-slug lookup, and program outcome-coverage reporting.
package portfolios

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Portfolio is a student-owned ePortfolio.
type Portfolio struct {
	ID           uuid.UUID
	OwnerID      uuid.UUID
	Title        string
	IntroText    string
	IsPublic     bool
	PublicSlug   *string
	SectionOrder []uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Artifact is a single piece of evidence within a portfolio.
type Artifact struct {
	ID                 uuid.UUID
	PortfolioID        uuid.UUID
	ArtifactType       string
	Title              string
	Description        string
	SourceSubmissionID *uuid.UUID
	SourceCourseID     *uuid.UUID
	FileKey            string
	FileName           string
	FileMime           string
	TextContent        string
	ExternalURL        string
	OutcomeIDs         []uuid.UUID
	IsPublic           bool
	SortOrder          int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Evaluation is a reviewer's rubric evaluation of an artifact.
type Evaluation struct {
	ID         uuid.UUID
	ArtifactID uuid.UUID
	ReviewerID uuid.UUID
	RubricJSON json.RawMessage
	ScoresJSON json.RawMessage
	TotalScore *float64
	Feedback   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ErrNotFound is returned when a portfolio or artifact does not exist (or is not owned by the caller).
var ErrNotFound = errors.New("portfolio: not found")

// ─── Portfolios ────────────────────────────────────────────────────────────────

const portfolioCols = `id, owner_id, title, intro_text, is_public, public_slug, section_order, created_at, updated_at`

func scanPortfolio(row pgx.Row) (*Portfolio, error) {
	var p Portfolio
	if err := row.Scan(&p.ID, &p.OwnerID, &p.Title, &p.IntroText, &p.IsPublic,
		&p.PublicSlug, &p.SectionOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePortfolio inserts a new portfolio for an owner.
func CreatePortfolio(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID, title, introText string) (*Portfolio, error) {
	return scanPortfolio(pool.QueryRow(ctx, `
INSERT INTO portfolio.portfolios (owner_id, title, intro_text)
VALUES ($1, $2, $3)
RETURNING `+portfolioCols, ownerID, title, introText))
}

// ListPortfoliosByOwner returns an owner's portfolios, most recently updated first.
func ListPortfoliosByOwner(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID) ([]Portfolio, error) {
	rows, err := pool.Query(ctx, `
SELECT `+portfolioCols+`
FROM portfolio.portfolios
WHERE owner_id = $1
ORDER BY updated_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Portfolio
	for rows.Next() {
		p, err := scanPortfolio(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPortfolioOwned returns a portfolio scoped to its owner, or ErrNotFound.
func GetPortfolioOwned(ctx context.Context, pool *pgxpool.Pool, ownerID, id uuid.UUID) (*Portfolio, error) {
	p, err := scanPortfolio(pool.QueryRow(ctx, `
SELECT `+portfolioCols+`
FROM portfolio.portfolios
WHERE id = $1 AND owner_id = $2`, id, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// GetPortfolioBySlug returns a public portfolio by its slug. Returns ErrNotFound
// if no portfolio with that slug exists or it is no longer public.
func GetPortfolioBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*Portfolio, error) {
	p, err := scanPortfolio(pool.QueryRow(ctx, `
SELECT `+portfolioCols+`
FROM portfolio.portfolios
WHERE public_slug = $1 AND is_public = TRUE`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// UpdatePortfolioInput carries editable portfolio fields.
type UpdatePortfolioInput struct {
	Title     string
	IntroText string
}

// UpdatePortfolio updates title/intro for an owned portfolio.
func UpdatePortfolio(ctx context.Context, pool *pgxpool.Pool, ownerID, id uuid.UUID, in UpdatePortfolioInput) (*Portfolio, error) {
	p, err := scanPortfolio(pool.QueryRow(ctx, `
UPDATE portfolio.portfolios
SET title = $3, intro_text = $4, updated_at = NOW()
WHERE id = $1 AND owner_id = $2
RETURNING `+portfolioCols, id, ownerID, in.Title, in.IntroText))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// SetPortfolioVisibility toggles public/private. When making a portfolio public
// for the first time it mints a stable, unguessable slug.
func SetPortfolioVisibility(ctx context.Context, pool *pgxpool.Pool, ownerID, id uuid.UUID, isPublic bool) (*Portfolio, error) {
	slug, err := newSlug()
	if err != nil {
		return nil, err
	}
	// COALESCE keeps an existing slug; only mint one when none exists and going public.
	p, err := scanPortfolio(pool.QueryRow(ctx, `
UPDATE portfolio.portfolios
SET is_public = $3,
    public_slug = CASE WHEN $3 THEN COALESCE(public_slug, $4) ELSE public_slug END,
    updated_at = NOW()
WHERE id = $1 AND owner_id = $2
RETURNING `+portfolioCols, id, ownerID, isPublic, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

// ReorderArtifacts persists a new artifact display order for an owned portfolio.
// Only ids that belong to the portfolio are applied to sort_order; section_order
// is stored verbatim for the editor.
func ReorderArtifacts(ctx context.Context, pool *pgxpool.Pool, ownerID, id uuid.UUID, order []uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE portfolio.portfolios
SET section_order = $3, updated_at = NOW()
WHERE id = $1 AND owner_id = $2`, id, ownerID, order)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	for i, aid := range order {
		if _, err := tx.Exec(ctx, `
UPDATE portfolio.portfolio_artifacts
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND portfolio_id = $2`, aid, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeletePortfolio removes an owned portfolio (cascades to artifacts/evaluations).
func DeletePortfolio(ctx context.Context, pool *pgxpool.Pool, ownerID, id uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM portfolio.portfolios WHERE id = $1 AND owner_id = $2`, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Artifacts ─────────────────────────────────────────────────────────────────

const artifactCols = `id, portfolio_id, artifact_type, title, description,
	source_submission_id, source_course_id, file_key, file_name, file_mime,
	text_content, external_url, outcome_ids, is_public, sort_order, created_at, updated_at`

func scanArtifact(row pgx.Row) (*Artifact, error) {
	var a Artifact
	if err := row.Scan(&a.ID, &a.PortfolioID, &a.ArtifactType, &a.Title, &a.Description,
		&a.SourceSubmissionID, &a.SourceCourseID, &a.FileKey, &a.FileName, &a.FileMime,
		&a.TextContent, &a.ExternalURL, &a.OutcomeIDs, &a.IsPublic, &a.SortOrder,
		&a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateArtifactInput carries the fields needed to add an artifact.
type CreateArtifactInput struct {
	ArtifactType       string
	Title              string
	Description        string
	SourceSubmissionID *uuid.UUID
	SourceCourseID     *uuid.UUID
	FileKey            string
	FileName           string
	FileMime           string
	TextContent        string
	ExternalURL        string
	OutcomeIDs         []uuid.UUID
	IsPublic           bool
}

// CreateArtifact inserts an artifact into an owned portfolio. The portfolio must
// belong to ownerID; otherwise ErrNotFound is returned. New artifacts are appended
// to the end (max sort_order + 1).
func CreateArtifact(ctx context.Context, pool *pgxpool.Pool, ownerID, portfolioID uuid.UUID, in CreateArtifactInput) (*Artifact, error) {
	if _, err := GetPortfolioOwned(ctx, pool, ownerID, portfolioID); err != nil {
		return nil, err
	}
	outcomes := in.OutcomeIDs
	if outcomes == nil {
		outcomes = []uuid.UUID{}
	}
	return scanArtifact(pool.QueryRow(ctx, `
INSERT INTO portfolio.portfolio_artifacts
	(portfolio_id, artifact_type, title, description, source_submission_id, source_course_id,
	 file_key, file_name, file_mime, text_content, external_url, outcome_ids, is_public, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
	COALESCE((SELECT MAX(sort_order) + 1 FROM portfolio.portfolio_artifacts WHERE portfolio_id = $1), 0))
RETURNING `+artifactCols,
		portfolioID, in.ArtifactType, in.Title, in.Description, in.SourceSubmissionID, in.SourceCourseID,
		in.FileKey, in.FileName, in.FileMime, in.TextContent, in.ExternalURL, outcomes, in.IsPublic))
}

// ListArtifacts returns all artifacts in a portfolio ordered by sort_order.
func ListArtifacts(ctx context.Context, pool *pgxpool.Pool, portfolioID uuid.UUID) ([]Artifact, error) {
	return queryArtifacts(ctx, pool, `
SELECT `+artifactCols+`
FROM portfolio.portfolio_artifacts
WHERE portfolio_id = $1
ORDER BY sort_order, created_at`, portfolioID)
}

// ListPublicArtifacts returns only the public artifacts in a portfolio.
func ListPublicArtifacts(ctx context.Context, pool *pgxpool.Pool, portfolioID uuid.UUID) ([]Artifact, error) {
	return queryArtifacts(ctx, pool, `
SELECT `+artifactCols+`
FROM portfolio.portfolio_artifacts
WHERE portfolio_id = $1 AND is_public = TRUE
ORDER BY sort_order, created_at`, portfolioID)
}

func queryArtifacts(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) ([]Artifact, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// GetArtifactOwned returns an artifact joined to its owning portfolio for ownerID.
func GetArtifactOwned(ctx context.Context, pool *pgxpool.Pool, ownerID, artifactID uuid.UUID) (*Artifact, error) {
	a, err := scanArtifact(pool.QueryRow(ctx, `
SELECT `+prefixedArtifactCols("a")+`
FROM portfolio.portfolio_artifacts a
JOIN portfolio.portfolios p ON p.id = a.portfolio_id
WHERE a.id = $1 AND p.owner_id = $2`, artifactID, ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// GetArtifactForReview returns an artifact by id without owner scoping (reviewer path).
func GetArtifactForReview(ctx context.Context, pool *pgxpool.Pool, artifactID uuid.UUID) (*Artifact, error) {
	a, err := scanArtifact(pool.QueryRow(ctx, `
SELECT `+artifactCols+`
FROM portfolio.portfolio_artifacts
WHERE id = $1`, artifactID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func prefixedArtifactCols(alias string) string {
	return alias + ".id, " + alias + ".portfolio_id, " + alias + ".artifact_type, " + alias + ".title, " +
		alias + ".description, " + alias + ".source_submission_id, " + alias + ".source_course_id, " +
		alias + ".file_key, " + alias + ".file_name, " + alias + ".file_mime, " + alias + ".text_content, " +
		alias + ".external_url, " + alias + ".outcome_ids, " + alias + ".is_public, " + alias + ".sort_order, " +
		alias + ".created_at, " + alias + ".updated_at"
}

// UpdateArtifactInput carries editable artifact fields (full replace of metadata).
type UpdateArtifactInput struct {
	Title       string
	Description string
	TextContent string
	ExternalURL string
	OutcomeIDs  []uuid.UUID
	IsPublic    bool
}

// UpdateArtifact updates an artifact that belongs to ownerID's portfolio.
func UpdateArtifact(ctx context.Context, pool *pgxpool.Pool, ownerID, artifactID uuid.UUID, in UpdateArtifactInput) (*Artifact, error) {
	outcomes := in.OutcomeIDs
	if outcomes == nil {
		outcomes = []uuid.UUID{}
	}
	a, err := scanArtifact(pool.QueryRow(ctx, `
UPDATE portfolio.portfolio_artifacts a
SET title = $3, description = $4, text_content = $5, external_url = $6,
    outcome_ids = $7, is_public = $8, updated_at = NOW()
FROM portfolio.portfolios p
WHERE a.id = $1 AND a.portfolio_id = p.id AND p.owner_id = $2
RETURNING `+prefixedArtifactCols("a"),
		artifactID, ownerID, in.Title, in.Description, in.TextContent, in.ExternalURL, outcomes, in.IsPublic))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// DeleteArtifact removes an artifact from ownerID's portfolio.
func DeleteArtifact(ctx context.Context, pool *pgxpool.Pool, ownerID, artifactID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM portfolio.portfolio_artifacts a
USING portfolio.portfolios p
WHERE a.id = $1 AND a.portfolio_id = p.id AND p.owner_id = $2`, artifactID, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Evaluations ───────────────────────────────────────────────────────────────

const evaluationCols = `id, artifact_id, reviewer_id, rubric_json, scores_json, total_score, feedback, created_at, updated_at`

func scanEvaluation(row pgx.Row) (*Evaluation, error) {
	var e Evaluation
	if err := row.Scan(&e.ID, &e.ArtifactID, &e.ReviewerID, &e.RubricJSON, &e.ScoresJSON,
		&e.TotalScore, &e.Feedback, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpsertEvaluationInput carries a reviewer's rubric evaluation.
type UpsertEvaluationInput struct {
	RubricJSON json.RawMessage
	ScoresJSON json.RawMessage
	TotalScore *float64
	Feedback   string
}

// UpsertEvaluation creates or replaces a reviewer's evaluation of an artifact.
func UpsertEvaluation(ctx context.Context, pool *pgxpool.Pool, artifactID, reviewerID uuid.UUID, in UpsertEvaluationInput) (*Evaluation, error) {
	scores := in.ScoresJSON
	if len(scores) == 0 {
		scores = json.RawMessage("{}")
	}
	return scanEvaluation(pool.QueryRow(ctx, `
INSERT INTO portfolio.portfolio_artifact_evaluations
	(artifact_id, reviewer_id, rubric_json, scores_json, total_score, feedback)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (artifact_id, reviewer_id) DO UPDATE
SET rubric_json = EXCLUDED.rubric_json,
    scores_json = EXCLUDED.scores_json,
    total_score = EXCLUDED.total_score,
    feedback = EXCLUDED.feedback,
    updated_at = NOW()
RETURNING `+evaluationCols,
		artifactID, reviewerID, in.RubricJSON, scores, in.TotalScore, in.Feedback))
}

// ListEvaluationsForArtifact returns all reviewer evaluations of an artifact.
func ListEvaluationsForArtifact(ctx context.Context, pool *pgxpool.Pool, artifactID uuid.UUID) ([]Evaluation, error) {
	rows, err := pool.Query(ctx, `
SELECT `+evaluationCols+`
FROM portfolio.portfolio_artifact_evaluations
WHERE artifact_id = $1
ORDER BY updated_at DESC`, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Evaluation
	for rows.Next() {
		e, err := scanEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// ListEvaluationsForPortfolio returns evaluations across all artifacts of a portfolio,
// keyed for the owner's view.
func ListEvaluationsForPortfolio(ctx context.Context, pool *pgxpool.Pool, portfolioID uuid.UUID) ([]Evaluation, error) {
	rows, err := pool.Query(ctx, `
SELECT e.id, e.artifact_id, e.reviewer_id, e.rubric_json, e.scores_json, e.total_score, e.feedback, e.created_at, e.updated_at
FROM portfolio.portfolio_artifact_evaluations e
JOIN portfolio.portfolio_artifacts a ON a.id = e.artifact_id
WHERE a.portfolio_id = $1
ORDER BY e.updated_at DESC`, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Evaluation
	for rows.Next() {
		e, err := scanEvaluation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// ─── Public view counter ──────────────────────────────────────────────────────

// RecordView inserts a privacy-safe public view event (no PII).
func RecordView(ctx context.Context, pool *pgxpool.Pool, portfolioID uuid.UUID) error {
	_, err := pool.Exec(ctx, `INSERT INTO portfolio.portfolio_views (portfolio_id) VALUES ($1)`, portfolioID)
	return err
}

// ViewCount returns the total public view count for a portfolio.
func ViewCount(ctx context.Context, pool *pgxpool.Pool, portfolioID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM portfolio.portfolio_views WHERE portfolio_id = $1`, portfolioID).Scan(&n)
	return n, err
}

// ─── Outcome coverage report ──────────────────────────────────────────────────

// OutcomeCoverageRow is one program outcome's artifact-evidence coverage.
type OutcomeCoverageRow struct {
	OutcomeID     uuid.UUID
	Title         string
	StudentCount  int // distinct portfolio owners with at least one artifact tagged to this outcome
	ArtifactCount int
}

// OutcomeCoverageReport aggregates, for each learning outcome of the given course
// (used as the "program" outcome set), how many distinct students have provided
// artifact evidence and how many artifacts are tagged. cohortSize is the number of
// distinct portfolio owners overall, used to compute submission-rate percentages.
func OutcomeCoverageReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (rows []OutcomeCoverageRow, cohortSize int, err error) {
	if err = pool.QueryRow(ctx, `SELECT COUNT(DISTINCT owner_id) FROM portfolio.portfolios`).Scan(&cohortSize); err != nil {
		return nil, 0, err
	}
	r, err := pool.Query(ctx, `
SELECT o.id, o.title,
       COUNT(DISTINCT p.owner_id) AS student_count,
       COUNT(a.id) AS artifact_count
FROM course.course_learning_outcomes o
LEFT JOIN portfolio.portfolio_artifacts a ON o.id = ANY(a.outcome_ids)
LEFT JOIN portfolio.portfolios p ON p.id = a.portfolio_id
WHERE o.course_id = $1
GROUP BY o.id, o.title, o.sort_order
ORDER BY o.sort_order`, courseID)
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	for r.Next() {
		var row OutcomeCoverageRow
		if err := r.Scan(&row.OutcomeID, &row.Title, &row.StudentCount, &row.ArtifactCount); err != nil {
			return nil, 0, err
		}
		rows = append(rows, row)
	}
	return rows, cohortSize, r.Err()
}

// ─── helpers ───────────────────────────────────────────────────────────────────

// newSlug returns a 24-char hex slug (96 bits of entropy) for public portfolio URLs.
func newSlug() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
