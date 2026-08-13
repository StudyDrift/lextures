package kernel

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
)

func TestPOST_SuccessAndShorterPath(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ResetRegistryForTest()
	cid := uuid.New()
	acc := &fakeAccess{
		user: uuid.New(), courseOK: true, courseCode: "BIO101", perm: true, courseID: &cid,
	}
	type in struct {
		Title string `json:"title"`
	}
	type out struct {
		Title string `json:"title"`
		Code  string `json:"courseCode"`
	}
	h := POST(acc, RequireCoursePermission("item:create", "no"), func(c *Ctx, body in) (out, error) {
		if body.Title == "" {
			return out{}, InvalidInput("Title is required.")
		}
		return out{Title: body.Title, Code: c.CourseCode}, nil
	}, WithStatus(http.StatusCreated), WithName("CreateOutcome"), WithDecodeOptions(DecodeOptions{RequireJSONContentType: false}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"CLO 1"}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got out
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "CLO 1" || got.Code != "BIO101" {
		t.Fatalf("got=%+v", got)
	}
}

func TestPOST_RepoNoRows(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	cid := uuid.New()
	acc := &fakeAccess{user: uuid.New(), courseOK: true, courseCode: "X", perm: true, courseID: &cid}
	h := POST(acc, RequireCourseAccess(), func(c *Ctx, _ None) (struct{}, error) {
		return struct{}{}, pgx.ErrNoRows
	})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestGET_NilGuardFailsClosedAndCounts(t *testing.T) {
	ResetRegistryForTest()
	acc := &fakeAccess{} // unauthenticated
	h := GET(acc, Guard{}, func(c *Ctx) (struct{ OK bool }, error) {
		return struct{ OK bool }{true}, nil
	}, WithName("secret"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if UnguardedCount() < 1 {
		t.Fatal("nil guard must be reported as unguarded")
	}
}

func TestGET_PublicCountsUnguarded(t *testing.T) {
	ResetRegistryForTest()
	h := GET[map[string]string](nil, Public(), func(c *Ctx) (map[string]string, error) {
		return map[string]string{"ok": "1"}, nil
	}, WithName("public-ping"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	found := false
	for _, r := range RegisteredRoutes() {
		if r.Name == "public-ping" && r.Unguarded {
			found = true
		}
	}
	if !found {
		t.Fatalf("routes=%+v", RegisteredRoutes())
	}
}

func TestPOST_OversizedBody(t *testing.T) {
	cid := uuid.New()
	acc := &fakeAccess{user: uuid.New(), courseOK: true, courseCode: "X", perm: true, courseID: &cid}
	h := POST(acc, RequireCourseAccess(), func(c *Ctx, body map[string]string) (struct{}, error) {
		return struct{}{}, nil
	}, WithDecodeOptions(DecodeOptions{MaxBytes: 32, RequireJSONContentType: false}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"`+strings.Repeat("x", 200)+`"}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTypesExported(t *testing.T) {
	t.Parallel()
	type in struct {
		Title string `json:"title"`
	}
	type out struct {
		ID string `json:"id"`
	}
	if InputType[in, out]().Name() != "in" {
		t.Fatal(InputType[in, out]())
	}
	if OutputType[in, out]().Name() != "out" {
		t.Fatal(OutputType[in, out]())
	}
}

func TestWriteErrorForbiddenEnvelope(t *testing.T) {
	t.Parallel()
	// Same envelope the hand-rolled check produced.
	rr := httptest.NewRecorder()
	WriteError(rr, httptest.NewRequest(http.MethodPost, "/", nil), Forbidden("You do not have permission to edit outcomes."))
	var env apierr.Body
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if rr.Code != 403 || env.Error.Code != apierr.CodeForbidden {
		t.Fatalf("%d %+v", rr.Code, env)
	}
}
