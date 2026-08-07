package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const (
	maxAdjustDatesInstructionChars = 2000
	maxBulkDueAtUpdates            = 500
)

// handlePostBulkStructureDueAt is POST /api/v1/courses/{course_code}/structure/dates/bulk
func (d Deps) handlePostBulkStructureDueAt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit course structure.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		var body struct {
			Updates []struct {
				ItemID string     `json:"itemId"`
				DueAt  *time.Time `json:"dueAt"`
			} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Updates) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "updates is required.")
			return
		}
		if len(body.Updates) > maxBulkDueAtUpdates {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				fmt.Sprintf("Too many updates (max %d).", maxBulkDueAtUpdates))
			return
		}

		updates := make([]coursestructurerepo.DueAtUpdate, 0, len(body.Updates))
		for _, u := range body.Updates {
			id, parseErr := uuid.Parse(strings.TrimSpace(u.ItemID))
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid itemId in updates.")
				return
			}
			if u.DueAt == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dueAt is required for each update.")
				return
			}
			t := u.DueAt.UTC()
			updates = append(updates, coursestructurerepo.DueAtUpdate{ItemID: id, DueAt: &t})
		}

		updated, failed, err := coursestructurerepo.BulkPatchChildDueAt(r.Context(), d.Pool, *cid, updates)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update due dates.")
			return
		}
		d.invalidateCourseStructureCache(r.Context(), *cid)
		if d.calendarFeedsEnabled() {
			d.invalidateCourseCalendarCache(r.Context(), *cid)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]int{
			"updated": updated,
			"failed":  failed,
		})
	}
}

type adjustDatesAIProposal struct {
	ItemID string `json:"itemId"`
	DueAt  string `json:"dueAt"`
}

type adjustDatesAIResponse struct {
	Reply     string                  `json:"reply"`
	Proposals []adjustDatesAIProposal `json:"proposals"`
}

const adjustDatesAISystemPrompt = `You are an instructor assistant that bulk-adjusts due dates for LMS course items.
Given the course schedule mode, term bounds (if any), and the current list of dated items, propose new dueAt values.

Respond with ONLY a JSON object (no markdown fences):
{"reply":"short plain-text explanation for the instructor","proposals":[{"itemId":"<uuid>","dueAt":"<RFC3339>"}]}

Rules:
- Only include items that should change; omit unchanged items.
- Never invent itemIds — use only ids from the provided list.
- dueAt must be RFC3339 UTC timestamps (e.g. 2026-09-15T23:59:00Z).
- Preserve relative spacing between items unless the instructor instruction requires re-pacing.
- For relative schedule mode, still propose absolute stored timestamps (the LMS re-anchors them per enrollment).
- Prefer keeping the same local time-of-day as each original due date when shifting by whole days.
- Prefer the smallest set of sensible changes that satisfy the request.
- If no change is needed, return proposals: [] and explain why in reply.`

// handlePostAdjustDatesAI is POST /api/v1/courses/{course_code}/structure/dates/ai-adjust
func (d Deps) handlePostAdjustDatesAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit course structure.")
			return
		}

		c, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		var body struct {
			Instruction string `json:"instruction"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		instruction := strings.TrimSpace(body.Instruction)
		if utf8.RuneCountInString(instruction) > maxAdjustDatesInstructionChars {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				fmt.Sprintf("Instruction too long (max %d characters).", maxAdjustDatesInstructionChars))
			return
		}
		if instruction == "" {
			instruction = "Intelligently adjust all due dates for the upcoming term while preserving relative spacing and workload balance."
		}

		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		items, err := coursestructurerepo.ListForCourseWithEnrichment(r.Context(), d.Pool, *cid, true)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course structure.")
			return
		}

		dated := filterDatedStructureItems(items)
		if len(dated) == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
				Reply:     "There are no items with due dates to adjust.",
				Proposals: []adjustDatesAIProposal{},
			})
			return
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil || strings.TrimSpace(model) == "" {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureDatesAIAdjust, model, instruction) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		userContext := formatAdjustDatesAIContext(c, dated)
		msgs := []aiprovider.Message{
			{Role: "system", Content: adjustDatesAISystemPrompt},
			{Role: "user", Content: userContext},
			{Role: "user", Content: "Instructor request:\n" + instruction},
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		generated, callMeta, err := bound.Complete(r.Context(), model, msgs)
		if err != nil {
			writeAIGenerationFailed(w, r, "AI generation failed: "+err.Error(), err)
			return
		}
		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureDatesAIAdjust, model, string(callMeta.Provider), instruction, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureDatesAIAdjust, Model: model,
		}, callMeta, true)

		parsed, parseErr := parseAdjustDatesAIResponse(generated.Text)
		if parseErr != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
				Reply:     strings.TrimSpace(generated.Text),
				Proposals: []adjustDatesAIProposal{},
			})
			return
		}
		parsed.Proposals = sanitizeAdjustDatesAIProposals(parsed.Proposals, dated)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(parsed)
	}
}

func filterDatedStructureItems(items []coursestructurerepo.ItemResponse) []coursestructurerepo.ItemResponse {
	out := make([]coursestructurerepo.ItemResponse, 0)
	for _, it := range items {
		if it.DueAt == nil {
			continue
		}
		switch it.Kind {
		case "assignment", "quiz", "content_page":
			out = append(out, it)
		}
	}
	return out
}

func formatAdjustDatesAIContext(c *course.CoursePublic, dated []coursestructurerepo.ItemResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Course title: %s\nCourse code: %s\nSchedule mode: %s\n", c.Title, c.CourseCode, c.ScheduleMode)
	if c.RelativeScheduleAnchorAt != nil {
		fmt.Fprintf(&b, "Relative schedule anchor: %s\n", c.RelativeScheduleAnchorAt.UTC().Format(time.RFC3339))
	}
	if c.StartsAt != nil {
		fmt.Fprintf(&b, "Course startsAt: %s\n", c.StartsAt.UTC().Format(time.RFC3339))
	}
	if c.EndsAt != nil {
		fmt.Fprintf(&b, "Course endsAt: %s\n", c.EndsAt.UTC().Format(time.RFC3339))
	}
	b.WriteString("\nDated items (id | kind | title | dueAt):\n")
	for _, it := range dated {
		due := ""
		if it.DueAt != nil {
			due = it.DueAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(&b, "- %s | %s | %q | %s\n", it.ID, it.Kind, it.Title, due)
	}
	return b.String()
}

func parseAdjustDatesAIResponse(raw string) (adjustDatesAIResponse, error) {
	clean := strings.TrimSpace(raw)
	if idx := strings.Index(clean, "```json"); idx != -1 {
		clean = clean[idx+7:]
		if endIdx := strings.Index(clean, "```"); endIdx != -1 {
			clean = clean[:endIdx]
		}
	} else if idx := strings.Index(clean, "```"); idx != -1 {
		clean = clean[idx+3:]
		if endIdx := strings.Index(clean, "```"); endIdx != -1 {
			clean = clean[:endIdx]
		}
	}
	clean = strings.TrimSpace(clean)
	if !strings.HasPrefix(clean, "{") {
		if start := strings.Index(clean, "{"); start != -1 {
			if end := strings.LastIndex(clean, "}"); end > start {
				clean = clean[start : end+1]
			}
		}
	}
	var parsed adjustDatesAIResponse
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return adjustDatesAIResponse{}, err
	}
	parsed.Reply = strings.TrimSpace(parsed.Reply)
	if parsed.Reply == "" {
		parsed.Reply = "Here are the proposed due date changes."
	}
	if parsed.Proposals == nil {
		parsed.Proposals = []adjustDatesAIProposal{}
	}
	return parsed, nil
}

func sanitizeAdjustDatesAIProposals(in []adjustDatesAIProposal, dated []coursestructurerepo.ItemResponse) []adjustDatesAIProposal {
	known := map[string]time.Time{}
	for _, it := range dated {
		if it.DueAt != nil {
			known[it.ID] = it.DueAt.UTC()
		}
	}
	var out []adjustDatesAIProposal
	seen := map[string]bool{}
	for _, p := range in {
		id := strings.TrimSpace(p.ItemID)
		if id == "" || seen[id] {
			continue
		}
		orig, ok := known[id]
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(p.DueAt))
		if err != nil {
			// try RFC3339Nano / common variants
			t, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(p.DueAt))
			if err != nil {
				continue
			}
		}
		t = t.UTC()
		if t.Equal(orig) {
			continue
		}
		seen[id] = true
		out = append(out, adjustDatesAIProposal{ItemID: id, DueAt: t.Format(time.RFC3339)})
	}
	return out
}
