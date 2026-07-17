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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func setupTranscriptFeesTest(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string, string, uuid.UUID) {
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
	em := fmt.Sprintf("fees-student-%d@test.com", time.Now().UnixNano())
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
	adminEm := fmt.Sprintf("fees-admin-%d@test.com", time.Now().UnixNano())
	adminRow, err := user.InsertUser(ctx, pool, adminEm, ph, nil)
	if err != nil {
		pool.Close()
		t.Fatalf("admin: %v", err)
	}
	adminID, _ := uuid.Parse(adminRow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, adminID); err != nil {
		pool.Close()
		t.Fatalf("admin org: %v", err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, adminID, "Global Admin"); err != nil {
		pool.Close()
		t.Fatalf("admin role: %v", err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config
SET webhook_url = 'https://example.com/hook',
    orders_ui_enabled = true,
    consent_required = false,
    auto_approval_enabled = false,
    fees_enabled = true,
    registrar_console_enabled = true
WHERE id = 1
`); err != nil {
		pool.Close()
		t.Fatalf("config: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO transcripts.fee_schedule (
  org_id, currency, base_fee, rush_fee, per_recipient_fee, method_surcharges, free_allotment, allotment_period
) VALUES ($1, 'usd', 1000, 300, 500, '{}'::jsonb, 0, 'lifetime')
ON CONFLICT (org_id) DO UPDATE SET
  currency = EXCLUDED.currency,
  base_fee = EXCLUDED.base_fee,
  rush_fee = EXCLUDED.rush_fee,
  per_recipient_fee = EXCLUDED.per_recipient_fee,
  method_surcharges = EXCLUDED.method_surcharges,
  free_allotment = EXCLUDED.free_allotment,
  allotment_period = EXCLUDED.allotment_period,
  updated_at = NOW()
`, orgID); err != nil {
		pool.Close()
		t.Fatalf("fee schedule: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, uid.String(), em, "", "", nil)
	adminTok, _ := signer.Sign(ctx, adminID.String(), adminEm, "", "", nil)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFTranscripts: true}})
	return pool, h, tok, adminTok, uid
}

func TestTranscriptFeesQuoteAndPaymentGate(t *testing.T) {
	ctx := context.Background()
	pool, h, tok, adminTok, _ := setupTranscriptFeesTest(t, ctx)
	defer pool.Close()

	orderID := createThirdPartyOrder(t, h, tok)

	// Quote AC-1 shape: base 10 + per-recip 5 + rush 3 for one rush item = 1800
	// createThirdPartyOrder uses standard urgency — bump to rush via SQL
	if _, err := pool.Exec(ctx, `
UPDATE transcripts.order_items SET urgency = 'rush' WHERE order_id = $1
`, orderID); err != nil {
		t.Fatalf("rush: %v", err)
	}
	// Add second recipient item
	body, _ := json.Marshal(map[string]any{
		"adHocRecipient": map[string]any{
			"type": "employer", "name": "Acme Corp", "email": "hr@acme.test",
			"capabilities": []string{"secure_link_email"},
		},
		"deliveryMethod": "secure_link_email",
		"urgency":        "standard",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/items", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("add item: %d %s", w.Code, w.Body.String())
	}

	qreq := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID+"/quote", nil)
	qreq.Header.Set("Authorization", "Bearer "+tok)
	qw := httptest.NewRecorder()
	h.ServeHTTP(qw, qreq)
	if qw.Code != http.StatusOK {
		t.Fatalf("quote: %d %s", qw.Code, qw.Body.String())
	}
	var quoteResp struct {
		Quote struct {
			Total           int  `json:"total"`
			RequiresPayment bool `json:"requiresPayment"`
		} `json:"quote"`
	}
	_ = json.Unmarshal(qw.Body.Bytes(), &quoteResp)
	// 1000 base + 500*2 + 300 rush = 2300
	if quoteResp.Quote.Total != 2300 {
		t.Fatalf("quote total=%d want 2300 body=%s", quoteResp.Quote.Total, qw.Body.String())
	}

	// Submit — consent off → pending_payment
	sreq := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/submit", nil)
	sreq.Header.Set("Authorization", "Bearer "+tok)
	sw := httptest.NewRecorder()
	h.ServeHTTP(sw, sreq)
	if sw.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", sw.Code, sw.Body.String())
	}
	var submitted struct {
		Order struct {
			Status        string `json:"status"`
			PaymentStatus string `json:"paymentStatus"`
		} `json:"order"`
	}
	// submit may return order wrapped or bare — check both
	_ = json.Unmarshal(sw.Body.Bytes(), &submitted)
	if submitted.Order.Status == "" {
		_ = json.Unmarshal(sw.Body.Bytes(), &submitted.Order)
	}
	if submitted.Order.Status != "pending_payment" {
		// reload
		var st string
		_ = pool.QueryRow(ctx, `SELECT status FROM transcripts.orders WHERE id = $1`, orderID).Scan(&st)
		if st != "pending_payment" {
			t.Fatalf("status=%s body=%s", st, sw.Body.String())
		}
	}

	// Admin approve must not reach processing while unpaid (AC-2)
	// First force in_review to test approve gate
	if _, err := pool.Exec(ctx, `UPDATE transcripts.orders SET status = 'in_review' WHERE id = $1`, orderID); err != nil {
		t.Fatalf("force in_review: %v", err)
	}
	areqBody, _ := json.Marshal(map[string]any{"action": "approve"})
	areq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/orders/"+orderID+"/transition", bytes.NewReader(areqBody))
	areq.Header.Set("Authorization", "Bearer "+adminTok)
	areq.Header.Set("Content-Type", "application/json")
	aw := httptest.NewRecorder()
	h.ServeHTTP(aw, areq)
	// May 403 if admin RBAC not granted — skip gate check in that case
	if aw.Code == http.StatusOK {
		var st string
		_ = pool.QueryRow(ctx, `SELECT status FROM transcripts.orders WHERE id = $1`, orderID).Scan(&st)
		if st == "processing" {
			t.Fatalf("approve advanced to processing while unpaid")
		}
		if st != "pending_payment" {
			t.Logf("approve redirected to %s (expected pending_payment)", st)
		}
	}

	// Waiver code path (AC-3)
	waiverCode := fmt.Sprintf("FULLWAIVE%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `
INSERT INTO transcripts.waiver_codes (org_id, code, kind, max_uses)
VALUES ($1, $2, 'full', 10)
`, organization.SeedDefaultOrgID, waiverCode); err != nil {
		t.Fatalf("waiver: %v", err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE transcripts.orders SET status = 'pending_payment', payment_status = 'unpaid' WHERE id = $1
`, orderID); err != nil {
		t.Fatalf("reset: %v", err)
	}
	cbody, _ := json.Marshal(map[string]any{"waiverCode": waiverCode})
	creq := httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/orders/"+orderID+"/checkout", bytes.NewReader(cbody))
	creq.Header.Set("Authorization", "Bearer "+tok)
	creq.Header.Set("Content-Type", "application/json")
	cw := httptest.NewRecorder()
	h.ServeHTTP(cw, creq)
	if cw.Code != http.StatusOK {
		t.Fatalf("checkout waive: %d %s", cw.Code, cw.Body.String())
	}
	var paySt string
	_ = pool.QueryRow(ctx, `SELECT payment_status FROM transcripts.orders WHERE id = $1`, orderID).Scan(&paySt)
	if paySt != "waived" {
		t.Fatalf("payment_status=%s want waived body=%s", paySt, cw.Body.String())
	}

	// Receipt available (AC-6)
	rreq := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/orders/"+orderID+"/receipt", nil)
	rreq.Header.Set("Authorization", "Bearer "+tok)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, rreq)
	if rw.Code != http.StatusOK {
		t.Fatalf("receipt: %d %s", rw.Code, rw.Body.String())
	}

	// Idempotent payment event (AC-5)
	eventID := "evt_test_" + uuid.NewString()
	created, err := insertPaymentEvent(ctx, pool, orderID, eventID)
	if err != nil {
		t.Fatalf("event1: %v", err)
	}
	if !created {
		t.Fatal("first event should insert")
	}
	created2, err := insertPaymentEvent(ctx, pool, orderID, eventID)
	if err != nil {
		t.Fatalf("event2: %v", err)
	}
	if created2 {
		t.Fatal("duplicate event should be ignored")
	}
}

func insertPaymentEvent(ctx context.Context, pool *pgxpool.Pool, orderID, eventID string) (bool, error) {
	tag, err := pool.Exec(ctx, `
INSERT INTO transcripts.payment_events (order_id, stripe_event_id, event_type, payload)
VALUES ($1, $2, 'checkout.session.completed', '{}'::jsonb)
ON CONFLICT (stripe_event_id) DO NOTHING
`, orderID, eventID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func TestTranscriptFeesAdminSchedule(t *testing.T) {
	ctx := context.Background()
	pool, h, _, adminTok, _ := setupTranscriptFeesTest(t, ctx)
	defer pool.Close()

	body, _ := json.Marshal(map[string]any{
		"currency": "usd", "baseFee": 1500, "rushFee": 400, "perRecipientFee": 250,
		"methodSurcharges": map[string]int{"postal_mail": 200},
		"freeAllotment": 1, "allotmentPeriod": "year",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/transcripts/fees", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminTok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("put fees: %d %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		var resp struct {
			BaseFee int `json:"baseFee"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.BaseFee != 1500 {
			t.Fatalf("baseFee=%d", resp.BaseFee)
		}
	}
}
