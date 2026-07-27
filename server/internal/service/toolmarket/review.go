package toolmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

// ReviewDecisionInput approves or rejects a release.
type ReviewDecisionInput struct {
	ReleaseID  uuid.UUID
	ReviewerID uuid.UUID
	Approve    bool
	Notes      string
}

// DecideReview applies human review (FR-3).
func (s Service) DecideReview(ctx context.Context, in ReviewDecisionInput) (*toolmarket.Release, error) {
	rel, err := toolmarket.GetRelease(ctx, s.Pool, in.ReleaseID)
	if err != nil || rel == nil {
		if err == nil {
			err = fmt.Errorf("release not found")
		}
		return nil, err
	}
	status := toolmarket.ReviewRejected
	toolStatus := toolmarket.StatusRejected
	event := webhooks.EventToolRejected
	action := "tool.rejected"
	if in.Approve {
		status = toolmarket.ReviewApproved
		toolStatus = toolmarket.StatusApproved
		event = webhooks.EventToolApproved
		action = "tool.approved"
		if strings.TrimSpace(in.Notes) == "" {
			in.Notes = "approved"
		}
	} else if strings.TrimSpace(in.Notes) == "" {
		return nil, fmt.Errorf("rejection requires notes")
	}
	updated, err := toolmarket.DecideRelease(ctx, s.Pool, in.ReleaseID, status, in.ReviewerID, in.Notes)
	if err != nil {
		return nil, err
	}
	if err := toolmarket.UpdateToolStatus(ctx, s.Pool, rel.ToolPK, toolStatus); err != nil {
		return nil, err
	}
	IncReviewQueue(-1)
	details, _ := json.Marshal(map[string]any{"notes": in.Notes, "status": status})
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &rel.ToolPK, &rel.ID, nil, action, &in.ReviewerID, details)
	tool, _ := toolmarket.GetToolByPK(ctx, s.Pool, rel.ToolPK)
	orgID := uuid.Nil
	if tool != nil && tool.OwnerOrgID != nil {
		orgID = *tool.OwnerOrgID
	}
	toolID := ""
	if tool != nil {
		toolID = tool.ToolID
	}
	webhooksvc.EmitAsync(s.Pool, s.Cfg, orgID, event, map[string]any{
		"tool_id":    toolID,
		"version":    rel.Version,
		"release_id": rel.ID.String(),
	})
	return updated, nil
}
