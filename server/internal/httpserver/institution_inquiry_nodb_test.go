package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPublicInstitutionInquiry_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/public/institution-inquiries", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestPublicInstitutionInquiry_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/institution-inquiries", bytes.NewBufferString("{bad"))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicInstitutionInquiry_MissingEmail(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	body, _ := json.Marshal(map[string]string{
		"organization_type":  "University",
		"organization_name":  "State U",
		"contact_name":       "Ada Lovelace",
		"enrollment_size":    "1,000 – 10,000",
		"hosting_preference": "Not sure yet",
		"message":            "Interested in a pilot.",
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/institution-inquiries", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing email, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicInstitutionInquiry_InvalidEmail(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	body, _ := json.Marshal(map[string]string{
		"organization_type":  "University",
		"organization_name":  "State U",
		"contact_name":       "Ada Lovelace",
		"email":              "not-an-email",
		"enrollment_size":    "1,000 – 10,000",
		"hosting_preference": "Not sure yet",
		"message":            "Interested in a pilot.",
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/institution-inquiries", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid email, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicInstitutionInquiry_NoDB_Returns503(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Pool: nil})
	body, _ := json.Marshal(map[string]string{
		"organization_type":  "University",
		"organization_name":  "State U",
		"contact_name":       "Ada Lovelace",
		"email":              "ada@example.edu",
		"enrollment_size":    "1,000 – 10,000",
		"hosting_preference": "Not sure yet",
		"message":            "Interested in a pilot.",
	})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/public/institution-inquiries", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without DB pool, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestInquiryHelpers(t *testing.T) {
	t.Parallel()
	if inquiryLooksLikeEmail("not-an-email") {
		t.Fatal("expected invalid email")
	}
	if !inquiryLooksLikeEmail("dean@university.edu") {
		t.Fatal("expected valid email")
	}
	if got := inquiryTrim("  hello  ", 10); got != "hello" {
		t.Fatalf("trim: %q", got)
	}
	long := strings.Repeat("x", 100)
	if got := inquiryTrim(long, 10); utf8.RuneCountInString(got) != 10 {
		t.Fatalf("truncate: len=%d", utf8.RuneCountInString(got))
	}
}
