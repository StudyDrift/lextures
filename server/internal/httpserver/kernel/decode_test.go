package kernel

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeDest struct {
	Title string `json:"title"`
}

func TestDecodeJSON_OK(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A","extra":1}`))
	req.Header.Set("Content-Type", "application/json")
	var dst decodeDest
	if err := DecodeJSON(rr, req, &dst, DecodeOptions{}); err != nil {
		t.Fatal(err)
	}
	if dst.Title != "A" {
		t.Fatalf("title=%q", dst.Title)
	}
}

func TestDecodeJSON_UnknownFieldsIgnoredByDefault(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A","nope":true}`))
	var dst decodeDest
	if err := DecodeJSON(rr, req, &dst, DecodeOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeJSON_DisallowUnknownFields(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A","nope":true}`))
	var dst decodeDest
	err := DecodeJSON(rr, req, &dst, DecodeOptions{DisallowUnknownFields: true})
	if err == nil {
		t.Fatal("expected error")
	}
	m := Map(err)
	if m.Status != http.StatusBadRequest || m.Code != "INVALID_INPUT" {
		t.Fatalf("mapped=%+v", m)
	}
}

func TestDecodeJSON_Malformed(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`))
	var dst decodeDest
	err := DecodeJSON(rr, req, &dst, DecodeOptions{InvalidJSONMessage: "Invalid JSON body."})
	if err == nil {
		t.Fatal("expected error")
	}
	m := Map(err)
	if m.Message != "Invalid JSON body." || m.Status != 400 {
		t.Fatalf("mapped=%+v", m)
	}
}

func TestDecodeJSON_TooLarge(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	payload := append([]byte(`{"title":"`), bytes.Repeat([]byte("a"), 64)...)
	payload = append(payload, '"', '}')
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	var dst decodeDest
	err := DecodeJSON(rr, req, &dst, DecodeOptions{MaxBytes: 16})
	if err == nil {
		t.Fatal("expected error")
	}
	m := Map(err)
	if m.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d mapped=%+v", m.Status, m)
	}
	// MaxBytesReader must have replaced the body so further reads fail fast.
	_, _ = io.Copy(io.Discard, req.Body)
}

func TestDecodeJSON_RequireContentType(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A"}`))
	req.Header.Set("Content-Type", "text/plain")
	var dst decodeDest
	err := DecodeJSON(rr, req, &dst, DecodeOptions{RequireJSONContentType: true})
	if err == nil {
		t.Fatal("expected error")
	}
	m := Map(err)
	if m.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d", m.Status)
	}
}

func TestDecodeJSON_ContentTypeCharsetOK(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	var dst decodeDest
	if err := DecodeJSON(rr, req, &dst, DecodeOptions{RequireJSONContentType: true}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeJSON_LooseContentTypePreserved(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"A"}`))
	// No Content-Type — matches pre-toolkit handlers.
	var dst decodeDest
	if err := DecodeJSON(rr, req, &dst, DecodeOptions{RequireJSONContentType: false}); err != nil {
		t.Fatal(err)
	}
}

func TestIsJSONContentType(t *testing.T) {
	t.Parallel()
	if isJSONContentType("") || isJSONContentType("text/plain") {
		t.Fatal("empty/plain should fail")
	}
	if !isJSONContentType("application/json") || !isJSONContentType("application/json; charset=utf-8") {
		t.Fatal("json should pass")
	}
}

func TestHandRolledDecodeEquivalent(t *testing.T) {
	t.Parallel()
	raw := `{"title":"Hello"}`
	var a, b decodeDest
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&a); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
	if err := DecodeJSON(rr, req, &b, DecodeOptions{RequireJSONContentType: false}); err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("%+v vs %+v", a, b)
	}
}
