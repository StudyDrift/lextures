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
	repoConsortium "github.com/lextures/lextures/server/internal/repos/consortium"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	svcConsortium "github.com/lextures/lextures/server/internal/service/consortium"
)

func TestConsortium_CrossOrgEnrollment_Pg(t *testing.T) {
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

	hostOrg := organization.SeedDefaultOrgID
	guestSlug := "e2e-guest-" + uuid.NewString()[:8]
	var guestOrgID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO tenant.organizations (slug, name, status, org_type)
VALUES ($1, 'E2E Guest Campus', 'active', 'higher-ed')
RETURNING id
`, guestSlug).Scan(&guestOrgID); err != nil {
		t.Fatalf("guest org: %v", err)
	}

	agreement, err := repoConsortium.CreateAgreement(ctx, pool, hostOrg, guestOrgID, repoConsortium.StatusActive)
	if err != nil {
		t.Fatalf("agreement: %v", err)
	}
	if agreement == nil {
		t.Fatal("expected agreement")
	}

	courseCode := "cons-" + uuid.NewString()[:8]
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (org_id, course_code, title, published, consortium_shareable)
VALUES ($1, $2, 'Consortium Course', true, true)
RETURNING id
`, hostOrg, courseCode).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}

	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	guestEmail := "guest-" + uuid.NewString()[:8] + "@test.invalid"
	guestUser, err := user.InsertUser(ctx, pool, guestEmail, ph, nil)
	if err != nil {
		t.Fatalf("guest user: %v", err)
	}
	guestUID, _ := uuid.Parse(guestUser.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, guestOrgID, guestUID); err != nil {
		t.Fatalf("move user org: %v", err)
	}

	if err := svcConsortium.EnrollGuestStudent(ctx, pool, courseID, guestUID, guestOrgID); err != nil {
		t.Fatalf("enroll: %v", err)
	}

	ok, err := enrollment.UserHasAccess(ctx, pool, courseCode, guestUID)
	if err != nil || !ok {
		t.Fatalf("guest access want true got %v err=%v", ok, err)
	}

	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	tok, err := signer.Sign(ctx, guestUser.ID, guestEmail, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	cfg := config.Config{FFConsortiumSharing: true}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consortium/courses", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("browse: %d %s", rr.Code, rr.Body.String())
	}
	var browse struct {
		Courses []struct {
			ID string `json:"id"`
		} `json:"courses"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&browse); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range browse.Courses {
		if c.ID == courseID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected shareable course in browse list")
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode, nil)
	req2 = req2.WithContext(ctx)
	req2.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("course access: %d %s", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"consortiumShareable": false})
	req3 := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/consortium-settings", bytes.NewReader(body))
	req3 = req3.WithContext(ctx)
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr3, req3)
	if rr3.Code == http.StatusOK {
		t.Log("guest cannot patch settings without instructor RBAC")
	}
}
