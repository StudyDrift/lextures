package migrate

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server"
)

// TestRepairMigration345RenumberCollision_Integration simulates a deploy droplet DB that
// applied submission_annotation_anchor as v345 before plan 17.10 renumbered deploy tracking
// to 345 and the anchor migration to 346.
func TestRepairMigration345RenumberCollision_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("use full go test to exercise migration repair with Postgres")
	}
	dsn := isolatedMigrationDSN(t)
	t.Setenv("MIGRATE_REPAIR_CHECKSUMS", "1")

	ctx := context.Background()
	release, err := acquireIntegrationMigrateGate()
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if err := runWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("baseline migrate: %v", err)
	}

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()

	// Roll back migration metadata to the pre-renumber demo state at v345.
	if _, err := conn.Exec(ctx, `DELETE FROM _sqlx_migrations WHERE version >= 345`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `ALTER TABLE _sqlx_migrations DROP COLUMN IF EXISTS deploy_id`); err != nil {
		t.Fatal(err)
	}
	anchorBody, err := serverdata.Migrations.ReadFile("migrations/346_submission_annotation_anchor.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, string(anchorBody)); err != nil {
		t.Fatal(err)
	}
	anchorSum := sqlxChecksum(anchorBody)
	if _, err := conn.Exec(ctx,
		`INSERT INTO _sqlx_migrations (version, description, success, checksum, execution_time)
		 VALUES (345, 'submission_annotation_anchor', true, $1, 0)`,
		anchorSum[:],
	); err != nil {
		t.Fatal(err)
	}

	if err := runWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("repair migrate: %v", err)
	}

	var v345Desc, v346Desc string
	if err := conn.QueryRow(ctx, `SELECT description FROM _sqlx_migrations WHERE version = 345`).Scan(&v345Desc); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, `SELECT description FROM _sqlx_migrations WHERE version = 346`).Scan(&v346Desc); err != nil {
		t.Fatal(err)
	}
	if v345Desc != "migration_deploy_tracking" {
		t.Fatalf("v345 description = %q, want migration_deploy_tracking", v345Desc)
	}
	if v346Desc != "submission_annotation_anchor" {
		t.Fatalf("v346 description = %q, want submission_annotation_anchor", v346Desc)
	}

	var latest int64
	if err := conn.QueryRow(ctx, `SELECT max(version) FROM _sqlx_migrations`).Scan(&latest); err != nil {
		t.Fatal(err)
	}
	if latest < 350 {
		t.Fatalf("latest migration version = %d, want >= 350", latest)
	}
}
