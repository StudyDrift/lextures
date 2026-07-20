package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestScreenShareCreateRefusedWhenFlagOff(t *testing.T) {
	d := Deps{Config: config.Config{ScreenShareEnabled: false}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/DEMO/screen-share/sessions", nil)
	_, ok := d.screenShareFlagsOK(rr, req, "DEMO")
	if ok {
		t.Fatal("expected flag off to refuse")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestTurnReadyRequiresSecretAndURLs(t *testing.T) {
	d := Deps{Config: config.Config{}}
	if d.turnReady() {
		t.Fatal("empty should not be ready")
	}
	d.Config.TURNSharedSecret = "secret"
	if d.turnReady() {
		t.Fatal("urls still required")
	}
	d.Config.TURNURLs = []string{"stun:localhost:3478"}
	if !d.turnReady() {
		t.Fatal("expected ready")
	}
}
