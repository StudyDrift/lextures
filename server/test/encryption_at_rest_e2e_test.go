package test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	gdprrepo "github.com/lextures/lextures/server/internal/repos/gdpr"
	"github.com/lextures/lextures/server/internal/repos/user"
	coppasvc "github.com/lextures/lextures/server/internal/service/coppa"
)

func TestEncryptionAtRest_COPPAAndDSAR(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	ph, err := auth.HashPassword("Passw0rd!enc")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	displayName := "Encryption Test User"
	email := "enc-" + time.Now().UTC().Format("20060102150405.000000") + "@test.invalid"
	created, err := user.InsertUser(ctx, pool, email, ph, &displayName)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(created.ID)

	dob := time.Date(2016, 6, 15, 0, 0, 0, 0, time.UTC)
	if err := coppasvc.FlagMinorAccount(ctx, pool, userID, dob, "parent@example.com"); err != nil {
		t.Fatalf("flag minor account: %v", err)
	}

	status, err := coppasvc.GetUserConsentStatus(ctx, pool, userID)
	if err != nil {
		t.Fatalf("get consent status: %v", err)
	}
	if status.ParentEmail == nil || *status.ParentEmail != "parent@example.com" {
		t.Fatalf("decrypted parent email mismatch: %#v", status.ParentEmail)
	}

	var rawParentEmail string
	var rawDOB string
	if err := pool.QueryRow(ctx, `SELECT parent_email, date_of_birth FROM "user".users WHERE id = $1`, userID).Scan(&rawParentEmail, &rawDOB); err != nil {
		t.Fatalf("select raw encrypted columns: %v", err)
	}
	if !strings.HasPrefix(rawParentEmail, "enc:v1:") {
		t.Fatalf("parent_email should be encrypted, got %q", rawParentEmail)
	}
	if !strings.HasPrefix(rawDOB, "enc:v1:") {
		t.Fatalf("date_of_birth should be encrypted, got %q", rawDOB)
	}

	dsarID, err := gdprrepo.InsertDSARRequest(ctx, pool, nil, userID, "access")
	if err != nil {
		t.Fatalf("insert dsar: %v", err)
	}
	archive := `{"hello":"world"}`
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	adminID := userID
	if err := gdprrepo.UpdateDSARStatus(ctx, pool, dsarID, adminID, "completed", &archive, &expiresAt, nil); err != nil {
		t.Fatalf("update dsar status: %v", err)
	}

	var rawArchive string
	if err := pool.QueryRow(ctx, `SELECT archive_url FROM compliance.dsar_requests WHERE id = $1`, dsarID).Scan(&rawArchive); err != nil {
		t.Fatalf("select raw archive url: %v", err)
	}
	if !strings.HasPrefix(rawArchive, "enc:v1:") {
		t.Fatalf("archive_url should be encrypted, got %q", rawArchive)
	}

	row, err := gdprrepo.GetDSARRequest(ctx, pool, dsarID)
	if err != nil {
		t.Fatalf("get dsar request: %v", err)
	}
	if row == nil || row.ArchiveURL == nil || *row.ArchiveURL != archive {
		t.Fatalf("decrypted archive_url mismatch: %#v", row)
	}
}
