package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestOIDCLink_Method(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/link", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET oidc link: %d", rr.Code)
	}
}

func TestOIDCLink_BadBody(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{OIDCSSOEnabled: true}}
	tok, err := d.JWTSigner.Sign(context.Background(), "a0000000-0000-4000-8000-000000000002", "x@y.com", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/link", strings.NewReader("notjson"))
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code: %d", rr.Code)
	}
}

func TestSAMLStatus_Disabled(t *testing.T) {
	d := Deps{Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != 200 {
		t.Fatalf("code: %d", rr.Code)
	}
	var m map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m["enabled"] != false {
		t.Fatalf("enabled: %v", m["enabled"])
	}
}

func TestOIDCStatus_DisabledShape(t *testing.T) {
	// Empty config still exposes appleNative (default audience com.lextures.ios); google stays off.
	d := Deps{Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != 200 {
		t.Fatalf("code: %d", rr.Code)
	}
	var m map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	// Native Apple is on by default (bundle ID audience), so enabled is true even without tenant OIDC.
	if m["enabled"] != true {
		t.Fatalf("enabled: %v (want true because appleNative defaults on)", m["enabled"])
	}
	if m["appleNative"] != true {
		t.Fatalf("appleNative: %v want true", m["appleNative"])
	}
	if m["googleNative"] != false {
		t.Fatalf("googleNative: %v want false without client id", m["googleNative"])
	}
}

func TestOIDCStatus_NativeFlags(t *testing.T) {
	d := Deps{Config: config.Config{
		OIDCAppleNativeAudience: "com.lextures.ios",
		OIDCGoogleClientID:      "google-server-client",
	}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != 200 {
		t.Fatalf("code: %d %s", rr.Code, rr.Body.String())
	}
	var m map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m["enabled"] != true {
		t.Fatalf("enabled with native: %v", m["enabled"])
	}
	if m["appleNative"] != true || m["googleNative"] != true {
		t.Fatalf("native: apple=%v google=%v", m["appleNative"], m["googleNative"])
	}
}

func TestOIDCNative_MissingFields(t *testing.T) {
	d := Deps{
		Pool:      nil,
		JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"),
		Config: config.Config{
			OIDCAppleNativeAudience: "com.lextures.ios",
			OIDCGoogleClientID:      "g",
		},
	}
	h := NewHandler(d)
	for _, path := range []string{"/api/v1/auth/oidc/apple/native", "/api/v1/auth/oidc/google/native"} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{}`)))
		h.ServeHTTP(rr, r)
		// No DB → 503 before body validation is fine; with nil pool we expect 503.
		if rr.Code != http.StatusServiceUnavailable && rr.Code != 400 {
			t.Fatalf("%s: code %d body %s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestSAMLStatus_EnabledNoPool(t *testing.T) {
	d := Deps{Config: config.Config{SAMLSSOEnabled: true, SAMLSPX509PEM: "x"}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code: %d", rr.Code)
	}
}

func TestAuthBody_InvalidJSON(t *testing.T) {
	d := Deps{Pool: nil, JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("not-json")))
	h.ServeHTTP(rr, r)
	if rr.Code != 400 {
		t.Fatalf("login: %d", rr.Code)
	}
}
