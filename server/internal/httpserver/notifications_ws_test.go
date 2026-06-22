package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func TestNotificationsWS_NoNotifHub_DoesNot503(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-jwt-here")
	h := NewHandler(Deps{JWTSigner: s, NotifHub: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws/notifications", nil)
	h.ServeHTTP(rr, r)
	if rr.Code == http.StatusServiceUnavailable {
		t.Fatalf("notifications ws: unexpected 503 when hub is nil: %d", rr.Code)
	}
}