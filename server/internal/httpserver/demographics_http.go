package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/demographics"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgunit"
	demosvc "github.com/lextures/lextures/server/internal/service/demographics"
)

func (d Deps) demographicsEnabled(w http.ResponseWriter) bool {
	if !d.Config.FFDemographics {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
			"Student demographics is not enabled.")
		return false
	}
	return true
}

type demographicsJSON struct {
	StudentID         string  `json:"studentId"`
	FreeLunch         *bool   `json:"freeLunch"`
	ReducedLunch      *bool   `json:"reducedLunch"`
	EllStatus         *bool   `json:"ellStatus"`
	DisabilityStatus  *bool   `json:"disabilityStatus"`
	RaceEthnicityCode *string `json:"raceEthnicityCode"`
	HomelessIndicator *bool   `json:"homelessIndicator"`
	MigrantIndicator  *bool   `json:"migrantIndicator"`
	DataSource        string  `json:"dataSource"`
	LastVerifiedAt    *string `json:"lastVerifiedAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

func rowToJSON(r *repo.Row) demographicsJSON {
	out := demographicsJSON{
		StudentID:         r.StudentID.String(),
		FreeLunch:         r.FreeLunch,
		ReducedLunch:      r.ReducedLunch,
		EllStatus:         r.EllStatus,
		DisabilityStatus:  r.DisabilityStatus,
		RaceEthnicityCode: r.RaceEthnicityCode,
		HomelessIndicator: r.HomelessIndicator,
		MigrantIndicator:  r.MigrantIndicator,
		DataSource:        r.DataSource,
		UpdatedAt:         r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if r.LastVerifiedAt != nil {
		s := r.LastVerifiedAt.UTC().Format("2006-01-02T15:04:05Z")
		out.LastVerifiedAt = &s
	}
	return out
}

// handleStudentDemographics — GET/PATCH /api/v1/admin/students/{studentId}/demographics
func (d Deps) handleStudentDemographics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.demographicsEnabled(w) {
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "studentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve student organization.")
			return
		}

		switch r.Method {
		case http.MethodGet:
			canRead, err := demosvc.CanReadIndividual(r.Context(), d.Pool, actorID, orgID, studentID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !canRead {
				// AC-1: omit demographics (200), do not return 403.
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{"studentId": studentID.String()})
				return
			}
			row, err := repo.GetByStudentID(r.Context(), d.Pool, studentID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load demographics.")
				return
			}
			if row == nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{"studentId": studentID.String()})
				return
			}
			if err := demosvc.LogView(r.Context(), d.Pool, orgID, actorID, studentID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to log demographic access.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(rowToJSON(row))

		case http.MethodPatch:
			canWrite, err := demosvc.CanWrite(r.Context(), d.Pool, actorID, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !canWrite {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to update demographics.")
				return
			}
			var body struct {
				FreeLunch         *bool   `json:"freeLunch"`
				ReducedLunch      *bool   `json:"reducedLunch"`
				EllStatus         *bool   `json:"ellStatus"`
				DisabilityStatus  *bool   `json:"disabilityStatus"`
				RaceEthnicityCode *string `json:"raceEthnicityCode"`
				HomelessIndicator *bool   `json:"homelessIndicator"`
				MigrantIndicator  *bool   `json:"migrantIndicator"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			row, err := repo.Upsert(r.Context(), d.Pool, studentID, repo.UpsertInput{
				FreeLunch:         body.FreeLunch,
				ReducedLunch:      body.ReducedLunch,
				EllStatus:         body.EllStatus,
				DisabilityStatus:  body.DisabilityStatus,
				RaceEthnicityCode: body.RaceEthnicityCode,
				HomelessIndicator: body.HomelessIndicator,
				MigrantIndicator:  body.MigrantIndicator,
				DataSource:        "manual",
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update demographics.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(rowToJSON(row))

		default:
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleSchoolDemographicsReport — GET /api/v1/admin/org-units/{orgUnitId}/demographics/report
func (d Deps) handleSchoolDemographicsReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.demographicsEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		schoolID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgUnitId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org unit id.")
			return
		}
		unit, err := orgunit.GetByID(r.Context(), d.Pool, schoolID)
		if err != nil || unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Org unit not found.")
			return
		}
		canReport, err := demosvc.CanRunReports(r.Context(), d.Pool, actorID, unit.OrgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canReport {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this report.")
			return
		}
		report, err := repo.Title1AggregateReport(r.Context(), d.Pool, schoolID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schoolId":              report.SchoolID.String(),
			"totalStudents":         report.TotalStudents,
			"freeLunchCount":          report.FreeLunchCount,
			"reducedLunchCount":       report.ReducedLunchCount,
			"economicDisadvantaged":   report.EconomicDisadvantaged,
			"economicDisadvantagePct": report.EconomicPct,
			"ellCount":                report.EllCount,
			"disabilityCount":         report.DisabilityCount,
			"homelessCount":           report.HomelessCount,
			"migrantCount":            report.MigrantCount,
			"raceBreakdown":           report.RaceBreakdown,
		})
	}
}

// handleSchoolDisaggregatedPerformance — GET /api/v1/admin/org-units/{orgUnitId}/demographics/disaggregated-performance
func (d Deps) handleSchoolDisaggregatedPerformance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.demographicsEnabled(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		schoolID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgUnitId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org unit id.")
			return
		}
		unit, err := orgunit.GetByID(r.Context(), d.Pool, schoolID)
		if err != nil || unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Org unit not found.")
			return
		}
		canReport, err := demosvc.CanRunReports(r.Context(), d.Pool, actorID, unit.OrgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canReport {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this report.")
			return
		}
		dimension := strings.TrimSpace(r.URL.Query().Get("dimension"))
		if dimension == "" {
			dimension = "ell"
		}
		report, err := repo.DisaggregatedPerformanceReport(r.Context(), d.Pool, schoolID, dimension)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(report)
	}
}

func (d Deps) registerDemographicsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/students/{studentId}/demographics", d.handleStudentDemographics())
	r.Patch("/api/v1/admin/students/{studentId}/demographics", d.handleStudentDemographics())
	r.Get("/api/v1/admin/org-units/{orgUnitId}/demographics/report", d.handleSchoolDemographicsReport())
	r.Get("/api/v1/admin/org-units/{orgUnitId}/demographics/disaggregated-performance", d.handleSchoolDisaggregatedPerformance())
}
