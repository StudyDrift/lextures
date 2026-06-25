package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/coursesections"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/groupspaces"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

type graderAgentRunFilterBody struct {
	SectionID     *string  `json:"sectionId"`
	GroupID       *string  `json:"groupId"`
	SubmissionIDs []string `json:"submissionIds"`
}

func parseGraderAgentRunFilterBody(body *graderAgentRunFilterBody) (*gradingagentrepo.RunFilter, error) {
	if body == nil {
		return nil, nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("invalid filter")
	}
	return gradingagentrepo.ParseRunFilterJSON(raw)
}

type graderAgentRunFilterContext struct {
	SectionLabel *string
	GroupLabel   *string
}

func (d Deps) graderAgentVisibleSectionIDs(
	ctx context.Context,
	courseID uuid.UUID,
	courseCode string,
	viewer uuid.UUID,
) ([]uuid.UUID, error) {
	return enrollment.GradebookStudentSectionFilter(ctx, d.Pool, courseID, courseCode, viewer, true)
}

func sectionAllowed(visible []uuid.UUID, sectionID uuid.UUID) bool {
	if len(visible) == 0 {
		return true
	}
	return slices.Contains(visible, sectionID)
}

func (d Deps) validateGraderAgentRunFilter(
	ctx context.Context,
	courseID, itemID uuid.UUID,
	courseCode string,
	viewer uuid.UUID,
	filter *gradingagentrepo.RunFilter,
) (*graderAgentRunFilterContext, error) {
	if filter == nil || filter.IsEmpty() {
		return nil, nil
	}
	visibleSections, err := d.graderAgentVisibleSectionIDs(ctx, courseID, courseCode, viewer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify section access")
	}
	meta := &graderAgentRunFilterContext{}
	if filter.SectionID != nil {
		sec, err := coursesections.GetByID(ctx, d.Pool, courseID, *filter.SectionID)
		if err != nil || sec == nil || sec.Status == "archived" {
			return nil, fmt.Errorf("section not found")
		}
		if !sectionAllowed(visibleSections, *filter.SectionID) {
			return nil, fmt.Errorf("you do not have access to that section")
		}
		label := sec.SectionCode
		if sec.Name != nil && strings.TrimSpace(*sec.Name) != "" {
			label = strings.TrimSpace(*sec.Name)
		}
		meta.SectionLabel = &label
	}
	if filter.GroupID != nil {
		group, err := groupspaces.GetGroupByCourseAndID(ctx, d.Pool, courseCode, *filter.GroupID)
		if err != nil || group == nil {
			return nil, fmt.Errorf("group not found")
		}
		name := strings.TrimSpace(group.Name)
		meta.GroupLabel = &name
	}
	if len(filter.SubmissionIDs) > 0 {
		for _, sid := range filter.SubmissionIDs {
			sub, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, courseID, sid)
			if err != nil || sub == nil || sub.ModuleItemID != itemID {
				return nil, fmt.Errorf("submission not found")
			}
		}
		if len(visibleSections) > 0 {
			rows, err := moduleassignmentsubmissions.ListForAssignmentFiltered(
				ctx, d.Pool, courseID, itemID, moduleassignmentsubmissions.GradedFilterAll,
				moduleassignmentsubmissions.ListFilter{
					SubmissionIDs:     filter.SubmissionIDs,
					VisibleSectionIDs: visibleSections,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to verify submission access")
			}
			if len(rows) != len(filter.SubmissionIDs) {
				return nil, fmt.Errorf("you do not have access to one or more selected submissions")
			}
		}
	}
	return meta, nil
}

func graderAgentListFilterFromRunFilter(filter *gradingagentrepo.RunFilter) moduleassignmentsubmissions.ListFilter {
	if filter == nil {
		return moduleassignmentsubmissions.ListFilter{}
	}
	return moduleassignmentsubmissions.ListFilter{
		SectionID:     filter.SectionID,
		GroupID:       filter.GroupID,
		SubmissionIDs: filter.SubmissionIDs,
	}
}

func formatGraderAgentRunTargetSummary(
	scope gradingagentrepo.RunScope,
	meta *graderAgentRunFilterContext,
	count int,
) string {
	scopeLabel := graderAgentScopeLabel(scope)
	if meta == nil || (meta.SectionLabel == nil && meta.GroupLabel == nil) {
		if count == 1 {
			return fmt.Sprintf("%s: 1 submission", scopeLabel)
		}
		return fmt.Sprintf("%s: %d submissions", scopeLabel, count)
	}
	target := ""
	switch {
	case meta.SectionLabel != nil && meta.GroupLabel != nil:
		target = fmt.Sprintf("%s in %s", *meta.GroupLabel, *meta.SectionLabel)
	case meta.SectionLabel != nil:
		target = *meta.SectionLabel
	case meta.GroupLabel != nil:
		target = *meta.GroupLabel
	}
	if count == 1 {
		return fmt.Sprintf("%s in %s: 1 submission", scopeLabel, target)
	}
	return fmt.Sprintf("%s in %s: %d submissions", scopeLabel, target, count)
}

func graderAgentScopeLabel(scope gradingagentrepo.RunScope) string {
	switch scope {
	case gradingagentrepo.RunScopeCurrent:
		return "Current submission"
	case gradingagentrepo.RunScopeUngraded:
		return "Ungraded"
	case gradingagentrepo.RunScopeAll:
		return "All"
	default:
		return string(scope)
	}
}

func runFilterToJSON(filter *gradingagentrepo.RunFilter) map[string]any {
	if filter == nil || filter.IsEmpty() {
		return nil
	}
	out := map[string]any{}
	if filter.SectionID != nil {
		out["sectionId"] = filter.SectionID.String()
	}
	if filter.GroupID != nil {
		out["groupId"] = filter.GroupID.String()
	}
	if len(filter.SubmissionIDs) > 0 {
		ids := make([]string, 0, len(filter.SubmissionIDs))
		for _, id := range filter.SubmissionIDs {
			ids = append(ids, id.String())
		}
		out["submissionIds"] = ids
	}
	return out
}

func runFilterToJSONFromBytes(raw []byte) map[string]any {
	parsed, err := gradingagentrepo.ParseRunFilterJSON(raw)
	if err != nil || parsed == nil {
		return nil
	}
	return runFilterToJSON(parsed)
}

func (d Deps) handleGetGraderAgentRunTarget() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentAccess(w, r)
		if !ok {
			return
		}
		if !d.graderAgentRunFiltersEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent run filters are not enabled.")
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
		if runFilter != nil && !runFilter.IsEmpty() {
			meta, valErr := d.validateGraderAgentRunFilter(r.Context(), *cid, itemID, courseCode, viewer, runFilter)
			if valErr != nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, valErr.Error())
				return
			}
			filterMeta = meta
		}
		overwrite := strings.EqualFold(strings.TrimSpace(q.Get("overwrite")), "true")
		submissions, runScope, err := d.resolveGraderAgentSubmissions(
			r.Context(), courseCode, *cid, itemID, viewer, scope, q.Get("submissionId"), overwrite, runFilter, d.graderAgentTextEntryGradingEnabled(),
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		count := len(submissions)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count":         count,
			"targetSummary": formatGraderAgentRunTargetSummary(runScope, filterMeta, count),
		})
	}
}
