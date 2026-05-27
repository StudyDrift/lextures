package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	reposoc2 "github.com/lextures/lextures/server/internal/repos/soc2"
	soc2service "github.com/lextures/lextures/server/internal/service/soc2"
)

func (d Deps) soc2Enabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().SOC2ModuleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "SOC 2 module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireSOC2Admin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := soc2service.CheckAdmin(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerSOC2Routes(r chi.Router) {
	r.Get("/api/v1/internal/compliance/soc2/evidence-summary", d.handleGetSOC2EvidenceSummary())
	r.Get("/api/v1/internal/compliance/soc2/access-reviews", d.handleGetSOC2AccessReviews())
	r.Post("/api/v1/internal/compliance/soc2/access-reviews", d.handlePostSOC2AccessReview())
	r.Post("/api/v1/internal/compliance/soc2/incidents", d.handlePostSOC2Incident())
	r.Get("/api/v1/internal/compliance/soc2/incidents", d.handleGetSOC2Incidents())
	r.Get("/api/v1/internal/compliance/soc2/incidents/{id}", d.handleGetSOC2Incident())
	r.Patch("/api/v1/internal/compliance/soc2/incidents/{id}", d.handlePatchSOC2Incident())
	r.Get("/api/v1/internal/compliance/soc2/vendor-risk", d.handleGetSOC2Vendors())
	r.Post("/api/v1/internal/compliance/soc2/vendor-risk", d.handlePostSOC2Vendor())
}

// GET /api/v1/internal/compliance/soc2/evidence-summary
func (d Deps) handleGetSOC2EvidenceSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		summary, err := soc2service.GetEvidenceSummary(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load evidence summary.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

// GET /api/v1/internal/compliance/soc2/access-reviews
func (d Deps) handleGetSOC2AccessReviews() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		reviews, err := soc2service.ListAccessReviews(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load access reviews.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reviews": accessReviewsToJSON(reviews)})
	}
}

type postAccessReviewBody struct {
	ReviewType    string  `json:"reviewType"`
	Findings      *string `json:"findings"`
	NextReviewDue *string `json:"nextReviewDue"` // RFC3339
}

// POST /api/v1/internal/compliance/soc2/access-reviews
func (d Deps) handlePostSOC2AccessReview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		reviewerID, ok := d.requireSOC2Admin(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postAccessReviewBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validTypes := map[string]bool{"privileged": true, "all_production": true, "third_party": true}
		if !validTypes[body.ReviewType] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "reviewType must be privileged, all_production, or third_party.")
			return
		}
		var nextReviewDue *time.Time
		if body.NextReviewDue != nil {
			t, err := time.Parse(time.RFC3339, *body.NextReviewDue)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "nextReviewDue must be RFC3339.")
				return
			}
			nextReviewDue = &t
		}
		id, err := soc2service.CreateAccessReview(r.Context(), d.Pool, reviewerID, body.ReviewType, body.Findings, nextReviewDue)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create access review.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

type postIncidentBody struct {
	Title       string   `json:"title"`
	Severity    string   `json:"severity"`
	TSCCriteria []string `json:"tscCriteria"`
}

// POST /api/v1/internal/compliance/soc2/incidents
func (d Deps) handlePostSOC2Incident() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postIncidentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
			return
		}
		validSeverities := map[string]bool{"P0": true, "P1": true, "P2": true, "P3": true}
		if !validSeverities[body.Severity] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "severity must be P0, P1, P2, or P3.")
			return
		}
		id, err := soc2service.OpenIncident(r.Context(), d.Pool, body.Title, body.Severity, body.TSCCriteria)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not log incident.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/internal/compliance/soc2/incidents
func (d Deps) handleGetSOC2Incidents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		status := r.URL.Query().Get("status")
		incidents, err := soc2service.ListIncidents(r.Context(), d.Pool, status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load incidents.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"incidents": incidentsToJSON(incidents)})
	}
}

// GET /api/v1/internal/compliance/soc2/incidents/{id}
func (d Deps) handleGetSOC2Incident() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid incident id.")
			return
		}
		inc, err := soc2service.GetIncident(r.Context(), d.Pool, id)
		if err != nil {
			if errors.Is(err, soc2service.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Incident not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load incident.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(incidentToJSON(*inc))
	}
}

type patchIncidentBody struct {
	Status        string  `json:"status"`
	ResolvedAt    *string `json:"resolvedAt"`    // RFC3339, optional
	PostMortemURL *string `json:"postMortemUrl"` // optional
}

// PATCH /api/v1/internal/compliance/soc2/incidents/{id}
func (d Deps) handlePatchSOC2Incident() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid incident id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchIncidentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validStatuses := map[string]bool{"open": true, "contained": true, "resolved": true, "closed": true}
		if !validStatuses[body.Status] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be open, contained, resolved, or closed.")
			return
		}
		var resolvedAt *time.Time
		if body.ResolvedAt != nil {
			t, err := time.Parse(time.RFC3339, *body.ResolvedAt)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "resolvedAt must be RFC3339.")
				return
			}
			resolvedAt = &t
		}
		if err := soc2service.UpdateIncidentStatus(r.Context(), d.Pool, id, body.Status, resolvedAt, body.PostMortemURL); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update incident.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/internal/compliance/soc2/vendor-risk
func (d Deps) handleGetSOC2Vendors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		vendors, err := soc2service.ListVendors(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load vendor risk register.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"vendors": vendorsToJSON(vendors)})
	}
}

type postVendorBody struct {
	VendorName    string  `json:"vendorName"`
	RiskTier      string  `json:"riskTier"`
	SOC2ReportURL *string `json:"soc2ReportUrl"`
	ReportDate    *string `json:"reportDate"`    // RFC3339 date
	NextReviewDue *string `json:"nextReviewDue"` // RFC3339 date
	Notes         *string `json:"notes"`
}

// POST /api/v1/internal/compliance/soc2/vendor-risk
func (d Deps) handlePostSOC2Vendor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.soc2Enabled(w) {
			return
		}
		if _, ok := d.requireSOC2Admin(w, r); !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postVendorBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.VendorName == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "vendorName is required.")
			return
		}
		validTiers := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
		if !validTiers[body.RiskTier] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "riskTier must be critical, high, medium, or low.")
			return
		}
		var reportDate *time.Time
		if body.ReportDate != nil {
			t, err := time.Parse("2006-01-02", *body.ReportDate)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "reportDate must be YYYY-MM-DD.")
				return
			}
			reportDate = &t
		}
		var nextReviewDue *time.Time
		if body.NextReviewDue != nil {
			t, err := time.Parse("2006-01-02", *body.NextReviewDue)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "nextReviewDue must be YYYY-MM-DD.")
				return
			}
			nextReviewDue = &t
		}
		id, err := soc2service.UpsertVendor(r.Context(), d.Pool, body.VendorName, body.RiskTier, body.SOC2ReportURL, reportDate, nextReviewDue, body.Notes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save vendor.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func accessReviewToJSON(r reposoc2.AccessReview) map[string]any {
	m := map[string]any{
		"id":         r.ID.String(),
		"reviewerId": r.ReviewerID.String(),
		"reviewType": r.ReviewType,
		"reviewedAt": r.ReviewedAt.UTC().Format(time.RFC3339),
	}
	if r.Findings != nil {
		m["findings"] = *r.Findings
	}
	if r.NextReviewDue != nil {
		m["nextReviewDue"] = r.NextReviewDue.UTC().Format(time.RFC3339)
	}
	return m
}

func accessReviewsToJSON(reviews []reposoc2.AccessReview) []map[string]any {
	out := make([]map[string]any, 0, len(reviews))
	for _, r := range reviews {
		out = append(out, accessReviewToJSON(r))
	}
	return out
}

func incidentToJSON(inc reposoc2.Incident) map[string]any {
	m := map[string]any{
		"id":          inc.ID.String(),
		"title":       inc.Title,
		"severity":    inc.Severity,
		"status":      inc.Status,
		"openedAt":    inc.OpenedAt.UTC().Format(time.RFC3339),
		"tscCriteria": inc.TSCCriteria,
	}
	if inc.ResolvedAt != nil {
		m["resolvedAt"] = inc.ResolvedAt.UTC().Format(time.RFC3339)
	}
	if inc.PostMortemURL != nil {
		m["postMortemUrl"] = *inc.PostMortemURL
	}
	return m
}

func incidentsToJSON(incidents []reposoc2.Incident) []map[string]any {
	out := make([]map[string]any, 0, len(incidents))
	for _, inc := range incidents {
		out = append(out, incidentToJSON(inc))
	}
	return out
}

func vendorToJSON(v reposoc2.VendorRisk) map[string]any {
	m := map[string]any{
		"id":         v.ID.String(),
		"vendorName": v.VendorName,
		"riskTier":   v.RiskTier,
	}
	if v.SOC2ReportURL != nil {
		m["soc2ReportUrl"] = *v.SOC2ReportURL
	}
	if v.ReportDate != nil {
		m["reportDate"] = v.ReportDate.UTC().Format("2006-01-02")
	}
	if v.NextReviewDue != nil {
		m["nextReviewDue"] = v.NextReviewDue.UTC().Format("2006-01-02")
	}
	if v.Notes != nil {
		m["notes"] = *v.Notes
	}
	return m
}

func vendorsToJSON(vendors []reposoc2.VendorRisk) []map[string]any {
	out := make([]map[string]any, 0, len(vendors))
	for _, v := range vendors {
		out = append(out, vendorToJSON(v))
	}
	return out
}
