package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoAdvising "github.com/lextures/lextures/server/internal/repos/advising"
	"github.com/lextures/lextures/server/internal/repos/organization"
	svcAdvising "github.com/lextures/lextures/server/internal/service/advising"
)

func (d Deps) advisingFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFAdvisingIntegration {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Advising integration is not enabled.")
		return true
	}
	return false
}

type advisingNoteJSON struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"studentId"`
	AdvisorID        string  `json:"advisorId"`
	Content          string  `json:"content"`
	VisibleToStudent bool    `json:"visibleToStudent"`
	CreatedAt        string  `json:"createdAt"`
	AdvisorEmail     string  `json:"advisorEmail,omitempty"`
	AdvisorDisplay   *string `json:"advisorDisplayName,omitempty"`
}

func noteToJSON(n repoAdvising.Note) advisingNoteJSON {
	return advisingNoteJSON{
		ID:               n.ID.String(),
		StudentID:        n.StudentID.String(),
		AdvisorID:        n.AdvisorID.String(),
		Content:          n.Content,
		VisibleToStudent: n.VisibleToStudent,
		CreatedAt:        n.CreatedAt.UTC().Format(time.RFC3339),
		AdvisorEmail:     n.AdvisorEmail,
		AdvisorDisplay:   n.AdvisorDisplay,
	}
}

type advisingConfigJSON struct {
	AppointmentURL        string  `json:"appointmentUrl"`
	DegreeAuditProvider   string  `json:"degreeAuditProvider"`
	DegreeAuditBaseURL    string  `json:"degreeAuditBaseUrl"`
	APICredentialsRef     string  `json:"apiCredentialsRef"`
	AtRiskBannerEnabled   bool    `json:"atRiskBannerEnabled"`
}

func configToAdvisingJSON(c *repoAdvising.Config) advisingConfigJSON {
	out := advisingConfigJSON{DegreeAuditProvider: repoAdvising.ProviderNone}
	if c == nil {
		return out
	}
	if c.AppointmentURL != nil {
		out.AppointmentURL = strings.TrimSpace(*c.AppointmentURL)
	}
	if c.DegreeAuditProvider != "" {
		out.DegreeAuditProvider = c.DegreeAuditProvider
	}
	if c.DegreeAuditBaseURL != nil {
		out.DegreeAuditBaseURL = strings.TrimSpace(*c.DegreeAuditBaseURL)
	}
	if c.APICredentialsRef != nil {
		out.APICredentialsRef = strings.TrimSpace(*c.APICredentialsRef)
	}
	out.AtRiskBannerEnabled = c.AtRiskBannerEnabled
	return out
}

type degreeProgressJSON struct {
	Configured             bool                          `json:"configured"`
	CompletionPercent      *int                          `json:"completionPercent,omitempty"`
	RemainingRequiredCount *int                          `json:"remainingRequiredCount,omitempty"`
	RemainingRequirements  []svcAdvising.RequirementGroup `json:"remainingRequirements,omitempty"`
	AtRisk                 bool                          `json:"atRisk,omitempty"`
	LastUpdated            *string                       `json:"lastUpdated,omitempty"`
	Stale                  bool                          `json:"stale,omitempty"`
	AppointmentURL         string                        `json:"appointmentUrl,omitempty"`
	RecentNotesCount       int                           `json:"recentNotesCount,omitempty"`
}

func (d Deps) registerAdvisingRoutes(r chi.Router) {
	r.Get("/api/v1/me/degree-progress", d.handleGetMyDegreeProgress())
	r.Get("/api/v1/me/advising-notes", d.handleGetMyAdvisingNotes())
	r.Get("/api/v1/me/advising/config", d.handleGetMyAdvisingConfig())
	r.Post("/api/v1/advisor/students/{uid}/notes", d.handlePostAdvisorNote())
	r.Get("/api/v1/advisor/students/{uid}/notes", d.handleGetAdvisorNotes())
	r.Get("/api/v1/admin/advising/config", d.handleGetAdminAdvisingConfig())
	r.Post("/api/v1/admin/advising/config", d.handlePostAdminAdvisingConfig())
	r.Post("/api/v1/admin/advising/links", d.handlePostAdminAdvisorLink())
}

func (d Deps) handleGetMyDegreeProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cfg, err := repoAdvising.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load advising config.")
			return
		}
		out := degreeProgressJSON{}
		if cfg.AppointmentURL != nil {
			out.AppointmentURL = strings.TrimSpace(*cfg.AppointmentURL)
		}
		since := time.Now().UTC().Add(-30 * 24 * time.Hour)
		if n, err := repoAdvising.CountRecentNotesForStudent(r.Context(), d.Pool, userID, since); err == nil {
			out.RecentNotesCount = n
		}
		if cfg.DegreeAuditProvider == repoAdvising.ProviderNone {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(out)
			return
		}
		summary, lastUpdated, stale, err := d.loadDegreeProgress(r.Context(), userID, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load degree progress.")
			return
		}
		out.Configured = true
		if summary != nil {
			out.CompletionPercent = &summary.CompletionPercent
			out.RemainingRequiredCount = &summary.RemainingRequiredCount
			out.RemainingRequirements = summary.RemainingRequirements
			out.AtRisk = summary.AtRisk && cfg.AtRiskBannerEnabled
		}
		if lastUpdated != nil {
			s := lastUpdated.UTC().Format(time.RFC3339)
			out.LastUpdated = &s
		}
		out.Stale = stale
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) loadDegreeProgress(ctx context.Context, userID uuid.UUID, cfg *repoAdvising.Config) (*svcAdvising.DegreeProgressSummary, *time.Time, bool, error) {
	now := time.Now().UTC()
	cached, err := repoAdvising.GetDegreeAuditCache(ctx, d.Pool, userID)
	if err != nil {
		return nil, nil, false, err
	}
	stale := false
	if cached != nil {
		stale = svcAdvising.CacheExpired(cached.FetchedAt, now)
		if !stale {
			summary, err := svcAdvising.ParseSummaryJSON(cached.Data)
			if err != nil {
				return nil, &cached.FetchedAt, true, nil
			}
			return &summary, &cached.FetchedAt, false, nil
		}
	}
	adapter := svcAdvising.AdapterFor(cfg.DegreeAuditProvider)
	if adapter == nil {
		if cached != nil {
			summary, _ := svcAdvising.ParseSummaryJSON(cached.Data)
			return &summary, &cached.FetchedAt, true, nil
		}
		return nil, nil, false, nil
	}
	adapterCfg := svcAdvising.AdapterConfig{Provider: cfg.DegreeAuditProvider}
	if cfg.DegreeAuditBaseURL != nil {
		adapterCfg.BaseURL = strings.TrimSpace(*cfg.DegreeAuditBaseURL)
	}
	if cfg.APICredentialsRef != nil {
		adapterCfg.CredentialsRef = strings.TrimSpace(*cfg.APICredentialsRef)
	}
	summary, fetchErr := adapter.FetchSummary(ctx, adapterCfg, userID)
	if fetchErr != nil {
		svcAdvising.RecordDegreeAuditAPICall(false)
		if cached != nil {
			s, _ := svcAdvising.ParseSummaryJSON(cached.Data)
			return &s, &cached.FetchedAt, true, nil
		}
		return nil, nil, false, fetchErr
	}
	svcAdvising.RecordDegreeAuditAPICall(true)
	raw, err := svcAdvising.SummaryToJSON(summary)
	if err != nil {
		return &summary, nil, false, nil
	}
	fetchedAt := now
	if err := repoAdvising.UpsertDegreeAuditCache(ctx, d.Pool, userID, raw, adapter.Provider(), fetchedAt); err != nil {
		return &summary, &fetchedAt, false, nil
	}
	return &summary, &fetchedAt, false, nil
}

func (d Deps) handleGetMyAdvisingNotes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		notes, err := repoAdvising.ListNotesForStudent(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load advising notes.")
			return
		}
		out := make([]advisingNoteJSON, 0, len(notes))
		for i := range notes {
			out = append(out, noteToJSON(notes[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"notes": out})
	}
}

func (d Deps) handleGetMyAdvisingConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		cfg, err := repoAdvising.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load advising config.")
			return
		}
		out := map[string]any{}
		if cfg.AppointmentURL != nil {
			out["appointmentUrl"] = strings.TrimSpace(*cfg.AppointmentURL)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) requireAdvisorAccess(w http.ResponseWriter, r *http.Request, advisorID, studentID uuid.UUID) bool {
	if _, ok := d.adminRbacUser(w, r); ok {
		return true
	}
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, advisorID)
	if err != nil || orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return false
	}
	linked, err := repoAdvising.ActiveAdvisorLinkBetween(r.Context(), d.Pool, orgID, advisorID, studentID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify advisor access.")
		return false
	}
	if !linked {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return false
	}
	return true
}

func (d Deps) handlePostAdvisorNote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		advisorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		if !d.requireAdvisorAccess(w, r, advisorID, studentID) {
			return
		}
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		content := strings.TrimSpace(body.Content)
		if content == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Content is required.")
			return
		}
		note, err := repoAdvising.InsertNote(r.Context(), d.Pool, studentID, advisorID, content)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create advising note.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"note": noteToJSON(*note)})
	}
}

func (d Deps) handleGetAdvisorNotes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		advisorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		if !d.requireAdvisorAccess(w, r, advisorID, studentID) {
			return
		}
		notes, err := repoAdvising.ListNotesForAdvisor(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load advising notes.")
			return
		}
		out := make([]advisingNoteJSON, 0, len(notes))
		for i := range notes {
			out = append(out, noteToJSON(notes[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"notes": out})
	}
}

func (d Deps) handleGetAdminAdvisingConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		cfg, err := repoAdvising.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load advising config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToAdvisingJSON(cfg))
	}
}

func (d Deps) handlePostAdminAdvisingConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		var body struct {
			AppointmentURL      *string `json:"appointmentUrl"`
			DegreeAuditProvider *string `json:"degreeAuditProvider"`
			DegreeAuditBaseURL  *string `json:"degreeAuditBaseUrl"`
			APICredentialsRef   *string `json:"apiCredentialsRef"`
			AtRiskBannerEnabled *bool   `json:"atRiskBannerEnabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		provider := repoAdvising.ProviderNone
		if body.DegreeAuditProvider != nil {
			provider = strings.TrimSpace(strings.ToLower(*body.DegreeAuditProvider))
		}
		switch provider {
		case repoAdvising.ProviderNone, repoAdvising.ProviderDegreeWorks, repoAdvising.ProviderStellic:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid degree audit provider.")
			return
		}
		if body.AppointmentURL != nil {
			u := strings.TrimSpace(*body.AppointmentURL)
			if u != "" {
				if _, err := url.ParseRequestURI(u); err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid appointment URL.")
					return
				}
				body.AppointmentURL = &u
			}
		}
		cfg, err := repoAdvising.UpsertConfig(
			r.Context(), d.Pool,
			body.AppointmentURL,
			provider,
			body.DegreeAuditBaseURL,
			body.APICredentialsRef,
			body.AtRiskBannerEnabled,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save advising config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToAdvisingJSON(cfg))
	}
}

func (d Deps) handlePostAdminAdvisorLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.advisingFeatureOff(w) {
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		var body struct {
			AdvisorUserID  string `json:"advisorUserId"`
			StudentUserID  string `json:"studentUserId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		advisorID, err := uuid.Parse(strings.TrimSpace(body.AdvisorUserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid advisor user id.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentUserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student user id.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, studentID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Student organization not found.")
			return
		}
		if err := repoAdvising.UpsertAdvisorLink(r.Context(), d.Pool, orgID, advisorID, studentID, &adminID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create advisor link.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// degreeProgressForUser loads cached or fresh degree progress for catalog badge enrichment.
func (d Deps) degreeProgressForUser(ctx context.Context, userID uuid.UUID) *svcAdvising.DegreeProgressSummary {
	cfg, err := repoAdvising.GetConfig(ctx, d.Pool)
	if err != nil || cfg == nil || cfg.DegreeAuditProvider == repoAdvising.ProviderNone {
		return nil
	}
	if !d.effectiveConfig().FFAdvisingIntegration {
		return nil
	}
	summary, _, _, err := d.loadDegreeProgress(ctx, userID, cfg)
	if err != nil || summary == nil {
		return nil
	}
	return summary
}

// catalogCourseCode builds a catalog course code from subject + number.
func catalogCourseCode(subject, courseNumber string) string {
	return strings.ToUpper(strings.TrimSpace(subject)) + strings.TrimSpace(courseNumber)
}

// enrichSectionWithDegreeRequirements adds fulfillsRequirements when applicable.
func enrichSectionWithDegreeRequirements(section map[string]any, summary *svcAdvising.DegreeProgressSummary) {
	if summary == nil {
		return
	}
	subject, _ := section["subject"].(string)
	num, _ := section["courseNumber"].(string)
	code := catalogCourseCode(subject, num)
	reqs := svcAdvising.FulfillsRequirements(summary, code)
	if len(reqs) > 0 {
		section["fulfillsRequirements"] = reqs
	}
}
