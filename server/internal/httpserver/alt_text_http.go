package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	imagealtrepo "github.com/lextures/lextures/server/internal/repos/imagealtrepo"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/alttextai"
	"github.com/lextures/lextures/server/internal/service/imagealt"
)

const (
	altTextFeature       = aigateway.FeatureAltTextSuggestion
	altTextSuggestModel  = alttextai.DefaultModel
	altTextRateLimitPerH = 100
)

type altTextRateEntry struct {
	count   int
	window  time.Time
}

var altTextRateMu sync.Mutex
var altTextRateByUser = map[uuid.UUID]altTextRateEntry{}

func (d Deps) altTextEnforcementEnabled() bool {
	return d.effectiveConfig().AltTextEnforcementEnabled
}

func (d Deps) altTextHardBlockEnabled() bool {
	return d.effectiveConfig().FFAltTextEnforcement
}

func (d Deps) requireAltTextEnforcement(w http.ResponseWriter) bool {
	if !d.altTextEnforcementEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Alt-text enforcement is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerAltTextRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/alt-text/suggest", d.handlePostAltTextSuggest())
	r.Get("/api/v1/courses/{course_code}/accessibility", d.handleGetCourseAccessibility())
}

// checkAltTextRateLimit enforces the per-user hourly cap. When a shared Redis is
// configured the counter lives in Redis (rate:* namespace) so the limit holds
// across all app instances (plan 17.2 FR-3 / AC-3); otherwise it falls back to
// the per-process counter for single-instance / Redis-down operation.
func (d Deps) checkAltTextRateLimit(ctx context.Context, userID uuid.UUID) bool {
	if d.Redis != nil {
		key := "rate:alttext:" + userID.String()
		if n, err := d.Redis.IncrWindow(ctx, key, time.Hour); err == nil {
			return n <= int64(altTextRateLimitPerH)
		}
		// Redis error: fall through to the in-process limiter rather than fail closed.
	}
	altTextRateMu.Lock()
	defer altTextRateMu.Unlock()
	now := time.Now()
	e, ok := altTextRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Hour {
		altTextRateByUser[userID] = altTextRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= altTextRateLimitPerH {
		return false
	}
	e.count++
	altTextRateByUser[userID] = e
	return true
}

type altTextSuggestRequest struct {
	ImageURL string `json:"imageUrl"`
	Language string `json:"language"`
}

type altTextSuggestResponse struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

func (d Deps) handlePostAltTextSuggest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireAltTextEnforcement(w) {
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
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to suggest alt text.")
			return
		}
		if !d.checkAltTextRateLimit(r.Context(), userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Alt-text suggestion rate limit exceeded.")
			return
		}
		var req altTextSuggestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		imageURL := strings.TrimSpace(req.ImageURL)
		if imageURL == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "imageUrl is required.")
			return
		}
		or := d.openRouterClient()
		if or == nil || d.effectiveConfig().OpenRouterAPIKey == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI provider not configured.")
			return
		}
		if !d.enforceAIGateway(w, r, userID, altTextFeature, altTextSuggestModel, imageURL) {
			return
		}
		suggestion, confidence, err := alttextai.Suggest(or, altTextSuggestModel, imageURL, req.Language)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Alt-text suggestion failed.")
			return
		}
		dec := aigateway.Decision{OptInConfirmed: true}
		d.logAIInferenceAllowed(r, userID, altTextFeature, altTextSuggestModel, imageURL, dec)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(altTextSuggestResponse{
			Suggestion: suggestion,
			Confidence: confidence,
		})
	}
}

type courseAccessibilityResponse struct {
	AltTextCoverage altTextCoverageJSON `json:"altTextCoverage"`
	HardBlockSave   bool                `json:"hardBlockSave"`
}

type altTextCoverageJSON struct {
	WithAlt        int                      `json:"withAlt"`
	Total          int                      `json:"total"`
	Percent        int                      `json:"percent"`
	UncoveredItems []imagealtrepo.ItemCoverage `json:"uncoveredItems"`
}

func (d Deps) handleGetCourseAccessibility() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireAltTextEnforcement(w) {
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
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view accessibility coverage.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		items, err := imagealtrepo.ListCourseMarkdownItems(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course content.")
			return
		}
		totalWith := 0
		totalImages := 0
		var uncovered []imagealtrepo.ItemCoverage
		for _, it := range items {
			imgs := imagealt.ScanMarkdown(it.Markdown)
			cov := imagealt.Summarize(imgs)
			totalWith += cov.WithAlt
			totalImages += cov.Total
			missing := cov.Total - cov.WithAlt
			if missing > 0 {
				uncovered = append(uncovered, imagealtrepo.ItemCoverage{
					ItemID:  it.ItemID,
					Title:   it.Title,
					Kind:    it.Kind,
					WithAlt: cov.WithAlt,
					Total:   cov.Total,
					Missing: missing,
				})
			}
		}
		pct := 100
		if totalImages > 0 {
			pct = (totalWith * 100) / totalImages
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(courseAccessibilityResponse{
			AltTextCoverage: altTextCoverageJSON{
				WithAlt:        totalWith,
				Total:          totalImages,
				Percent:        pct,
				UncoveredItems: uncovered,
			},
			HardBlockSave: d.altTextHardBlockEnabled(),
		})
	}
}
