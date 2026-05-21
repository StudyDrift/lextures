package avscan_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/workers/avscan"
)

func TestWorkerQuarantinesEICAR(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	root := t.TempDir()
	local := &filestorage.LocalDriver{Root: root}
	eicar := []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*")
	key := "uploads/test/eicar.txt"
	if err := local.PutObject(ctx, key, bytes.NewReader(eicar), int64(len(eicar)), "text/plain"); err != nil {
		t.Fatal(err)
	}

	var tenantID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM tenant.organizations LIMIT 1`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	objID, err := storageobjects.Upsert(ctx, pool, tenantID, nil, key, "local", "text/plain", int64(len(eicar)), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := avscan.EnqueueForObject(ctx, pool, objID); err != nil {
		t.Fatal(err)
	}

	w := avscan.New(pool, local, clamav.NewClient("", true))
	w.LocalRoot = root
	for i := 0; i < 5; i++ {
		done, err := w.ProcessNext(ctx)
		if err != nil {
			t.Fatalf("process: %v", err)
		}
		if done {
			break
		}
	}

	obj, err := storageobjects.LoadByID(ctx, pool, objID)
	if err != nil || obj == nil {
		t.Fatalf("load: %v", obj)
	}
	if obj.ScanStatus != storageobjects.ScanQuarantined {
		t.Fatalf("status = %s, want quarantined", obj.ScanStatus)
	}
	qPath := filepath.Join(root, filepath.FromSlash(obj.ObjectKey))
	if _, err := os.Stat(qPath); err != nil {
		t.Fatalf("quarantine file missing: %v", err)
	}
}
