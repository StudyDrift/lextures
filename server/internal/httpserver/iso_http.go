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
	repoiso "github.com/lextures/lextures/server/internal/repos/iso"
	isoservice "github.com/lextures/lextures/server/internal/service/iso"
)

func (d Deps) isoIsmsEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().IsoIsmsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "ISO ISMS module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireISOAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := isoservice.CheckAdmin(r.Context(), d.Pool, uid)
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

func (d Deps) registerISORoutes(r chi.Router) {
	r.Get("/api/v1/trust/iso", d.handleGetTrustISO())
	r.Get("/api/v1/compliance/iso/dashboard", d.handleGetISODashboard())
	r.Get("/api/v1/compliance/iso/audit-findings", d.handleListISOAuditFindings())
	r.Post("/api/v1/compliance/iso/audit-findings", d.handlePostISOAuditFinding())
	r.Patch("/api/v1/compliance/iso/audit-findings/{id}", d.handlePatchISOAuditFinding())
	r.Get("/api/v1/compliance/iso/risk-register", d.handleListISORisks())
	r.Post("/api/v1/compliance/iso/risk-register", d.handlePostISORisk())
	r.Get("/api/v1/compliance/iso/supplier-reviews", d.handleListISOSuppliers())
	r.Post("/api/v1/compliance/iso/supplier-reviews", d.handlePostISOSupplier())
	r.Get("/api/v1/compliance/iso/training", d.handleListISOTraining())
	r.Post("/api/v1/compliance/iso/training", d.handlePostISOTraining())
	r.Get("/api/v1/compliance/iso/soa", d.handleListISOSoA())
	r.Patch("/api/v1/compliance/iso/soa/{controlId}", d.handlePatchISOSoA())
	r.Patch("/api/v1/compliance/iso/program", d.handlePatchISOProgram())
}

// GET /api/v1/trust/iso — public ISMS program summary for trust center (plan 10.10).
func (d Deps) handleGetTrustISO() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Service unavailable.")
			return
		}
		program, soa, err := isoservice.GetTrustProgram(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load ISMS program status.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(trustISOJSON(program, soa))
	}
}

func trustISOJSON(program repoiso.ProgramStatus, soa repoiso.SoASummary) map[string]any {
	out := map[string]any{
		"scopeStatement":    program.ScopeStatement,
		"iso27001Status":    program.ISO27001Status,
		"iso27701Status":    program.ISO27701Status,
		"soa": map[string]any{
			"total":       soa.Total,
			"implemented": soa.Implemented,
			"planned":     soa.Planned,
			"excluded":    soa.Excluded,
		},
	}
	if program.ISO27001CertURL != nil {
		out["iso27001CertUrl"] = *program.ISO27001CertURL
	}
	if program.ISO27001LastAudit != nil {
		out["iso27001LastAudit"] = program.ISO27001LastAudit.Format("2006-01-02")
	}
	if program.SoALastReview != nil {
		out["soaLastReview"] = program.SoALastReview.Format("2006-01-02")
	}
	return out
}

func (d Deps) handleGetISODashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		dash, err := isoservice.GetDashboard(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load dashboard.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"program":          trustISOJSON(dash.Program, dash.SoA),
			"openFindings":     dash.OpenFindings,
			"highRisks":        dash.HighRisks,
			"pendingSuppliers": dash.PendingSuppliers,
			"trainingYear":     dash.TrainingYear,
			"trainingCount":    dash.TrainingCount,
		})
	}
}

func (d Deps) handleListISOAuditFindings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		items, err := isoservice.ListAuditFindings(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list findings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"findings": auditFindingsToJSON(items)})
	}
}

type postAuditFindingBody struct {
	AuditCycle       string  `json:"auditCycle"`
	FindingType      string  `json:"findingType"`
	ISOClause        string  `json:"isoClause"`
	Description      string  `json:"description"`
	CorrectiveAction *string `json:"correctiveAction"`
	DueDate          *string `json:"dueDate"`
}

func (d Deps) handlePostISOAuditFinding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		var body postAuditFindingBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.AuditCycle == "" || body.FindingType == "" || body.ISOClause == "" || body.Description == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "auditCycle, findingType, isoClause, and description are required.")
			return
		}
		due, err := parseOptionalDate(body.DueDate)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dueDate.")
			return
		}
		id, err := isoservice.CreateAuditFinding(r.Context(), d.Pool, body.AuditCycle, body.FindingType, body.ISOClause, body.Description, body.CorrectiveAction, due)
		if err != nil {
			if errors.Is(err, isoservice.ErrInvalidInput) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid finding fields.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create finding.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

type patchAuditFindingBody struct {
	Status           string  `json:"status"`
	CorrectiveAction *string `json:"correctiveAction"`
	DueDate          *string `json:"dueDate"`
}

func (d Deps) handlePatchISOAuditFinding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid id.")
			return
		}
		var body patchAuditFindingBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		due, err := parseOptionalDate(body.DueDate)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dueDate.")
			return
		}
		if err := isoservice.PatchAuditFinding(r.Context(), d.Pool, id, body.Status, body.CorrectiveAction, due, body.Status == "closed"); err != nil {
			if errors.Is(err, isoservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Finding not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid patch.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleListISORisks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		items, err := isoservice.ListRiskEntries(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list risks.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"risks": risksToJSON(items)})
	}
}

type postRiskBody struct {
	RiskTitle  string  `json:"riskTitle"`
	Likelihood int     `json:"likelihood"`
	Impact     int     `json:"impact"`
	Treatment  string  `json:"treatment"`
	OwnerID    *string `json:"ownerId"`
	ReviewDate *string `json:"reviewDate"`
}

func (d Deps) handlePostISORisk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		var body postRiskBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var ownerID *uuid.UUID
		if body.OwnerID != nil && *body.OwnerID != "" {
			oid, err := uuid.Parse(*body.OwnerID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid ownerId.")
				return
			}
			ownerID = &oid
		}
		review, err := parseOptionalDate(body.ReviewDate)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid reviewDate.")
			return
		}
		id, err := isoservice.CreateRiskEntry(r.Context(), d.Pool, body.RiskTitle, body.Likelihood, body.Impact, body.Treatment, ownerID, review)
		if err != nil {
			if errors.Is(err, isoservice.ErrInvalidInput) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid risk fields.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create risk.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleListISOSuppliers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		items, err := isoservice.ListSupplierReviews(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list suppliers.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"suppliers": suppliersToJSON(items)})
	}
}

type postSupplierBody struct {
	VendorName      string  `json:"vendorName"`
	ReviewStatus    string  `json:"reviewStatus"`
	CertificateType *string `json:"certificateType"`
	CertificateURL  *string `json:"certificateUrl"`
	Notes           *string `json:"notes"`
	NextReviewDue   *string `json:"nextReviewDue"`
}

func (d Deps) handlePostISOSupplier() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		var body postSupplierBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		now := time.Now().UTC()
		var reviewedAt *time.Time
		if body.ReviewStatus == "approved" {
			reviewedAt = &now
		}
		nextDue, err := parseOptionalDate(body.NextReviewDue)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid nextReviewDue.")
			return
		}
		id, err := isoservice.UpsertSupplierReview(r.Context(), d.Pool, body.VendorName, body.ReviewStatus, body.CertificateType, body.CertificateURL, body.Notes, reviewedAt, nextDue)
		if err != nil {
			if errors.Is(err, isoservice.ErrInvalidInput) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid supplier fields.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save supplier review.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleListISOTraining() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		year := time.Now().UTC().Year()
		items, err := isoservice.ListTraining(r.Context(), d.Pool, year)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list training.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"year": year, "completions": trainingToJSON(items)})
	}
}

type postTrainingBody struct {
	UserID string `json:"userId"`
	Year   int    `json:"year"`
}

func (d Deps) handlePostISOTraining() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		adminID, ok := d.requireISOAdmin(w, r)
		if !ok {
			return
		}
		var body postTrainingBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		userID := adminID
		if body.UserID != "" {
			uid, err := uuid.Parse(body.UserID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userId.")
				return
			}
			userID = uid
		}
		year := body.Year
		if year == 0 {
			year = time.Now().UTC().Year()
		}
		id, err := isoservice.RecordTraining(r.Context(), d.Pool, userID, year)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid training record.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func (d Deps) handleListISOSoA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		controls, err := isoservice.ListSoAControls(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list SoA controls.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"controls": soaToJSON(controls)})
	}
}

type patchSoABody struct {
	Status                 string  `json:"status"`
	ExclusionJustification *string `json:"exclusionJustification"`
}

func (d Deps) handlePatchISOSoA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		controlID := chi.URLParam(r, "controlId")
		var body patchSoABody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := isoservice.PatchSoAControl(r.Context(), d.Pool, controlID, body.Status, body.ExclusionJustification); err != nil {
			if errors.Is(err, isoservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Control not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type patchProgramBody struct {
	ScopeStatement    string  `json:"scopeStatement"`
	ISO27001Status    string  `json:"iso27001Status"`
	ISO27701Status    string  `json:"iso27701Status"`
	ISO27001CertURL   *string `json:"iso27001CertUrl"`
	ISO27001LastAudit *string `json:"iso27001LastAudit"`
	SoALastReview     *string `json:"soaLastReview"`
}

func (d Deps) handlePatchISOProgram() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.isoIsmsEnabled(w) {
			return
		}
		if _, ok := d.requireISOAdmin(w, r); !ok {
			return
		}
		var body patchProgramBody
		if err := decodeJSONBody(r, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		lastAudit, err := parseOptionalDate(body.ISO27001LastAudit)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid iso27001LastAudit.")
			return
		}
		soaReview, err := parseOptionalDate(body.SoALastReview)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid soaLastReview.")
			return
		}
		if err := isoservice.UpdateProgramStatus(r.Context(), d.Pool, body.ScopeStatement, body.ISO27001Status, body.ISO27701Status, body.ISO27001CertURL, lastAudit, soaReview); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid program update.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func decodeJSONBody(r *http.Request, dest any) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	return json.Unmarshal(b, dest)
}

func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func auditFindingsToJSON(items []repoiso.AuditFinding) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, f := range items {
		row := map[string]any{
			"id": f.ID.String(), "auditCycle": f.AuditCycle, "findingType": f.FindingType,
			"isoClause": f.ISOClause, "description": f.Description, "status": f.Status,
			"createdAt": f.CreatedAt.Format(time.RFC3339),
		}
		if f.CorrectiveAction != nil {
			row["correctiveAction"] = *f.CorrectiveAction
		}
		if f.DueDate != nil {
			row["dueDate"] = f.DueDate.Format("2006-01-02")
		}
		if f.ClosedAt != nil {
			row["closedAt"] = f.ClosedAt.Format(time.RFC3339)
		}
		out = append(out, row)
	}
	return out
}

func risksToJSON(items []repoiso.RiskEntry) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, r := range items {
		row := map[string]any{
			"id": r.ID.String(), "riskTitle": r.RiskTitle, "likelihood": r.Likelihood,
			"impact": r.Impact, "treatment": r.Treatment, "residualScore": r.ResidualScore,
			"createdAt": r.CreatedAt.Format(time.RFC3339),
		}
		if r.OwnerID != nil {
			row["ownerId"] = r.OwnerID.String()
		}
		if r.ReviewDate != nil {
			row["reviewDate"] = r.ReviewDate.Format("2006-01-02")
		}
		out = append(out, row)
	}
	return out
}

func suppliersToJSON(items []repoiso.SupplierReview) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, s := range items {
		row := map[string]any{
			"id": s.ID.String(), "vendorName": s.VendorName, "reviewStatus": s.ReviewStatus,
			"createdAt": s.CreatedAt.Format(time.RFC3339),
		}
		if s.CertificateType != nil {
			row["certificateType"] = *s.CertificateType
		}
		if s.CertificateURL != nil {
			row["certificateUrl"] = *s.CertificateURL
		}
		if s.ReviewedAt != nil {
			row["reviewedAt"] = s.ReviewedAt.Format(time.RFC3339)
		}
		if s.NextReviewDue != nil {
			row["nextReviewDue"] = s.NextReviewDue.Format("2006-01-02")
		}
		if s.Notes != nil {
			row["notes"] = *s.Notes
		}
		out = append(out, row)
	}
	return out
}

func trainingToJSON(items []repoiso.TrainingCompletion) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, t := range items {
		out = append(out, map[string]any{
			"id": t.ID.String(), "userId": t.UserID.String(),
			"trainingYear": t.TrainingYear, "completedAt": t.CompletedAt.Format(time.RFC3339),
		})
	}
	return out
}

func soaToJSON(items []repoiso.SoAControlRow) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, c := range items {
		row := map[string]any{
			"controlId": c.ControlID, "theme": c.Theme, "title": c.Title,
			"status": c.Status, "updatedAt": c.UpdatedAt.Format(time.RFC3339),
		}
		if c.ExclusionJustification != nil {
			row["exclusionJustification"] = *c.ExclusionJustification
		}
		out = append(out, row)
	}
	return out
}
