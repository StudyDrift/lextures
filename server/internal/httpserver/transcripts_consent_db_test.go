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
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupTranscriptConsentTest(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	orgID := organization.SeedDefaultOrgID
	em := fmt.Sprintf("consent-student-%d@test.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		pool.Close()
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, uid); err != nil {
		pool.Close()
		t.Fatalf("org: %v", err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config
SET webhook_url = 'https://example.com/hook',
    orders_ui_enabled = true,
    consent_required = true,
    auto_approval_enabled = false
WHERE id = 1
`); err != nil {
		pool.Close()
		t.Fatalf("config: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, uid.String(), em, "", "", nil)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFTranscripts: true}})
	return pool, h, tok, uid
}

func createThirdPartyOrder(t *testing.T, h http.Handler, tok string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{
				"adHocRecipient": map[string]any{
					"type":         "institution",
					"name":         "State University",
					"email":        "registrar@state.edu",
					"capabilities": []string{"secure_link_email"},
				},
				"deliveryMethod": "secure_link_email",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	return created.Order.ID
}

func submitOrder(t *testing.T, h http.Handler, tok, orderID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/submit", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out.Order.Status
}

func TestTranscriptConsent_SignAdvancesPastPending_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, _ := setupTranscriptConsentTest(t, ctx)
	defer pool.Close()

	orderID := createThirdPartyOrder(t, h, tok)
	if status := submitOrder(t, h, tok, orderID); status != "pending_consent" {
		t.Fatalf("AC-2: want pending_consent got %s", status)
	}

	// Preview
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID+"/consent/preview?locale=en", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", w.Code, w.Body.String())
	}
	var preview struct {
		Preview struct {
			RequiresConsent   bool   `json:"requiresConsent"`
			TextVersion       string `json:"textVersion"`
			AuthorizationText string `json:"authorizationText"`
		} `json:"preview"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &preview)
	if !preview.Preview.RequiresConsent || preview.Preview.TextVersion == "" || preview.Preview.AuthorizationText == "" {
		t.Fatalf("preview incomplete: %+v", preview.Preview)
	}

	// AC-1: sign
	signBody, _ := json.Marshal(map[string]any{
		"method":        "typed",
		"signatureData": "Alex Student",
		"agree":         true,
		"locale":        "en",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/consent", bytes.NewReader(signBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("User-Agent", "ConsentTest/1.0")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("sign: %d %s", w.Code, w.Body.String())
	}
	var signed struct {
		Consent struct {
			ID          string `json:"id"`
			TextVersion string `json:"textVersion"`
			PayloadHash string `json:"payloadHash"`
			SignerRole  string `json:"signerRole"`
		} `json:"consent"`
		Order struct {
			Status    string  `json:"status"`
			ConsentID *string `json:"consentId"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &signed)
	if signed.Consent.SignerRole != "student" || signed.Consent.PayloadHash == "" {
		t.Fatalf("consent: %+v", signed.Consent)
	}
	if signed.Order.Status == "pending_consent" {
		t.Fatal("AC-1: order should advance past pending_consent")
	}
	if signed.Order.ConsentID == nil {
		t.Fatal("expected consentId on order")
	}

	// AC-6: export contains exact text version
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID+"/consent/export", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("export: %d %s", w.Code, w.Body.String())
	}
	var exp struct {
		Export map[string]any `json:"export"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &exp)
	if exp.Export["textVersion"] != signed.Consent.TextVersion {
		t.Fatalf("export textVersion mismatch: %v", exp.Export["textVersion"])
	}
	if exp.Export["authorizationText"] == nil || exp.Export["authorizationText"] == "" {
		t.Fatal("export missing authorizationText")
	}
}

func TestTranscriptConsent_UnsignedBlockedAtPending_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, _ := setupTranscriptConsentTest(t, ctx)
	defer pool.Close()

	orderID := createThirdPartyOrder(t, h, tok)
	if status := submitOrder(t, h, tok, orderID); status != "pending_consent" {
		t.Fatalf("AC-2: want pending_consent got %s", status)
	}
}

func TestTranscriptConsent_RevokeBlocksUndelivered_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, _ := setupTranscriptConsentTest(t, ctx)
	defer pool.Close()

	orderID := createThirdPartyOrder(t, h, tok)
	_ = submitOrder(t, h, tok, orderID)
	signBody, _ := json.Marshal(map[string]any{
		"method": "typed", "signatureData": "Alex Student", "agree": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/consent", bytes.NewReader(signBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("sign: %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/consent/revoke", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", w.Code, w.Body.String())
	}
	var revoked struct {
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
		Consent struct {
			RevokedAt *string `json:"revokedAt"`
		} `json:"consent"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &revoked)
	if revoked.Order.Status != "pending_consent" {
		t.Fatalf("AC-4: want pending_consent got %s", revoked.Order.Status)
	}
	if revoked.Consent.RevokedAt == nil {
		t.Fatal("AC-4: expected revokedAt")
	}
}

func TestTranscriptConsent_SelfRecipientSkipsGate_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, _ := setupTranscriptConsentTest(t, ctx)
	defer pool.Close()

	_, status := createAndSubmitOrder(t, h, tok)
	if status == "pending_consent" {
		t.Fatal("AC-5: self-recipient should not require third-party consent")
	}
	if status != "in_review" && status != "processing" {
		t.Fatalf("AC-5: unexpected status %s", status)
	}
}

func TestTranscriptConsent_MinorRequiresGuardian_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tok, studentID := setupTranscriptConsentTest(t, ctx)
	defer pool.Close()

	if _, err := pool.Exec(ctx, `UPDATE "user".users SET is_minor = TRUE WHERE id = $1`, studentID); err != nil {
		t.Fatalf("minor: %v", err)
	}
	orgID := organization.SeedDefaultOrgID
	parentEmail := fmt.Sprintf("consent-parent-%d@test.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	prow, err := user.InsertUser(ctx, pool, parentEmail, ph, nil)
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	parentID, _ := uuid.Parse(prow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1, account_type = 'parent' WHERE id = $2`, orgID, parentID); err != nil {
		t.Fatalf("parent org: %v", err)
	}
	if _, err := parentlinks.UpsertActive(ctx, pool, orgID, parentID, studentID, "guardian", &parentID); err != nil {
		t.Fatalf("link: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	parentTok, _ := signer.Sign(ctx, parentID.String(), parentEmail, "", "", nil)

	orderID := createThirdPartyOrder(t, h, tok)
	if status := submitOrder(t, h, tok, orderID); status != "pending_consent" {
		t.Fatalf("want pending_consent got %s", status)
	}

	// AC-3: student alone cannot authorize
	signBody, _ := json.Marshal(map[string]any{
		"method": "typed", "signatureData": "Minor Student", "agree": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/consent", bytes.NewReader(signBody))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("AC-3 student sign: want 403 got %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/parent/transcripts/orders/"+orderID+"/consent", bytes.NewReader(signBody))
	req.Header.Set("Authorization", "Bearer "+parentTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("AC-3 guardian sign: %d %s", w.Code, w.Body.String())
	}
	var signed struct {
		Consent struct {
			SignerRole           string  `json:"signerRole"`
			GuardianRelationship *string `json:"guardianRelationship"`
		} `json:"consent"`
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &signed)
	if signed.Consent.SignerRole != "guardian" {
		t.Fatalf("want guardian role got %s", signed.Consent.SignerRole)
	}
	if signed.Consent.GuardianRelationship == nil || *signed.Consent.GuardianRelationship == "" {
		t.Fatal("expected guardian relationship recorded")
	}
	if signed.Order.Status == "pending_consent" {
		t.Fatal("guardian sign should advance order")
	}
}
