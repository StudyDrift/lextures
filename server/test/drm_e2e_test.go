// Package test holds end-to-end tests.
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	gofpdf "github.com/jung-kurt/gofpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/httpserver"
	"github.com/lextures/lextures/server/internal/migrate"
	drmrepo "github.com/lextures/lextures/server/internal/repos/drm"
	"github.com/lextures/lextures/server/internal/repos/user"
	drmservice "github.com/lextures/lextures/server/internal/service/drm"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// drmEnv holds the wired server and helpers for DRM e2e tests.
type drmEnv struct {
	srv       *httptest.Server
	pool      *pgxpool.Pool
	signer    *auth.JWTSigner
	drmSvc    *drmservice.Service
	userID    uuid.UUID
	objectID  uuid.UUID
	objectKey string
}

const drmTestSecret = "drm-e2e-secret-32byteslong-xxxxx"

func setupDRM(t *testing.T) *drmEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping DRM e2e tests")
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

	signer := auth.NewJWTSigner("drm-e2e-jwt-secret-min32chars-xx")

	// Create a test user with display name.
	email := fmt.Sprintf("drm-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!drm")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "Alice Johnson"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	// Minimal course.
	cc := fmt.Sprintf("C-DM%04d", time.Now().UnixNano()%10000)
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO course.courses (course_code, title, created_by_user_id)
		VALUES ($1, 'DRM E2E Course', $2) RETURNING id
	`, cc, userID).Scan(&courseID)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO course.course_enrollments (course_id, user_id, role)
		VALUES ($1, $2, 'student')
	`, courseID, userID)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	// Resolve tenant_id from the user's org.
	var tenantID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, userID).Scan(&tenantID)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}

	// Register a storage.objects row.
	objectKey := fmt.Sprintf("drm-e2e/%s/lecture.pdf", userID)
	var objectID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO storage.objects (tenant_id, course_id, object_key, bucket, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, 'e2e-bucket', 'application/pdf', 1024, $4)
		RETURNING id
	`, tenantID, courseID, objectKey, userID).Scan(&objectID)
	if err != nil {
		t.Fatalf("storage object: %v", err)
	}

	// Write a real PDF to local storage so the watermark handler can read it.
	storageDir := t.TempDir()
	pdfBytes := drmTestPDF(t)
	storage, err := filestorage.New(filestorage.BackendConfig{Backend: "local", LocalRoot: storageDir})
	if err != nil {
		t.Fatalf("storage driver: %v", err)
	}
	if err := storage.PutObject(ctx, objectKey, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf"); err != nil {
		t.Fatalf("put pdf: %v", err)
	}

	drmSvc := drmservice.New(pool, drmservice.Config{
		Secret:   []byte(drmTestSecret),
		TokenTTL: time.Hour,
	})

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Storage:   storage,
		DRM:       drmSvc,
	}))
	t.Cleanup(srv.Close)

	return &drmEnv{
		srv:       srv,
		pool:      pool,
		signer:    signer,
		drmSvc:    drmSvc,
		userID:    userID,
		objectID:  objectID,
		objectKey: objectKey,
	}
}

func drmTestPDF(t *testing.T) []byte {
	t.Helper()
	f := gofpdf.New("P", "mm", "A4", "")
	f.AddPage()
	f.SetFont("Arial", "", 12)
	f.Cell(40, 10, "DRM E2E lecture notes")
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		t.Fatalf("create test PDF: %v", err)
	}
	return buf.Bytes()
}

func drmBearer(t *testing.T, env *drmEnv) string {
	t.Helper()
	tok, err := env.signer.Sign(context.Background(), env.userID.String(), "test@example.com", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return "Bearer " + tok
}

// TestDRM_E2E_LicenseNoAuth — no JWT returns 401.
func TestDRM_E2E_LicenseNoAuth(t *testing.T) {
	env := setupDRM(t)
	resp, err := http.Post(env.srv.URL+"/api/v1/files/"+env.objectID.String()+"/license", "", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestDRM_E2E_LicenseNoneType — drm_type=none returns token + granted=true.
func TestDRM_E2E_LicenseNoneType(t *testing.T) {
	env := setupDRM(t)

	req, _ := http.NewRequest(http.MethodPost,
		env.srv.URL+"/api/v1/files/"+env.objectID.String()+"/license", nil)
	req.Header.Set("Authorization", drmBearer(t, env))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Granted bool   `json:"granted"`
		DRMType string `json:"drmType"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Granted {
		t.Fatal("expected granted=true for drm_type=none")
	}
	if body.Token == "" {
		t.Fatal("expected a non-empty token in response")
	}
}

// TestDRM_E2E_WatermarkPDF — drm_type=watermark_only returns a PDF with the user's identity stamped.
func TestDRM_E2E_WatermarkPDF(t *testing.T) {
	env := setupDRM(t)

	// Mark the object as watermark_only.
	ctx := context.Background()
	if err := drmrepo.SetObjectDRM(ctx, env.pool, env.objectID, drmrepo.DRMTypeWatermark, nil, nil); err != nil {
		t.Fatalf("set drm: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost,
		env.srv.URL+"/api/v1/files/"+env.objectID.String()+"/license", nil)
	req.Header.Set("Authorization", drmBearer(t, env))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// AC-1: response must be a PDF.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/pdf") {
		t.Fatalf("expected Content-Type application/pdf, got %q", ct)
	}
	if resp.Header.Get("X-DRM-Type") != "watermark_only" {
		t.Fatalf("expected X-DRM-Type header watermark_only, got %q", resp.Header.Get("X-DRM-Type"))
	}
	// Cache-Control must be no-store (AC-1 / FR-5: do not cache user-bound content).
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %q", cc)
	}

	// Verify output is a valid PDF.
	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.HasPrefix(out.String(), "%PDF") {
		t.Fatal("response body does not start with %PDF")
	}
}

// TestDRM_E2E_TokenBinding — token must be user-bound; different user's token is invalid (AC-2).
func TestDRM_E2E_TokenBinding(t *testing.T) {
	env := setupDRM(t)

	aliceTok := env.drmSvc.SignToken(env.objectID, env.userID)

	// Alice's own token validates.
	if !env.drmSvc.ValidateToken(aliceTok, env.objectID, env.userID) {
		t.Fatal("token must validate for issuing user")
	}

	// Same token does NOT validate for a different user (AC-2).
	bob := uuid.New()
	if env.drmSvc.ValidateToken(aliceTok, env.objectID, bob) {
		t.Fatal("Alice's token must not validate for Bob (AC-2)")
	}
}

// TestDRM_E2E_LicenseRequestLogged — every license request is logged to the audit table (FR-6).
func TestDRM_E2E_LicenseRequestLogged(t *testing.T) {
	env := setupDRM(t)

	req, _ := http.NewRequest(http.MethodPost,
		env.srv.URL+"/api/v1/files/"+env.objectID.String()+"/license", nil)
	req.Header.Set("Authorization", drmBearer(t, env))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Audit row should exist.
	ctx := context.Background()
	var count int64
	if err := env.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM storage.drm_license_requests
		WHERE object_id = $1 AND user_id = $2 AND granted = true
	`, env.objectID, env.userID).Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one audit row (FR-6)")
	}
}

// TestDRM_E2E_AdminSetDRMType_Unauthorized — non-admin receives 403.
func TestDRM_E2E_AdminSetDRMType_Unauthorized(t *testing.T) {
	env := setupDRM(t)

	body := `{"drmType":"watermark_only"}`
	req, _ := http.NewRequest(http.MethodPut,
		env.srv.URL+"/api/v1/admin/files/"+env.objectID.String()+"/drm",
		strings.NewReader(body))
	req.Header.Set("Authorization", drmBearer(t, env))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", resp.StatusCode)
	}
}

// TestDRM_E2E_AnomaliesEndpoint_Unauthorized — non-admin receives 403.
func TestDRM_E2E_AnomaliesEndpoint_Unauthorized(t *testing.T) {
	env := setupDRM(t)

	req, _ := http.NewRequest(http.MethodGet,
		env.srv.URL+"/api/v1/admin/drm/anomalies", nil)
	req.Header.Set("Authorization", drmBearer(t, env))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", resp.StatusCode)
	}
}

// TestDRM_E2E_InvalidObjectID — malformed UUID returns 400.
func TestDRM_E2E_InvalidObjectID(t *testing.T) {
	env := setupDRM(t)

	req, _ := http.NewRequest(http.MethodPost,
		env.srv.URL+"/api/v1/files/not-a-uuid/license", nil)
	req.Header.Set("Authorization", drmBearer(t, env))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", resp.StatusCode)
	}
}
