package httpserver

import (
	"bytes"
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
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func TestQuizgameAdmin_SettingsQuotasReviewForceEnd_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupQuizKitTest(t, ctx, "teacher", true, true)
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
	h = NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			FFInteractiveQuizzes: true,
			FFIqLiveHosting:      true,
			FFIqPublicKitCatalog: true,
			CourseFilesRoot:      t.TempDir(),
			PublicWebOrigin:      "http://localhost:5173",
		},
	})

	authHdr := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
	}

	// Platform settings get/patch
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/interactive-quizzes", nil)
	authHdr(req)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get settings: %d %s", rec.Code, rec.Body.String())
	}

	// Clear leftover live sessions from other tests in the shared seed org so quotas are deterministic.
	if _, err := pool.Exec(ctx, `
		UPDATE quizgame.sessions
		SET status = 'ended', current_phase = 'ended', ended_at = COALESCE(ended_at, NOW()), join_code = NULL
		WHERE status IN ('lobby', 'running', 'paused')
	`); err != nil {
		t.Fatalf("clear live sessions: %v", err)
	}

	patchRec := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/settings/interactive-quizzes",
		strings.NewReader(`{"maxConcurrentGames":1,"maxPlayersPerGame":2,"maxKitsPerCourse":50,"guestJoinPolicy":"disabled","retentionDays":180}`))
	authHdr(patchReq)
	h.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch settings: %d %s", patchRec.Code, patchRec.Body.String())
	}
	var settings quizgame.PlatformSettings
	if err := json.Unmarshal(patchRec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.MaxConcurrentGames == nil || *settings.MaxConcurrentGames != 1 {
		t.Fatalf("maxConcurrent=%v", settings.MaxConcurrentGames)
	}
	if settings.MaxPlayersPerGame != 2 {
		t.Fatalf("maxPlayers=%d", settings.MaxPlayersPerGame)
	}

	kitID := seedReadyKit(t, h, tok, cc)

	// Start first game OK
	body, _ := json.Marshal(map[string]any{"pacing": "manual"})
	startRec := httptest.NewRecorder()
	startReq := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	authHdr(startReq)
	h.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusCreated {
		t.Fatalf("start game: %d %s", startRec.Code, startRec.Body.String())
	}
	var started map[string]any
	_ = json.Unmarshal(startRec.Body.Bytes(), &started)
	gameID, _ := started["gameId"].(string)

	// Second concurrent game refused
	start2 := httptest.NewRecorder()
	start2Req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	authHdr(start2Req)
	h.ServeHTTP(start2, start2Req)
	if start2.Code != http.StatusConflict {
		t.Fatalf("expected concurrent quota 409, got %d %s", start2.Code, start2.Body.String())
	}

	// Analytics
	anRec := httptest.NewRecorder()
	anReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/interactive-quizzes/analytics", nil)
	authHdr(anReq)
	h.ServeHTTP(anRec, anReq)
	if anRec.Code != http.StatusOK {
		t.Fatalf("analytics: %d %s", anRec.Code, anRec.Body.String())
	}

	// Catalog submit → review queue
	subRec := httptest.NewRecorder()
	subReq := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/submit-to-catalog", nil)
	authHdr(subReq)
	h.ServeHTTP(subRec, subReq)
	if subRec.Code != http.StatusOK {
		t.Fatalf("submit catalog: %d %s", subRec.Code, subRec.Body.String())
	}

	qRec := httptest.NewRecorder()
	qReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/interactive-quizzes/review-queue", nil)
	authHdr(qReq)
	h.ServeHTTP(qRec, qReq)
	if qRec.Code != http.StatusOK {
		t.Fatalf("review queue: %d %s", qRec.Code, qRec.Body.String())
	}
	var qBody struct {
		Items []quizgame.ModerationQueueItem `json:"items"`
	}
	_ = json.Unmarshal(qRec.Body.Bytes(), &qBody)
	if len(qBody.Items) == 0 {
		t.Fatal("expected pending review item")
	}
	itemID := qBody.Items[0].ID

	rejRec := httptest.NewRecorder()
	rejReq := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/interactive-quizzes/review-queue/"+itemID+"/reject",
		strings.NewReader(`{"reason":"Not suitable"}`))
	authHdr(rejReq)
	h.ServeHTTP(rejRec, rejReq)
	if rejRec.Code != http.StatusOK {
		t.Fatalf("reject: %d %s", rejRec.Code, rejRec.Body.String())
	}

	// Force-end frees concurrency slot
	endRec := httptest.NewRecorder()
	endReq := httptest.NewRequest(http.MethodPost,
		"/api/v1/admin/interactive-quizzes/games/"+gameID+"/force-end", nil)
	authHdr(endReq)
	h.ServeHTTP(endRec, endReq)
	if endRec.Code != http.StatusOK {
		t.Fatalf("force-end: %d %s", endRec.Code, endRec.Body.String())
	}

	start3 := httptest.NewRecorder()
	start3Req := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+cc+"/live-quizzes/kits/"+kitID+"/games", bytes.NewReader(body))
	authHdr(start3Req)
	h.ServeHTTP(start3, start3Req)
	if start3.Code != http.StatusCreated {
		t.Fatalf("start after force-end: %d %s", start3.Code, start3.Body.String())
	}

	// Retention dry-run path
	res, err := quizgame.RunRetention(ctx, pool, time.Now().UTC().AddDate(0, 0, 1), time.Now().UTC().AddDate(0, 0, 1), 10)
	if err != nil {
		t.Fatalf("retention: %v", err)
	}
	_ = res

	// DSAR export/erase smoke (use a throwaway user id that owns no kits)
	dummy := uuid.New()
	export, err := quizgame.ExportUserContent(ctx, pool, dummy)
	if err != nil {
		t.Fatalf("dsar export: %v", err)
	}
	_ = export
	if err := quizgame.EraseUserContent(ctx, pool, dummy); err != nil {
		t.Fatalf("dsar erase: %v", err)
	}

	// Restore non-restrictive settings for other shared-DB tests.
	restore := httptest.NewRecorder()
	restoreReq := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/settings/interactive-quizzes",
		strings.NewReader(`{"maxConcurrentGames":50,"maxPlayersPerGame":300,"guestJoinPolicy":"teacher_mediated"}`))
	authHdr(restoreReq)
	h.ServeHTTP(restore, restoreReq)
	if restore.Code != http.StatusOK {
		t.Fatalf("restore settings: %d %s", restore.Code, restore.Body.String())
	}
}
