package toolmarket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

// Service implements CT.9 marketplace lifecycle.
type Service struct {
	Pool *pgxpool.Pool
	Cfg  config.Config
}

func (s Service) enabled() bool {
	return s.Cfg.FFContentToolMarketplace
}

// CreateToolInput creates a draft tool.
type CreateToolInput struct {
	ToolID        string
	OwnerUserID   uuid.UUID
	OwnerOrgID    *uuid.UUID
	DisplayName   string
	Summary       string
	DescriptionMD string
	SubjectTags   []string
	GradeTags     []string
	SupportURL    *string
	PrivacyURL    *string
	Visibility    string
	PricingModel  string
}

// CreateTool creates a draft marketplace tool.
func (s Service) CreateTool(ctx context.Context, in CreateToolInput) (*toolmarket.Tool, error) {
	if !s.enabled() {
		return nil, fmt.Errorf("content tool marketplace disabled")
	}
	if err := ValidateMarketplaceToolID(in.ToolID); err != nil {
		return nil, err
	}
	vis := in.Visibility
	if vis == "" {
		vis = toolmarket.VisibilityPrivate
	}
	switch vis {
	case toolmarket.VisibilityPrivate, toolmarket.VisibilityUnlisted, toolmarket.VisibilityPublic:
	default:
		return nil, fmt.Errorf("invalid visibility")
	}
	pricing := in.PricingModel
	if pricing == "" {
		pricing = toolmarket.PricingFree
	}
	// v1 only permits free (FR-16).
	if pricing != toolmarket.PricingFree {
		return nil, fmt.Errorf("v1 only supports free pricing")
	}
	t, err := toolmarket.CreateTool(ctx, s.Pool, toolmarket.CreateToolParams{
		ToolID:        in.ToolID,
		OwnerUserID:   in.OwnerUserID,
		OwnerOrgID:    in.OwnerOrgID,
		DisplayName:   in.DisplayName,
		Summary:       in.Summary,
		DescriptionMD: in.DescriptionMD,
		SubjectTags:   in.SubjectTags,
		GradeTags:     in.GradeTags,
		SupportURL:    in.SupportURL,
		PrivacyURL:    in.PrivacyURL,
		Visibility:    vis,
		PricingModel:  pricing,
	})
	if err != nil {
		return nil, err
	}
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &t.ID, nil, nil, "tool.created", &in.OwnerUserID, nil)
	return t, nil
}

// CreateReleaseInput uploads a new version.
type CreateReleaseInput struct {
	ToolID             string
	OwnerUserID        uuid.UUID
	Version            string
	ManifestJSON       json.RawMessage
	DataSheetJSON      json.RawMessage
	BundleBase64       string
	AxeStatus          string
	KeyboardTestStatus string
	I18nKeys           map[string]string
}

// CreateReleaseResult includes check outcomes.
type CreateReleaseResult struct {
	Release *toolmarket.Release `json:"release"`
	Checks  ChecksReport        `json:"checks"`
}

// CreateRelease stores a release and runs automated checks (does not submit).
func (s Service) CreateRelease(ctx context.Context, in CreateReleaseInput) (*CreateReleaseResult, error) {
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, in.ToolID)
	if err != nil {
		return nil, err
	}
	if tool == nil {
		return nil, fmt.Errorf("tool not found")
	}
	if tool.OwnerUserID == nil || *tool.OwnerUserID != in.OwnerUserID {
		return nil, fmt.Errorf("forbidden")
	}
	bundle, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.BundleBase64))
	if err != nil {
		return nil, fmt.Errorf("invalid bundleBase64")
	}
	forced, err := ForceIframeManifest(in.ManifestJSON, tool.ToolID, in.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	sheet := in.DataSheetJSON
	if len(sheet) == 0 {
		// Prefer dataSheet embedded in manifest.
		var m ctsvc.Manifest
		_ = json.Unmarshal(forced, &m)
		if m.DataSheet != nil {
			sheet, _ = json.Marshal(m.DataSheet)
		} else {
			sheet = json.RawMessage(`{}`)
		}
	}
	checks := RunAutomatedChecks(tool.ToolID, in.Version, forced, sheet, bundle, in.AxeStatus, in.KeyboardTestStatus, in.I18nKeys)
	checksJSON, _ := json.Marshal(checks)
	soak := time.Now().UTC().Add(time.Duration(toolmarket.DefaultSoakDays) * 24 * time.Hour)
	rel, err := toolmarket.CreateRelease(ctx, s.Pool, toolmarket.CreateReleaseParams{
		ToolPK:        tool.ID,
		Version:       in.Version,
		ManifestJSON:  forced,
		DataSheetJSON: sheet,
		BundleSRI:     ComputeSRI(bundle),
		BundleBytes:   len(bundle),
		ChecksJSON:    checksJSON,
		SoakUntil:     &soak,
	})
	if err != nil {
		return nil, err
	}
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &tool.ID, &rel.ID, nil, "release.created", &in.OwnerUserID, checksJSON)
	return &CreateReleaseResult{Release: rel, Checks: checks}, nil
}

// SubmitRelease moves a passing release into the human review queue (AC-1).
func (s Service) SubmitRelease(ctx context.Context, toolID, version string, ownerUserID uuid.UUID) (*toolmarket.Release, error) {
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, toolID)
	if err != nil {
		return nil, err
	}
	if tool == nil {
		return nil, fmt.Errorf("tool not found")
	}
	if tool.OwnerUserID == nil || *tool.OwnerUserID != ownerUserID {
		return nil, fmt.Errorf("forbidden")
	}
	rel, err := toolmarket.GetReleaseByVersion(ctx, s.Pool, tool.ID, version)
	if err != nil || rel == nil {
		if err == nil {
			err = fmt.Errorf("release not found")
		}
		return nil, err
	}
	var checks ChecksReport
	_ = json.Unmarshal(rel.ChecksJSON, &checks)
	if !checks.OK {
		failing := []string{}
		for _, c := range checks.Checks {
			if !c.OK {
				failing = append(failing, c.Name)
			}
		}
		return nil, &SubmitRejectedError{FailingChecks: failing}
	}
	if err := toolmarket.UpdateToolStatus(ctx, s.Pool, tool.ID, toolmarket.StatusInReview); err != nil {
		return nil, err
	}
	details, _ := json.Marshal(map[string]any{"version": version})
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &tool.ID, &rel.ID, nil, "tool.submitted", &ownerUserID, details)
	IncReviewQueue(1)
	orgID := uuid.Nil
	if tool.OwnerOrgID != nil {
		orgID = *tool.OwnerOrgID
	}
	webhooksvc.EmitAsync(s.Pool, s.Cfg, orgID, webhooks.EventToolSubmitted, map[string]any{
		"tool_id":    tool.ToolID,
		"version":    version,
		"release_id": rel.ID.String(),
	})
	return rel, nil
}

// SubmitRejectedError is returned when automated checks fail (AC-1).
type SubmitRejectedError struct {
	FailingChecks []string
}

func (e *SubmitRejectedError) Error() string {
	return "automated checks failed: " + strings.Join(e.FailingChecks, ", ")
}
