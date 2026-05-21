package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/legalack"
)

// Current legal document versions — keep in sync with clients/web/src/lib/legal-documents.ts.
var currentLegalVersions = map[string]struct {
	Version       string
	EffectiveDate string
}{
	legalack.DocumentPrivacyPolicy: {
		Version:       "2026-05-21",
		EffectiveDate: "2026-05-21",
	},
	legalack.DocumentTermsOfService: {
		Version:       "2026-05-21",
		EffectiveDate: "2026-05-21",
	},
}

type legalPendingResponse struct {
	Documents []legalPendingDoc `json:"documents"`
}

type legalPendingDoc struct {
	Document      string `json:"document"`
	Version       string `json:"version"`
	EffectiveDate string `json:"effectiveDate"`
}

type legalAcknowledgeBody struct {
	Document string `json:"document"`
	Version  string `json:"version"`
}

// GET /api/v1/legal/pending
func (d Deps) handleLegalPending() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pending, err := legalack.Pending(r.Context(), d.Pool, userID, currentLegalVersions)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load pending legal documents.")
			return
		}
		docs := make([]legalPendingDoc, 0, len(pending))
		for _, p := range pending {
			docs = append(docs, legalPendingDoc{
				Document:      p.Document,
				Version:       p.Version,
				EffectiveDate: p.EffectiveDate,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(legalPendingResponse{Documents: docs})
	}
}

// POST /api/v1/legal/acknowledge
func (d Deps) handleLegalAcknowledge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body legalAcknowledgeBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		doc := strings.TrimSpace(body.Document)
		ver := strings.TrimSpace(body.Version)
		cur, known := currentLegalVersions[doc]
		if !known {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown document.")
			return
		}
		if ver != cur.Version {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Version does not match the current published document.")
			return
		}
		if err := legalack.RecordAck(r.Context(), d.Pool, userID, doc, ver); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record acknowledgement.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) registerLegalRoutes(r chi.Router) {
	r.Get("/api/v1/legal/pending", d.handleLegalPending())
	r.Post("/api/v1/legal/acknowledge", d.handleLegalAcknowledge())
}
