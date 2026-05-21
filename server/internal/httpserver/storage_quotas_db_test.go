package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/storagequota"
)

// setupQuotaDB migrates and returns a pool plus a signed admin JWT.
func setupQuotaDB(t *testing.T) (*pgxpool.Pool, *auth.JWTSigner, string) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	em := fmt.Sprintf("quota-admin-%d@test.invalid", time.Now().UnixNano())
	ph, err := auth.HashPassword("longtestpassword1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if err = rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("rbac: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return pool, signer, tok
}

// ---------------------------------------------------------------------------
// Admin: list quotas (empty initially)
// ---------------------------------------------------------------------------

func TestAdminStorageQuotasList_OK_Pg(t *testing.T) {
	pool, signer, tok := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, StorageQuota: svc})

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/storage-quotas", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body.String())
	}
	var entries []storagequota.QuotaEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// May be non-empty from prior test runs; just verify it's a valid JSON array.
}

// ---------------------------------------------------------------------------
// Admin: set quota and read it back via storage-usage endpoint
// ---------------------------------------------------------------------------

func TestAdminStorageQuotasPut_ThenGetCourseUsage_Pg(t *testing.T) {
	pool, signer, tok := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, StorageQuota: svc})
	ctx := context.Background()

	// Create a course.
	courseCode := fmt.Sprintf("quota-test-%d", time.Now().UnixNano())
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO course.courses (course_code, title, created_by_user_id)
		VALUES ($1, 'Quota Test', (SELECT id FROM "user".users ORDER BY created_at LIMIT 1))
		RETURNING id`, courseCode).Scan(&courseID); err != nil {
		t.Fatalf("create course: %v", err)
	}

	// Set a 1 GB quota via PUT.
	limitBytes := int64(1 << 30) // 1 GiB
	body := map[string]int64{"limit_bytes": limitBytes}
	b, _ := json.Marshal(body)
	putURL := fmt.Sprintf("/api/v1/admin/storage-quotas/course/%s", courseID)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, putURL, bytes.NewReader(b))
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("PUT quota: expected 204 got %d: %s", rr.Code, rr.Body.String())
	}

	// Enroll the admin user so requireCourseAccess works.
	if _, err := pool.Exec(ctx, `
		INSERT INTO course.course_enrollments (course_id, user_id, role, active)
		SELECT $1, id, 'instructor', true FROM "user".users ORDER BY created_at LIMIT 1`,
		courseID); err != nil {
		t.Fatalf("enroll: %v", err)
	}

	// GET storage-usage for the course.
	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/storage-usage", nil)
	r2.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET usage: expected 200 got %d: %s", rr2.Code, rr2.Body.String())
	}
	var info storagequota.UsageInfo
	if err := json.NewDecoder(rr2.Body).Decode(&info); err != nil {
		t.Fatalf("decode usage: %v", err)
	}
	if info.LimitBytes == nil || *info.LimitBytes != limitBytes {
		t.Fatalf("limit_bytes: expected %d got %v", limitBytes, info.LimitBytes)
	}
	if info.UsedBytes != 0 {
		t.Fatalf("used_bytes: expected 0 got %d", info.UsedBytes)
	}
}

// ---------------------------------------------------------------------------
// Admin: reconcile runs without error
// ---------------------------------------------------------------------------

func TestAdminStorageQuotasReconcile_OK_Pg(t *testing.T) {
	pool, signer, tok := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, StorageQuota: svc})

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/storage-quotas/reconcile", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Quota enforcement: CheckAndReserve rejects upload that would breach limit
// ---------------------------------------------------------------------------

func TestStorageQuota_CheckAndReserve_ExceedsLimit_Pg(t *testing.T) {
	pool, _, _ := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	ctx := context.Background()

	tenantID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	limitBytes := int64(1000)

	// Set a 1000-byte quota on the course.
	if err := svc.SetQuota(ctx, "course", courseID, &limitBytes); err != nil {
		t.Fatalf("set quota: %v", err)
	}

	// First upload: 600 bytes — should pass.
	v, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 600)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if v != nil {
		t.Fatalf("first reserve: unexpected violation %+v", v)
	}

	// Second upload: 500 bytes — should fail (600+500 > 1000).
	v2, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 500)
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if v2 == nil {
		t.Fatal("second reserve: expected quota violation, got nil")
	}
	if v2.QuotaType != "course" {
		t.Fatalf("violation type: expected 'course' got %q", v2.QuotaType)
	}
	if v2.LimitBytes != limitBytes {
		t.Fatalf("violation limit: expected %d got %d", limitBytes, v2.LimitBytes)
	}
}

// ---------------------------------------------------------------------------
// Quota enforcement: Release decrements counters
// ---------------------------------------------------------------------------

func TestStorageQuota_Release_Pg(t *testing.T) {
	pool, _, _ := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	ctx := context.Background()

	tenantID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	limitBytes := int64(1000)

	if err := svc.SetQuota(ctx, "course", courseID, &limitBytes); err != nil {
		t.Fatalf("set quota: %v", err)
	}

	// Reserve 800 bytes.
	if v, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 800); err != nil || v != nil {
		t.Fatalf("reserve: v=%v err=%v", v, err)
	}

	// Release 800 bytes.
	if err := svc.Release(ctx, tenantID, &courseID, userID, 800); err != nil {
		t.Fatalf("release: %v", err)
	}

	// Now 900 bytes should succeed.
	v, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 900)
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if v != nil {
		t.Fatalf("after release: unexpected violation %+v", v)
	}
}

// ---------------------------------------------------------------------------
// Admin: invalid scope returns 400
// ---------------------------------------------------------------------------

func TestAdminStorageQuotasPut_InvalidScope_Returns400_Pg(t *testing.T) {
	pool, signer, tok := setupQuotaDB(t)
	svc := &storagequota.Service{Pool: pool}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, StorageQuota: svc})

	b, _ := json.Marshal(map[string]int64{"limit_bytes": 100})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut,
		"/api/v1/admin/storage-quotas/invalid-scope/00000000-0000-0000-0000-000000000001",
		bytes.NewReader(b))
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d: %s", rr.Code, rr.Body.String())
	}
}
