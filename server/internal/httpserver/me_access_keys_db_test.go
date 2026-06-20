package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAccessKeys_CreateListRevokeScope_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	jwtSecret := "01234567890123456789012345678901"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	em := "ak-" + time.Now().Format("20060102150405") + "@e.com"
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, uuid.MustParse(row.ID), "Student"); err != nil {
		t.Fatalf("role: %v", err)
	}

	signer := auth.NewJWTSignerWithPool(jwtSecret, pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{FFAPITokens: true, FFPublicAPI: true, JWTSecret: jwtSecret},
	})

	createBody := []byte(`{"label":"Test key","scopes":["courses:read"]}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/access-keys", bytes.NewReader(createBody))
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.ID == "" {
		t.Fatalf("missing token/id: %+v", created)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/access-keys", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
	}
	var listed struct {
		Tokens []struct {
			ID        string `json:"id"`
			TokenMask string `json:"tokenMask"`
		} `json:"tokens"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Tokens) != 1 || listed.Tokens[0].TokenMask == created.Token {
		t.Fatalf("expected masked token in list: %+v", listed.Tokens)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("courses with key: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/me/access-keys/"+created.ID, nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("revoked key: expected 401, got %d", rr.Code)
	}
}
