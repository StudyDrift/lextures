package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
	"github.com/lextures/lextures/server/internal/service/transcriptissue"
)

func (d Deps) registerTranscriptDocumentRoutes(r chi.Router) {
	r.Post("/api/v1/transcripts/documents", d.handlePostTranscriptDocument())
	r.Get("/api/v1/transcripts/documents", d.handleListTranscriptDocuments())
	r.Get("/api/v1/transcripts/documents/{id}", d.handleGetTranscriptDocument())
	r.Get("/api/v1/transcripts/documents/{id}/download", d.handleDownloadTranscriptDocument())
	r.Get("/api/v1/transcripts/preview", d.handleTranscriptPreview())
	r.Get("/api/v1/admin/transcripts/students/{uid}/documents", d.handleAdminListStudentTranscriptDocuments())
	r.Post("/api/v1/admin/transcripts/students/{uid}/documents", d.handleAdminGenerateStudentTranscript())
}

type transcriptDocumentJSON struct {
	ID              string   `json:"id"`
	Variant         string   `json:"variant"`
	Version         int      `json:"version"`
	SchemaVersion   string   `json:"schemaVersion"`
	TemplateVersion string   `json:"templateVersion"`
	ContentHash     string   `json:"contentHash"`
	GPACumulative   *float64 `json:"gpaCumulative,omitempty"`
	CreditsEarned   *float64 `json:"creditsEarned,omitempty"`
	GeneratedAt     string   `json:"generatedAt"`
	HasPDF          bool     `json:"hasPdf"`
	HasXML          bool     `json:"hasXml"`
}

func documentToJSON(doc *transcriptsrepo.Document) transcriptDocumentJSON {
	return transcriptDocumentJSON{
		ID:              doc.ID.String(),
		Variant:         string(doc.Variant),
		Version:         doc.Version,
		SchemaVersion:   doc.SchemaVersion,
		TemplateVersion: doc.TemplateVersion,
		ContentHash:     doc.ContentHash,
		GPACumulative:   doc.GPACumulative,
		CreditsEarned:   doc.CreditsEarned,
		GeneratedAt:     doc.GeneratedAt.UTC().Format(time.RFC3339),
		HasPDF:          len(doc.PDFBytes) > 0 || (doc.PDFKey != nil && *doc.PDFKey != ""),
		HasXML:          len(doc.PESCXMLBytes) > 0 || (doc.PESCXMLKey != nil && *doc.PESCXMLKey != ""),
	}
}

type postTranscriptDocumentBody struct {
	Variant string   `json:"variant"`
	Terms   []string `json:"terms"`
	Format  []string `json:"format"`
}

func parseGenerateFormats(formats []string) transcriptissue.GenerateFormats {
	out := transcriptissue.GenerateFormats{}
	if len(formats) == 0 {
		out.PDF = true
		out.XML = true
		return out
	}
	for _, f := range formats {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "pdf":
			out.PDF = true
		case "xml", "pesc":
			out.XML = true
		}
	}
	if !out.PDF && !out.XML {
		out.PDF = true
		out.XML = true
	}
	return out
}

func parseVariant(s string) (academicrecord.Variant, string) {
	v := academicrecord.Variant(strings.ToLower(strings.TrimSpace(s)))
	switch v {
	case academicrecord.VariantOfficial, academicrecord.VariantUnofficial,
		academicrecord.VariantPartial, academicrecord.VariantInProgress:
		return v, ""
	case "":
		return academicrecord.VariantUnofficial, ""
	default:
		return "", "variant must be official, unofficial, partial, or in_progress."
	}
}

func parseTermIDs(raw []string) ([]uuid.UUID, string) {
	if len(raw) == 0 {
		return nil, ""
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, "terms must be UUID strings."
		}
		out = append(out, id)
	}
	return out, ""
}

func (d Deps) officialGenerationOff(w http.ResponseWriter, r *http.Request, variant academicrecord.Variant) bool {
	if variant != academicrecord.VariantOfficial {
		return false
	}
	cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
	if err != nil || cfg == nil || !cfg.OfficialEnabled {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Official transcript generation is not enabled.")
		return true
	}
	return false
}

// POST /api/v1/transcripts/documents
func (d Deps) handlePostTranscriptDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postTranscriptDocumentBody
		if len(b) > 0 {
			if err := json.Unmarshal(b, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		variant, msg := parseVariant(body.Variant)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		if variant == academicrecord.VariantUnofficial {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Use GET /api/v1/transcripts/preview for unofficial previews.")
			return
		}
		if d.officialGenerationOff(w, r, variant) {
			return
		}
		termIDs, msg := parseTermIDs(body.Terms)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		result, err := transcriptissue.Generate(r.Context(), d.Pool, transcriptissue.GenerateParams{
			UserID:      userID,
			GeneratedBy: userID,
			Variant:     variant,
			TermIDs:     termIDs,
			Formats:     parseGenerateFormats(body.Format),
			Persist:     true,
			GeneratedAt: time.Now().UTC(),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate transcript.")
			return
		}
		if result.Document == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to persist transcript.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": documentToJSON(result.Document),
			"record":   result.Record,
		})
	}
}

// GET /api/v1/transcripts/documents
func (d Deps) handleListTranscriptDocuments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		list, err := transcriptsrepo.ListDocumentsByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load documents.")
			return
		}
		out := make([]transcriptDocumentJSON, 0, len(list))
		for i := range list {
			out = append(out, documentToJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"documents": out})
	}
}

// GET /api/v1/transcripts/documents/{id}
func (d Deps) handleGetTranscriptDocument() http.HandlerFunc {
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
		doc, err := transcriptsrepo.GetDocumentByID(r.Context(), d.Pool, userID, docID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}
		if !transcriptsrepo.VerifyDocumentHash(doc) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInternal, "Document integrity check failed.")
			return
		}
		var record academicrecord.AcademicRecord
		_ = json.Unmarshal(doc.Canonical, &record)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": documentToJSON(doc),
			"record":   record,
		})
	}
}

// GET /api/v1/transcripts/documents/{id}/download?format=pdf|xml
func (d Deps) handleDownloadTranscriptDocument() http.HandlerFunc {
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
		doc, err := transcriptsrepo.GetDocumentByID(r.Context(), d.Pool, userID, docID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}
		if !transcriptsrepo.VerifyDocumentHash(doc) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInternal, "Document integrity check failed.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "pdf"
		}
		switch format {
		case "pdf":
			if len(doc.PDFBytes) == 0 {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "PDF not available.")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="transcript.pdf"`)
			_, _ = w.Write(doc.PDFBytes)
		case "xml", "pesc":
			if len(doc.PESCXMLBytes) == 0 {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "XML not available.")
				return
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="transcript.xml"`)
			_, _ = w.Write(doc.PESCXMLBytes)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be pdf or xml.")
		}
	}
}

// GET /api/v1/transcripts/preview — unofficial watermarked preview (no persistence).
func (d Deps) handleTranscriptPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		formats := transcriptissue.GenerateFormats{PDF: true, XML: false}
		if format == "xml" || format == "pesc" {
			formats = transcriptissue.GenerateFormats{PDF: false, XML: true}
		}
		if format == "json" || format == "" {
			formats = transcriptissue.GenerateFormats{PDF: true, XML: true}
		}
		result, err := transcriptissue.Generate(r.Context(), d.Pool, transcriptissue.GenerateParams{
			UserID:      userID,
			GeneratedBy: userID,
			Variant:     academicrecord.VariantUnofficial,
			Formats:     formats,
			Persist:     false,
			GeneratedAt: time.Now().UTC(),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build preview.")
			return
		}
		if format == "pdf" {
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `inline; filename="transcript-unofficial.pdf"`)
			_, _ = w.Write(result.PDF)
			return
		}
		if format == "xml" || format == "pesc" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			_, _ = w.Write(result.PESCXML)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"record":      result.Record,
			"contentHash": result.Hash,
			"variant":     "unofficial",
			"persisted":   false,
		})
	}
}

// GET /api/v1/admin/transcripts/students/{uid}/documents
func (d Deps) handleAdminListStudentTranscriptDocuments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		uid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		list, err := transcriptsrepo.ListDocumentsByStudentAdmin(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load documents.")
			return
		}
		out := make([]transcriptDocumentJSON, 0, len(list))
		for i := range list {
			out = append(out, documentToJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"documents": out})
	}
}

// POST /api/v1/admin/transcripts/students/{uid}/documents — registrar generate/reissue.
func (d Deps) handleAdminGenerateStudentTranscript() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		uid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postTranscriptDocumentBody
		if len(b) > 0 {
			if err := json.Unmarshal(b, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		variant, msg := parseVariant(body.Variant)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		if variant == "" || variant == academicrecord.VariantUnofficial {
			variant = academicrecord.VariantOfficial
		}
		if d.officialGenerationOff(w, r, variant) {
			return
		}
		termIDs, msg := parseTermIDs(body.Terms)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		result, err := transcriptissue.Generate(r.Context(), d.Pool, transcriptissue.GenerateParams{
			UserID:      uid,
			GeneratedBy: adminID,
			Variant:     variant,
			TermIDs:     termIDs,
			Formats:     parseGenerateFormats(body.Format),
			Persist:     true,
			GeneratedAt: time.Now().UTC(),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate transcript.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": documentToJSON(result.Document),
			"record":   result.Record,
		})
	}
}
