package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func TestBoardAdmin_PoliciesOverviewAnalytics_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	var teacherID uuid.UUID
	if err := pool.QueryRow(ctx, `
		SELECT ce.user_id
		FROM course.course_enrollments ce
		INNER JOIN course.courses c ON c.id = ce.course_id
		WHERE c.course_code = $1
		LIMIT 1
	`, cc).Scan(&teacherID); err != nil {
		t.Fatalf("teacher id: %v", err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, teacherID, "Global Admin"); err != nil {
		t.Fatalf("rbac: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			FFVisualBoards:          true,
			FFBoardsExternalSharing: true,
			CoppaWorkflowEnabled:    true,
			CourseFilesRoot:         t.TempDir(),
			PublicWebOrigin:         "http://localhost:5173",
		},
	})

	authHdr := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
	}

	// Reset to known defaults (shared seed org may retain prior test state).
	resetRec := httptest.NewRecorder()
	resetReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/boards/policies",
		strings.NewReader(`{"externalSharing":false,"minorModerationFloor":true,"clearBoardCap":true,"defaultAttribution":"named"}`))
	authHdr(resetReq)
	h.ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset policies: %d %s", resetRec.Code, resetRec.Body.String())
	}
	var pol board.OrgPolicies
	if err := json.Unmarshal(resetRec.Body.Bytes(), &pol); err != nil {
		t.Fatalf("decode reset: %v", err)
	}
	if pol.ExternalSharing {
		t.Fatal("external sharing should be off after reset")
	}
	if !pol.MinorModerationFloor {
		t.Fatal("minor floor should be on after reset")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/boards/policies", nil)
	authHdr(req)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get policies: %d %s", rec.Code, rec.Body.String())
	}

	patchRec := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/boards/policies",
		strings.NewReader(`{"externalSharing":true,"boardCapPerCourse":2}`))
	authHdr(patchReq)
	h.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch policies: %d %s", patchRec.Code, patchRec.Body.String())
	}
	if err := json.Unmarshal(patchRec.Body.Bytes(), &pol); err != nil {
		t.Fatalf("decode patched: %v", err)
	}
	if !pol.ExternalSharing || pol.BoardCapPerCourse == nil || *pol.BoardCapPerCourse != 2 {
		t.Fatalf("unexpected policies: %+v", pol)
	}

	for i, title := range []string{"A", "B"} {
		cr := httptest.NewRecorder()
		cq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards",
			strings.NewReader(`{"title":"`+title+`"}`))
		authHdr(cq)
		h.ServeHTTP(cr, cq)
		if cr.Code != http.StatusCreated {
			t.Fatalf("create board %d: %d %s", i, cr.Code, cr.Body.String())
		}
	}
	capRec := httptest.NewRecorder()
	capReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards",
		strings.NewReader(`{"title":"Over Cap"}`))
	authHdr(capReq)
	h.ServeHTTP(capRec, capReq)
	if capRec.Code != http.StatusForbidden {
		t.Fatalf("expected cap 403, got %d %s", capRec.Code, capRec.Body.String())
	}

	ovRec := httptest.NewRecorder()
	ovReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/boards/overview", nil)
	authHdr(ovReq)
	h.ServeHTTP(ovRec, ovReq)
	if ovRec.Code != http.StatusOK {
		t.Fatalf("overview: %d %s", ovRec.Code, ovRec.Body.String())
	}
	var overview board.AdminOverview
	if err := json.Unmarshal(ovRec.Body.Bytes(), &overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overview.BoardCount < 2 {
		t.Fatalf("boardCount=%d, want >= 2", overview.BoardCount)
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards", nil)
	authHdr(listReq)
	h.ServeHTTP(listRec, listReq)
	var boards []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &boards); err != nil {
		var wrapped struct {
			Boards []map[string]any `json:"boards"`
		}
		if err2 := json.Unmarshal(listRec.Body.Bytes(), &wrapped); err2 != nil || len(wrapped.Boards) == 0 {
			t.Fatalf("list boards: %s", listRec.Body.String())
		}
		boards = wrapped.Boards
	}
	boardID, _ := boards[0]["id"].(string)

	postRec := httptest.NewRecorder()
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards/"+boardID+"/posts",
		strings.NewReader(`{"contentType":"text","title":"Hello","body":{"text":"hi"}}`))
	authHdr(postReq)
	h.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", postRec.Code, postRec.Body.String())
	}

	if _, err := board.RefreshAnalyticsDaily(ctx, pool, nil, time.Now().UTC()); err != nil {
		t.Fatalf("rollup: %v", err)
	}

	anRec := httptest.NewRecorder()
	anReq := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/analytics", nil)
	authHdr(anReq)
	h.ServeHTTP(anRec, anReq)
	if anRec.Code != http.StatusOK {
		t.Fatalf("analytics: %d %s", anRec.Code, anRec.Body.String())
	}
	var sum board.BoardAnalyticsSummary
	if err := json.Unmarshal(anRec.Body.Bytes(), &sum); err != nil {
		t.Fatalf("decode analytics: %v", err)
	}
	if sum.CardCount < 1 {
		t.Fatalf("cardCount=%d", sum.CardCount)
	}

	orgID, err := organization.OrgIDForUser(ctx, pool, teacherID)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	resolved, err := board.ResolveOrgPolicies(ctx, pool, orgID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !board.ExternalSharingAllowed(true, resolved) {
		t.Fatal("expected external sharing allowed after patch")
	}
}
