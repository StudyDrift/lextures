package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	ctrepo "github.com/lextures/lextures/server/internal/repos/coursetranslation"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	tmsvc "github.com/lextures/lextures/server/internal/service/translationmemory"
)

const courseTranslationAIModel = "openai/gpt-4o-mini"

func courseTranslatePerm(courseCode string) string {
	return "course:" + courseCode + ":content:translate"
}

func (d Deps) translationMemoryEnabled() bool {
	return d.effectiveConfig().TranslationMemoryEnabled
}

func (d Deps) requireTranslationMemoryEnabled(w http.ResponseWriter) bool {
	if !d.translationMemoryEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Translation memory is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireCourseTranslate(w http.ResponseWriter, r *http.Request, userID uuid.UUID, courseCode string) bool {
	ctx := r.Context()
	translatePerm := courseTranslatePerm(courseCode)
	editPerm := "course:" + courseCode + ":item:create"
	canTranslate, err := courseroles.UserHasPermission(ctx, d.Pool, userID, translatePerm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !canTranslate {
		canTranslate, err = courseroles.UserHasPermission(ctx, d.Pool, userID, editPerm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return false
		}
	}
	if !canTranslate {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to translate course content.")
		return false
	}
	return true
}

func (d Deps) registerCourseTranslationRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/translations", d.handleListCourseTranslations())
	r.Put("/api/v1/courses/{course_code}/translations/{item_id}", d.handlePutCourseTranslation())
	r.Post("/api/v1/courses/{course_code}/translations/{item_id}/ai-draft", d.handlePostCourseTranslationAIDraft())
	r.Post("/api/v1/courses/{course_code}/translations/{item_id}/publish", d.handlePostCourseTranslationPublish())
	r.Get("/api/v1/courses/{course_code}/glossary", d.handleGetCourseGlossary())
	r.Post("/api/v1/courses/{course_code}/glossary", d.handlePostCourseGlossary())
	r.Get("/api/v1/courses/{course_code}/translation-coverage", d.handleGetCourseTranslationCoverage())
	r.Patch("/api/v1/courses/{course_code}/me/content-locale", d.handlePatchMyContentLocale())
	r.Get("/api/v1/translation-memory", d.handleQueryTranslationMemory())
}

func (d Deps) resolveCourseTranslationItem(w http.ResponseWriter, r *http.Request) (courseCode string, courseID, itemID uuid.UUID, itemType ctrepo.ItemType, ok bool) {
	courseCode, _, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, "", false
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	itemType, err = ctrepo.ResolveItemType(r.Context(), d.Pool, *cid, itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Item not found.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This item type does not support translation.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	return courseCode, *cid, itemID, itemType, true
}

type translationItemJSON struct {
	ItemID                  uuid.UUID  `json:"itemId"`
	ItemType                string     `json:"itemType"`
	Title                   string     `json:"title"`
	Body                    string     `json:"body"`
	HasPublished            bool       `json:"hasPublished"`
	HasDraft                bool       `json:"hasDraft"`
	TargetLocale            string     `json:"targetLocale,omitempty"`
	TranslatedTitle         *string    `json:"translatedTitle,omitempty"`
	TranslatedBody          *string    `json:"translatedBody,omitempty"`
	IsDraft                 bool       `json:"isDraft"`
	MachineTranslationDraft bool       `json:"machineTranslationDraft"`
	PublishedAt             *time.Time `json:"publishedAt,omitempty"`
	Version                 int64      `json:"version,omitempty"`
	GlossaryMatches         []tmsvc.GlossaryMatch `json:"glossaryMatches,omitempty"`
	TMSuggestions           []ctrepo.TMMatch      `json:"tmSuggestions,omitempty"`
}

func (d Deps) handleListCourseTranslations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		targetLocale := strings.TrimSpace(r.URL.Query().Get("target_locale"))
		if targetLocale == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "target_locale is required.")
			return
		}
		if _, err := normalizeLocaleInput(targetLocale); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid target_locale.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		items, err := ctrepo.ListTranslatableItems(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list items.")
			return
		}
		cov, err := ctrepo.GetCoverage(r.Context(), d.Pool, *cid, targetLocale)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute coverage.")
			return
		}
		out := make([]translationItemJSON, 0, len(items))
		sourceLocale := "en"
		glossary, _ := ctrepo.ListGlossary(r.Context(), d.Pool, *cid, sourceLocale, targetLocale)
		gEntries := glossaryToService(glossary)
		for _, it := range items {
			row := translationItemJSON{
				ItemID:   it.ItemID,
				ItemType: string(it.ItemType),
				Title:    it.Title,
				Body:     it.Body,
			}
			tr, _ := ctrepo.GetTranslation(r.Context(), d.Pool, it.ItemID, it.ItemType, targetLocale)
			if tr != nil {
				row.TargetLocale = targetLocale
				row.TranslatedTitle = tr.TranslatedTitle
				row.TranslatedBody = tr.TranslatedBody
				row.IsDraft = tr.IsDraft
				row.MachineTranslationDraft = tr.MachineTranslationDraft
				row.PublishedAt = tr.PublishedAt
				row.Version = tr.Version
				row.HasPublished = tr.PublishedAt != nil && !tr.IsDraft
				row.HasDraft = tr.IsDraft
			}
			if len(gEntries) > 0 && it.Body != "" {
				row.GlossaryMatches = tmsvc.FindGlossaryMatches(it.Body, gEntries)
			}
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":    out,
			"coverage": cov,
		})
	}
}

func glossaryToService(rows []ctrepo.GlossaryRow) []tmsvc.GlossaryEntry {
	out := make([]tmsvc.GlossaryEntry, len(rows))
	for i, g := range rows {
		out[i] = tmsvc.GlossaryEntry{SourceTerm: g.SourceTerm, TargetTerm: g.TargetTerm}
	}
	return out
}

type putTranslationRequest struct {
	TargetLocale            string  `json:"targetLocale"`
	SourceLocale            string  `json:"sourceLocale"`
	TranslatedTitle         *string `json:"translatedTitle"`
	TranslatedBody          *string `json:"translatedBody"`
	IsDraft                 *bool   `json:"isDraft"`
	MachineTranslationDraft *bool   `json:"machineTranslationDraft"`
	Version                 *int64  `json:"version"`
}

func (d Deps) handlePutCourseTranslation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, courseID, itemID, itemType, ok := d.resolveCourseTranslationItem(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		var req putTranslationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req.TargetLocale = strings.TrimSpace(req.TargetLocale)
		if req.TargetLocale == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetLocale is required.")
			return
		}
		if _, err := normalizeLocaleInput(req.TargetLocale); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid targetLocale.")
			return
		}
		sourceLocale := strings.TrimSpace(req.SourceLocale)
		if sourceLocale == "" {
			sourceLocale = "en"
		}
		isDraft := true
		if req.IsDraft != nil {
			isDraft = *req.IsDraft
		}
		machineDraft := false
		if req.MachineTranslationDraft != nil {
			machineDraft = *req.MachineTranslationDraft
		}
		tr, err := ctrepo.UpsertTranslation(r.Context(), d.Pool, ctrepo.UpsertTranslationInput{
			SourceItemID:            itemID,
			SourceItemType:          itemType,
			SourceLocale:            sourceLocale,
			TargetLocale:            req.TargetLocale,
			TranslatedTitle:         req.TranslatedTitle,
			TranslatedBody:          req.TranslatedBody,
			IsDraft:                 isDraft,
			MachineTranslationDraft: machineDraft,
			ExpectedVersion:         req.Version,
		})
		if err != nil && strings.Contains(err.Error(), "version conflict") {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Translation was updated by another user.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save translation.")
			return
		}
		_ = courseID
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(translationRowJSON(tr))
	}
}

func translationRowJSON(tr *ctrepo.Translation) map[string]any {
	if tr == nil {
		return map[string]any{}
	}
	return map[string]any{
		"itemId":                  tr.SourceItemID,
		"itemType":                tr.SourceItemType,
		"targetLocale":            tr.TargetLocale,
		"translatedTitle":         tr.TranslatedTitle,
		"translatedBody":            tr.TranslatedBody,
		"isDraft":                 tr.IsDraft,
		"machineTranslationDraft": tr.MachineTranslationDraft,
		"publishedAt":             tr.PublishedAt,
		"version":                 tr.Version,
	}
}

func (d Deps) handlePostCourseTranslationAIDraft() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, itemID, itemType, ok := d.resolveCourseTranslationItem(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		var req struct {
			TargetLocale string `json:"targetLocale"`
			SourceLocale string `json:"sourceLocale"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req.TargetLocale = strings.TrimSpace(req.TargetLocale)
		if req.TargetLocale == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetLocale is required.")
			return
		}
		sourceLocale := strings.TrimSpace(req.SourceLocale)
		if sourceLocale == "" {
			sourceLocale = "en"
		}
		title, body, err := ctrepo.GetSourceContent(r.Context(), d.Pool, itemID, itemType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load source content.")
			return
		}
		text := strings.TrimSpace(body)
		if text == "" {
			text = strings.TrimSpace(title)
		}
		or := d.openRouterClient()
		if or == nil || d.effectiveConfig().OpenRouterAPIKey == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI provider not configured.")
			return
		}
		if !d.enforceAIGateway(w, r, userID, aigateway.FeatureContentTranslation, courseTranslationAIModel, text) {
			return
		}
		translated, _, err := callLLMTranslation(or, text, req.TargetLocale)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "AI draft translation failed.")
			return
		}
		var tTitle *string
		if strings.TrimSpace(title) != "" {
			tTitleStr, _, tErr := callLLMTranslation(or, strings.TrimSpace(title), req.TargetLocale)
			if tErr == nil {
				tTitle = &tTitleStr
			}
		}
		tBody := translated
		draft := true
		machine := true
		tr, err := ctrepo.UpsertTranslation(r.Context(), d.Pool, ctrepo.UpsertTranslationInput{
			SourceItemID:            itemID,
			SourceItemType:          itemType,
			SourceLocale:            sourceLocale,
			TargetLocale:            req.TargetLocale,
			TranslatedTitle:         tTitle,
			TranslatedBody:          &tBody,
			IsDraft:                 draft,
			MachineTranslationDraft: machine,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save AI draft.")
			return
		}
		d.logAIInferenceAllowed(r, userID, aigateway.FeatureContentTranslation, courseTranslationAIModel, text, aigateway.Decision{OptInConfirmed: true})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(translationRowJSON(tr))
	}
}

func (d Deps) handlePostCourseTranslationPublish() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, itemID, itemType, ok := d.resolveCourseTranslationItem(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		var req struct {
			TargetLocale string `json:"targetLocale"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req.TargetLocale = strings.TrimSpace(req.TargetLocale)
		if req.TargetLocale == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetLocale is required.")
			return
		}
		tr, err := ctrepo.PublishTranslation(r.Context(), d.Pool, itemID, itemType, req.TargetLocale, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Translation not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to publish translation.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(translationRowJSON(tr))
	}
}

func (d Deps) handleQueryTranslationMemory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode := strings.TrimSpace(r.URL.Query().Get("course_code"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "course_code is required.")
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		sourceLocale := strings.TrimSpace(r.URL.Query().Get("source_locale"))
		targetLocale := strings.TrimSpace(r.URL.Query().Get("target_locale"))
		text := strings.TrimSpace(r.URL.Query().Get("text"))
		if sourceLocale == "" {
			sourceLocale = "en"
		}
		if targetLocale == "" || text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "target_locale and text are required.")
			return
		}
		matches, err := ctrepo.QueryTM(r.Context(), d.Pool, sourceLocale, targetLocale, text, 8)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to query translation memory.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"matches": matches})
	}
}

func (d Deps) handleGetCourseGlossary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		sourceLocale := strings.TrimSpace(r.URL.Query().Get("source_locale"))
		targetLocale := strings.TrimSpace(r.URL.Query().Get("target_locale"))
		if sourceLocale == "" {
			sourceLocale = "en"
		}
		if targetLocale == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "target_locale is required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := ctrepo.ListGlossary(r.Context(), d.Pool, *cid, sourceLocale, targetLocale)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load glossary.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": rows})
	}
}

func (d Deps) handlePostCourseGlossary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		var req struct {
			SourceLocale string `json:"sourceLocale"`
			TargetLocale string `json:"targetLocale"`
			SourceTerm   string `json:"sourceTerm"`
			TargetTerm   string `json:"targetTerm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req.SourceTerm = strings.TrimSpace(req.SourceTerm)
		req.TargetTerm = strings.TrimSpace(req.TargetTerm)
		if req.SourceTerm == "" || req.TargetTerm == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sourceTerm and targetTerm are required.")
			return
		}
		if strings.TrimSpace(req.SourceLocale) == "" {
			req.SourceLocale = "en"
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		row, err := ctrepo.AddGlossaryEntry(r.Context(), d.Pool, *cid, req.SourceLocale, req.TargetLocale, req.SourceTerm, req.TargetTerm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save glossary entry.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(row)
	}
}

func (d Deps) handleGetCourseTranslationCoverage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if !d.requireCourseTranslate(w, r, userID, courseCode) {
			return
		}
		targetLocale := strings.TrimSpace(r.URL.Query().Get("target_locale"))
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if targetLocale != "" {
			cov, err := ctrepo.GetCoverage(r.Context(), d.Pool, *cid, targetLocale)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute coverage.")
				return
			}
			writeJSON(w, http.StatusOK, cov)
			return
		}
		locales, err := ctrepo.LocalesWithCoverage(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list locales.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"locales": locales})
	}
}

func (d Deps) handlePatchMyContentLocale() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireTranslationMemoryEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		var req struct {
			ContentLocale *string `json:"contentLocale"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.ContentLocale != nil {
			loc := strings.TrimSpace(*req.ContentLocale)
			if loc == "" {
				req.ContentLocale = nil
			} else if _, err := normalizeLocaleInput(loc); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid contentLocale.")
				return
			} else {
				req.ContentLocale = &loc
			}
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if err := ctrepo.SetEnrollmentContentLocale(r.Context(), d.Pool, *cid, viewer, req.ContentLocale); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save content locale.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// enrichModuleItemWithTranslation applies published translations for students (plan 11.5 AC-2, AC-3).
func (d Deps) enrichModuleItemWithTranslation(
	ctx context.Context,
	courseID uuid.UUID,
	itemID uuid.UUID,
	itemType ctrepo.ItemType,
	viewer uuid.UUID,
	canEdit bool,
	resp *moduleAssignmentGetResponse,
) {
	if !d.translationMemoryEnabled() || resp == nil {
		return
	}
	var targetLocale *string
	if canEdit {
		return
	}
	loc, err := ctrepo.GetEnrollmentContentLocale(ctx, d.Pool, courseID, viewer)
	if err != nil || loc == nil || strings.TrimSpace(*loc) == "" {
		return
	}
	targetLocale = loc
	tr, err := ctrepo.GetPublishedForStudent(ctx, d.Pool, itemID, itemType, *targetLocale)
	if err != nil || tr == nil {
		return
	}
	if tr.TranslatedTitle != nil && strings.TrimSpace(*tr.TranslatedTitle) != "" {
		resp.Title = *tr.TranslatedTitle
	}
	if tr.TranslatedBody != nil && strings.TrimSpace(*tr.TranslatedBody) != "" {
		resp.Markdown = *tr.TranslatedBody
	}
}
