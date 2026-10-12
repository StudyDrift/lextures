package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/userai"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
	"github.com/lextures/lextures/server/internal/service/starterlibrary"
)

func (d Deps) registerStarterLibraryRoutes(r chi.Router) {
	r.Get("/api/v1/starter-library", d.handleListStarterLibrary())
	r.Post("/api/v1/starter-library/generate", d.handleGenerateStarterCourse())
	r.Get("/api/v1/starter-library/{slug}", d.handleGetStarterCourse())
	r.Post("/api/v1/starter-library/{slug}/use", d.handleUseStarterCourse())
}

type starterModuleSummary struct {
	Title     string `json:"title"`
	PageCount int    `json:"pageCount"`
}

type starterSummary struct {
	Slug             string                 `json:"slug"`
	Title            string                 `json:"title"`
	Summary          string                 `json:"summary"`
	GradeBand        string                 `json:"gradeBand"`
	Subject          string                 `json:"subject"`
	SubjectLabel     string                 `json:"subjectLabel"`
	EstimatedMinutes int                    `json:"estimatedMinutes"`
	Standards        []string               `json:"standards"`
	Modules          []starterModuleSummary `json:"modules"`
}

type starterPageDTO struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

type starterModuleDTO struct {
	Title string           `json:"title"`
	Pages []starterPageDTO `json:"pages"`
}

type starterDetail struct {
	starterSummary
	Modules []starterModuleDTO `json:"modules"`
}

func summarizeStarter(c starterlibrary.Course) starterSummary {
	mods := make([]starterModuleSummary, 0, len(c.Modules))
	for _, m := range c.Modules {
		mods = append(mods, starterModuleSummary{Title: m.Title, PageCount: len(m.Pages)})
	}
	standards := c.Standards
	if standards == nil {
		standards = []string{}
	}
	return starterSummary{
		Slug:             c.Slug,
		Title:            c.Title,
		Summary:          c.Summary,
		GradeBand:        c.GradeBand,
		Subject:          c.Subject,
		SubjectLabel:     starterlibrary.SubjectLabel(c.Subject),
		EstimatedMinutes: c.EstimatedMinutes,
		Standards:        standards,
		Modules:          mods,
	}
}

func detailStarter(c starterlibrary.Course) starterDetail {
	mods := make([]starterModuleDTO, 0, len(c.Modules))
	for _, m := range c.Modules {
		pages := make([]starterPageDTO, 0, len(m.Pages))
		for _, p := range m.Pages {
			pages = append(pages, starterPageDTO{Title: p.Title, Markdown: p.Markdown})
		}
		mods = append(mods, starterModuleDTO{Title: m.Title, Pages: pages})
	}
	sum := summarizeStarter(c)
	return starterDetail{starterSummary: sum, Modules: mods}
}

func (d Deps) handleListStarterLibrary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		q := r.URL.Query()
		courses := starterlibrary.Filter(q.Get("gradeBand"), q.Get("subject"))
		out := make([]starterSummary, 0, len(courses))
		for _, c := range courses {
			out = append(out, summarizeStarter(c))
		}
		subjects := make([]map[string]string, 0, len(starterlibrary.Subjects))
		for _, s := range starterlibrary.Subjects {
			subjects = append(subjects, map[string]string{"id": s.ID, "label": s.Label})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"gradeBands": starterlibrary.GradeBands,
			"subjects":   subjects,
			"courses":    out,
		})
	}
}

func (d Deps) handleGetStarterCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		c, ok := starterlibrary.Get(chi.URLParam(r, "slug"))
		if !ok {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Starter course not found.")
			return
		}
		writeJSON(w, http.StatusOK, detailStarter(c))
	}
}

type starterUseBody struct {
	LearnerUserID string `json:"learnerUserId"`
}

type starterCopyResponse struct {
	CourseCode    string `json:"courseCode"`
	Title         string `json:"title"`
	Source        string `json:"source,omitempty"`
	Enrolled      bool   `json:"enrolled"`
	LearnerUserID string `json:"learnerUserId,omitempty"`
}

func (d Deps) requireCourseCreate(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	allowed, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, "global:app:course:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !allowed {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to create courses.")
		return false
	}
	return true
}

func (d Deps) starterLearner(w http.ResponseWriter, r *http.Request, buyer uuid.UUID, learnerRaw string) (*uuid.UUID, bool) {
	learnerRaw = strings.TrimSpace(learnerRaw)
	if learnerRaw == "" {
		return nil, true
	}
	if !d.effectiveConfig().FFHomeschoolManagedLearners {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Managed learners are not enabled.")
		return nil, false
	}
	id, err := uuid.Parse(learnerRaw)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learnerUserId.")
		return nil, false
	}
	if id == buyer {
		return nil, true
	}
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, buyer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify learner.")
		return nil, false
	}
	can, err := managedlearners.CanEnrollManaged(r.Context(), d.Pool, buyer, id, orgID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify learner.")
		return nil, false
	}
	if !can {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You cannot assign this course to that learner.")
		return nil, false
	}
	return &id, true
}

func (d Deps) copyStarter(w http.ResponseWriter, r *http.Request, userID uuid.UUID, outline starterlibrary.Course, learnerRaw, source string) {
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
		return
	}
	if !d.requireCourseCreate(w, r, userID) {
		return
	}
	enrollID, ok := d.starterLearner(w, r, userID, learnerRaw)
	if !ok {
		return
	}
	copied, err := starterlibrary.CopyForUser(r.Context(), d.Pool, userID, outline)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not copy this starter course.")
		return
	}
	resp := starterCopyResponse{
		CourseCode: copied.CourseCode,
		Title:      copied.Title,
		Source:     source,
	}
	if enrollID != nil {
		if _, err := courseroles.EnrollStudentWithGrants(r.Context(), d.Pool, copied.CourseID, *enrollID, copied.CourseCode); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "The course was copied, but the learner could not be enrolled.")
			return
		}
		resp.Enrolled = true
		resp.LearnerUserID = enrollID.String()
	}
	d.notifyCourses(userID)
	writeJSON(w, http.StatusCreated, resp)
}

func (d Deps) handleUseStarterCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		outline, found := starterlibrary.Get(chi.URLParam(r, "slug"))
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Starter course not found.")
			return
		}
		var body starterUseBody
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		d.copyStarter(w, r, userID, outline, body.LearnerUserID, "library")
	}
}

func (d Deps) handleGenerateStarterCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			GradeBand     string `json:"gradeBand"`
			Subject       string `json:"subject"`
			Focus         string `json:"focus"`
			LearnerUserID string `json:"learnerUserId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		outline, source, ok := d.starterOutline(w, r, userID, body.GradeBand, body.Subject, body.Focus)
		if !ok {
			return
		}
		d.copyStarter(w, r, userID, outline, body.LearnerUserID, source)
	}
}

// starterOutline prefers a model draft and falls back to the curated template when AI is unavailable or unusable.
func (d Deps) starterOutline(w http.ResponseWriter, r *http.Request, userID uuid.UUID, band, subject, focus string) (starterlibrary.Course, string, bool) {
	template, err := starterlibrary.TemplateCourse(band, subject, focus)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return starterlibrary.Course{}, "", false
	}
	orgID := d.orgIDPtrForUser(r.Context(), userID)
	if d.Pool == nil || !d.aiConfigured(r.Context(), orgID) {
		return template, "template", true
	}
	model, mErr := userai.GetCourseSetupModelID(r.Context(), d.Pool, userID)
	if mErr != nil || strings.TrimSpace(model) == "" {
		model = userai.DefaultCourseSetupModelID
	}
	prompt := "Grade band: " + strings.TrimSpace(band) + "\nSubject: " + starterlibrary.SubjectLabel(subject)
	if strings.TrimSpace(focus) != "" {
		prompt += "\nFamily focus: " + strings.TrimSpace(focus)
	}
	if !d.enforceAIGateway(w, r, userID, aigateway.FeatureStarterLibrary, model, prompt) {
		return starterlibrary.Course{}, "", false
	}
	temp := 0.4
	bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
	result, callMeta, genErr := bound.Complete(r.Context(), model, []aiprovider.Message{
		{Role: "system", Content: starterlibrary.OutlineSystemPrompt},
		{Role: "user", Content: prompt},
	}, aiprovider.ChatOptions{JSONMode: true, MaxTokens: 1800, Temperature: &temp})
	if genErr != nil {
		return template, "template", true
	}
	parsed, parseErr := starterlibrary.ParseOutline(result.Text, strings.TrimSpace(band), strings.TrimSpace(subject))
	if parseErr != nil {
		return template, "template", true
	}
	gwDec := aigateway.Decision{
		UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
		OptInConfirmed: true,
	}
	d.logAIInferenceAllowedWithProvider(r, userID, aigateway.FeatureStarterLibrary, model, string(callMeta.Provider), prompt, gwDec)
	d.recordAIProviderUsage(r.Context(), AIUsageMeta{
		UserID: userID, Feature: aigateway.FeatureStarterLibrary, Model: model,
	}, callMeta, true)
	if strings.TrimSpace(focus) != "" {
		parsed = starterlibrary.ApplyFocus(parsed, focus)
	}
	return parsed, "ai", true
}
