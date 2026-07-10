package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/hibp"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	pfrepo "github.com/lextures/lextures/server/internal/repos/productfeedback"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestFeedback_SubmitAndAdmin_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	cfg := config.Load()
	cfg.FFFeedback = true
	stub := hibp.StubChecker{Result: hibp.Result{BreachFound: false, HIBPAvailable: true}}
	jwtSecret := "01234567890123456789012345678901"
	signer := auth.NewJWTSignerWithPool(jwtSecret, pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg, PasswordChecker: stub})

	userEmail := "feedback-user-" + time.Now().Format("20060102150405.000") + "@e.invalid"
	password := "J7q#xM2pL9vRkW4$hN8zT1cY5bU6nM0aS"
	signupBody, _ := json.Marshal(map[string]any{
		"email":        userEmail,
		"password":     password,
		"display_name": "Feedback User",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("signup: %d %s", rr.Code, rr.Body.String())
	}
	var signupResp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&signupResp); err != nil {
		t.Fatal(err)
	}
	userToken, _ := signupResp["access_token"].(string)

	submitBody, _ := json.Marshal(map[string]any{
		"message":  "Love the new dashboard!",
		"category": "praise",
		"source":   "web",
		"context":  map[string]string{"route": "/dashboard"},
	})
	submitRR := httptest.NewRecorder()
	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(submitBody))
	submitReq.Header.Set("Authorization", "Bearer "+userToken)
	submitReq = submitReq.WithContext(ctx)
	h.ServeHTTP(submitRR, submitReq)
	if submitRR.Code != http.StatusCreated {
		t.Fatalf("submit: %d %s", submitRR.Code, submitRR.Body.String())
	}
	var submitResp map[string]any
	if err := json.NewDecoder(submitRR.Body).Decode(&submitResp); err != nil {
		t.Fatal(err)
	}
	feedbackID, _ := submitResp["id"].(string)
	if feedbackID == "" {
		t.Fatalf("missing id: %+v", submitResp)
	}

	emptyBody, _ := json.Marshal(map[string]any{"message": "  ", "source": "web"})
	badRR := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(emptyBody))
	badReq.Header.Set("Authorization", "Bearer "+userToken)
	badReq = badReq.WithContext(ctx)
	h.ServeHTTP(badRR, badReq)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("empty message: %d %s", badRR.Code, badRR.Body.String())
	}

	adminEmail := "feedback-admin-" + time.Now().Format("20060102150405.000") + "@e.invalid"
	ph, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	adminRow, err := user.InsertUser(ctx, pool, adminEmail, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	adminID, _ := uuid.Parse(adminRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, adminID, "Global Admin"); err != nil {
		t.Fatal(err)
	}
	adminToken, err := signer.Sign(ctx, adminRow.ID, adminEmail, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	listRR := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback?status=new", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminToken)
	listReq = listReq.WithContext(ctx)
	h.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("admin list: %d %s", listRR.Code, listRR.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatal(err)
	}
	if len(listResp.Items) == 0 {
		t.Fatal("expected at least one feedback item")
	}

	detailRR := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback/"+feedbackID, nil)
	detailReq.Header.Set("Authorization", "Bearer "+adminToken)
	detailReq = detailReq.WithContext(ctx)
	h.ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("admin get: %d %s", detailRR.Code, detailRR.Body.String())
	}

	patchBody, _ := json.Marshal(map[string]any{"status": "resolved", "admin_note": "Thanks!"})
	patchRR := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/feedback/"+feedbackID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Authorization", "Bearer "+adminToken)
	patchReq = patchReq.WithContext(ctx)
	h.ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("admin patch: %d %s", patchRR.Code, patchRR.Body.String())
	}
	var patchResp map[string]any
	if err := json.NewDecoder(patchRR.Body).Decode(&patchResp); err != nil {
		t.Fatal(err)
	}
	if patchResp["status"] != "resolved" {
		t.Fatalf("status: %+v", patchResp)
	}

	authUser, err := signer.Verify(ctx, userToken)
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.MustParse(authUser.UserID)
	if err := pfrepo.DeleteByUser(ctx, pool, uid); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM feedback.submissions WHERE user_id = $1`, uid).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected erasure to delete feedback, got %d rows", count)
	}
}

func TestFeedback_OversizedMessage_Returns400_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	cfg := config.Config{FFFeedback: true}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg, PasswordChecker: hibp.StubChecker{}})

	em := "feedback-big-" + time.Now().Format("20060102150405") + "@e.invalid"
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"message": strings.Repeat("x", 5001),
		"source":  "web",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rr.Code)
	}
}
