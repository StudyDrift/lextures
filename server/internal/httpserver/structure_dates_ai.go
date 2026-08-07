package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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
	// adjustDatesAITimeout is longer than the shared 120s client default: bulk
	// initial schedules can be slow on free/slower course-setup models.
	adjustDatesAITimeout = 180 * time.Second
	// adjustDatesAIMaxTokens caps completion size; index-based proposals stay compact.
	adjustDatesAIMaxTokens = 6000
)

type adjustDatesAIProposal struct {
	ItemID string `json:"itemId"`
	DueAt  string `json:"dueAt"`
}

// adjustDatesAIPlan is a compact bulk schedule the model (or a deterministic
// fast path) can return instead of enumerating every item due date.
type adjustDatesAIPlan struct {
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	DurationDays int    `json:"durationDays"`
	// ApplyTo: "undated" (default when any undated) or "all".
	ApplyTo string `json:"applyTo"`
}

type adjustDatesAIResponse struct {
	Reply     string                  `json:"reply"`
	Proposals []adjustDatesAIProposal `json:"proposals"`
	// Plan is accepted from the model but never required in the HTTP response
	// after expansion (proposals are the source of truth for the client).
	Plan *adjustDatesAIPlan `json:"plan,omitempty"`
}

// adjustDatesAIRaw is the wire form the model may emit (indexes and/or plan).
type adjustDatesAIRaw struct {
	Reply     string `json:"reply"`
	Plan      *adjustDatesAIPlan
	Proposals []struct {
		I      *int   `json:"i"`
		ItemID string `json:"itemId"`
		DueAt  string `json:"dueAt"`
	} `json:"proposals"`
}

const adjustDatesAISystemPrompt = `You are an instructor assistant that sets and bulk-adjusts due dates for LMS course items.
Given the course schedule mode, term bounds (if any), and the numbered list of dateable items (some may have no due date yet), propose due dates.

Respond with ONLY a JSON object (no markdown fences). Prefer the compact plan form when scheduling many undated items evenly:

{"reply":"short plain-text explanation","plan":{"startDate":"YYYY-MM-DD","endDate":"YYYY-MM-DD","applyTo":"undated"},"proposals":[]}

Or return per-item proposals using the 1-based index "i" from the list (preferred over full UUIDs):

{"reply":"short plain-text explanation","proposals":[{"i":1,"dueAt":"2026-09-15T23:59:00Z"}]}

Rules:
- Never invent indexes or itemIds — use only numbers/ids from the provided list.
- dueAt must be RFC3339 UTC timestamps (e.g. 2026-09-15T23:59:00Z). Dates in plan may be YYYY-MM-DD (interpreted as 23:59:00Z).
- When many items have dueAt "none", prefer plan with applyTo "undated" so the server spaces them evenly in outline order — do NOT list every item.
- You MAY also include proposals for specific overrides (by "i") on top of a plan.
- For items that already have due dates: only include them in proposals if they should change; omit unchanged items.
- Preserve relative spacing between already-dated items unless the instructor asks to re-pace.
- For relative schedule mode, still propose absolute stored timestamps (the LMS re-anchors them per enrollment).
- Prefer end-of-day deadlines (23:59:00Z) unless the instructor specifies otherwise.
- Prefer the smallest set of sensible changes that satisfy the request when adjusting existing dates.
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

		dateable := filterDateableStructureItems(items)
		if len(dateable) == 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
				Reply:     "There are no assignments, quizzes, or pages that can receive due dates.",
				Proposals: []adjustDatesAIProposal{},
			})
			return
		}

		datedCount := 0
		for _, it := range dateable {
			if it.DueAt != nil {
				datedCount++
			}
		}

		// Fast path: pure initial schedules with a clear duration (e.g. "4 week course")
		// or empty guidance — no model call. Avoids free-tier timeouts on large item lists.
		if datedCount == 0 {
			if plan, reply, ok := resolveDeterministicInitialPlan(instruction, c); ok {
				proposals := expandEvenSchedule(dateable, plan, nil)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
					Reply:     reply,
					Proposals: sanitizeAdjustDatesAIProposals(proposals, dateable),
				})
				return
			}
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		aiInstruction := instruction
		if aiInstruction == "" {
			if datedCount == 0 {
				aiInstruction = "Propose a sensible initial due-date schedule for all undated items based on module/outline order and course term bounds (if any), spacing work evenly with end-of-day deadlines."
			} else if datedCount < len(dateable) {
				aiInstruction = "Set due dates for any undated items and intelligently adjust existing due dates for the upcoming term while preserving relative spacing and workload balance."
			} else {
				aiInstruction = "Intelligently adjust all due dates for the upcoming term while preserving relative spacing and workload balance."
			}
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil || strings.TrimSpace(model) == "" {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureDatesAIAdjust, model, aiInstruction) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		userContext := formatAdjustDatesAIContext(c, items, dateable)
		msgs := []aiprovider.Message{
			{Role: "system", Content: adjustDatesAISystemPrompt},
			{Role: "user", Content: userContext},
			{Role: "user", Content: "Instructor request:\n" + aiInstruction},
		}

		temp := 0.2
		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		generated, callMeta, err := bound.Complete(r.Context(), model, msgs, aiprovider.ChatOptions{
			JSONMode:    true,
			MaxTokens:   adjustDatesAIMaxTokens,
			Timeout:     adjustDatesAITimeout,
			Temperature: &temp,
		})
		if err != nil {
			// Last-resort fallback: even spacing when the model times out on an undated-heavy course.
			if isTimeoutError(err) && datedCount < len(dateable) {
				if plan, reply, ok := resolveDeterministicInitialPlan(instruction, c); ok {
					proposals := expandEvenSchedule(dateable, plan, nil)
					if len(proposals) > 0 {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
							Reply:     reply + " (Used an even schedule because the AI model timed out.)",
							Proposals: sanitizeAdjustDatesAIProposals(proposals, dateable),
						})
						return
					}
				}
				writeAIGenerationFailed(w, r,
					"The AI model took too long to respond. Try a simpler instruction (e.g. \"4 week course\"), or select a faster model in Settings → AI.",
					err)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+err.Error(), err)
			return
		}
		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureDatesAIAdjust, model, string(callMeta.Provider), aiInstruction, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureDatesAIAdjust, Model: model,
		}, callMeta, true)

		parsed, parseErr := parseAdjustDatesAIResponse(generated.Text, dateable)
		if parseErr != nil {
			// If the model returned prose only, try deterministic undated fill.
			if datedCount < len(dateable) {
				if plan, reply, ok := resolveDeterministicInitialPlan(instruction, c); ok {
					proposals := expandEvenSchedule(dateable, plan, nil)
					if len(proposals) > 0 {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
							Reply:     reply,
							Proposals: sanitizeAdjustDatesAIProposals(proposals, dateable),
						})
						return
					}
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(adjustDatesAIResponse{
				Reply:     strings.TrimSpace(generated.Text),
				Proposals: []adjustDatesAIProposal{},
			})
			return
		}
		parsed.Proposals = sanitizeAdjustDatesAIProposals(parsed.Proposals, dateable)
		// Drop plan from client payload after expansion.
		parsed.Plan = nil
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(parsed)
	}
}
