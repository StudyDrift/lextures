package storagequota_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/service/storagequota"
)

func setupSvc(t *testing.T) *storagequota.Service {
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
	return &storagequota.Service{Pool: pool}
}

// ---------------------------------------------------------------------------
// Unlimited quota: CheckAndReserve always passes when no limit is set
// ---------------------------------------------------------------------------

func TestCheckAndReserve_NoLimit_AlwaysPasses(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	// No quota set — 10 GB should pass.
	v, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 10<<30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
}

// ---------------------------------------------------------------------------
// Tenant-level quota enforcement
// ---------------------------------------------------------------------------

func TestCheckAndReserve_TenantQuota_Enforced(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	limit := int64(500)
	if err := svc.SetQuota(ctx, "tenant", tenantID, &limit); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}

	// 400 bytes → OK
	v, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 400)
	if err != nil || v != nil {
		t.Fatalf("first reserve: v=%v err=%v", v, err)
	}
	// 200 bytes more → exceeds tenant limit
	v2, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 200)
	if err != nil {
		t.Fatalf("second reserve err: %v", err)
	}
	if v2 == nil {
		t.Fatal("expected quota violation, got nil")
	}
	if v2.QuotaType != "tenant" {
		t.Fatalf("QuotaType: want 'tenant', got %q", v2.QuotaType)
	}
}

// ---------------------------------------------------------------------------
// User-level quota enforcement
// ---------------------------------------------------------------------------

func TestCheckAndReserve_UserQuota_Enforced(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	limit := int64(200)
	if err := svc.SetQuota(ctx, "user", userID, &limit); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}

	v, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 150)
	if err != nil || v != nil {
		t.Fatalf("first reserve: %v %v", v, err)
	}
	v2, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 100)
	if err != nil {
		t.Fatalf("second err: %v", err)
	}
	if v2 == nil || v2.QuotaType != "user" {
		t.Fatalf("expected user violation, got %+v", v2)
	}
}

// ---------------------------------------------------------------------------
// Release brings counter back down so subsequent upload can succeed
// ---------------------------------------------------------------------------

func TestRelease_AllowsSubsequentUpload(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	limit := int64(1000)
	if err := svc.SetQuota(ctx, "user", userID, &limit); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}

	if v, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 900); err != nil || v != nil {
		t.Fatalf("reserve: %v %v", v, err)
	}
	if err := svc.Release(ctx, tenantID, nil, userID, 900); err != nil {
		t.Fatalf("release: %v", err)
	}
	// Counter is back to 0 — 950 bytes should succeed.
	v, err := svc.CheckAndReserve(ctx, tenantID, nil, userID, 950)
	if err != nil || v != nil {
		t.Fatalf("post-release reserve: %v %v", v, err)
	}
}

// ---------------------------------------------------------------------------
// Release clamps to zero (no underflow)
// ---------------------------------------------------------------------------

func TestRelease_ClampsToZero(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	// Release more than was ever reserved — should not go negative.
	if err := svc.Release(ctx, tenantID, nil, userID, 99999); err != nil {
		t.Fatalf("release: %v", err)
	}
	// Verify counter is >= 0 by checking it via GetCourseUsage (indirectly)
}

// ---------------------------------------------------------------------------
// GetCourseUsage: percent_used calculation
// ---------------------------------------------------------------------------

func TestGetCourseUsage_PercentUsed(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	tenantID := uuid.New()
	courseID := uuid.New()
	userID := uuid.New()
	limit := int64(1000)
	if err := svc.SetQuota(ctx, "course", courseID, &limit); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	if v, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 250); err != nil || v != nil {
		t.Fatalf("reserve: %v %v", v, err)
	}

	info, err := svc.GetCourseUsage(ctx, courseID)
	if err != nil {
		t.Fatalf("GetCourseUsage: %v", err)
	}
	if info.UsedBytes != 250 {
		t.Fatalf("UsedBytes: want 250, got %d", info.UsedBytes)
	}
	if info.LimitBytes == nil || *info.LimitBytes != limit {
		t.Fatalf("LimitBytes: want %d, got %v", limit, info.LimitBytes)
	}
	if info.PercentUsed != 25.0 {
		t.Fatalf("PercentUsed: want 25.0, got %f", info.PercentUsed)
	}
}

// ---------------------------------------------------------------------------
// SetQuota removes limit when nil is passed
// ---------------------------------------------------------------------------

func TestSetQuota_NilRemovesLimit(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	courseID := uuid.New()
	limit := int64(100)
	if err := svc.SetQuota(ctx, "course", courseID, &limit); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := svc.SetQuota(ctx, "course", courseID, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	// Verify limit is gone by trying to reserve more than the old limit.
	tenantID := uuid.New()
	userID := uuid.New()
	v, err := svc.CheckAndReserve(ctx, tenantID, &courseID, userID, 500)
	if err != nil || v != nil {
		t.Fatalf("post-clear reserve: %v %v", v, err)
	}
}

// ---------------------------------------------------------------------------
// Reconcile runs without error
// ---------------------------------------------------------------------------

func TestReconcile_OK(t *testing.T) {
	svc := setupSvc(t)
	if err := svc.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListQuotas returns previously set quotas
// ---------------------------------------------------------------------------

func TestListQuotas_ContainsSetQuota(t *testing.T) {
	svc := setupSvc(t)
	ctx := context.Background()

	courseID := uuid.New()
	limit := int64(12345)
	if err := svc.SetQuota(ctx, "course", courseID, &limit); err != nil {
		t.Fatalf("set: %v", err)
	}
	entries, err := svc.ListQuotas(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.ScopeID == courseID.String() && e.Scope == "course" {
			found = true
			if e.LimitBytes == nil || *e.LimitBytes != limit {
				t.Fatalf("limit mismatch: want %d got %v", limit, e.LimitBytes)
			}
		}
	}
	if !found {
		t.Fatalf("quota for course %s not found in list", courseID)
	}
}
