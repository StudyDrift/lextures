package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupTranscriptLifecycleTest(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string, string, uuid.UUID, uuid.UUID) {
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
	mkUser := func(prefix string) (uuid.UUID, string) {
		em := fmt.Sprintf("%s-%d@test.com", prefix, time.Now().UnixNano())
		ph, _ := auth.HashPassword("longpassword0longpassword0")
		row, err := user.InsertUser(ctx, pool, em, ph, nil)
		if err != nil {
			pool.Close()
			t.Fatalf("user: %v", err)
		}
		uid, _ := uuid.Parse(row.ID)
		if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, uid); err != nil {
			pool.Close()
			t.Fatalf("set org: %v", err)
		}
		return uid, em
	}

	studentID, studentEmail := mkUser("transcript-life-student")
	adminID, adminEmail := mkUser("transcript-life-admin")
	if err := rbac.AssignUserRoleByName(ctx, pool, adminID, "Global Admin"); err != nil {
		pool.Close()
		t.Fatalf("admin role: %v", err)
	}

	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config
SET webhook_url = 'https://example.com/hook',
    webhook_secret = 'hold-webhook-secret',
    orders_ui_enabled = true,
    auto_approval_enabled = false,
    registrar_console_enabled = true
WHERE id = 1
`); err != nil {
		pool.Close()
		t.Fatalf("config: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tokStudent, _ := signer.Sign(ctx, studentID.String(), studentEmail, "", "", nil)
	tokAdmin, _ := signer.Sign(ctx, adminID.String(), adminEmail, "", "", nil)

	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFTranscripts: true}})
	return pool, h, tokStudent, tokAdmin, studentID, adminID
}

func createAndSubmitOrder(t *testing.T, h http.Handler, tok string) (string, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/recipients?type=self", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("search: %d %s", w.Code, w.Body.String())
	}
	var search struct {
		Recipients []struct {
			ID string `json:"id"`
		} `json:"recipients"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &search)
	if len(search.Recipients) == 0 {
		t.Fatal("expected self recipient")
	}
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"recipientId": search.Recipients[0].ID, "deliveryMethod": "secure_link_email"},
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+created.Order.ID+"/submit", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}
	var submitted struct {
		Order struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &submitted)
	return submitted.Order.ID, submitted.Order.Status
}

func TestTranscriptLifecycle_HoldBlocksSubmit_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tokStudent, tokAdmin, studentID, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	// AC-1: place financial hold, then submit → on_hold
	body, _ := json.Marshal(map[string]any{
		"userId": studentID.String(),
		"type":   "financial",
		"reason": "owes tuition $500",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/holds", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("place hold: %d %s", w.Code, w.Body.String())
	}

	orderID, status := createAndSubmitOrder(t, h, tokStudent)
	if status != "on_hold" {
		t.Fatalf("AC-1: want on_hold got %s", status)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID, nil)
	req.Header.Set("Authorization", "Bearer "+tokStudent)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var detail struct {
		Order struct {
			Status         string  `json:"status"`
			StudentMessage *string `json:"studentMessage"`
			OnHold         bool    `json:"onHold"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if !detail.Order.OnHold || detail.Order.StudentMessage == nil || *detail.Order.StudentMessage == "" {
		t.Fatalf("student should see hold message: %+v", detail.Order)
	}
	if detail.Order.StudentMessage != nil && *detail.Order.StudentMessage == "owes tuition $500" {
		t.Fatal("student message must not leak internal reason")
	}
}

func TestTranscriptLifecycle_ReleaseResumes_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tokStudent, tokAdmin, studentID, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	body, _ := json.Marshal(map[string]any{
		"userId": studentID.String(),
		"type":   "financial",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/holds", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var placed struct {
		Hold struct {
			ID string `json:"id"`
		} `json:"hold"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &placed)

	orderID, _ := createAndSubmitOrder(t, h, tokStudent)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/holds/"+placed.Hold.ID+"/release", nil)
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("release: %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID, nil)
	req.Header.Set("Authorization", "Bearer "+tokStudent)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var detail struct {
		Order struct {
			Status string `json:"status"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if detail.Order.Status != "in_review" {
		t.Fatalf("AC-2: want in_review got %s", detail.Order.Status)
	}
}

func TestTranscriptLifecycle_RejectAndIllegal_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tokStudent, tokAdmin, _, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	orderID, status := createAndSubmitOrder(t, h, tokStudent)
	if status != "in_review" {
		t.Fatalf("want in_review got %s", status)
	}

	// AC-3 reject
	body, _ := json.Marshal(map[string]any{"action": "reject", "reason": "Missing documentation"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("reject: %d %s", w.Code, w.Body.String())
	}
	var rejected struct {
		Order struct {
			Status          string  `json:"status"`
			RejectionReason *string `json:"rejectionReason"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rejected)
	if rejected.Order.Status != "rejected" {
		t.Fatalf("want rejected got %s", rejected.Order.Status)
	}
	if rejected.Order.RejectionReason == nil || *rejected.Order.RejectionReason != "Missing documentation" {
		t.Fatalf("rejection reason missing: %+v", rejected.Order)
	}

	// AC-4 illegal transition
	body, _ = json.Marshal(map[string]any{"action": "approve"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("illegal: want 400 got %d %s", w.Code, w.Body.String())
	}

	// Student cannot transition
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokStudent)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("student transition: want 403 got %d", w.Code)
	}
}

func TestTranscriptLifecycle_AutoApproval_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tokStudent, _, _, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config SET auto_approval_enabled = true WHERE id = 1
`); err != nil {
		t.Fatalf("auto: %v", err)
	}
	_, status := createAndSubmitOrder(t, h, tokStudent)
	if status != "processing" {
		t.Fatalf("AC-6: want processing got %s", status)
	}
}

func TestTranscriptLifecycle_ExternalHoldIdempotent_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, _, _, studentID, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	payload := map[string]any{
		"userId":     studentID.String(),
		"type":       "financial",
		"externalId": fmt.Sprintf("sis-hold-%d", time.Now().UnixNano()),
		"reason":     "bursar sync",
	}
	raw, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte("hold-webhook-secret"))
	_, _ = mac.Write(raw)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	post := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/transcripts/holds", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Lextures-Signature", sig)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Fatalf("integration hold: %d %s", w.Code, w.Body.String())
		}
		var resp struct {
			Hold struct {
				ID string `json:"id"`
			} `json:"hold"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.Hold.ID
	}
	id1 := post()
	id2 := post()
	if id1 == "" || id1 != id2 {
		t.Fatalf("AC-5: want same hold id, got %q vs %q", id1, id2)
	}
	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM transcripts.holds WHERE external_id = $1
`, payload["externalId"]).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("want 1 hold got %d", count)
	}
}

func TestTranscriptLifecycle_ApproveComplete_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, h, tokStudent, tokAdmin, _, _ := setupTranscriptLifecycleTest(t, ctx)
	defer pool.Close()

	orderID, _ := createAndSubmitOrder(t, h, tokStudent)

	body, _ := json.Marshal(map[string]any{"action": "approve"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", w.Code, w.Body.String())
	}
	var approved struct {
		Order struct {
			Status string `json:"status"`
			Items  []struct {
				Status string `json:"status"`
			} `json:"items"`
			Events []struct {
				ToState string `json:"toState"`
			} `json:"events"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &approved)
	if approved.Order.Status != "processing" {
		t.Fatalf("want processing got %s", approved.Order.Status)
	}
	if len(approved.Order.Items) == 0 || approved.Order.Items[0].Status != "ready" {
		t.Fatalf("items should be ready: %+v", approved.Order.Items)
	}
	if len(approved.Order.Events) == 0 {
		t.Fatal("expected audit events")
	}

	body, _ = json.Marshal(map[string]any{"action": "complete"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", w.Code, w.Body.String())
	}
}
