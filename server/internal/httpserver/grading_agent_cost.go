package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
)

func graderAgentCostEstimateToJSON(estimate gradingagentrepo.CostEstimate, targetSummary string) map[string]any {
	out := map[string]any{
		"submissionCount": estimate.SubmissionCount,
		"hasSample":       estimate.HasSample,
		"targetSummary":   targetSummary,
	}
	if estimate.PromptTokens != nil {
		out["estimatedPromptTokens"] = *estimate.PromptTokens
	}
	if estimate.CompletionTokens != nil {
		out["estimatedCompletionTokens"] = *estimate.CompletionTokens
	}
	if estimate.CostMinUSD != nil {
		out["estimatedCostMinUsd"] = *estimate.CostMinUSD
	}
	if estimate.CostMaxUSD != nil {
		out["estimatedCostMaxUsd"] = *estimate.CostMaxUSD
	}
	if estimate.TokensOnly {
		out["tokensOnly"] = true
	}
	return out
}

func (d Deps) handleGetGraderAgentRunEstimate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		if !d.graderAgentCostEstimateEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent cost estimates are not enabled.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, _, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || cid == nil {
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		q := r.URL.Query()
		scope := gradingagentrepo.RunScope(strings.ToLower(strings.TrimSpace(q.Get("scope"))))
		if scope == "" {
			scope = gradingagentrepo.RunScopeUngraded
		}
		var filterBody graderAgentRunFilterBody
		if sid := strings.TrimSpace(q.Get("sectionId")); sid != "" {
			filterBody.SectionID = &sid
		}
		if gid := strings.TrimSpace(q.Get("groupId")); gid != "" {
			filterBody.GroupID = &gid
		}
		if rawIDs := strings.TrimSpace(q.Get("submissionIds")); rawIDs != "" {
			filterBody.SubmissionIDs = strings.Split(rawIDs, ",")
		}
		runFilter, parseErr := parseGraderAgentRunFilterBody(&filterBody)
		if parseErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, parseErr.Error())
			return
		}
		var filterMeta *graderAgentRunFilterContext
		if d.graderAgentRunFiltersEnabled() && runFilter != nil && !runFilter.IsEmpty() {
			meta, valErr := d.validateGraderAgentRunFilter(r.Context(), *cid, itemID, courseCode, viewer, runFilter)
			if valErr != nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, valErr.Error())
				return
			}
			filterMeta = meta
		}
		overwrite := strings.EqualFold(strings.TrimSpace(q.Get("overwrite")), "true")
		submissions, runScope, resolveErr := d.resolveGraderAgentSubmissions(
			r.Context(), courseCode, *cid, itemID, viewer, scope, q.Get("submissionId"), overwrite, runFilter, d.graderAgentTextEntryGradingEnabled(),
		)
		if resolveErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, resolveErr.Error())
			return
		}
		sample, sampleErr := gradingagentrepo.GetLatestDryRunSample(r.Context(), d.Pool, cfg.ID)
		if sampleErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load cost sample.")
			return
		}
		estimate := gradingagentrepo.EstimateRunCost(len(submissions), sample)
		targetSummary := formatGraderAgentRunTargetSummary(runScope, filterMeta, len(submissions))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(graderAgentCostEstimateToJSON(estimate, targetSummary))
	}
}

func runUsageToJSON(totals gradingagentrepo.RunUsageTotals) map[string]any {
	out := map[string]any{}
	if totals.PromptTokens > 0 {
		out["promptTokens"] = totals.PromptTokens
	}
	if totals.CompletionTokens > 0 {
		out["completionTokens"] = totals.CompletionTokens
	}
	if totals.CostUSD > 0 {
		out["costUsd"] = totals.CostUSD
	}
	return out
}
