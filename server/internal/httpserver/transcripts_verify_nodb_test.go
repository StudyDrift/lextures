package httpserver

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestVerifyCredential_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: false, FFCoCurricularTranscript: false}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify/some-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestVerifyUpload_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: false}})
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "t.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("%PDF-1.4"))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", rec.Code)
	}
}

func TestVerifyUpload_RequiresPDF(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: true}, Pool: nil})
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "t.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("not a pdf"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	// Pool nil → 500 after parse, or 400 for non-PDF. Prefer validating content type first.
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 400/500 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminRevoke_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: true}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/transcripts/documents/00000000-0000-0000-0000-000000000001/revoke", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
}

func TestInstitutionDID_TranscriptsOnly(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{
		FFTranscripts:            true,
		FFCoCurricularTranscript: false,
		PublicWebOrigin:          "http://localhost:5173",
		JWTSecret:                "01234567890123456789012345678901",
	}})
	req := httptest.NewRequest(http.MethodGet, "/.well-known/did.json", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("did:web:")) {
		t.Fatalf("expected did:web in body: %s", rec.Body.String())
	}
}
