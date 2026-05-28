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

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/hibp"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

func TestPutSettingsTimezone_InvalidIANA_Returns422_Pg(t *testing.T) {
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

	jwt := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	stub := hibp.StubChecker{Result: hibp.Result{BreachFound: false, HIBPAvailable: true}}
	email := "tz-invalid-" + time.Now().Format("20060102150405") + "@test.invalid"
	res, err := authservice.Signup(ctx, pool, jwt, config.Config{}, stub, authservice.SignupRequest{
		Email:    email,
		Password: "J7q#xM2pL9vRkW4$hN8zT1cY5bU6nM0aS",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	d := Deps{Pool: pool, JWTSigner: jwt, Config: config.Config{}, PasswordChecker: stub}
	h := NewHandler(d)
	body, _ := json.Marshal(map[string]string{"timezone": "Not/A_Real_Zone"})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings/timezone", bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+res.AccessToken)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d %s", rr.Code, rr.Body.String())
	}
}
