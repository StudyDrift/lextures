package httpserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/lti"
)

func ltiTestDeps(t *testing.T) Deps {
	t.Helper()
	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	pemS := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pair, err := lti.FromPKCS8PEM(pemS, "k1")
	if err != nil {
		t.Fatal(err)
	}
	return Deps{
		Config: config.Config{
			LTIEnabled:          true,
			LTIRSAPrivateKeyPEM: pemS,
			LTIRSAKeyID:         "k1",
			LTIAPIBaseURL:       "http://platform.test",
			PublicWebOrigin:     "http://app.test",
		},
		Lti: &lti.Runtime{Enabled: true, Keys: pair, APIBaseURL: "http://platform.test"},
	}
}

func TestLtiConsumerEndpoints_LtiOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{}})
	for _, path := range []string{
		"/api/v1/lti/deep-link",
		"/api/v1/lti/callback",
		"/api/v1/lti/launch/00000000-0000-0000-0000-000000000001",
	} {
		rr := httptest.NewRecorder()
		method := http.MethodGet
		if strings.Contains(path, "deep-link") || strings.Contains(path, "launch") {
			method = http.MethodPost
		}
		r := httptest.NewRequest(method, path, nil)
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400 when LTI off, got %d %s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestLtiConsumerCallback_MissingParams(t *testing.T) {
	h := NewHandler(ltiTestDeps(t))
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/lti/callback", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("callback: %d %s", rr.Code, rr.Body.String())
	}
}

func TestLtiDeepLink_MissingJWT(t *testing.T) {
	h := NewHandler(ltiTestDeps(t))
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/lti/deep-link", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("deep-link: %d %s", rr.Code, rr.Body.String())
	}
}

func TestLtiPlatformLaunch_Unauthorized(t *testing.T) {
	h := NewHandler(ltiTestDeps(t))
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/lti/launch/00000000-0000-0000-0000-000000000001", strings.NewReader(`{"courseId":"00000000-0000-0000-0000-000000000002"}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("launch: %d %s", rr.Code, rr.Body.String())
	}
}

func TestLtiConsumerTarget_Redirects(t *testing.T) {
	h := NewHandler(ltiTestDeps(t))
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/lti/consumer/target", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("target: %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "http://app.test/") {
		t.Fatalf("expected redirect to app origin, got %s", rr.Body.String())
	}
}