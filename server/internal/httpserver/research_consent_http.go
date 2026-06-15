package httpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	repoConsent "github.com/lextures/lextures/server/internal/repos/researchconsent"
	svcConsent "github.com/lextures/lextures/server/internal/service/research_consent"
)

const permResearchManage = "global:app:research:manage"

func (d Deps) researchConsentFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFResearchConsent {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Research consent is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerResearchConsentRoutes(r chi.Router) {
	r.Post("/api/v1/admin/consent-studies", d.handleCreateConsentStudy())
	r.Get("/api/v1/admin/consent-studies", d.handleListConsentStudies())
	r.Get("/api/v1/admin/consent-studies/{id}", d.handleGetConsentStudy())
	r.Patch("/api/v1/admin/consent-studies/{id}", d.handleUpdateConsentStudy())
	r.Get("/api/v1/admin/consent-studies/{id}/records", d.handleConsentStudyRecords())
	r.Get("/api/v1/admin/consent-studies/{id}/export", d.handleConsentStudyExport())
	r.Get("/api/v1/me/consent-studies", d.handleMyPendingConsentStudies())
	r.Get("/api/v1/me/consent-studies/history", d.handleMyConsentHistory())
	r.Post("/api/v1/me/consent-studies/{id}/respond", d.handleRespondConsentStudy())
}

// researchManager authenticates and authorizes a researcher/admin. Returns the
// user id and whether the user is a full admin (can access all studies).
func (d Deps) researchManager(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool, bool) {
	userID, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false, false
	}
	ctx := r.Context()
	isAdmin, err := rbac.UserHasPermission(ctx, d.Pool, userID, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false, false
	}
	if isAdmin {
		return userID, true, true
	}
	isResearcher, err := rbac.UserHasPermission(ctx, d.Pool, userID, permResearchManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, false, false
	}
	if !isResearcher {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false, false
	}
	return userID, false, true
}

// loadOwnedStudy fetches a study and enforces that the caller owns it (or is admin).
func (d Deps) loadOwnedStudy(w http.ResponseWriter, r *http.Request, userID uuid.UUID, isAdmin bool) (*repoConsent.Study, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid study id.")
		return nil, false
	}
	study, err := repoConsent.GetStudy(r.Context(), d.Pool, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load study.")
		return nil, false
	}
	if study == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Study not found.")
		return nil, false
	}
	if !isAdmin && study.ResearcherID != userID {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return nil, false
	}
	return study, true
}

type consentStudyJSON struct {
	ID             string          `json:"id"`
	ResearcherID   string          `json:"researcherId"`
	Title          string          `json:"title"`
	IRBProtocol    string          `json:"irbProtocol"`
	ConsentText    string          `json:"consentText"`
	DataUseDesc    string          `json:"dataUseDescription"`
	TargetCriteria json.RawMessage `json:"targetCriteria"`
	Status         string          `json:"status"`
	CreatedAt      string          `json:"createdAt"`
}

func studyToJSON(s repoConsent.Study) consentStudyJSON {
	tc := s.TargetCriteria
	if len(tc) == 0 {
		tc = json.RawMessage(`{}`)
	}
	return consentStudyJSON{
		ID:             s.ID.String(),
		ResearcherID:   s.ResearcherID.String(),
		Title:          s.Title,
		IRBProtocol:    s.IRBProtocol,
		ConsentText:    s.ConsentText,
		DataUseDesc:    s.DataUseDesc,
		TargetCriteria: tc,
		Status:         s.Status,
		CreatedAt:      s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (d Deps) handleCreateConsentStudy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, _, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		var body struct {
			Title          string          `json:"title"`
			IRBProtocol    string          `json:"irbProtocol"`
			ConsentText    string          `json:"consentText"`
			DataUseDesc    string          `json:"dataUseDescription"`
			TargetCriteria json.RawMessage `json:"targetCriteria"`
			Status         string          `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		title := strings.TrimSpace(body.Title)
		irb := strings.TrimSpace(body.IRBProtocol)
		consentText := strings.TrimSpace(body.ConsentText)
		dataUse := strings.TrimSpace(body.DataUseDesc)
		if title == "" || irb == "" || consentText == "" || dataUse == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title, irbProtocol, consentText, and dataUseDescription are required.")
			return
		}
		status := strings.TrimSpace(strings.ToLower(body.Status))
		switch status {
		case "", repoConsent.StatusDraft:
			status = repoConsent.StatusDraft
		case repoConsent.StatusActive, repoConsent.StatusClosed:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		study, err := repoConsent.CreateStudy(r.Context(), d.Pool, repoConsent.Study{
			OrgID:          orgID,
			ResearcherID:   userID,
			Title:          title,
			IRBProtocol:    irb,
			ConsentText:    consentText,
			DataUseDesc:    dataUse,
			TargetCriteria: body.TargetCriteria,
			Status:         status,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create study.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"study": studyToJSON(*study)})
	}
}

func (d Deps) handleListConsentStudies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, isAdmin, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		studies, err := repoConsent.ListStudiesForOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load studies.")
			return
		}
		out := make([]map[string]any, 0, len(studies))
		for i := range studies {
			// Researchers only see their own studies; admins see all (NFR Security).
			if !isAdmin && studies[i].ResearcherID != userID {
				continue
			}
			rate, _ := repoConsent.GetConsentRate(r.Context(), d.Pool, studies[i].ID)
			out = append(out, map[string]any{
				"study":       studyToJSON(studies[i]),
				"consentRate": rate,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"studies": out})
	}
}

func (d Deps) handleGetConsentStudy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, isAdmin, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		study, ok := d.loadOwnedStudy(w, r, userID, isAdmin)
		if !ok {
			return
		}
		rate, _ := repoConsent.GetConsentRate(r.Context(), d.Pool, study.ID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"study":       studyToJSON(*study),
			"consentRate": rate,
		})
	}
}

func (d Deps) handleUpdateConsentStudy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, isAdmin, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		study, ok := d.loadOwnedStudy(w, r, userID, isAdmin)
		if !ok {
			return
		}
		var body struct {
			Title          *string         `json:"title"`
			IRBProtocol    *string         `json:"irbProtocol"`
			ConsentText    *string         `json:"consentText"`
			DataUseDesc    *string         `json:"dataUseDescription"`
			Status         *string         `json:"status"`
			TargetCriteria json.RawMessage `json:"targetCriteria"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Status != nil {
			s := strings.TrimSpace(strings.ToLower(*body.Status))
			switch s {
			case repoConsent.StatusDraft, repoConsent.StatusActive, repoConsent.StatusClosed:
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
				return
			}
			body.Status = &s
		}
		// Activation requires an IRB protocol number (Open Question #2 resolved: required).
		if body.Status != nil && *body.Status == repoConsent.StatusActive {
			irb := study.IRBProtocol
			if body.IRBProtocol != nil {
				irb = strings.TrimSpace(*body.IRBProtocol)
			}
			if irb == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "An IRB protocol number is required to activate a study.")
				return
			}
		}
		updated, err := repoConsent.UpdateStudy(
			r.Context(), d.Pool, study.ID,
			trimPtr(body.Title), trimPtr(body.IRBProtocol), trimPtr(body.ConsentText),
			trimPtr(body.DataUseDesc), body.Status, body.TargetCriteria,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update study.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"study": studyToJSON(*updated)})
	}
}

func (d Deps) handleConsentStudyRecords() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, isAdmin, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		study, ok := d.loadOwnedStudy(w, r, userID, isAdmin)
		if !ok {
			return
		}
		records, err := repoConsent.ListRecordsForStudy(r.Context(), d.Pool, study.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load consent records.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"records": consentRecordsToJSON(records)})
	}
}

func (d Deps) handleConsentStudyExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, isAdmin, ok := d.researchManager(w, r)
		if !ok {
			return
		}
		study, ok := d.loadOwnedStudy(w, r, userID, isAdmin)
		if !ok {
			return
		}
		participants, err := repoConsent.ExportConsenting(r.Context(), d.Pool, study.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export participants.")
			return
		}
		if participants == nil {
			participants = []repoConsent.Participant{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studyId":      study.ID.String(),
			"participants": participants,
			"count":        len(participants),
		})
	}
}

func (d Deps) handleMyPendingConsentStudies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Organization not found.")
			return
		}
		studies, err := repoConsent.PendingStudiesForUser(r.Context(), d.Pool, orgID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load consent studies.")
			return
		}
		out := make([]consentStudyJSON, 0, len(studies))
		for i := range studies {
			out = append(out, studyToJSON(studies[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"studies": out})
	}
}

func (d Deps) handleMyConsentHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		records, err := repoConsent.ListHistoryForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load consent history.")
			return
		}
		// Enrich with study titles for display.
		titles := map[uuid.UUID]string{}
		out := make([]map[string]any, 0, len(records))
		for i := range records {
			title, seen := titles[records[i].StudyID]
			if !seen {
				if s, err := repoConsent.GetStudy(r.Context(), d.Pool, records[i].StudyID); err == nil && s != nil {
					title = s.Title
				}
				titles[records[i].StudyID] = title
			}
			j := consentRecordToJSON(records[i])
			out = append(out, map[string]any{
				"id":         j.ID,
				"studyId":    j.StudyID,
				"studyTitle": title,
				"decision":   j.Decision,
				"createdAt":  j.CreatedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"history": out})
	}
}

func (d Deps) handleRespondConsentStudy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.researchConsentFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studyID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid study id.")
			return
		}
		var body struct {
			Decision string `json:"decision"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		decision := strings.TrimSpace(strings.ToLower(body.Decision))
		if !svcConsent.ValidDecision(decision) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "decision must be granted, declined, or withdrawn.")
			return
		}
		study, err := repoConsent.GetStudy(r.Context(), d.Pool, studyID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load study.")
			return
		}
		if study == nil || study.Status != repoConsent.StatusActive {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Study not found or not active.")
			return
		}
		// The student must be targeted by the study to respond.
		targeted, err := repoConsent.UserTargetedBy(r.Context(), d.Pool, study, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify targeting.")
			return
		}
		if !targeted {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not eligible for this study.")
			return
		}
		now := time.Now().UTC()
		ip := clientIP(r)
		ua := r.UserAgent()
		sig := svcConsent.SignRecord(d.effectiveConfig().JWTSecret, studyID, userID, decision, now)
		rec := repoConsent.Record{
			StudyID:  studyID,
			UserID:   userID,
			Decision: decision,
			HMAC:     strPtrOrNil(sig),
		}
		if ip != "" {
			rec.IPAddress = &ip
		}
		if ua != "" {
			rec.UserAgent = &ua
		}
		saved, err := repoConsent.InsertRecord(r.Context(), d.Pool, rec)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record decision.")
			return
		}
		svcConsent.RecordDecision(studyID.String(), decision)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"record": consentRecordToJSON(*saved)})
	}
}

// --- helpers ---

type consentRecordJSON struct {
	ID        string  `json:"id"`
	StudyID   string  `json:"studyId"`
	UserID    string  `json:"userId"`
	Decision  string  `json:"decision"`
	IPAddress *string `json:"ipAddress,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	HMAC      *string `json:"hmac,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

func consentRecordToJSON(r repoConsent.Record) consentRecordJSON {
	return consentRecordJSON{
		ID:        r.ID.String(),
		StudyID:   r.StudyID.String(),
		UserID:    r.UserID.String(),
		Decision:  r.Decision,
		IPAddress: r.IPAddress,
		UserAgent: r.UserAgent,
		HMAC:      r.HMAC,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func consentRecordsToJSON(records []repoConsent.Record) []consentRecordJSON {
	out := make([]consentRecordJSON, 0, len(records))
	for i := range records {
		out = append(out, consentRecordToJSON(records[i]))
	}
	return out
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// clientIP extracts a valid caller IP from X-Forwarded-For or RemoteAddr, or ""
// when none can be parsed (so it can be stored as NULL in an inet column).
func clientIP(r *http.Request) string {
	candidate := strings.TrimSpace(r.RemoteAddr)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		candidate = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(candidate); err == nil {
		if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil {
			return ip.String()
		}
	}
	return ""
}
