package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
)

// adminAuditSetup runs migrations and creates an audit-read-capable user.
// Returns pool, signer, token, userID.
func adminAuditSetup(t *testing.T) (*pgxpool.Pool, *auth.JWTSigner, string, uuid.UUID) {
	t.Helper()
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
	t.Cleanup(func() { pool.Close() })

	em := "audit-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("role: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return pool, signer, tok, uid
}

func TestAdminAuditLog_RecordAndQuery_Pg(t *testing.T) {
	pool, signer, tok, actorID := adminAuditSetup(t)
	ctx := context.Background()

	targetType := "user"
	targetID := uuid.New()
	_, err := auditservice.Record(ctx, pool, auditservice.RecordParams{
		EventType:   auditservice.EventRoleGrant,
		ActorID:     actorID,
		ActorIP:     strPtr("127.0.0.1"),
		TargetType:  &targetType,
		TargetID:    &targetID,
		AfterValue:  []byte(`{"role":"instructor"}`),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminAuditLogEnabled: true},
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/compliance/audit-log?eventType=role_grant", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
	}

	var out struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Events) == 0 {
		t.Fatal("expected at least one audit event")
	}
	found := false
	for _, e := range out.Events {
		if e["actorId"] == actorID.String() && e["eventType"] == "role_grant" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("role_grant event for actor not found in response: %v", out.Events)
	}
}

func TestAdminAuditLog_GetByID_Pg(t *testing.T) {
	pool, signer, tok, actorID := adminAuditSetup(t)
	ctx := context.Background()

	eventID, err := auditservice.Record(ctx, pool, auditservice.RecordParams{
		EventType: auditservice.EventGradeOverride,
		ActorID:   actorID,
		BeforeValue: []byte(`{"grade":"B"}`),
		AfterValue:  []byte(`{"grade":"A"}`),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminAuditLogEnabled: true},
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/compliance/audit-log/"+eventID.String(), nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("getByID: %d %s", rr.Code, rr.Body.String())
	}

	var e map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&e); err != nil {
		t.Fatal(err)
	}
	if e["eventId"] != eventID.String() {
		t.Errorf("eventId=%v want %s", e["eventId"], eventID)
	}
	if e["eventType"] != "grade_override" {
		t.Errorf("eventType=%v want grade_override", e["eventType"])
	}
}

func TestAdminAuditLog_GetByID_NotFound_Pg(t *testing.T) {
	pool, signer, tok, _ := adminAuditSetup(t)
	ctx := context.Background()

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminAuditLogEnabled: true},
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/compliance/audit-log/"+uuid.New().String(), nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status=%d want 404", rr.Code)
	}
}

func TestAdminAuditLog_Export_CSV_Pg(t *testing.T) {
	pool, signer, tok, actorID := adminAuditSetup(t)
	ctx := context.Background()

	_, err := auditservice.Record(ctx, pool, auditservice.RecordParams{
		EventType: auditservice.EventDataExport,
		ActorID:   actorID,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminAuditLogEnabled: true},
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/compliance/audit-log/export?format=csv", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("export: %d %s", rr.Code, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type=%s want text/csv", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event_id") {
		t.Error("CSV missing header row")
	}
	if !strings.Contains(body, "data_export") {
		t.Error("CSV missing data_export event type")
	}
}

func TestAdminAuditLog_AppendOnlyTrigger_Pg(t *testing.T) {
	pool, _, _, actorID := adminAuditSetup(t)
	ctx := context.Background()

	eventID, err := auditservice.Record(ctx, pool, auditservice.RecordParams{
		EventType: auditservice.EventContentDelete,
		ActorID:   actorID,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Attempt DELETE — must be rejected by the append-only trigger (AC-2).
	_, err = pool.Exec(ctx,
		`DELETE FROM compliance.admin_audit_log WHERE event_id = $1`, eventID)
	if err == nil {
		t.Fatal("DELETE succeeded; expected trigger to reject it")
	}
	if !strings.Contains(err.Error(), "append-only") {
		t.Errorf("unexpected error: %v", err)
	}

	// Attempt UPDATE — must also be rejected.
	_, err = pool.Exec(ctx,
		`UPDATE compliance.admin_audit_log SET event_type = 'tampered' WHERE event_id = $1`, eventID)
	if err == nil {
		t.Fatal("UPDATE succeeded; expected trigger to reject it")
	}
	if !strings.Contains(err.Error(), "append-only") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAdminAuditLog_Forbidden_WithoutAuditPermission_Pg(t *testing.T) {
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

	// Create a regular user (no Global Admin role, no audit:read permission).
	em := "noaudit-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminAuditLogEnabled: true},
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-log", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusForbidden {
		t.Errorf("status=%d want 403", rr.Code)
	}
}

