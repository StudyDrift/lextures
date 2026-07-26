// Package contenttools holds JSON shapes for Content Tools HTTP APIs (plan CT.1+).
package contenttools

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Settings is the course-scoped Content Tools configuration.
type Settings struct {
	AllowedToolIDs      []string  `json:"allowedToolIds"`
	StudentResetAllowed bool      `json:"studentResetAllowed"`
	MaxInstancesPerItem int16     `json:"maxInstancesPerItem"`
	UpdatedAt           time.Time `json:"updatedAt,omitempty"`
}

// CatalogTool is one tool available in a course after flag/allowlist/role filtering.
type CatalogTool struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Capabilities  []string `json:"capabilities"`
	I18nNamespace string   `json:"i18nNamespace"`
	UI            ToolUI   `json:"ui"`
}

// ToolUI is the manifest ui block exposed to clients.
type ToolUI struct {
	Renderer string `json:"renderer"`
	Icon     string `json:"icon"`
	Group    string `json:"group"`
}

// ToolManifestPublic is a manifest returned by GET .../manifests/{tool_id}
// (sensitive schema annotations stripped).
type ToolManifestPublic struct {
	ID            string          `json:"id"`
	Version       string          `json:"version"`
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Capabilities  []string        `json:"capabilities"`
	ConfigSchema  json.RawMessage `json:"configSchema"`
	StateSchema   json.RawMessage `json:"stateSchema"`
	Scoring       Scoring         `json:"scoring"`
	AI            *AIBlock        `json:"ai,omitempty"`
	Network       *NetworkBlock   `json:"network,omitempty"`
	Storage       StorageBlock    `json:"storage"`
	Roles         RolesBlock      `json:"roles"`
	A11y          A11yBlock       `json:"a11y"`
	I18nNamespace string          `json:"i18nNamespace"`
	UI            ToolUI          `json:"ui"`
}

// Scoring is the manifest scoring block.
type Scoring struct {
	Mode     string   `json:"mode"`
	MaxScore *float64 `json:"maxScore,omitempty"`
}

// AIBlock declares AI usage for a tool.
type AIBlock struct {
	FeatureID string `json:"featureId"`
	Required  bool   `json:"required"`
}

// NetworkBlock declares allowed network hosts.
type NetworkBlock struct {
	AllowedHosts []string `json:"allowedHosts"`
}

// StorageBlock declares state size budget.
type StorageBlock struct {
	MaxStateBytes int `json:"maxStateBytes"`
}

// RolesBlock declares who may interact.
type RolesBlock struct {
	Interact []string `json:"interact"`
}

// A11yBlock is mandatory accessibility metadata.
type A11yBlock struct {
	KeyboardOperable bool   `json:"keyboardOperable"`
	SRPattern        string `json:"srPattern"`
	WCAGNotes        string `json:"wcagNotes,omitempty"`
}

// ToolInstance is a placed tool returned by instance CRUD.
type ToolInstance struct {
	ID              uuid.UUID       `json:"id"`
	ToolID          string          `json:"toolId"`
	ToolVersion     string          `json:"toolVersion"`
	HostKind        string          `json:"hostKind"`
	StructureItemID *uuid.UUID      `json:"structureItemId"`
	SectionKey      *string         `json:"sectionKey"`
	Title           *string         `json:"title"`
	Config          json.RawMessage `json:"config"`
	Status          string          `json:"status"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// CreateInstanceRequest is POST .../instances.
type CreateInstanceRequest struct {
	ToolID          string          `json:"toolId"`
	HostKind        string          `json:"hostKind"`
	StructureItemID *uuid.UUID      `json:"structureItemId"`
	SectionKey      *string         `json:"sectionKey"`
	Title           *string         `json:"title"`
	Config          json.RawMessage `json:"config"`
}

// PatchInstanceRequest is PATCH .../instances/{id}.
type PatchInstanceRequest struct {
	Title      *string          `json:"title"`
	SectionKey *string          `json:"sectionKey"`
	Config     *json.RawMessage `json:"config"`
	Status     *string          `json:"status"`
}

// ToolInstanceUsage is GET .../instances/{id}/usage (plan CT.2 delete confirmation).
type ToolInstanceUsage struct {
	InstanceID        uuid.UUID `json:"instanceId"`
	LearnersWithState int       `json:"learnersWithState"`
	LearnersCompleted int       `json:"learnersCompleted"`
	LastInteractionAt *string   `json:"lastInteractionAt"`
	ReferencedInBody  bool      `json:"referencedInBody"`
}

// FieldError is one JSON Schema validation failure (HTTP 422).
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidationErrorBody is the 422 error payload for config validation.
type ValidationErrorBody struct {
	Error struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Errors  []FieldError `json:"errors"`
	} `json:"error"`
}
