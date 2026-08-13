package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
)

type fakeAccess struct {
	user       uuid.UUID
	authOK     bool
	authWrote  bool
	courseCode string
	courseOK   bool
	perm       bool
	permErr    error
	courseID   *uuid.UUID
	lookupErr  error
}

func (f *fakeAccess) Authenticate(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if !f.authOK {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		f.authWrote = true
		return uuid.UUID{}, false
	}
	return f.user, true
}

func (f *fakeAccess) RequireCourseAccess(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, bool) {
	if !f.courseOK {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, false
	}
	return f.courseCode, f.user, true
}

func (f *fakeAccess) UserHasPermission(ctx context.Context, userID uuid.UUID, perm string) (bool, error) {
	if f.permErr != nil {
		return false, f.permErr
	}
	return f.perm, nil
}

func (f *fakeAccess) LookupCourseID(ctx context.Context, courseCode string) (*uuid.UUID, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	return f.courseID, nil
}

func TestAuthenticated_DeniesByDefault(t *testing.T) {
	t.Parallel()
	acc := &fakeAccess{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: acc}
	err := Authenticated().Check(ctx)
	if !errors.Is(err, ErrWritten) {
		t.Fatalf("err=%v", err)
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestAuthenticated_Allows(t *testing.T) {
	t.Parallel()
	uid := uuid.New()
	acc := &fakeAccess{authOK: true, user: uid}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: acc}
	if err := Authenticated().Check(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Viewer != uid {
		t.Fatalf("viewer=%s", ctx.Viewer)
	}
}

func TestRequireCoursePermission_LearnerDenied(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	acc := &fakeAccess{
		user:       uuid.New(),
		courseOK:   true,
		courseCode: "BIO101",
		perm:       false,
		courseID:   &cid,
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: acc}
	err := RequireCoursePermission("item:create", "You do not have permission to edit outcomes.").Check(ctx)
	if err == nil {
		t.Fatal("expected deny")
	}
	WriteError(rr, req, err)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
	var env apierr.Body
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != apierr.CodeForbidden || env.Error.Message != "You do not have permission to edit outcomes." {
		t.Fatalf("envelope=%+v", env)
	}
}

func TestRequireCoursePermission_AllowsStaff(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	uid := uuid.New()
	acc := &fakeAccess{
		user:       uid,
		courseOK:   true,
		courseCode: "BIO101",
		perm:       true,
		courseID:   &cid,
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: acc}
	if err := RequireCoursePermission("item:create", "no").Check(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.CourseID != cid || ctx.CourseCode != "BIO101" || ctx.Viewer != uid {
		t.Fatalf("ctx=%+v", ctx)
	}
}

func TestRequireCourseAccess_NoAccess(t *testing.T) {
	t.Parallel()
	acc := &fakeAccess{courseOK: false}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: acc}
	err := RequireCourseAccess().Check(ctx)
	if !errors.Is(err, ErrWritten) {
		t.Fatalf("err=%v", err)
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestZeroGuard_FailsClosed(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Ctx{Context: req.Context(), W: rr, R: req, Access: &fakeAccess{authOK: true, user: uuid.New()}}
	err := Guard{}.Check(ctx)
	if err == nil {
		t.Fatal("zero guard must not authorise")
	}
	m := Map(err)
	if m.Status != http.StatusInternalServerError {
		t.Fatalf("status=%d", m.Status)
	}
}

func TestPublic_MarksCtx(t *testing.T) {
	t.Parallel()
	ctx := &Ctx{}
	if err := Public().Check(ctx); err != nil {
		t.Fatal(err)
	}
	if !ctx.public || !Public().Public() {
		t.Fatal("expected public")
	}
}
