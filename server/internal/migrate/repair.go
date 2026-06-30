package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func migrateRepairChecksumsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MIGRATE_REPAIR_CHECKSUMS"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// demoChecksumRepairMigrations lists versions whose SQL files may legitimately drift
// from what the demo droplet (or a persistent dev DB) recorded (abbreviated deploys,
// follow-up PR edits) while remaining idempotent (IF NOT EXISTS / ON CONFLICT DO NOTHING).
// When MIGRATE_REPAIR_CHECKSUMS is enabled (docker-compose.yml and docker-compose.deploy.yml),
// stored checksums are updated to match the embedded files so migrate can proceed.
var demoChecksumRepairMigrations = []struct {
	version int64
	file    string
}{
	{120, "120_clever_classlink.sql"},
	{135, "135_org_role_grants.sql"},
	{171, "171_mastery_heatmap_cache.sql"},
	// Idempotent ADD COLUMN IF NOT EXISTS; file may change while feature-flag columns evolve.
	{172, "172_platform_feature_flags.sql"},
	// Idempotent (CREATE ... IF NOT EXISTS + ADD COLUMN IF NOT EXISTS); file drifted on persistent dev/demo DBs.
	{220, "220_behavior_pbis.sql"},
	// Idempotent ADD COLUMN IF NOT EXISTS; feature-flag columns may evolve as flags migrate off env.
	{267, "267_feature_flags_env_to_db.sql"},
	// Idempotent ADD COLUMN IF NOT EXISTS; backfills the ff_ui_mode column the repo expected.
	{268, "268_ff_ui_mode_column.sql"},
	// Idempotent CREATE TABLE/INDEX IF NOT EXISTS; table may exist on DBs that applied billing SQL manually.
	{278, "278_billing_stripe.sql"},
	// Edited after apply in PR #288: extend billing.user_entitlements from 278 instead of recreating it.
	{279, "279_learning_paths.sql"},
	// Idempotent INSERT ... WHERE NOT EXISTS; AI Grader prompt refined after initial seed (319 updates existing rows).
	{318, "318_grading_agent_default_templates.sql"},
}

// repairMigration289RenumberCollision fixes dev/demo DBs that applied grading_agent as v289
// and grader_agent_model as v290 before main's 289_study_reminders.sql landed and the agent
// migrations were renumbered to 290/291 (commit aa57337a).
func repairMigration289RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v289Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 289`,
	).Scan(&v289Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v289 renumber: %w", err)
	}
	if v289Desc != "grading_agent" {
		return nil
	}
	var v290Desc string
	err = c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 290`,
	).Scan(&v290Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v289 renumber: %w", err)
	}
	if v290Desc != "grader_agent_model" {
		return nil
	}
	var studyRemindersExists bool
	err = c.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'studyreminders' AND table_name = 'configs')`,
	).Scan(&studyRemindersExists)
	if err != nil {
		return fmt.Errorf("migrate repair v289 renumber: %w", err)
	}
	if studyRemindersExists {
		return nil
	}

	studyBody, err := fs.ReadFile(fsys, dir+"/289_study_reminders.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v289 renumber: read study_reminders: %w", err)
	}
	if _, err := c.Exec(ctx, string(studyBody)); err != nil {
		return fmt.Errorf("migrate repair v289 renumber: apply study_reminders: %w", err)
	}

	studySum := sqlxChecksum(studyBody)
	gradingBody, err := fs.ReadFile(fsys, dir+"/290_grading_agent.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v289 renumber: read grading_agent: %w", err)
	}
	gradingSum := sqlxChecksum(gradingBody)
	modelBody, err := fs.ReadFile(fsys, dir+"/291_grader_agent_model.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v289 renumber: read grader_agent_model: %w", err)
	}
	modelSum := sqlxChecksum(modelBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 289`,
		"study_reminders", studySum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v289 renumber: update v289: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 290`,
		"grading_agent", gradingSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v289 renumber: update v290: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
		int64(291), "grader_agent_model", modelSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v289 renumber: insert v291: %w", err)
	}
	return tx.Commit(ctx)
}

// repairMigration294RenumberCollision fixes dev/demo DBs that applied study_buddy as v294
// before main's 294_course_files_submission_size_500mb.sql landed and study_buddy was renumbered to 296.
func repairMigration294RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v294Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 294`,
	).Scan(&v294Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v294 renumber: %w", err)
	}
	if v294Desc != "study_buddy" {
		return nil
	}
	var courseFilesAt500mb bool
	err = c.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'course_files_byte_size_check'
      AND pg_get_constraintdef(oid) LIKE '%524288000%'
)`).Scan(&courseFilesAt500mb)
	if err != nil {
		return fmt.Errorf("migrate repair v294 renumber: %w", err)
	}
	if courseFilesAt500mb {
		return nil
	}

	courseFilesBody, err := fs.ReadFile(fsys, dir+"/294_course_files_submission_size_500mb.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v294 renumber: read course_files: %w", err)
	}
	if _, err := c.Exec(ctx, string(courseFilesBody)); err != nil {
		return fmt.Errorf("migrate repair v294 renumber: apply course_files: %w", err)
	}

	courseFilesSum := sqlxChecksum(courseFilesBody)
	studyBuddyBody, err := fs.ReadFile(fsys, dir+"/296_study_buddy.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v294 renumber: read study_buddy: %w", err)
	}
	studyBuddySum := sqlxChecksum(studyBuddyBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 294`,
		"course_files_submission_size_500mb", courseFilesSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v294 renumber: update v294: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
		int64(296), "study_buddy", studyBuddySum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v294 renumber: insert v296: %w", err)
	}
	return tx.Commit(ctx)
}

// repairMigration292RenumberCollision fixes dev/demo DBs that applied instructor_comments_json
// as v292 before main's 292_learner_goals.sql landed and instructor_comments was renumbered to 293.
func repairMigration292RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v292Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 292`,
	).Scan(&v292Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v292 renumber: %w", err)
	}
	if v292Desc != "instructor_comments_json" {
		return nil
	}
	var learnerGoalsExists bool
	err = c.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'user' AND table_name = 'learner_goals')`,
	).Scan(&learnerGoalsExists)
	if err != nil {
		return fmt.Errorf("migrate repair v292 renumber: %w", err)
	}
	if learnerGoalsExists {
		return nil
	}

	learnerBody, err := fs.ReadFile(fsys, dir+"/292_learner_goals.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v292 renumber: read learner_goals: %w", err)
	}
	if _, err := c.Exec(ctx, string(learnerBody)); err != nil {
		return fmt.Errorf("migrate repair v292 renumber: apply learner_goals: %w", err)
	}

	learnerSum := sqlxChecksum(learnerBody)
	commentsBody, err := fs.ReadFile(fsys, dir+"/293_instructor_comments_json.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v292 renumber: read instructor_comments_json: %w", err)
	}
	commentsSum := sqlxChecksum(commentsBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 292`,
		"learner_goals", learnerSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v292 renumber: update v292: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
		int64(293), "instructor_comments_json", commentsSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v292 renumber: insert v293: %w", err)
	}
	return tx.Commit(ctx)
}

// repairMigration308RenumberCollision fixes dev/demo DBs that applied submission_attachments as v308
// before main's 308_grade_curving.sql landed and submission_attachments was renumbered to 309 (PR #332).
func repairMigration308RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v308Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 308`,
	).Scan(&v308Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v308 renumber: %w", err)
	}
	var gradeCurvingApplied bool
	err = c.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'settings'
      AND table_name = 'platform_app_settings'
      AND column_name = 'ff_grade_curving'
)`).Scan(&gradeCurvingApplied)
	if err != nil {
		return fmt.Errorf("migrate repair v308 renumber: %w", err)
	}
	if gradeCurvingApplied {
		return nil
	}
	if v308Desc != "submission_attachments" {
		return nil
	}

	gradeBody, err := fs.ReadFile(fsys, dir+"/308_grade_curving.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v308 renumber: read grade_curving: %w", err)
	}
	if _, err := c.Exec(ctx, string(gradeBody)); err != nil {
		return fmt.Errorf("migrate repair v308 renumber: apply grade_curving: %w", err)
	}

	gradeSum := sqlxChecksum(gradeBody)
	attachmentsBody, err := fs.ReadFile(fsys, dir+"/309_submission_attachments.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v308 renumber: read submission_attachments: %w", err)
	}
	attachmentsSum := sqlxChecksum(attachmentsBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 308`,
		"grade_curving", gradeSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v308 renumber: update v308: %w", err)
	}

	var v309Exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+sqlxMigrationsTable+` WHERE version = 309)`,
	).Scan(&v309Exists); err != nil {
		return fmt.Errorf("migrate repair v308 renumber: check v309: %w", err)
	}
	if !v309Exists {
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
			int64(309), "submission_attachments", attachmentsSum[:],
		); err != nil {
			return fmt.Errorf("migrate repair v308 renumber: insert v309: %w", err)
		}
	} else if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 309`,
		"submission_attachments", attachmentsSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v308 renumber: update v309: %w", err)
	}

	return tx.Commit(ctx)
}

// repairMigration345RenumberCollision fixes demo DBs that applied submission_annotation_anchor
// as v345 before main's 345_migration_deploy_tracking.sql landed and the anchor migration
// was renumbered to 346 (plan 17.10 / admin console renumber).
func repairMigration345RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v345Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 345`,
	).Scan(&v345Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v345 renumber: %w", err)
	}
	if v345Desc != "submission_annotation_anchor" {
		return nil
	}
	var deployIDColumn bool
	err = c.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = $1
			  AND column_name = 'deploy_id'
		)`, sqlxMigrationsTable).Scan(&deployIDColumn)
	if err != nil {
		return fmt.Errorf("migrate repair v345 renumber: %w", err)
	}
	if deployIDColumn {
		return nil
	}

	deployBody, err := fs.ReadFile(fsys, dir+"/345_migration_deploy_tracking.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v345 renumber: read deploy_tracking: %w", err)
	}
	if _, err := c.Exec(ctx, string(deployBody)); err != nil {
		return fmt.Errorf("migrate repair v345 renumber: apply deploy_tracking: %w", err)
	}

	deploySum := sqlxChecksum(deployBody)
	anchorBody, err := fs.ReadFile(fsys, dir+"/346_submission_annotation_anchor.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v345 renumber: read submission_annotation_anchor: %w", err)
	}
	anchorSum := sqlxChecksum(anchorBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 345`,
		"migration_deploy_tracking", deploySum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v345 renumber: update v345: %w", err)
	}

	var v346Exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+sqlxMigrationsTable+` WHERE version = 346)`,
	).Scan(&v346Exists); err != nil {
		return fmt.Errorf("migrate repair v345 renumber: check v346: %w", err)
	}
	if !v346Exists {
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
			int64(346), "submission_annotation_anchor", anchorSum[:],
		); err != nil {
			return fmt.Errorf("migrate repair v345 renumber: insert v346: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// repairMigration334RenumberCollision fixes dev/demo DBs that applied
// quiz_manual_grading_is_correct_backfill as v334 before main's 334_marketplace.sql
// landed and the backfill was renumbered to 335.
func repairMigration334RenumberCollision(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	var v334Desc string
	err := c.QueryRow(ctx,
		`SELECT description FROM `+sqlxMigrationsTable+` WHERE version = 334`,
	).Scan(&v334Desc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v334 renumber: %w", err)
	}
	if v334Desc != "quiz_manual_grading_is_correct_backfill" {
		return nil
	}
	var marketplaceExists bool
	err = c.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'marketplace')`,
	).Scan(&marketplaceExists)
	if err != nil {
		return fmt.Errorf("migrate repair v334 renumber: %w", err)
	}
	if marketplaceExists {
		return nil
	}

	marketplaceBody, err := fs.ReadFile(fsys, dir+"/334_marketplace.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v334 renumber: read marketplace: %w", err)
	}
	if _, err := c.Exec(ctx, string(marketplaceBody)); err != nil {
		return fmt.Errorf("migrate repair v334 renumber: apply marketplace: %w", err)
	}

	marketplaceSum := sqlxChecksum(marketplaceBody)
	backfillBody, err := fs.ReadFile(fsys, dir+"/335_quiz_manual_grading_is_correct_backfill.sql")
	if err != nil {
		return fmt.Errorf("migrate repair v334 renumber: read backfill: %w", err)
	}
	backfillSum := sqlxChecksum(backfillBody)

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET description = $1, checksum = $2 WHERE version = 334`,
		"marketplace", marketplaceSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v334 renumber: update v334: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO `+sqlxMigrationsTable+` (version, description, success, checksum, execution_time) VALUES ($1, $2, true, $3, 0)`,
		int64(335), "quiz_manual_grading_is_correct_backfill", backfillSum[:],
	); err != nil {
		return fmt.Errorf("migrate repair v334 renumber: insert v335: %w", err)
	}
	return tx.Commit(ctx)
}

// repairDemoMigrationChecksums updates _sqlx_migrations when a listed version's stored
// checksum does not match the embedded file. Only runs when MIGRATE_REPAIR_CHECKSUMS
// is enabled.
func repairDemoMigrationChecksums(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string) error {
	if !migrateRepairChecksumsEnabled() {
		return nil
	}
	for _, m := range demoChecksumRepairMigrations {
		if err := repairOneMigrationChecksum(ctx, c, fsys, dir, m.version, m.file); err != nil {
			return err
		}
	}
	return nil
}

func repairOneMigrationChecksum(ctx context.Context, c *pgx.Conn, fsys fs.FS, dir string, version int64, file string) error {
	rel := dir + "/" + file
	body, err := fs.ReadFile(fsys, rel)
	if err != nil {
		return fmt.Errorf("migrate repair v%d: read %q: %w", version, rel, err)
	}
	currentSum := sqlxChecksum(body)

	var rowChecksum []byte
	err = c.QueryRow(ctx,
		`SELECT checksum FROM `+sqlxMigrationsTable+` WHERE version = $1`,
		version,
	).Scan(&rowChecksum)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("migrate repair v%d: %w", version, err)
	}
	if bytes.Equal(rowChecksum, currentSum[:]) {
		return nil
	}
	_, err = c.Exec(ctx,
		`UPDATE `+sqlxMigrationsTable+` SET checksum = $1 WHERE version = $2`,
		currentSum[:], version,
	)
	if err != nil {
		return fmt.Errorf("migrate repair v%d: %w", version, err)
	}
	return nil
}
