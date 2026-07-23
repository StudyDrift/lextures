package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const (
	maxModulesAIMessageChars = 4000
	maxModulesAIHistory      = 20
)

func (d Deps) registerModulesAIRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/modules-ai/chat", d.handlePostModulesAIChat())
}

type modulesAIHistoryMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type modulesAIProposal struct {
	Op          string `json:"op"`
	Title       string `json:"title,omitempty"`
	ItemID      string `json:"itemId,omitempty"`
	ModuleID    string `json:"moduleId,omitempty"`
	ModuleTitle string `json:"moduleTitle,omitempty"`
	Published   *bool  `json:"published,omitempty"`
}

type modulesAIChatResponse struct {
	Reply     string              `json:"reply"`
	Proposals []modulesAIProposal `json:"proposals"`
}

const modulesAISystemPrompt = `You are an instructor assistant for editing a course Modules outline in an LMS.
Given the current outline and the instructor's request, respond with ONLY a JSON object (no markdown fences) of the form:
{"reply":"short explanation for the instructor","proposals":[...]}

Each proposal is one of:
- {"op":"create_module","title":"..."}
- {"op":"rename","itemId":"<uuid>","title":"..."}
- {"op":"set_published","itemId":"<uuid>","published":true|false}
- {"op":"create_content_page","moduleId":"<uuid>","title":"..."} OR {"op":"create_content_page","moduleTitle":"<exact module title>","title":"..."}
- {"op":"create_assignment","moduleId":"<uuid>","title":"..."} OR {"op":"create_assignment","moduleTitle":"<exact module title>","title":"..."}
- {"op":"create_quiz","moduleId":"<uuid>","title":"..."} OR {"op":"create_quiz","moduleTitle":"<exact module title>","title":"..."}
- {"op":"create_heading","moduleId":"<uuid>","title":"..."} OR {"op":"create_heading","moduleTitle":"<exact module title>","title":"..."}

Rules:
- You CAN create modules and then create items under them in the SAME proposals list.
- When adding items under a NEW module from this response: emit create_module first, then child ops with moduleTitle set to that exact new module title (do NOT invent UUIDs).
- When adding items under an EXISTING module: use moduleId from the outline.
- Prefer the smallest set of proposals that satisfies the request.
- Never invent item/module UUIDs. Never claim you cannot create quizzes/assignments under new modules — use moduleTitle.
- Do not propose deletes or archives.
- reply must be plain text for the instructor (not JSON).`

// handlePostModulesAIChat is POST /api/v1/courses/{course_code}/modules-ai/chat.
func (d Deps) handlePostModulesAIChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

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
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
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
		if !c.ModulesAiAssistantEnabled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Modules AI assistant is not enabled for this course.")
			return
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		var body struct {
			Message string                `json:"message"`
			History []modulesAIHistoryMsg `json:"history"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		message := strings.TrimSpace(body.Message)
		if message == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Message is required.")
			return
		}
		if utf8.RuneCountInString(message) > maxModulesAIMessageChars {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				fmt.Sprintf("Message too long (max %d characters).", maxModulesAIMessageChars))
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

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil || strings.TrimSpace(model) == "" {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureModulesAIAssistant, model, message) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, viewer),
			OptInConfirmed: true,
		}

		msgs := []aiprovider.Message{{Role: "system", Content: modulesAISystemPrompt}}
		msgs = append(msgs, aiprovider.Message{
			Role:    "user",
			Content: "Course title: " + c.Title + "\nCourse code: " + courseCode + "\n\nCurrent outline:\n" + formatModulesAIOutline(items),
		})
		hist := body.History
		if len(hist) > maxModulesAIHistory {
			hist = hist[len(hist)-maxModulesAIHistory:]
		}
		for _, h := range hist {
			role := strings.TrimSpace(h.Role)
			if role != "user" && role != "assistant" {
				continue
			}
			content := strings.TrimSpace(h.Content)
			if content == "" {
				continue
			}
			msgs = append(msgs, aiprovider.Message{Role: role, Content: content})
		}
		msgs = append(msgs, aiprovider.Message{Role: "user", Content: message})

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		generated, callMeta, err := bound.Complete(r.Context(), model, msgs)
		if err != nil {
			writeAIGenerationFailed(w, r, "AI generation failed: "+err.Error(), err)
			return
		}
		d.logAIInferenceAllowedWithProvider(r, viewer, aigateway.FeatureModulesAIAssistant, model, string(callMeta.Provider), message, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: viewer, CourseCode: courseCode, Feature: aigateway.FeatureModulesAIAssistant, Model: model,
		}, callMeta, true)

		parsed, parseErr := parseModulesAIChatResponse(generated.Text)
		if parseErr != nil {
			// Soft-fail: still return a useful reply without proposals.
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(modulesAIChatResponse{
				Reply:     strings.TrimSpace(generated.Text),
				Proposals: []modulesAIProposal{},
			})
			return
		}
		parsed.Proposals = sanitizeModulesAIProposals(parsed.Proposals, items)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(parsed)
	}
}

func formatModulesAIOutline(items []coursestructurerepo.ItemResponse) string {
	if len(items) == 0 {
		return "(empty — no modules yet)"
	}
	byParent := map[string][]coursestructurerepo.ItemResponse{}
	var modules []coursestructurerepo.ItemResponse
	for _, it := range items {
		if it.Kind == "module" {
			modules = append(modules, it)
			continue
		}
		if it.ParentID != nil {
			byParent[*it.ParentID] = append(byParent[*it.ParentID], it)
		}
	}
	var b strings.Builder
	for _, m := range modules {
		pub := "draft"
		if m.Published {
			pub = "published"
		}
		fmt.Fprintf(&b, "- module id=%s title=%q status=%s\n", m.ID, m.Title, pub)
		for _, child := range byParent[m.ID] {
			cpub := "draft"
			if child.Published {
				cpub = "published"
			}
			fmt.Fprintf(&b, "  - %s id=%s title=%q status=%s\n", child.Kind, child.ID, child.Title, cpub)
		}
	}
	return b.String()
}

func parseModulesAIChatResponse(raw string) (modulesAIChatResponse, error) {
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
	var parsed modulesAIChatResponse
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		return modulesAIChatResponse{}, err
	}
	parsed.Reply = strings.TrimSpace(parsed.Reply)
	if parsed.Reply == "" {
		parsed.Reply = "Here are the proposed outline changes."
	}
	if parsed.Proposals == nil {
		parsed.Proposals = []modulesAIProposal{}
	}
	return parsed, nil
}

func normalizeModulesAITitle(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

func sanitizeModulesAIProposals(in []modulesAIProposal, items []coursestructurerepo.ItemResponse) []modulesAIProposal {
	known := map[string]string{}
	modules := map[string]bool{}
	moduleTitles := map[string]bool{}
	for _, it := range items {
		known[it.ID] = it.Kind
		if it.Kind == "module" {
			modules[it.ID] = true
			if key := normalizeModulesAITitle(it.Title); key != "" {
				moduleTitles[key] = true
			}
		}
	}
	for _, p := range in {
		if strings.TrimSpace(p.Op) != "create_module" {
			continue
		}
		if key := normalizeModulesAITitle(p.Title); key != "" {
			moduleTitles[key] = true
		}
	}

	var creates []modulesAIProposal
	var rest []modulesAIProposal
	for _, p := range in {
		op := strings.TrimSpace(p.Op)
		title := strings.TrimSpace(p.Title)
		itemID := strings.TrimSpace(p.ItemID)
		moduleID := strings.TrimSpace(p.ModuleID)
		moduleTitle := strings.TrimSpace(p.ModuleTitle)
		switch op {
		case "create_module":
			if title == "" {
				continue
			}
			creates = append(creates, modulesAIProposal{Op: op, Title: title})
		case "rename":
			if title == "" || itemID == "" || known[itemID] == "" {
				continue
			}
			rest = append(rest, modulesAIProposal{Op: op, ItemID: itemID, Title: title})
		case "set_published":
			if itemID == "" || known[itemID] == "" || p.Published == nil {
				continue
			}
			pub := *p.Published
			rest = append(rest, modulesAIProposal{Op: op, ItemID: itemID, Published: &pub})
		case "create_content_page", "create_assignment", "create_quiz", "create_heading":
			if title == "" {
				continue
			}
			if moduleID != "" && modules[moduleID] {
				rest = append(rest, modulesAIProposal{Op: op, ModuleID: moduleID, Title: title})
				continue
			}
			if key := normalizeModulesAITitle(moduleTitle); key != "" && moduleTitles[key] {
				rest = append(rest, modulesAIProposal{Op: op, ModuleTitle: strings.TrimSpace(moduleTitle), Title: title})
				continue
			}
		default:
			continue
		}
	}
	// create_module first so clients can apply parents before children in one batch.
	return append(creates, rest...)
}
