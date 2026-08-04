package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/coursechecklist"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCourseChecklist_AuthzDismissRestoreSummary_Pg(t *testing.T) {
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

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	ts := time.Now().Format("20060102150405")
	instEmail := "cc2-inst-" + ts + "@e.com"
	stuEmail := "cc2-stu-" + ts + "@e.com"
	taEmail := "cc2-ta-" + ts + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	inst, _ := user.InsertUser(ctx, pool, instEmail, ph, strPtr("Instructor"))
	stu, _ := user.InsertUser(ctx, pool, stuEmail, ph, strPtr("Student"))
	ta, _ := user.InsertUser(ctx, pool, taEmail, ph, strPtr("TA"))
	instID, _ := uuid.Parse(inst.ID)
	stuID, _ := uuid.Parse(stu.ID)
	taID, _ := uuid.Parse(ta.ID)

	courseCode := "C-" + fmt.Sprintf("%06X", time.Now().UnixNano()%0xFFFFFF)
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Checklist API Test', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id
`, instID, courseCode).Scan(&courseID)
	if err != nil {
		t.Fatalf("course: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	for _, pair := range []struct {
		uid  uuid.UUID
		role string
	}{
		{instID, "teacher"},
		{stuID, "student"},
		{taID, "ta"},
	} {
		_, err = tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`, courseID, pair.uid, pair.role)
		if err != nil {
			t.Fatalf("enroll %s: %v", pair.role, err)
		}
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, pair.uid, courseID, courseCode); err != nil {
			t.Fatalf("grants: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	instTok, _ := signer.Sign(ctx, inst.ID, instEmail, "", "", nil)
	stuTok, _ := signer.Sign(ctx, stu.ID, stuEmail, "", "", nil)
	taTok, _ := signer.Sign(ctx, ta.ID, taEmail, "", "", nil)

	d := Deps{Pool: pool, JWTSigner: signer, Config: config.Config{ChecklistSnapshotTTL: 15 * time.Minute}}
	h := NewHandler(d)

	// Student → 403, no item titles leaked.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+stuTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("student GET: %d %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("Set course")) {
		t.Fatalf("student body leaked titles: %s", w.Body.String())
	}

	// TA → 403
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+taTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("ta GET: %d %s", w.Code, w.Body.String())
	}

	// Unknown course / unenrolled shape: same 404 body.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-DOESNOTEXIST/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing course: %d", w.Code)
	}
	missingBody := w.Body.String()

	// Teacher GET (cold) then warm hit.
	beforeHit := testutil.ToFloat64(coursechecklist.SnapshotHitsCounter().WithLabelValues("hit"))
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("teacher GET cold: %d %s", w.Code, w.Body.String())
	}
	var checklist coursechecklist.ChecklistResponse
	if err := json.NewDecoder(w.Body).Decode(&checklist); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if checklist.Summary.OutstandingEssential < 1 {
		t.Fatalf("expected outstanding essentials, got %+v", checklist.Summary)
	}
	outstandingBefore := checklist.Summary.OutstandingEssential

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("teacher GET warm: %d %s", w.Code, w.Body.String())
	}
	afterHit := testutil.ToFloat64(coursechecklist.SnapshotHitsCounter().WithLabelValues("hit"))
	if afterHit < beforeHit+1 {
		t.Fatalf("expected snapshot hit increment, before=%v after=%v", beforeHit, afterHit)
	}

	// Dismiss course.dates (essential todo in the CC.1 reference catalog).
	body := []byte(`{"reason":"not_applicable","note":"pass/fail seminar"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/checklist/items/course.dates/dismiss", bytes.NewReader(body))
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dismiss: %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if err := json.NewDecoder(w.Body).Decode(&checklist); err != nil {
		t.Fatalf("decode after dismiss: %v", err)
	}
	if len(checklist.Dismissed) != 1 || checklist.Dismissed[0].ID != "course.dates" {
		t.Fatalf("dismissed pile: %+v", checklist.Dismissed)
	}
	for _, cat := range checklist.Categories {
		for _, it := range cat.Items {
			if it.ID == "course.dates" {
				t.Fatal("dismissed item still inline")
			}
		}
	}
	if checklist.Summary.Dismissed != 1 || checklist.Summary.OutstandingEssential != outstandingBefore-1 {
		t.Fatalf("summary after dismiss: %+v (before essential %d)", checklist.Summary, outstandingBefore)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist/summary", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("summary: %d %s", w.Code, w.Body.String())
	}
	var summary coursechecklist.ChecklistSummary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("summary decode: %v", err)
	}
	if summary.Dismissed != 1 {
		t.Fatalf("summary dismissed=%d", summary.Dismissed)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist/history", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("history: %d %s", w.Code, w.Body.String())
	}
	var hist coursechecklist.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&hist); err != nil {
		t.Fatalf("history decode: %v", err)
	}
	if len(hist.Events) < 1 || hist.Events[0].Action != "dismiss" {
		t.Fatalf("history events: %+v", hist.Events)
	}

	// Restore
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/checklist/items/course.dates/restore", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("restore: %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if err := json.NewDecoder(w.Body).Decode(&checklist); err != nil {
		t.Fatalf("decode after restore: %v", err)
	}
	if checklist.Summary.Dismissed != 0 {
		t.Fatalf("expected 0 dismissed, got %+v", checklist.Summary)
	}
	found := false
	for _, cat := range checklist.Categories {
		for _, it := range cat.Items {
			if it.ID == "course.dates" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("restored item missing from categories")
	}

	// Unknown item_id → 404 not_found, no row
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/checklist/items/not.a.real.item/dismiss", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+instTok)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown item: %d %s", w.Code, w.Body.String())
	}
	var errBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &errBody)
	if errBody.Error.Code != "NOT_FOUND" {
		t.Fatalf("code=%q body=%s", errBody.Error.Code, w.Body.String())
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_checklist_item_state WHERE course_id = $1 AND item_id = 'not.a.real.item'`, courseID).Scan(&n)
	if n != 0 {
		t.Fatalf("unexpected state row for unknown item")
	}

	// Path injection / SQL metacharacters as item_id → 404 (validated via ResolveItemID).
	for _, bad := range []string{"../../etc/passwd", "course.sections';DROP TABLE"} {
		path := "/api/v1/courses/" + courseCode + "/checklist/items/" + url.PathEscape(bad) + "/dismiss"
		req = httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{}`)))
		req = req.WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+instTok)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Fatalf("bad item %q: %d %s", bad, w.Code, w.Body.String())
		}
	}

	// Unrelated user same 404 body as missing course.
	strangerEmail := "cc2-str-" + ts + "@e.com"
	stranger, _ := user.InsertUser(ctx, pool, strangerEmail, ph, strPtr("Stranger"))
	strTok, _ := signer.Sign(ctx, stranger.ID, strangerEmail, "", "", nil)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+strTok)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("stranger: %d", w.Code)
	}
	if w.Body.String() != missingBody {
		t.Fatalf("404 bodies differ:\nmissing=%s\nstranger=%s", missingBody, w.Body.String())
	}
}

func TestCourseChecklist_SingleFlight_Pg(t *testing.T) {
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

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	ts := time.Now().Format("20060102150405.000")
	email := "cc2-sf-" + ts + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	u, _ := user.InsertUser(ctx, pool, email, ph, strPtr("Teacher"))
	uid, _ := uuid.Parse(u.ID)
	courseCode := "C-" + fmt.Sprintf("%06X", time.Now().UnixNano()%0xFFFFFF)
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Checklist SF', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id
`, uid, courseCode).Scan(&courseID)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	tx, _ := pool.Begin(ctx)
	_, _ = tx.Exec(ctx, `INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'teacher')`, courseID, uid)
	_ = courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, uid, courseID, courseCode)
	_ = tx.Commit(ctx)
	tok, _ := signer.Sign(ctx, u.ID, email, "", "", nil)

	// Ensure cold by deleting any snapshot.
	_, _ = pool.Exec(ctx, `DELETE FROM course.course_checklist_snapshots WHERE course_id = $1`, courseID)

	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{ChecklistSnapshotTTL: 15 * time.Minute}})
	const n = 20
	var wg sync.WaitGroup
	computed := make([]string, n)
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/checklist", nil)
			req = req.WithContext(ctx)
			req.Header.Set("Authorization", "Bearer "+tok)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				errs[i] = fmt.Errorf("status %d %s", w.Code, w.Body.String())
				return
			}
			var resp coursechecklist.ChecklistResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				errs[i] = err
				return
			}
			computed[i] = resp.ComputedAt.UTC().Format(time.RFC3339Nano)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i := 1; i < n; i++ {
		if computed[i] != computed[0] {
			t.Fatalf("computedAt mismatch: %s vs %s", computed[0], computed[i])
		}
	}
}

func TestCourseChecklist_Unauthenticated401_NoDB(t *testing.T) {
	h := NewHandler(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-X/checklist", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
}
