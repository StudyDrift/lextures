package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoCatalog "github.com/lextures/lextures/server/internal/repos/catalog"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/workers/catalogsync"
)

func (d Deps) meOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
	if err != nil || orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
		return uuid.UUID{}, false
	}
	return orgID, true
}

func (d Deps) registerCatalogRoutes(r chi.Router) {
	r.Method(http.MethodGet, "/api/v1/catalog/sections", d.handleCatalogSectionsList())
	r.Method(http.MethodGet, "/api/v1/catalog/sections/{id}", d.handleCatalogSectionDetail())
	r.Method(http.MethodGet, "/api/v1/catalog/schedule", d.handleCatalogSchedule())
	r.Method(http.MethodPost, "/api/v1/admin/catalog/sync", d.handleAdminCatalogSync())
	r.Method(http.MethodGet, "/api/v1/admin/catalog/sync-status", d.handleAdminCatalogSyncStatus())
	r.Get("/api/v1/courses/{course_code}/catalog-info", d.handleCourseCatalogInfo())
}

func (d Deps) catalogFeatureOff(w http.ResponseWriter) bool {
	if !d.Config.FFCatalogIntegration {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Course catalog integration is not enabled.")
		return true
	}
	return false
}

func (d Deps) handleCatalogSectionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		_ = userID

		f := repoCatalog.ListFilter{Limit: 50}
		if v := strings.TrimSpace(r.URL.Query().Get("term_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				f.TermID = &id
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("department")); v != "" {
			f.Department = &v
		}
		if v := strings.TrimSpace(r.URL.Query().Get("days")); v != "" {
			f.Days = &v
		}
		if v := strings.TrimSpace(r.URL.Query().Get("min_credits")); v != "" {
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				f.MinCredits = &n
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("max_credits")); v != "" {
			if n, err := strconv.ParseFloat(v, 64); err == nil {
				f.MaxCredits = &n
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("q")); v != "" {
			f.Query = &v
		}
		if v := strings.TrimSpace(r.URL.Query().Get("cursor")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				f.Cursor = &id
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				f.Limit = n
			}
		}

		sections, err := repoCatalog.ListSections(r.Context(), d.Pool, orgID, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list catalog sections.")
			return
		}
		out := make([]map[string]any, 0, len(sections))
		for i := range sections {
			out = append(out, sectionToPublicJSON(&sections[i]))
		}
		var nextCursor *string
		if len(sections) == f.Limit && len(sections) > 0 {
			c := sections[len(sections)-1].ID.String()
			nextCursor = &c
		}
		lastSync, _ := repoCatalog.GetLastSyncStatus(r.Context(), d.Pool, orgID)
		resp := map[string]any{"sections": out, "nextCursor": nextCursor}
		if lastSync != nil {
			resp["lastSyncedAt"] = lastSync.StartedAt
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleCatalogSectionDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		sectionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid section id.")
			return
		}
		sec, err := repoCatalog.GetSection(r.Context(), d.Pool, orgID, sectionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load section.")
			return
		}
		if sec == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Section not found.")
			return
		}
		out := sectionToPublicJSON(sec)
		prereqStatus, _ := repoCatalog.GetPrereqStatusForUser(r.Context(), d.Pool, userID, sectionID)
		if len(prereqStatus) > 0 {
			out["prerequisiteStatus"] = prereqStatus
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"section": out})
	}
}

func (d Deps) handleCatalogSchedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		entries, err := repoCatalog.ListScheduleForUser(r.Context(), d.Pool, orgID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load schedule.")
			return
		}
		out := make([]map[string]any, 0, len(entries))
		for i := range entries {
			e := entries[i]
			item := map[string]any{
				"section":        sectionToPublicJSON(&e.Section),
				"registrationStatus": e.Registration.Status,
			}
			if e.CourseCode != nil {
				item["courseCode"] = *e.CourseCode
			}
			if e.CourseTitle != nil {
				item["courseTitle"] = *e.CourseTitle
			}
			if len(e.Registration.PrereqStatus) > 0 {
				item["prerequisiteStatus"] = e.Registration.PrereqStatus
			}
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"schedule": out})
	}
}

func (d Deps) handleAdminCatalogSync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		var body struct {
			ConnectionID string `json:"connectionId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		connIDStr := strings.TrimSpace(body.ConnectionID)

		var conn *repoSIS.Connection
		var err error
		if connIDStr != "" {
			connID, parseErr := uuid.Parse(connIDStr)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connectionId.")
				return
			}
			conn, err = repoSIS.GetConnection(r.Context(), d.Pool, orgID, connID)
		} else {
			conns, listErr := repoSIS.ListConnections(r.Context(), d.Pool, orgID)
			if listErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list SIS connections.")
				return
			}
			for i := range conns {
				if conns[i].Active && repoSIS.VendorMarket(conns[i].Vendor) == "he" {
					conn = &conns[i]
					break
				}
			}
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load SIS connection.")
			return
		}
		if conn == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No active HE SIS connection found.")
			return
		}

		result, err := catalogsync.RunSync(r.Context(), d.Pool, *conn)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Catalog sync failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"logId":          result.LogID,
			"status":         result.Status,
			"sectionsSynced": result.SectionsSynced,
			"shellsCreated":  result.ShellsCreated,
			"shellsUpdated":  result.ShellsUpdated,
			"errors":         result.Errors,
		})
	}
}

func (d Deps) handleAdminCatalogSyncStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		log, err := repoCatalog.GetLastSyncStatus(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load sync status.")
			return
		}
		if log == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"synced": false})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"synced":         true,
			"lastSyncedAt":   log.StartedAt,
			"finishedAt":     log.FinishedAt,
			"status":         log.Status,
			"sectionsSynced": log.SectionsSynced,
			"shellsCreated":  log.ShellsCreated,
			"errorCount":     len(log.Errors),
			"errors":         log.Errors,
		})
	}
}

func (d Deps) handleCourseCatalogInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.catalogFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		orgID, ok := d.meOrgID(w, r)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course code.")
			return
		}
		course, err := repoCourse.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(course.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}
		sec, err := repoCatalog.GetSectionByLMSCourseID(r.Context(), d.Pool, orgID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load catalog info.")
			return
		}
		if sec == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"catalogInfo": nil})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"catalogInfo": sectionToPublicJSON(sec)})
	}
}

func sectionToPublicJSON(s *repoCatalog.Section) map[string]any {
	out := map[string]any{
		"id":           s.ID,
		"orgId":        s.OrgID,
		"termId":       s.TermID,
		"sisCourseId":  s.SISCourseID,
		"sisSectionId": s.SISSectionID,
		"subject":      s.Subject,
		"courseNumber": s.CourseNumber,
		"title":        s.Title,
		"status":       s.Status,
	}
	if s.CRN != nil {
		out["crn"] = *s.CRN
	}
	if s.SectionNumber != nil {
		out["sectionNumber"] = *s.SectionNumber
	}
	if s.Credits != nil {
		out["credits"] = *s.Credits
	}
	if s.MeetingPattern != nil {
		out["meetingPattern"] = s.MeetingPattern
	}
	if s.Room != nil {
		out["room"] = *s.Room
	}
	if s.Department != nil {
		out["department"] = *s.Department
	}
	if len(s.Prerequisites) > 0 {
		out["prerequisites"] = s.Prerequisites
	}
	if s.InstructorName != nil {
		out["instructorName"] = *s.InstructorName
	}
	if s.LMSCourseID != nil {
		out["lmsCourseId"] = *s.LMSCourseID
	}
	if s.SyncedAt != nil {
		out["syncedAt"] = s.SyncedAt
	}
	return out
}
