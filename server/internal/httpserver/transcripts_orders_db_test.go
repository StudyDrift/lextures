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
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupTranscriptOrdersTest(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string, string, uuid.UUID) {
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

	uidA, emA := mkUser("transcript-orders-a")
	uidB, emB := mkUser("transcript-orders-b")

	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config
SET webhook_url = 'https://example.com/hook', orders_ui_enabled = true
WHERE id = 1
`); err != nil {
		pool.Close()
		t.Fatalf("config: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tokA, _ := signer.Sign(ctx, uidA.String(), emA, "", "", nil)
	tokB, _ := signer.Sign(ctx, uidB.String(), emB, "", "", nil)

	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFTranscripts: true}})
	return pool, h, tokA, tokB, uidA
}

func TestTranscriptOrders_SearchCreateSubmitScope_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tokA, tokB, _ := setupTranscriptOrdersTest(t, ctx)
	defer pool.Close()

	// AC-1: search seeded university
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/recipients?q=State%20University", nil)
	req.Header.Set("Authorization", "Bearer "+tokA)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("search: want 200 got %d body=%s", w.Code, w.Body.String())
	}
	var search struct {
		Recipients []struct {
			ID           string   `json:"id"`
			Name         string   `json:"name"`
			Capabilities []string `json:"capabilities"`
		} `json:"recipients"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(search.Recipients) == 0 {
		t.Fatal("expected seeded State University")
	}
	uniID := search.Recipients[0].ID
	hasPostal := false
	for _, c := range search.Recipients[0].Capabilities {
		if c == "postal_mail" {
			hasPostal = true
		}
	}
	if !hasPostal {
		t.Fatal("expected postal_mail capability on seeded university")
	}

	// Self recipient
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/recipients?type=self", nil)
	req.Header.Set("Authorization", "Bearer "+tokA)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("self search: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &search)
	var selfID string
	for _, r := range search.Recipients {
		if r.Name != "" {
			selfID = r.ID
			break
		}
	}
	if selfID == "" {
		t.Fatal("expected self recipient")
	}

	// AC-2: create multi-item order + submit
	body := map[string]any{
		"items": []map[string]any{
			{"recipientId": uniID, "deliveryMethod": "secure_link_email", "urgency": "standard"},
			{"recipientId": selfID, "deliveryMethod": "secure_link_email", "urgency": "rush"},
		},
	}
	raw, _ := json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create order: want 201 got %d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Order struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"order"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if len(created.Order.Items) != 2 {
		t.Fatalf("want 2 items got %d", len(created.Order.Items))
	}
	orderID := created.Order.ID

	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/submit", nil)
	req.Header.Set("Authorization", "Bearer "+tokA)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit: want 200 got %d body=%s", w.Code, w.Body.String())
	}

	// AC-3: reject unsupported delivery method
	body = map[string]any{
		"items": []map[string]any{
			{"recipientId": selfID, "deliveryMethod": "postal_mail", "urgency": "standard"},
		},
	}
	raw, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid method: want 400 got %d body=%s", w.Code, w.Body.String())
	}

	// AC-5: cross-user access denied
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID, nil)
	req.Header.Set("Authorization", "Bearer "+tokB)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-user: want 404 got %d", w.Code)
	}

	// AC-6: ad-hoc with existing canonical key dedupes
	key := fmt.Sprintf("ceeb:test-%d", time.Now().UnixNano())
	adhoc1 := map[string]any{
		"items": []map[string]any{
			{
				"adHocRecipient": map[string]any{
					"type": "institution", "name": "Dedupe U", "canonicalKey": key,
					"capabilities": []string{"secure_link_email"},
				},
				"deliveryMethod": "secure_link_email",
			},
		},
	}
	raw, _ = json.Marshal(adhoc1)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("adhoc1: %d %s", w.Code, w.Body.String())
	}
	var o1 struct {
		Order struct {
			Items []struct {
				RecipientID string `json:"recipientId"`
			} `json:"items"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &o1)
	raw, _ = json.Marshal(adhoc1)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("adhoc2: %d %s", w.Code, w.Body.String())
	}
	var o2 struct {
		Order struct {
			Items []struct {
				RecipientID string `json:"recipientId"`
			} `json:"items"`
		} `json:"order"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &o2)
	if o1.Order.Items[0].RecipientID == "" || o1.Order.Items[0].RecipientID != o2.Order.Items[0].RecipientID {
		t.Fatalf("expected canonical-key dedupe: %q vs %q", o1.Order.Items[0].RecipientID, o2.Order.Items[0].RecipientID)
	}
}

func TestTranscriptOrders_LegacyBackfill_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, _, _, uid := setupTranscriptOrdersTest(t, ctx)
	defer pool.Close()

	orgID := organization.SeedDefaultOrgID
	var legacyID uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO transcripts.transcript_requests (
  user_id, org_id, status, delivery_type, delivery_email, urgency_days, urgency_unit
)
VALUES ($1, $2, 'submitted', 'email', 'a@example.com', 1, 'days')
RETURNING id
`, uid, orgID).Scan(&legacyID)
	if err != nil {
		t.Fatalf("insert legacy: %v", err)
	}

	// Re-run the backfill statements from migration (idempotent path for new rows).
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.orders (user_id, org_id, status, legacy_request_id, created_at, submitted_at)
SELECT r.user_id, r.org_id,
       CASE r.status WHEN 'submitted' THEN 'completed' WHEN 'failed' THEN 'failed' ELSE 'in_review' END,
       r.id, r.created_at, r.submitted_at
FROM transcripts.transcript_requests r
WHERE r.id = $1
  AND NOT EXISTS (SELECT 1 FROM transcripts.orders o WHERE o.legacy_request_id = r.id)
`, legacyID)
	if err != nil {
		t.Fatalf("backfill order: %v", err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.order_items (order_id, recipient_id, delivery_method, urgency, status, created_at)
SELECT o.id, 'a0000000-0000-4000-8000-000000000001'::uuid, 'secure_link_email', 'standard', 'delivered', r.created_at
FROM transcripts.orders o
JOIN transcripts.transcript_requests r ON r.id = o.legacy_request_id
WHERE o.legacy_request_id = $1
  AND NOT EXISTS (SELECT 1 FROM transcripts.order_items oi WHERE oi.order_id = o.id)
`, legacyID)
	if err != nil {
		t.Fatalf("backfill item: %v", err)
	}

	var orderStatus string
	var itemCount int
	err = pool.QueryRow(ctx, `
SELECT o.status, (SELECT COUNT(*) FROM transcripts.order_items oi WHERE oi.order_id = o.id)
FROM transcripts.orders o
WHERE o.legacy_request_id = $1
`, legacyID).Scan(&orderStatus, &itemCount)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if orderStatus != "completed" || itemCount != 1 {
		t.Fatalf("backfill mismatch status=%s items=%d", orderStatus, itemCount)
	}
}
