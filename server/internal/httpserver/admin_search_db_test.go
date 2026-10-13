package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	adminsearchrepo "github.com/lextures/lextures/server/internal/repos/adminsearch"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAdminSearch_OrgScoped_Pg(t *testing.T) {
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
	unique := "as" + ts
	emOrg := "as-org-" + ts + "@e.com"
	aliceName := "Alice Johnson" + unique
	orgRow, err := user.InsertUser(ctx, pool, emOrg, ph, &aliceName)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	orgUID := uuid.MustParse(orgRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, orgUID, "Student"); err != nil {
		t.Fatalf("role: %v", err)
	}
	slugOrg, err := organization.OrgSlugForUser(ctx, pool, orgUID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orgroles.Create(ctx, pool, defOrg, orgUID, nil, orgroles.RoleOrgAdmin, &orgUID, nil); err != nil {
		t.Fatalf("grant org admin: %v", err)
	}
	_, err = pool.Exec(ctx, `
UPDATE "user".users SET first_name = 'Alice', last_name = $2 WHERE id = $1
`, orgUID, "Johnson"+unique)
	if err != nil {
		t.Fatalf("update alice: %v", err)
	}

	otherOrg, err := organization.Create(ctx, pool, "Other Org "+ts, "other-"+ts, nil, nil, "us-east", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	emOther := "as-other-" + ts + "@e.com"
	bobName := "Bob Johnson" + unique
	otherRow, err := user.InsertUser(ctx, pool, emOther, ph, &bobName)
	if err != nil {
		t.Fatalf("other user: %v", err)
	}
	otherUID := uuid.MustParse(otherRow.ID)
	_, err = pool.Exec(ctx, `
UPDATE "user".users SET org_id = $1, first_name = 'Bob', last_name = $3 WHERE id = $2
`, otherOrg.ID, otherUID, "Johnson"+unique)
	if err != nil {
		t.Fatalf("move other user: %v", err)
	}

	courseTitle := "Biology " + unique
	_, err = course.CreateCourse(ctx, pool, orgUID, courseTitle, "Intro biology course", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("course: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	orgTok, err := signer.Sign(ctx, orgRow.ID, emOrg, defOrg.String(), slugOrg, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminConsoleEnabled: true, AdminSearchEnabled: true},
	})

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q="+unique+"&types=users", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+orgTok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, emOrg) {
		t.Fatalf("expected org user in results: %s", body)
	}
	if strings.Contains(body, emOther) {
		t.Fatalf("cross-org user leaked: %s", body)
	}

	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=biologe&types=courses", nil)
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+orgTok)
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("fuzzy search status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), courseTitle) && !strings.Contains(rr2.Body.String(), "Biology") {
		t.Fatalf("expected Biology course via fuzzy match: %s", rr2.Body.String())
	}
}

// TestAdminSearch_WholeWordMatching_Pg covers #791: "QA" must not match inside or at the
// start of another word ("Qamets"), and whole-word course hits must make the 5-item preview.
func TestAdminSearch_WholeWordMatching_Pg(t *testing.T) {
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

	ts := time.Now().Format("20060102150405.000000")
	slug := "ww-" + strings.ReplaceAll(ts, ".", "")
	org, err := organization.Create(ctx, pool, "Word Match Org "+ts, slug, nil, nil, "us-east", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	addUser := func(local, display string) uuid.UUID {
		t.Helper()
		row, err := user.InsertUser(ctx, pool, local+"-"+slug+"@e.com", ph, &display)
		if err != nil {
			t.Fatalf("user %s: %v", display, err)
		}
		id := uuid.MustParse(row.ID)
		if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, org.ID, id); err != nil {
			t.Fatalf("move user: %v", err)
		}
		return id
	}
	creator := addUser("creator", "Course Creator")
	addUser("telling", "Telling Qamets")
	addUser("lead", "QA Lead")

	addCourse := func(title, description string) string {
		t.Helper()
		c, err := course.CreateCourse(ctx, pool, creator, title, description, "", nil, nil, nil)
		if err != nil {
			t.Fatalf("course %s: %v", title, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE course.courses SET org_id = $1 WHERE id = $2`, org.ID, uuid.MustParse(c.ID)); err != nil {
			t.Fatalf("move course: %v", err)
		}
		return c.CourseCode
	}
	// Five courses that only match "qa" as a word prefix sort ahead alphabetically...
	for _, title := range []string{"Aqaba History", "Qadi Law", "Qanat Systems", "Qatar Studies", "Qawwali Music"} {
		addCourse(title, "")
	}
	// ...but the whole-word match must still make the 5-item preview.
	qaCode := addCourse("Software QA", "")

	users, _, err := adminsearchrepo.SearchUsers(ctx, pool, org.ID, "QA", 5, 0)
	if err != nil {
		t.Fatalf("search users: %v", err)
	}
	var names []string
	for _, u := range users {
		names = append(names, u.Title)
	}
	if len(names) != 1 || names[0] != "QA Lead" {
		t.Fatalf("users for QA = %v, want only [QA Lead]", names)
	}

	courses, _, err := adminsearchrepo.SearchCourses(ctx, pool, org.ID, "QA", 5, 0)
	if err != nil {
		t.Fatalf("search courses: %v", err)
	}
	if len(courses) != 1 || courses[0].Subtitle != qaCode {
		var got []string
		for _, c := range courses {
			got = append(got, c.Title+" ("+c.Subtitle+")")
		}
		t.Fatalf("courses for QA = %v, want only Software QA (%s)", got, qaCode)
	}

	// Longer queries still match word prefixes, ranked below whole words.
	prefix, _, err := adminsearchrepo.SearchCourses(ctx, pool, org.ID, "Qat", 5, 0)
	if err != nil {
		t.Fatalf("search courses prefix: %v", err)
	}
	if len(prefix) == 0 || prefix[0].Title != "Qatar Studies" {
		t.Fatalf("expected Qatar Studies first for prefix query, got %+v", prefix)
	}
	// Course codes are found by their code.
	byCode, _, err := adminsearchrepo.SearchCourses(ctx, pool, org.ID, qaCode, 5, 0)
	if err != nil {
		t.Fatalf("search by code: %v", err)
	}
	if len(byCode) == 0 || byCode[0].Subtitle != qaCode {
		t.Fatalf("expected %s first when searching its code, got %+v", qaCode, byCode)
	}
}
