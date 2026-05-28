package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	reporeports "github.com/lextures/lextures/server/internal/repos/securityreports"
	reportservice "github.com/lextures/lextures/server/internal/service/securityreports"
)

func (d Deps) securityDisclosureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().SecurityDisclosureModuleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Security disclosure module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireSecurityAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := reportservice.CheckAdmin(r.Context(), d.Pool, uid)
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

func (d Deps) registerSecurityReportsRoutes(r chi.Router) {
	r.Get("/api/v1/trust/security", d.handleGetTrustSecurity())
	r.Get("/api/v1/compliance/security-reports", d.handleGetSecurityReports())
	r.Post("/api/v1/compliance/security-reports", d.handlePostSecurityReport())
	r.Get("/api/v1/compliance/security-reports/export", d.handleGetSecurityReportsExport())
	r.Get("/api/v1/compliance/security-reports/{id}", d.handleGetSecurityReport())
	r.Patch("/api/v1/compliance/security-reports/{id}", d.handlePatchSecurityReport())
}

// GET /api/v1/trust/security — public responsible-disclosure policy metadata (AC-5).
func (d Deps) handleGetTrustSecurity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy := reportservice.DefaultTrustPolicy()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(policy)
	}
}

// GET /api/v1/compliance/security-reports
func (d Deps) handleGetSecurityReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.securityDisclosureEnabled(w) {
			return
		}
		if _, ok := d.requireSecurityAdmin(w, r); !ok {
			return
		}
		reports, err := reportservice.ListReports(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load security reports.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reports": securityReportsToJSON(reports)})
	}
}

type postSecurityReportBody struct {
	ReporterHandle *string  `json:"reporterHandle"`
	ReportDate     string   `json:"reportDate"` // YYYY-MM-DD
	CVSSScore      *float64 `json:"cvssScore"`
	Severity       *string  `json:"severity"`
	Summary        string   `json:"summary"`
}

// POST /api/v1/compliance/security-reports
func (d Deps) handlePostSecurityReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.securityDisclosureEnabled(w) {
			return
		}
		if _, ok := d.requireSecurityAdmin(w, r); !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postSecurityReportBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Summary == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "summary is required.")
			return
		}
		reportDate := time.Now().UTC().Truncate(24 * time.Hour)
		if body.ReportDate != "" {
			t, err := time.Parse("2006-01-02", body.ReportDate)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "reportDate must be YYYY-MM-DD.")
				return
			}
			reportDate = t
		}
		if body.Severity != nil && !validSecuritySeverity(*body.Severity) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "severity must be critical, high, medium, low, or informational.")
			return
		}
		id, err := reportservice.CreateReport(r.Context(), d.Pool, body.ReporterHandle, reportDate, body.CVSSScore, body.Severity, body.Summary)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create security report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/security-reports/{id}
func (d Deps) handleGetSecurityReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.securityDisclosureEnabled(w) {
			return
		}
		if _, ok := d.requireSecurityAdmin(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid report id.")
			return
		}
		report, err := reportservice.GetReport(r.Context(), d.Pool, id)
		if err != nil {
			if errors.Is(err, reportservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load security report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(securityReportToJSON(*report))
	}
}

type patchSecurityReportBody struct {
	Status     string   `json:"status"`
	Severity   *string  `json:"severity"`
	CVSSScore  *float64 `json:"cvssScore"`
	TriagedAt  *string  `json:"triagedAt"`  // RFC3339
	PatchDate  *string  `json:"patchDate"`  // YYYY-MM-DD
	BountyPaid *bool    `json:"bountyPaid"`
}

// PATCH /api/v1/compliance/security-reports/{id}
func (d Deps) handlePatchSecurityReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.securityDisclosureEnabled(w) {
			return
		}
		if _, ok := d.requireSecurityAdmin(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid report id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchSecurityReportBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if !validSecurityStatus(body.Status) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be triaging, accepted, patched, disputed, or wont_fix.")
			return
		}
		if body.Severity != nil && !validSecuritySeverity(*body.Severity) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "invalid severity.")
			return
		}
		var triagedAt *time.Time
		if body.TriagedAt != nil {
			t, err := time.Parse(time.RFC3339, *body.TriagedAt)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "triagedAt must be RFC3339.")
				return
			}
			triagedAt = &t
		}
		var patchDate *time.Time
		if body.PatchDate != nil {
			t, err := time.Parse("2006-01-02", *body.PatchDate)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "patchDate must be YYYY-MM-DD.")
				return
			}
			patchDate = &t
		}
		if err := reportservice.UpdateReport(r.Context(), d.Pool, id, reportservice.UpdateReportInput{
			Status:     body.Status,
			Severity:   body.Severity,
			CVSSScore:  body.CVSSScore,
			TriagedAt:  triagedAt,
			PatchDate:  patchDate,
			BountyPaid: body.BountyPaid,
		}); err != nil {
			if errors.Is(err, reportservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Report not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update security report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/security-reports/export — CSV export for auditors (AC-4).
func (d Deps) handleGetSecurityReportsExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.securityDisclosureEnabled(w) {
			return
		}
		if _, ok := d.requireSecurityAdmin(w, r); !ok {
			return
		}
		reports, err := reportservice.ListReports(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not export security reports.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="security_reports.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "reporter_handle", "report_date", "triaged_at", "cvss_score", "severity", "summary", "status", "patch_date", "sla_met", "bounty_paid", "created_at"})
		for _, rep := range reports {
			row := []string{
				rep.ID.String(),
				strPtrOrEmpty(rep.ReporterHandle),
				rep.ReportDate.Format("2006-01-02"),
				timePtrRFC3339(rep.TriagedAt),
				floatPtrStr(rep.CVSSScore),
				strPtrOrEmpty(rep.Severity),
				rep.Summary,
				rep.Status,
				datePtrStr(rep.PatchDate),
				boolPtrStr(rep.SLAMet),
				strconv.FormatBool(rep.BountyPaid),
				rep.CreatedAt.UTC().Format(time.RFC3339),
			}
			_ = cw.Write(row)
		}
		cw.Flush()
	}
}

func validSecuritySeverity(s string) bool {
	switch s {
	case "critical", "high", "medium", "low", "informational":
		return true
	default:
		return false
	}
}

func validSecurityStatus(s string) bool {
	switch s {
	case "triaging", "accepted", "patched", "disputed", "wont_fix":
		return true
	default:
		return false
	}
}

func securityReportToJSON(rep reporeports.Report) map[string]any {
	m := map[string]any{
		"id":         rep.ID.String(),
		"reportDate": rep.ReportDate.Format("2006-01-02"),
		"summary":    rep.Summary,
		"status":     rep.Status,
		"bountyPaid": rep.BountyPaid,
		"createdAt":  rep.CreatedAt.UTC().Format(time.RFC3339),
	}
	if rep.ReporterHandle != nil {
		m["reporterHandle"] = *rep.ReporterHandle
	}
	if rep.TriagedAt != nil {
		m["triagedAt"] = rep.TriagedAt.UTC().Format(time.RFC3339)
	}
	if rep.CVSSScore != nil {
		m["cvssScore"] = *rep.CVSSScore
	}
	if rep.Severity != nil {
		m["severity"] = *rep.Severity
	}
	if rep.PatchDate != nil {
		m["patchDate"] = rep.PatchDate.Format("2006-01-02")
	}
	if rep.SLAMet != nil {
		m["slaMet"] = *rep.SLAMet
	}
	return m
}

func securityReportsToJSON(reports []reporeports.Report) []map[string]any {
	out := make([]map[string]any, 0, len(reports))
	for _, rep := range reports {
		out = append(out, securityReportToJSON(rep))
	}
	return out
}

func strPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func floatPtrStr(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', 1, 64)
}

func boolPtrStr(b *bool) string {
	if b == nil {
		return ""
	}
	return strconv.FormatBool(*b)
}

func timePtrRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func datePtrStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
