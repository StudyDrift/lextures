package httpserver

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestParentLinks_IsolationAndBulk_Pg(t *testing.T) {
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

	defOrg := organization.SeedDefaultOrgID
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ts := time.Now().Format("20060102150405")

	emGA := "pl-ga-" + ts + "@e.com"
	gaRow, err := user.InsertUser(ctx, pool, emGA, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	gaID := uuid.MustParse(gaRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, gaID, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}
	slugGA, err := organization.OrgSlugForUser(ctx, pool, gaID)
	if err != nil {
		t.Fatal(err)
	}

	emAdmin := "pl-adm-" + ts + "@e.com"
	adminRow, err := user.InsertUser(ctx, pool, emAdmin, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	adminID := uuid.MustParse(adminRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, adminID, "Student"); err != nil {
		t.Fatalf("student: %v", err)
	}
	slugAdm, err := organization.OrgSlugForUser(ctx, pool, adminID)
	if err != nil {
		t.Fatal(err)
	}

	emParent := "pl-par-" + ts + "@e.com"
	dn := "Pat Parent"
	parentRow, err := user.InsertUser(ctx, pool, emParent, ph, &dn)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET account_type = 'parent' WHERE id = $1`, parentID); err != nil {
		t.Fatal(err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, parentID, "Student"); err != nil {
		t.Fatalf("student role: %v", err)
	}
	slugPar, err := organization.OrgSlugForUser(ctx, pool, parentID)
	if err != nil {
		t.Fatal(err)
	}

	emStu := "pl-stu-" + ts + "@e.com"
	sdn := "Sam Student"
	stuRow, err := user.InsertUser(ctx, pool, emStu, ph, &sdn)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	stuID := uuid.MustParse(stuRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, stuID, "Student"); err != nil {
		t.Fatalf("student: %v", err)
	}

	emOther := "pl-oth-" + ts + "@e.com"
	othRow, err := user.InsertUser(ctx, pool, emOther, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	othID := uuid.MustParse(othRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, othID, "Student"); err != nil {
		t.Fatalf("student: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	gaTok, err := signer.Sign(ctx, gaRow.ID, emGA, defOrg.String(), slugGA, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	grantBody := []byte(`{"userId":"` + adminRow.ID + `","role":"org_admin"}`)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+defOrg.String()+"/role-grants", bytes.NewReader(grantBody))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+gaTok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("grant org_admin: %d %s", rr.Code, rr.Body.String())
	}

	adminTok, err := signer.Sign(ctx, adminRow.ID, emAdmin, defOrg.String(), slugAdm, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	linkBody := []byte(`{"parentUserId":"` + parentID.String() + `","studentUserId":"` + stuID.String() + `","relationship":"guardian"}`)
	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+defOrg.String()+"/parent-links", bytes.NewReader(linkBody))
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+adminTok)
	r2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("parent link: %d %s", rr2.Code, rr2.Body.String())
	}

	parentTok, err := signer.Sign(ctx, parentRow.ID, emParent, defOrg.String(), slugPar, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	rr3 := httptest.NewRecorder()
	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/parent/children", nil)
	r3 = r3.WithContext(ctx)
	r3.Header.Set("Authorization", "Bearer "+parentTok)
	h.ServeHTTP(rr3, r3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("children: %d %s", rr3.Code, rr3.Body.String())
	}
	var ch struct {
		Children []struct {
			StudentUserID string `json:"studentUserId"`
		} `json:"children"`
	}
	if err := json.NewDecoder(rr3.Body).Decode(&ch); err != nil {
		t.Fatal(err)
	}
	if len(ch.Children) != 1 || ch.Children[0].StudentUserID != stuID.String() {
		t.Fatalf("unexpected children: %#v", ch)
	}

	rr4 := httptest.NewRecorder()
	r4 := httptest.NewRequest(http.MethodGet, "/api/v1/parent/students/"+othID.String()+"/grades", nil)
	r4 = r4.WithContext(ctx)
	r4.Header.Set("Authorization", "Bearer "+parentTok)
	h.ServeHTTP(rr4, r4)
	if rr4.Code != http.StatusForbidden {
		t.Fatalf("cross-child want 403 got %d %s", rr4.Code, rr4.Body.String())
	}

	var csvBuf bytes.Buffer
	w := csv.NewWriter(&csvBuf)
	_ = w.Write([]string{"parent_email", "student_email"})
	for i := 0; i < 3; i++ {
		pe := fmt.Sprintf("pl-bp-%s-%d@e.com", ts, i)
		se := fmt.Sprintf("pl-bs-%s-%d@e.com", ts, i)
		pr, err := user.InsertUser(ctx, pool, pe, ph, nil)
		if err != nil {
			t.Fatal(err)
		}
		sr, err := user.InsertUser(ctx, pool, se, ph, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := rbac.AssignUserRoleByName(ctx, pool, uuid.MustParse(pr.ID), "Student"); err != nil {
			t.Fatal(err)
		}
		if err := rbac.AssignUserRoleByName(ctx, pool, uuid.MustParse(sr.ID), "Student"); err != nil {
			t.Fatal(err)
		}
		_ = w.Write([]string{pe, se})
	}
	w.Flush()

	rr5 := httptest.NewRecorder()
	r5 := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+defOrg.String()+"/parent-links/bulk", bytes.NewReader(csvBuf.Bytes()))
	r5 = r5.WithContext(ctx)
	r5.Header.Set("Authorization", "Bearer "+adminTok)
	r5.Header.Set("Content-Type", "text/csv")
	h.ServeHTTP(rr5, r5)
	if rr5.Code != http.StatusOK {
		t.Fatalf("bulk: %d %s", rr5.Code, rr5.Body.String())
	}
	var bulkOut struct {
		Created int `json:"created"`
	}
	if err := json.NewDecoder(rr5.Body).Decode(&bulkOut); err != nil {
		t.Fatal(err)
	}
	if bulkOut.Created != 3 {
		t.Fatalf("created want 3 got %d", bulkOut.Created)
	}
}

func TestParentNotificationPrefs_Pg(t *testing.T) {
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

	defOrg := organization.SeedDefaultOrgID
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ts := time.Now().Format("20060102150405.999")
	emParent := "notifp-par-" + ts + "@e.com"
	dn := "Notif Parent"
	parentRow, err := user.InsertUser(ctx, pool, emParent, ph, &dn)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET account_type = 'parent' WHERE id = $1`, parentID); err != nil {
		t.Fatal(err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, parentID, "Student"); err != nil {
		t.Fatal(err)
	}
	slugPar, err := organization.OrgSlugForUser(ctx, pool, parentID)
	if err != nil {
		t.Fatal(err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	parentTok, err := signer.Sign(ctx, parentRow.ID, emParent, defOrg.String(), slugPar, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	// GET defaults (no row yet).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/notification-prefs", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+parentTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET prefs: %d %s", rr.Code, rr.Body.String())
	}
	var prefs struct {
		GradePosted       bool `json:"gradePosted"`
		MissingAssignment bool `json:"missingAssignment"`
		AttendanceEvent   bool `json:"attendanceEvent"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&prefs); err != nil {
		t.Fatal(err)
	}
	if !prefs.GradePosted || !prefs.MissingAssignment {
		t.Fatalf("unexpected defaults: %+v", prefs)
	}

	// PATCH to disable grade_posted.
	patchBody := []byte(`{"gradePosted":false}`)
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/parent/notification-prefs", bytes.NewReader(patchBody))
	req2 = req2.WithContext(ctx)
	req2.Header.Set("Authorization", "Bearer "+parentTok)
	req2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("PATCH prefs: %d %s", rr2.Code, rr2.Body.String())
	}
	var prefs2 struct {
		GradePosted bool `json:"gradePosted"`
	}
	if err := json.NewDecoder(rr2.Body).Decode(&prefs2); err != nil {
		t.Fatal(err)
	}
	if prefs2.GradePosted {
		t.Fatalf("expected gradePosted=false after PATCH, got true")
	}

	// GET reflects updated value.
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/parent/notification-prefs", nil)
	req3 = req3.WithContext(ctx)
	req3.Header.Set("Authorization", "Bearer "+parentTok)
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("GET prefs after PATCH: %d %s", rr3.Code, rr3.Body.String())
	}
	var prefs3 struct {
		GradePosted bool `json:"gradePosted"`
	}
	if err := json.NewDecoder(rr3.Body).Decode(&prefs3); err != nil {
		t.Fatal(err)
	}
	if prefs3.GradePosted {
		t.Fatalf("expected gradePosted=false persisted, got true")
	}

	// Non-parent gets 403.
	emOther := "notifp-oth-" + ts + "@e.com"
	othRow, err := user.InsertUser(ctx, pool, emOther, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, uuid.MustParse(othRow.ID), "Student"); err != nil {
		t.Fatal(err)
	}
	slugOth, err := organization.OrgSlugForUser(ctx, pool, uuid.MustParse(othRow.ID))
	if err != nil {
		t.Fatal(err)
	}
	othTok, err := signer.Sign(ctx, othRow.ID, emOther, defOrg.String(), slugOth, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/parent/notification-prefs", nil)
	req4 = req4.WithContext(ctx)
	req4.Header.Set("Authorization", "Bearer "+othTok)
	h.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusForbidden {
		t.Fatalf("non-parent want 403 got %d", rr4.Code)
	}
}

func TestParentWeeklySummary_Pg(t *testing.T) {
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

	defOrg := organization.SeedDefaultOrgID
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ts := time.Now().Format("20060102150405.999")
	emParent := "weekly-par-" + ts + "@e.com"
	parentRow, err := user.InsertUser(ctx, pool, emParent, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	parentID := uuid.MustParse(parentRow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET account_type = 'parent' WHERE id = $1`, parentID); err != nil {
		t.Fatal(err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, parentID, "Student"); err != nil {
		t.Fatal(err)
	}
	slugPar, err := organization.OrgSlugForUser(ctx, pool, parentID)
	if err != nil {
		t.Fatal(err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	parentTok, err := signer.Sign(ctx, parentRow.ID, emParent, defOrg.String(), slugPar, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	// Weekly summary for a parent with no linked children should return empty items.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/weekly-summary", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+parentTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("weekly-summary: %d %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Items     []any  `json:"items"`
		WeekStart string `json:"weekStart"`
		WeekEnd   string `json:"weekEnd"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 0 {
		t.Fatalf("expected 0 items for unlinked parent, got %d", len(out.Items))
	}
	if out.WeekStart == "" || out.WeekEnd == "" {
		t.Fatalf("expected weekStart/weekEnd in response")
	}
}
