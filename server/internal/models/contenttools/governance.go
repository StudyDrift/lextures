package contenttools

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrgPolicy is GET/PUT /api/v1/orgs/{orgId}/content-tool-policy (CT.8).
type OrgPolicy struct {
	DeniedCapabilities      []string   `json:"deniedCapabilities"`
	DeniedToolIDs           []string   `json:"deniedToolIds"`
	AllowedToolIDs          []string   `json:"allowedToolIds"`
	AIDisclosureMode        string     `json:"aiDisclosureMode"`
	FreeTextFilterAction    string     `json:"freeTextFilterAction"`
	CrisisEscalationEnabled *bool      `json:"crisisEscalationEnabled,omitempty"`
	AILogRetentionDays      int        `json:"aiLogRetentionDays"`
	UpdatedAt               *time.Time `json:"updatedAt,omitempty"`
}

// AIConsentRequest is POST .../content-tools/ai-consent.
type AIConsentRequest struct {
	ToolID   string `json:"toolId,omitempty"`
	Decision string `json:"decision"`
}

// AIConsentResponse is the consent acknowledgement payload.
type AIConsentResponse struct {
	Decision  string    `json:"decision"`
	ToolID    string    `json:"toolId,omitempty"`
	DecidedAt time.Time `json:"decidedAt"`
}

// ModerationRequest is POST .../report or .../moderate.
type ModerationRequest struct {
	Action        string     `json:"action,omitempty"`
	Category      string     `json:"category,omitempty"`
	Reason        *string    `json:"reason,omitempty"`
	ContentPath   *string    `json:"contentPath,omitempty"`
	SubjectUserID *uuid.UUID `json:"subjectUserId,omitempty"`
	StateID       *uuid.UUID `json:"stateId,omitempty"`
}

// CategoryPtr returns a pointer to Category when non-empty.
func (m ModerationRequest) CategoryPtr() *string {
	c := strings.TrimSpace(m.Category)
	if c == "" {
		return nil
	}
	return &c
}

// ModerationAction is one moderation audit row.
type ModerationAction struct {
	ID            uuid.UUID  `json:"id"`
	InstanceID    uuid.UUID  `json:"instanceId"`
	StateID       *uuid.UUID `json:"stateId,omitempty"`
	ContentPath   *string    `json:"contentPath,omitempty"`
	Action        string     `json:"action"`
	Category      *string    `json:"category,omitempty"`
	Reason        *string    `json:"reason,omitempty"`
	ActorUserID   *uuid.UUID `json:"actorUserId,omitempty"`
	SubjectUserID *uuid.UUID `json:"subjectUserId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// KillRequest is POST /api/v1/admin/content-tools/kill.
type KillRequest struct {
	Scope   string  `json:"scope"`
	Target  string  `json:"target"`
	Engaged *bool   `json:"engaged,omitempty"`
	Reason  *string `json:"reason,omitempty"`
}

// FreeTextBlockedBody is returned when a free-text submission is blocked (CT.8 AC-4).
type FreeTextBlockedBody struct {
	Error struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Guidance string `json:"guidance"`
		Category string `json:"category"`
	} `json:"error"`
}
