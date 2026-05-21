package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminQuarantineFeatureOff(t *testing.T) {
	d := Deps{Config: config.Config{AvScanningEnabled: false}}
	srv := httptest.NewServer(NewHandler(d))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/admin/quarantine")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusUnauthorized && res.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 401 or 501", res.StatusCode)
	}
}
