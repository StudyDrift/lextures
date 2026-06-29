package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RollbackLatest applies the down.sql companion for the most recently applied migration.
// Returns ErrRollbackNotSupported when the companion is a documented no-op stub.
func RollbackLatest(ctx context.Context, fsys fs.FS, dsn string) error {
	if dsn == "" {
		return fmt.Errorf("migrate: empty database URL")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("migrate: parse config: %w", err)
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("migrate: connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	return rollbackLatestLocked(ctx, conn, fsys, "migrations")
}

// RollbackLatestFromPool is like RollbackLatest but uses an existing pool's DSN.
func RollbackLatestFromPool(ctx context.Context, fsys fs.FS, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("migrate: nil pool")
	}
	return RollbackLatest(ctx, fsys, pool.Config().ConnString())
}

// ErrRollbackNotSupported is returned when the down.sql file is a documented stub.
var ErrRollbackNotSupported = errors.New("migrate: rollback not supported for this migration")

func rollbackLatestLocked(ctx context.Context, conn *pgx.Conn, fsys fs.FS, dir string) error {
	lid, err := takeAdvisoryLock(ctx, conn)
	if err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	defer func() { _ = releaseAdvisoryLock(context.Background(), conn, lid) }()

	var version int64
	var description string
	sel := fmt.Sprintf(
		"SELECT version, description FROM %s WHERE success = true ORDER BY version DESC LIMIT 1",
		sqlxMigrationsTable,
	)
	if err := conn.QueryRow(ctx, sel).Scan(&version, &description); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("migrate: no applied migrations to roll back")
		}
		return fmt.Errorf("migrate: latest migration: %w", err)
	}

	upName, err := findMigrationNameByVersion(fsys, dir, int(version))
	if err != nil {
		return err
	}
	upPath := dir + "/" + upName
	downPath := downMigrationPath(upPath)

	downBody, err := fs.ReadFile(fsys, downPath)
	if err != nil {
		return fmt.Errorf("migrate: read %q: %w", downPath, err)
	}
	if !rollbackSupported(downBody) {
		return fmt.Errorf("%w (%s)", ErrRollbackNotSupported, upName)
	}

	t0 := time.Now()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(downBody)); err != nil {
		return fmt.Errorf("migrate: rollback v%d: %w", version, err)
	}
	del := fmt.Sprintf("DELETE FROM %s WHERE version = $1", sqlxMigrationsTable)
	if _, err := tx.Exec(ctx, del, version); err != nil {
		return fmt.Errorf("migrate: remove migration row v%d: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	elapsed := time.Since(t0)
	slog.Info("migration rolled back",
		"version", version,
		"description", description,
		"file", upName,
		"duration_ms", elapsed.Milliseconds(),
	)
	return nil
}

func findMigrationNameByVersion(fsys fs.FS, dir string, version int) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", fmt.Errorf("migrate: readdir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !isUpMigration(e.Name()) {
			continue
		}
		mf, perr := parseMigrationName(dir + "/" + e.Name())
		if perr != nil {
			continue
		}
		if mf.Version == version {
			return mf.Name, nil
		}
	}
	return "", fmt.Errorf("migrate: no up migration file for version %d", version)
}
