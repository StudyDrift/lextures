package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/logging"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/transcriptverify"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	transcriptVerifyLimit  = 60
	transcriptUploadLimit  = 20
	transcriptVerifyWindow = time.Minute
	maxVerifyUploadBytes   = 12 << 20
)

type transcriptVerifyAttempt struct {
	count int
	start time.Time
}

var (
	transcriptVerifyMu       sync.Mutex
	transcriptVerifyAttempts = map[string]*transcriptVerifyAttempt{}
)

func transcriptVerifyRateLimited(key string, limit int) bool {
	transcriptVerifyMu.Lock()
	defer transcriptVerifyMu.Unlock()
	now := time.Now()
	e := transcriptVerifyAttempts[key]
	if e == nil || now.Sub(e.start) > transcriptVerifyWindow {
		transcriptVerifyAttempts[key] = &transcriptVerifyAttempt{count: 1, start: now}
		return false
	}
	e.count++
	return e.count > limit
}

func (d Deps) registerTranscriptVerifyRoutes(r chi.Router) {
	r.Get("/api/v1/verify/{shareToken}", d.handleUnifiedCredentialVerify())
	r.Post("/api/v1/verify/upload", d.handleUnifiedCredentialVerifyUpload())
	r.Post("/api/v1/admin/transcripts/documents/{id}/revoke", d.handleAdminRevokeTranscriptDocument())
	r.Post("/api/v1/admin/transcripts/documents/{id}/unrevoke", d.handleAdminUnrevokeTranscriptDocument())
	r.Patch("/api/v1/transcripts/documents/{id}/disclosure", d.handlePatchTranscriptDocumentDisclosure())
}

func (d Deps) verifySigningAvailable() bool {
	cfg := d.effectiveConfig()
	return cfg.FFTranscripts || cfg.FFCoCurricularTranscript || cfg.FFDiplomas
}

// GET /api/v1/verify/{shareToken} — unified transcript + CLR verify (T08; preserves CLR path).
func (d Deps) handleUnifiedCredentialVerify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.verifySigningAvailable() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential verification is not enabled.")
			return
		}
		if transcriptVerifyRateLimited("link:"+r.RemoteAddr, transcriptVerifyLimit) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "shareToken"))
		method := transcriptsrepo.VerifyMethodLink
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("via")), "qr") {
			method = transcriptsrepo.VerifyMethodQR
		}
		out, err := transcriptverify.VerifyByToken(r.Context(), d.Pool, d.effectiveConfig(), token, transcriptverify.Context{
			Method: method,
			IP:     clientIP(r),
			UA:     r.UserAgent(),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		logging.GlobalCredentialVerifyMetrics.IncResult(out.Result)
		logging.GlobalCCRMetrics.IncVerifications()
		telemetry.RecordBusinessEvent("credential.verified")

		if out.Result == transcriptverify.ResultNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// POST /api/v1/verify/upload — PDF hash check against issued transcripts (T08).
func (d Deps) handleUnifiedCredentialVerifyUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFTranscripts {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Transcript verification is not enabled.")
			return
		}
		if transcriptVerifyRateLimited("upload:"+r.RemoteAddr, transcriptUploadLimit) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		if err := r.ParseMultipartForm(maxVerifyUploadBytes); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid multipart form or file too large.")
			return
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "file is required.")
			return
		}
		defer f.Close()
		if header.Size > maxVerifyUploadBytes {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File too large.")
			return
		}
		limited := io.LimitReader(f, maxVerifyUploadBytes+1)
		pdfBytes, err := io.ReadAll(limited)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read file.")
			return
		}
		if len(pdfBytes) > maxVerifyUploadBytes {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "File too large.")
			return
		}
		if len(pdfBytes) < 5 || string(pdfBytes[:4]) != "%PDF" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Upload must be a PDF.")
			return
		}

		out, err := transcriptverify.VerifyByPDFHash(r.Context(), d.Pool, d.effectiveConfig(), pdfBytes, transcriptverify.Context{
			Method: transcriptsrepo.VerifyMethodUpload,
			IP:     clientIP(r),
			UA:     r.UserAgent(),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		logging.GlobalCredentialVerifyMetrics.IncResult(out.Result)
		logging.GlobalCredentialVerifyMetrics.IncUpload()
		telemetry.RecordBusinessEvent("credential.verified")

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if out.Result == transcriptverify.ResultNotFound {
			w.WriteHeader(http.StatusNotFound)
		}
		_ = json.NewEncoder(w).Encode(out)
	}
}

type revokeBody struct {
	Reason string `json:"reason"`
}

// POST /api/v1/admin/transcripts/documents/{id}/revoke
func (d Deps) handleAdminRevokeTranscriptDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		docID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid document id.")
			return
		}
		var body revokeBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		doc, err := transcriptsrepo.RevokeDocument(r.Context(), d.Pool, docID, body.Reason)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}
		telemetry.RecordBusinessEvent("credential.revoked")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": documentToJSON(doc)})
	}
}

// POST /api/v1/admin/transcripts/documents/{id}/unrevoke
func (d Deps) handleAdminUnrevokeTranscriptDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		docID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid document id.")
			return
		}
		doc, err := transcriptsrepo.UnrevokeDocument(r.Context(), d.Pool, docID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to unrevoke document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": documentToJSON(doc)})
	}
}

type disclosureBody struct {
	DisclosePublicly *bool `json:"disclosePublicly"`
}

// PATCH /api/v1/transcripts/documents/{id}/disclosure — holder-controlled verify disclosure (T08).
func (d Deps) handlePatchTranscriptDocumentDisclosure() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		docID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid document id.")
			return
		}
		var body disclosureBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DisclosePublicly == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "disclosePublicly is required.")
			return
		}
		doc, err := transcriptsrepo.SetDocumentDisclosePublicly(r.Context(), d.Pool, userID, docID, *body.DisclosePublicly)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update disclosure.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"document": documentToJSON(doc)})
	}
}
