package quizgame

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

// DeepCopyOpts controls kit deep-copy behaviour (duplicate / template / import).
type DeepCopyOpts struct {
	SourceKitID      string
	TargetCourseCode string
	CreatedBy        uuid.UUID
	Title            string // optional override; empty → "{title} (copy)"
	AsTemplate       bool
	TemplateScope    string // system|org|course when AsTemplate
	DropBankLinks    bool   // true for cross-course / import
	DerivedFromKitID *string
	Attribution      string
	Subject          *string
	GradeBand        *string
	Language         *string
	CopyCatalogMeta  bool // copy subject/grade/language/tags from source when true
}

// DeepCopyKit transactionally copies a kit and all questions into a target course.
// Media references are shared (not blob-duplicated). Bank links are dropped when
// DropBankLinks is true or the target course differs from the source.
func DeepCopyKit(ctx context.Context, pool *pgxpool.Pool, opts DeepCopyOpts) (*Kit, error) {
	srcID, err := uuid.Parse(opts.SourceKitID)
	if err != nil {
		return nil, fmt.Errorf("quizgame: invalid source kit id")
	}
	if strings.TrimSpace(opts.TargetCourseCode) == "" {
		return nil, fmt.Errorf("quizgame: target course is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	src, err := getKitByIDTx(ctx, tx, srcID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}

	var targetCourseID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM course.courses WHERE course_code = $1
	`, opts.TargetCourseCode).Scan(&targetCourseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("quizgame: target course not found")
		}
		return nil, err
	}

	crossCourse := src.CourseID == "" || src.CourseID != targetCourseID.String()
	dropBank := opts.DropBankLinks || crossCourse

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = src.Title + " (copy)"
	}
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen]
	}

	baseSlug := organization.SuggestSlugFromName(title)
	if baseSlug == "" {
		baseSlug = "kit"
	}

	var derived *uuid.UUID
	if opts.DerivedFromKitID != nil && strings.TrimSpace(*opts.DerivedFromKitID) != "" {
		d, err := uuid.Parse(strings.TrimSpace(*opts.DerivedFromKitID))
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid derived_from_kit_id")
		}
		derived = &d
	} else {
		derived = &srcID
	}

	attribution := strings.TrimSpace(opts.Attribution)
	if attribution == "" && src.CreatedBy != nil {
		attribution = "Based on \"" + src.Title + "\""
	}

	subject := opts.Subject
	gradeBand := opts.GradeBand
	language := opts.Language
	if opts.CopyCatalogMeta {
		if subject == nil {
			subject = src.Subject
		}
		if gradeBand == nil {
			gradeBand = src.GradeBand
		}
		if language == nil {
			language = src.Language
		}
	}

	isTemplate := opts.AsTemplate
	var templateScope *string
	if isTemplate {
		scope := strings.TrimSpace(strings.ToLower(opts.TemplateScope))
		if scope == "" {
			scope = "course"
		}
		switch scope {
		case "system", "org", "course":
		default:
			return nil, fmt.Errorf("quizgame: invalid template scope")
		}
		templateScope = &scope
	}

	var created *Kit
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := baseSlug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
			if len(slug) > 48 {
				slug = slug[:48]
			}
		}
		sp := fmt.Sprintf("kit_copy_%d", attempt)
		if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
			return nil, err
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO quizgame.kits (
				course_id, title, description, slug, cover_image_ref,
				status, visibility, tags, created_by,
				is_template, template_scope, derived_from_kit_id, attribution,
				subject, grade_band, language, catalog_status
			) VALUES (
				$1, $2, $3, $4, $5,
				'draft', 'course', $6, $7,
				$8, $9, $10, $11,
				$12, $13, $14, 'unlisted'
			)
			RETURNING `+selectKitColsReturning(),
			targetCourseID, title, src.Description, slug, src.CoverImageRef,
			src.Tags, opts.CreatedBy,
			isTemplate, templateScope, derived, attribution,
			subject, gradeBand, language,
		)
		k, err := scanKit(row)
		if err == nil {
			if _, relErr := tx.Exec(ctx, "RELEASE SAVEPOINT "+sp); relErr != nil {
				return nil, relErr
			}
			created = &k
			break
		}
		if _, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp); rbErr != nil {
			return nil, rbErr
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}
	if created == nil {
		return nil, fmt.Errorf("quizgame: could not allocate unique slug")
	}

	newKitID, err := uuid.Parse(created.ID)
	if err != nil {
		return nil, err
	}

	type qRow struct {
		position, timeLimit int
		qType, pointsStyle  string
		prompt              string
		mediaRef, mediaAlt  *string
		optsJSON, corrJSON  []byte
		shuffle             bool
		explanation         *string
		bankSource          uuid.NullUUID
		source              string
		needsReview         bool
		confidence          *float64
	}
	rows, err := tx.Query(ctx, `
		SELECT position, question_type::text, prompt, prompt_media_ref, prompt_media_alt,
			options, correct_answer, time_limit_seconds, points_style::text, answer_shuffle,
			explanation, source_question_id, source, needs_review, generation_confidence
		FROM quizgame.questions
		WHERE kit_id = $1
		ORDER BY position ASC
	`, srcID)
	if err != nil {
		return nil, err
	}
	srcQuestions := make([]qRow, 0)
	for rows.Next() {
		var qr qRow
		if err := rows.Scan(
			&qr.position, &qr.qType, &qr.prompt, &qr.mediaRef, &qr.mediaAlt,
			&qr.optsJSON, &qr.corrJSON, &qr.timeLimit, &qr.pointsStyle, &qr.shuffle,
			&qr.explanation, &qr.bankSource, &qr.source, &qr.needsReview, &qr.confidence,
		); err != nil {
			rows.Close()
			return nil, err
		}
		if len(qr.optsJSON) == 0 {
			qr.optsJSON = []byte("[]")
		}
		if qr.source == "" {
			qr.source = QuestionSourceAuthored
		}
		srcQuestions = append(srcQuestions, qr)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for _, qr := range srcQuestions {
		var sourceID any
		if !dropBank && qr.bankSource.Valid {
			sourceID = qr.bankSource.UUID
		}
		var corr any
		if len(qr.corrJSON) > 0 {
			corr = qr.corrJSON
		}
		var conf any
		if qr.confidence != nil {
			conf = *qr.confidence
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO quizgame.questions (
				kit_id, position, question_type, prompt, prompt_media_ref, prompt_media_alt,
				options, correct_answer, time_limit_seconds, points_style, answer_shuffle,
				explanation, source_question_id, source, needs_review, generation_confidence
			) VALUES (
				$1, $2, $3::quizgame.question_type, $4, $5, $6,
				$7::jsonb, $8::jsonb, $9, $10::quizgame.points_style, $11,
				$12, $13, $14, $15, $16
			)
		`, newKitID, qr.position, qr.qType, qr.prompt, qr.mediaRef, qr.mediaAlt,
			qr.optsJSON, corr, qr.timeLimit, qr.pointsStyle, qr.shuffle,
			qr.explanation, sourceID, qr.source, qr.needsReview, conf); err != nil {
			return nil, err
		}
	}

	// Re-load to pick up question_count from trigger.
	row := tx.QueryRow(ctx, `
		SELECT `+selectKitCols()+`
		FROM quizgame.kits k
		WHERE k.id = $1
	`, newKitID)
	final, err := scanKit(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &final, nil
}

func selectKitColsReturning() string {
	return `id, course_id, title, description, slug, cover_image_ref,
		status::text, visibility::text, tags, question_count, archived,
		created_by, created_at, updated_at,
		is_template, template_scope, derived_from_kit_id, attribution,
		subject, grade_band, language, catalog_status`
}

func getKitByIDTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*Kit, error) {
	row := tx.QueryRow(ctx, `
		SELECT `+selectKitCols()+`
		FROM quizgame.kits k
		WHERE k.id = $1
	`, id)
	k, err := scanKit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// GetByID returns a kit by id without course scoping (templates / library).
func GetByID(ctx context.Context, pool *pgxpool.Pool, kitID string) (*Kit, error) {
	id, err := uuid.Parse(kitID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectKitCols()+`
		FROM quizgame.kits k
		WHERE k.id = $1
	`, id)
	k, err := scanKit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}
