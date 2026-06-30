package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func integrationMigrateGatePath() string {
	if dir := os.Getenv("TMPDIR"); dir != "" {
		return filepath.Join(dir, "lextures-migrate-integration.lock")
	}
	return filepath.Join(os.TempDir(), "lextures-migrate-integration.lock")
}

// acquireIntegrationMigrateGate serializes Postgres-backed integration tests that share
// one DATABASE_URL across parallel `go test` package binaries. No-op when unset.
func acquireIntegrationMigrateGate() (func(), error) {
	if os.Getenv("DATABASE_URL") == "" {
		return func() {}, nil
	}
	path := integrationMigrateGatePath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open gate %q: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock gate %q: %w", path, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
