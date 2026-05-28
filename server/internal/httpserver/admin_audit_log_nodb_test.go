package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminAuditLog_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{AdminAuditLogEnabled: false}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/audit-log"},
		{http.MethodGet, "/api/v1/compliance/audit-log/export"},
		{http.MethodGet, "/api/v1/compliance/audit-log/00000000-0000-0000-0000-000000000001"},
	}
	for _, p := range paths {
		r := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: status=%d want 404", p.method, p.path, w.Code)
		}
	}
}

func TestAdminAuditLog_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{AdminAuditLogEnabled: true}}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/audit-log"},
		{http.MethodGet, "/api/v1/compliance/audit-log/export"},
		{http.MethodGet, "/api/v1/compliance/audit-log/00000000-0000-0000-0000-000000000001"},
	}
	for _, p := range paths {
		r := httptest.NewRequest(p.method, p.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status=%d want 401", p.method, p.path, w.Code)
		}
	}
}

func TestAdminAuditLog_EventIDNotUUID_Returns401(t *testing.T) {
	// Without auth, auth check fires before UUID parse; expect 401.
	d := Deps{Config: config.Config{AdminAuditLogEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-log/not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}

func TestAdminAuditLog_ExportUnauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{AdminAuditLogEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/audit-log/export?format=csv", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d want 401", w.Code)
	}
}
