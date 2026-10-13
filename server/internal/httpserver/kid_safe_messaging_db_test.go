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
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

// TestKidSafeMessaging_Pg covers #775: managed learners message only their parent and
// parent-approved teachers (403 otherwise), and parents can read their kid's threads.
func TestKidSafeMessaging_Pg(t *testing.T) {
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

	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().Format("20060102150405.000000")
	mk := func(prefix, name string) *user.Row {
		t.Helper()
		dn := name
		row, err := user.InsertUser(ctx, pool, prefix+"-"+ts+"@e.com", ph, &dn)
		if err != nil {
			t.Fatalf("user %s: %v", prefix, err)
		}
		return row
	}
	parent := mk("ksm-par", "Kid Parent")
	teacher := mk("ksm-tch", "Co-op Teacher")
	stranger := mk("ksm-str", "Stranger")
	parentID := uuid.MustParse(parent.ID)

	dep, err := managedlearners.Create(ctx, pool, managedlearners.CreateParams{ActorID: parentID, DisplayName: "Messaging Kid"})
	if err != nil {
		t.Fatalf("dependent: %v", err)
	}
	kidID := uuid.MustParse(dep.ID)
	kidRow, err := user.FindByID(ctx, pool, kidID)
	if err != nil || kidRow == nil {
		t.Fatalf("kid row: %v", err)
	}

	var courseID uuid.UUID
	cc := "C-" + strings.ToUpper(uuid.NewString()[:6])
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Co-op Science', $2) RETURNING id
`, cc, parentID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student'), ($1, $3, 'teacher')
`, courseID, kidID, uuid.MustParse(teacher.ID)); err != nil {
		t.Fatalf("enroll: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{FFHomeschoolManagedLearners: true}})
	tokenFor := func(row *user.Row) string {
		t.Helper()
		uid := uuid.MustParse(row.ID)
		slug, _ := organization.OrgSlugForUser(ctx, pool, uid)
		oid, _ := organization.OrgIDForUser(ctx, pool, uid)
		tok, err := signer.Sign(ctx, row.ID, row.Email, oid.String(), slug, nil)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return tok
	}
	sess, err := managedlearners.StartSession(ctx, pool, signer, parentID, kidID)
	if err != nil {
		t.Fatalf("learn-as: %v", err)
	}
	kidTok := sess.Token
	parentTok := tokenFor(parent)
	teacherTok := tokenFor(teacher)
	strangerTok := tokenFor(stranger)

	do := func(tok, method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var r *http.Request
		if body != nil {
			b, _ := json.Marshal(body)
			r = httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(ctx)
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, path, nil).WithContext(ctx)
		}
		r.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)
		return rr
	}
	send := func(tok string, body map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		body["subject"] = "Hi"
		body["body"] = "Hello"
		return do(tok, http.MethodPost, "/api/v1/communication/messages", body)
	}
	recipientIDs := func(tok string) (bool, []string) {
		t.Helper()
		rr := do(tok, http.MethodGet, "/api/v1/communication/recipients", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("recipients: %d %s", rr.Code, rr.Body.String())
		}
		var out struct {
			Restricted bool `json:"restricted"`
			Recipients []struct {
				ID    string `json:"id"`
				Kind  string `json:"kind"`
				Email string `json:"email"`
			} `json:"recipients"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		ids := make([]string, 0, len(out.Recipients))
		for _, rc := range out.Recipients {
			if rc.Email != "" {
				t.Fatalf("recipients must not expose email: %+v", rc)
			}
			ids = append(ids, rc.ID+":"+rc.Kind)
		}
		return out.Restricted, ids
	}

	// Adults are unrestricted and keep free-text To.
	if restricted, ids := recipientIDs(parentTok); restricted || len(ids) != 0 {
		t.Fatalf("parent recipients: restricted=%v ids=%v", restricted, ids)
	}
	if rr := send(parentTok, map[string]any{"to_email": stranger.Email}); rr.Code != http.StatusOK {
		t.Fatalf("adult send: %d %s", rr.Code, rr.Body.String())
	}

	// Kid sees only the parent.
	restricted, ids := recipientIDs(kidTok)
	if !restricted || len(ids) != 1 || ids[0] != parent.ID+":parent" {
		t.Fatalf("kid recipients: restricted=%v ids=%v", restricted, ids)
	}

	// Kid → stranger / unapproved teacher / unknown address: 403.
	for name, body := range map[string]map[string]any{
		"stranger email": {"to_email": stranger.Email},
		"stranger id":    {"toUserId": stranger.ID},
		"teacher id":     {"toUserId": teacher.ID},
		"unknown email":  {"to_email": "nobody-" + ts + "@e.com"},
		"self via id":    {"toUserId": kidID.String()},
	} {
		if rr := send(kidTok, body); rr.Code != http.StatusForbidden {
			t.Fatalf("kid → %s: want 403 got %d %s", name, rr.Code, rr.Body.String())
		}
	}
	// Others cannot reach the kid either.
	if rr := send(strangerTok, map[string]any{"to_email": kidRow.Email}); rr.Code != http.StatusForbidden {
		t.Fatalf("stranger → kid: want 403 got %d %s", rr.Code, rr.Body.String())
	}
	if rr := send(teacherTok, map[string]any{"toUserId": kidID.String()}); rr.Code != http.StatusForbidden {
		t.Fatalf("unapproved teacher → kid: want 403 got %d %s", rr.Code, rr.Body.String())
	}

	// Kid → parent works (by id and by email).
	if rr := send(kidTok, map[string]any{"toUserId": parent.ID}); rr.Code != http.StatusOK {
		t.Fatalf("kid → parent: %d %s", rr.Code, rr.Body.String())
	}
	if rr := send(parentTok, map[string]any{"to_email": kidRow.Email}); rr.Code != http.StatusOK {
		t.Fatalf("parent → kid: %d %s", rr.Code, rr.Body.String())
	}

	// Parent approves the course teacher; strangers cannot be approved.
	base := "/api/v1/parent/students/" + kidID.String()
	rr := do(parentTok, http.MethodGet, base+"/messaging/contacts", nil)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), teacher.ID) {
		t.Fatalf("contacts: %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(parentTok, http.MethodPut, base+"/messaging/contacts/"+stranger.ID, nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("approve stranger: want 400 got %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(strangerTok, http.MethodPut, base+"/messaging/contacts/"+teacher.ID, nil); rr.Code != http.StatusForbidden {
		t.Fatalf("stranger approve: want 403 got %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(kidTok, http.MethodPut, base+"/messaging/contacts/"+teacher.ID, nil); rr.Code != http.StatusForbidden {
		t.Fatalf("kid self-approve: want 403 got %d %s", rr.Code, rr.Body.String())
	}
	rr = do(parentTok, http.MethodPut, base+"/messaging/contacts/"+teacher.ID, nil)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"approved":true`) {
		t.Fatalf("approve teacher: %d %s", rr.Code, rr.Body.String())
	}
	if _, ids := recipientIDs(kidTok); len(ids) != 2 || ids[1] != teacher.ID+":teacher" {
		t.Fatalf("kid recipients after approval: %v", ids)
	}
	if rr := send(kidTok, map[string]any{"toUserId": teacher.ID}); rr.Code != http.StatusOK {
		t.Fatalf("kid → approved teacher: %d %s", rr.Code, rr.Body.String())
	}
	if rr := send(teacherTok, map[string]any{"toUserId": kidID.String()}); rr.Code != http.StatusOK {
		t.Fatalf("approved teacher → kid: %d %s", rr.Code, rr.Body.String())
	}

	// Parent sees the kid's threads (sent + inbox); synthetic kid email is redacted.
	rr = do(parentTok, http.MethodGet, base+"/messages?folder=sent", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("parent kid sent: %d %s", rr.Code, rr.Body.String())
	}
	var list struct {
		Messages []struct {
			ID   string `json:"id"`
			To   string `json:"to"`
			From struct {
				Email string `json:"email"`
			} `json:"from"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Messages) != 2 {
		t.Fatalf("kid sent messages: want 2 got %d %s", len(list.Messages), rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), kidRow.Email) {
		t.Fatalf("synthetic kid email leaked: %s", rr.Body.String())
	}
	rr = do(parentTok, http.MethodGet, base+"/messages", nil)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), teacher.Email) {
		t.Fatalf("parent kid inbox: %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(parentTok, http.MethodGet, base+"/messages/"+list.Messages[0].ID, nil); rr.Code != http.StatusOK {
		t.Fatalf("parent kid message: %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(parentTok, http.MethodGet, base+"/messages?folder=trash", nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("parent kid trash: want 400 got %d", rr.Code)
	}
	if rr := do(strangerTok, http.MethodGet, base+"/messages", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("stranger kid messages: want 403 got %d", rr.Code)
	}
	if rr := do(kidTok, http.MethodGet, base+"/messages", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("kid reading parent view: want 403 got %d", rr.Code)
	}

	// Revoking the teacher closes the channel again.
	if rr := do(parentTok, http.MethodDelete, base+"/messaging/contacts/"+teacher.ID, nil); rr.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rr.Code, rr.Body.String())
	}
	if rr := send(kidTok, map[string]any{"toUserId": teacher.ID}); rr.Code != http.StatusForbidden {
		t.Fatalf("kid → revoked teacher: want 403 got %d %s", rr.Code, rr.Body.String())
	}

	// Kid Settings: simplified profile only, inside the Learn-as session.
	rr = do(kidTok, http.MethodGet, "/api/v1/me/learner-settings", nil)
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), "email") {
		t.Fatalf("learner settings get: %d %s", rr.Code, rr.Body.String())
	}
	rr = do(kidTok, http.MethodPatch, "/api/v1/me/learner-settings", map[string]any{"displayName": "  Kiddo  ", "uiTheme": "dark"})
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"displayName":"Kiddo"`) || !strings.Contains(rr.Body.String(), `"uiTheme":"dark"`) {
		t.Fatalf("learner settings patch: %d %s", rr.Code, rr.Body.String())
	}
	if rr := do(kidTok, http.MethodPatch, "/api/v1/me/learner-settings", map[string]any{"displayName": "   "}); rr.Code != http.StatusBadRequest {
		t.Fatalf("blank display name: want 400 got %d", rr.Code)
	}
	if rr := do(parentTok, http.MethodGet, "/api/v1/me/learner-settings", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("parent learner settings: want 403 got %d", rr.Code)
	}
	if rr := do(kidTok, http.MethodDelete, "/api/v1/settings/account", nil); rr.Code != http.StatusForbidden {
		t.Fatalf("kid delete account: want 403 got %d %s", rr.Code, rr.Body.String())
	}
	// The Learn-as session is still valid after loading Settings.
	if rr := do(kidTok, http.MethodGet, "/api/v1/me", nil); rr.Code != http.StatusOK {
		t.Fatalf("kid session after settings: %d %s", rr.Code, rr.Body.String())
	}
}
