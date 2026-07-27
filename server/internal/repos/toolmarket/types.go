// Package toolmarket provides DB access for CT.9 Content Tools marketplace.
package toolmarket

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	VisibilityPrivate  = "private"
	VisibilityUnlisted = "unlisted"
	VisibilityPublic   = "public"

	PricingFree  = "free"
	PricingPaid  = "paid"
	PricingTrial = "trial"

	StatusDraft     = "draft"
	StatusInReview  = "in_review"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusSuspended = "suspended"
	StatusSunset    = "sunset"

	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"

	InstallActive    = "active"
	InstallRevoked   = "revoked"
	InstallSuspended = "suspended"

	DefaultSoakDays   = 7
	DefaultSunsetDays = 90
)

// Tool is a publishable marketplace tool.
type Tool struct {
	ID            uuid.UUID
	ToolID        string
	OwnerUserID   *uuid.UUID
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
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Release is an immutable published version.
type Release struct {
	ID             uuid.UUID
	ToolPK         uuid.UUID
	Version        string
	ManifestJSON   json.RawMessage
	DataSheetJSON  json.RawMessage
	BundleObjectID *uuid.UUID
	BundleSRI      string
	BundleBytes    int
	ChecksJSON     json.RawMessage
	ReviewStatus   string
	ReviewedBy     *uuid.UUID
	ReviewNotes    *string
	PublishedAt    *time.Time
	SunsetAt       *time.Time
	SoakUntil      *time.Time
	CreatedAt      time.Time
}

// Installation is an org-scoped install with frozen consent.
type Installation struct {
	ID                    uuid.UUID
	OrgID                 uuid.UUID
	ToolPK                uuid.UUID
	PinnedMajor           int
	CurrentVersion        string
	ConsentedCapabilities []string
	ConsentedHosts        []string
	AutoUpdateMinor       bool
	Status                string
	InstalledBy           *uuid.UUID
	InstalledAt           time.Time
	RevokedAt             *time.Time
	ToolID                string // joined from tools.tool_id when loaded with join
	DisplayName           string
}

// AccessGrant invites an org to an unlisted/private tool.
type AccessGrant struct {
	ToolPK    uuid.UUID
	OrgID     uuid.UUID
	GrantedBy *uuid.UUID
	GrantedAt time.Time
}

// BrowseFilters filters marketplace listings.
type BrowseFilters struct {
	Subject string
	Grade   string
	Query   string
	OrgID   *uuid.UUID // when set, include unlisted tools granted to this org
	Limit   int
	Offset  int
}

// Listing is a public browse card.
type Listing struct {
	ToolID       string
	DisplayName  string
	Summary      string
	SubjectTags  []string
	GradeTags    []string
	Visibility   string
	PricingModel string
	Status       string
	Version      string
	WCAGLevel    string
	Capabilities []string
	SupportURL   *string
	PrivacyURL   *string
	SunsetAt     *time.Time
}
