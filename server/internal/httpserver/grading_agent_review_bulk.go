package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/gradecomment"
	gradingagentrepo "github.com/lextures/lextures/server/internal/repos/gradingagent"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

const gradingAgentBulkChunkSize = 50

type postGraderAgentReviewBulkBody struct {
	Action        string                         `json:"action"`
	ResultIDs     []string                       `json:"resultIds"`
	MinConfidence *float64                       `json:"minConfidence"`
	Items         []postGraderAgentReviewBulkItem `json:"items"`
}

type postGraderAgentReviewBulkItem struct {
	ResultID     string   `json:"resultId"`
	PointsEarned *float64 `json:"pointsEarned"`
	Comment      *string  `json:"comment"`
}

type graderAgentBulkOutcome struct {
	ResultID string  `json:"resultId"`
	Status   string  `json:"status"`
	Error    *string `json:"error,omitempty"`
}

func (d Deps) graderAgentSuggestModeEnabled() bool {
	return d.effectiveConfig().GraderAgentSuggestModeEnabled
}

func (d Deps) requireGraderAgentSuggestModeAccess(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool) {
	if !d.graderAgentSuggestModeEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent suggest mode is not enabled.")
		return "", uuid.Nil, false
	}
	return d.requireGraderAgentAccess(w, r)
}

func (d Deps) handlePostGraderAgentReviewBulk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireGraderAgentSuggestModeAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil || cid == nil {
			return
		}
		cfg, err := gradingagentrepo.GetConfigByItem(r.Context(), d.Pool, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grader agent not found.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body postGraderAgentReviewBulkBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		action := strings.ToLower(strings.TrimSpace(body.Action))
		switch action {
		case "approve", "approve_all", "reject":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Action must be approve, approve_all, or reject.")
			return
		}

		held, _, err := gradingagentrepo.ListReviewQueueByConfig(r.Context(), d.Pool, cfg.ID, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load review queue.")
			return
		}

		overrideByID := make(map[uuid.UUID]postGraderAgentReviewBulkItem, len(body.Items))
		for _, item := range body.Items {
			id, parseErr := uuid.Parse(strings.TrimSpace(item.ResultID))
			if parseErr != nil {
				continue
			}
			overrideByID[id] = item
		}

		selected := make([]gradingagentrepo.ReviewQueueItem, 0)
		if len(overrideByID) > 0 {
			for _, item := range held {
				if _, ok := overrideByID[item.ID]; ok {
					selected = append(selected, item)
				}
			}
		} else if len(body.ResultIDs) > 0 {
			want := make(map[uuid.UUID]struct{}, len(body.ResultIDs))
			for _, raw := range body.ResultIDs {
				id, parseErr := uuid.Parse(strings.TrimSpace(raw))
				if parseErr != nil {
					continue
				}
				want[id] = struct{}{}
			}
			for _, item := range held {
				if _, ok := want[item.ID]; ok {
					selected = append(selected, item)
				}
			}
		} else if body.MinConfidence != nil {
			floor := *body.MinConfidence
			if floor > 1 {
				floor = floor / 100
			}
			for _, item := range held {
				if item.Confidence != nil && *item.Confidence >= floor {
					selected = append(selected, item)
				}
			}
		} else if action == "approve_all" || action == "approve" {
			selected = held
		}

		if action == "reject" && len(selected) == 0 && len(body.ResultIDs) > 0 {
			want := make(map[uuid.UUID]struct{}, len(body.ResultIDs))
			for _, raw := range body.ResultIDs {
				id, parseErr := uuid.Parse(strings.TrimSpace(raw))
				if parseErr == nil {
					want[id] = struct{}{}
				}
			}
			for _, item := range held {
				if _, ok := want[item.ID]; ok {
					selected = append(selected, item)
				}
			}
		}

		outcomes := make([]graderAgentBulkOutcome, 0, len(selected))
		for start := 0; start < len(selected); start += gradingAgentBulkChunkSize {
			end := start + gradingAgentBulkChunkSize
			if end > len(selected) {
				end = len(selected)
			}
			chunk := selected[start:end]
			for _, item := range chunk {
				outcome := graderAgentBulkOutcome{ResultID: item.ID.String()}
				switch action {
				case "approve", "approve_all":
					override := overrideByID[item.ID]
					status, applyErr := d.applyGradingAgentSuggestion(
						r.Context(), *cid, itemID, cfg, assignRow, &item.ResultRow, viewer,
						override.PointsEarned, override.Comment,
					)
					if applyErr != nil {
						msg := applyErr.Error()
						outcome.Status = "error"
						outcome.Error = &msg
					} else if status == gradingagentrepo.ItemApplied || status == gradingagentrepo.ItemOverridden {
						outcome.Status = string(status)
					} else {
						outcome.Status = "noop"
					}
				case "reject":
					updated, rejectErr := gradingagentrepo.UpdateResultStatus(
						r.Context(), d.Pool, item.ID, gradingagentrepo.ItemSkipped, nil, &viewer,
					)
					if rejectErr != nil || updated == nil {
						msg := "Failed to reject suggestion."
						if rejectErr != nil {
							msg = rejectErr.Error()
						}
						outcome.Status = "error"
						outcome.Error = &msg
					} else if updated.Status == gradingagentrepo.ItemSkipped {
						outcome.Status = string(updated.Status)
					} else {
						outcome.Status = "noop"
					}
				}
				outcomes = append(outcomes, outcome)
			}
		}

		_ = courseCode
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"outcomes": outcomes})
	}
}

func (d Deps) applyGradingAgentSuggestion(
	ctx context.Context,
	courseID, itemID uuid.UUID,
	cfg *gradingagentrepo.ConfigRow,
	assignRow *coursemoduleassignments.CourseItemAssignmentRow,
	result *gradingagentrepo.ResultRow,
	viewer uuid.UUID,
	overridePoints *float64,
	overrideComment *string,
) (gradingagentrepo.ItemStatus, error) {
	if result == nil {
		return "", fmt.Errorf("result not found")
	}
	if result.ConfigID != cfg.ID {
		return "", fmt.Errorf("result not found")
	}
	if result.Status == gradingagentrepo.ItemApplied || result.Status == gradingagentrepo.ItemOverridden {
		return result.Status, nil
	}
	if result.Status != gradingagentrepo.ItemSuggested {
		return "", fmt.Errorf("result is not a held suggestion")
	}
	points := result.SuggestedPoints
	if overridePoints != nil {
		points = overridePoints
	}
	if points == nil {
		return "", fmt.Errorf("missing suggested score")
	}
	comment := result.Comment
	edited := overridePoints != nil || overrideComment != nil
	if overrideComment != nil {
		trimmed := strings.TrimSpace(*overrideComment)
		comment = &trimmed
	}
	subRow, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, courseID, result.SubmissionID)
	if err != nil || subRow == nil {
		return "", fmt.Errorf("submission not found")
	}
	posting := gradingAgentCellPosting(assignRow.PostingPolicy, cfg.PostPolicy)
	var flatComment *string
	var commentsJSON []byte
	if comment != nil && strings.TrimSpace(*comment) != "" {
		_, commentsJSON, flatComment, _ = gradecomment.Append(nil, gradecomment.Comment{
			DisplayName: "Grading agent",
			Body:        strings.TrimSpace(*comment),
			Source:      "lextures",
		})
	}
	if err := coursegrades.UpsertCellWithFlags(
		ctx, d.Pool, courseID, subRow.SubmittedBy, itemID,
		*points, result.SuggestedRubric, flatComment, commentsJSON, posting, true,
	); err != nil {
		return "", err
	}
	status := gradingagentrepo.ItemApplied
	if edited {
		status = gradingagentrepo.ItemOverridden
	}
	updated, err := gradingagentrepo.UpdateResultStatus(ctx, d.Pool, result.ID, status, nil, &viewer)
	if err != nil || updated == nil {
		return "", fmt.Errorf("failed to update result")
	}
	return updated.Status, nil
}