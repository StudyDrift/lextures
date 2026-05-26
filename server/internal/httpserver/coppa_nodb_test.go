package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCoppa_FeatureOff_Status(t *testing.T) {
	h := NewHandler(Deps{JWTSigner: nil, Config: config.Config{CoppaWorkflowEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/coppa/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature off, got %d", rr.Code)
	}
}

func TestCoppa_FeatureOff_ConsentToken(t *testing.T) {
	h := NewHandler(Deps{JWTSigner: nil, Config: config.Config{CoppaWorkflowEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/consent-token", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature off, got %d", rr.Code)
	}
}

func TestCoppa_FeatureOff_BulkImport(t *testing.T) {
	h := NewHandler(Deps{JWTSigner: nil, Config: config.Config{CoppaWorkflowEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/bulk-import/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature off, got %d", rr.Code)
	}
}

func TestCoppa_Status_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405 for wrong method, got %d", rr.Code)
	}
}

func TestCoppa_Status_RequiresAuth(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/coppa/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when no token, got %d", rr.Code)
	}
}

func TestCoppa_ConsentToken_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/coppa/consent-token", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestCoppa_ConsentRevoke_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/coppa/consent/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestCoppa_AIOptIn_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/coppa/ai-opt-in", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestCoppa_BulkImport_RequiresAuth(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/bulk-import/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when no token, got %d", rr.Code)
	}
}

func TestCoppa_ParentDashboard_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/parent-dashboard", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestCoppa_Initiate_RequiresAuth(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{JWTSigner: s, Config: config.Config{CoppaWorkflowEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/coppa/initiate", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when no token, got %d", rr.Code)
	}
}
