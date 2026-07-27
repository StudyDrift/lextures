package contenttools

import (
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
)

// FilterCatalog returns tools available for a course after allowlist + role filter.
// orgPolicyDenied, when non-nil, excludes tool ids denied by org policy.
func FilterCatalog(reg *Registry, allowedToolIDs []string, interactRole string, orgPolicyDenied map[string]struct{}) []ctmodel.CatalogTool {
	if reg == nil {
		return []ctmodel.CatalogTool{}
	}
	out := make([]ctmodel.CatalogTool, 0)
	for _, m := range reg.List() {
		// FR-16 / AC-11: deprecated tools are hidden from the authoring palette.
		if m.Deprecated {
			continue
		}
		if !ToolAllowedByAllowlist(allowedToolIDs, m.ID) {
			continue
		}
		if orgPolicyDenied != nil {
			if _, denied := orgPolicyDenied[m.ID]; denied {
				continue
			}
		}
		if interactRole != "" && !roleMayInteract(m, interactRole) {
			continue
		}
		caps := m.Capabilities
		if caps == nil {
			caps = []string{}
		}
		out = append(out, ctmodel.CatalogTool{
			ID:            m.ID,
			Version:       m.Version,
			Name:          m.Name,
			Category:      m.Category,
			Capabilities:  caps,
			I18nNamespace: m.I18nNamespace,
			UI: ctmodel.ToolUI{
				Renderer: m.UI.Renderer,
				Icon:     m.UI.Icon,
				Group:    m.UI.Group,
			},
		})
	}
	return out
}

func roleMayInteract(m *CompiledManifest, role string) bool {
	for _, r := range m.Roles.Interact {
		if r == role {
			return true
		}
	}
	return false
}

// ManifestToPublic converts a compiled manifest to the public API shape with
// sensitive schema annotations stripped (FR-15).
func ManifestToPublic(m *CompiledManifest) (ctmodel.ToolManifestPublic, error) {
	cfg, err := StripSensitiveSchemaAnnotations(m.ConfigSchema)
	if err != nil {
		return ctmodel.ToolManifestPublic{}, err
	}
	state, err := StripSensitiveSchemaAnnotations(m.StateSchema)
	if err != nil {
		return ctmodel.ToolManifestPublic{}, err
	}
	caps := m.Capabilities
	if caps == nil {
		caps = []string{}
	}
	out := ctmodel.ToolManifestPublic{
		ID:           m.ID,
		Version:      m.Version,
		Name:         m.Name,
		Category:     m.Category,
		Capabilities: caps,
		ConfigSchema: cfg,
		StateSchema:  state,
		Scoring: ctmodel.Scoring{
			Mode:     m.Scoring.Mode,
			MaxScore: m.Scoring.MaxScore,
		},
		Storage: ctmodel.StorageBlock{MaxStateBytes: m.Storage.MaxStateBytes},
		Roles:   ctmodel.RolesBlock{Interact: append([]string{}, m.Roles.Interact...)},
		A11y: ctmodel.A11yBlock{
			KeyboardOperable: m.A11y.KeyboardOperable,
			SRPattern:        m.A11y.SRPattern,
			WCAGNotes:        m.A11y.WCAGNotes,
		},
		I18nNamespace:   m.I18nNamespace,
		UI: ctmodel.ToolUI{
			Renderer: m.UI.Renderer,
			Icon:     m.UI.Icon,
			Group:    m.UI.Group,
		},
		ConflictPolicy:  EffectiveConflictPolicy(m),
		AutosaveMs:      EffectiveAutosaveMs(m),
		RespectsDueDate: m.RespectsDueDate,
		AllowsSelfReset: m.AllowsSelfReset,
		Deprecated:          m.Deprecated,
		SunsetAt:            m.SunsetAt,
		Contract:            m.Contract,
		StateSchemaVersion:  m.StateSchemaVersion,
		ConfigSchemaVersion: m.ConfigSchemaVersion,
	}
	if m.Sandbox == "" {
		out.Sandbox = SandboxInProcess
	} else {
		out.Sandbox = m.Sandbox
	}
	if out.Contract <= 0 {
		out.Contract = ContractVersion
	}
	if out.StateSchemaVersion <= 0 {
		out.StateSchemaVersion = 1
	}
	if out.ConfigSchemaVersion <= 0 {
		out.ConfigSchemaVersion = 1
	}
	if len(m.Actions) > 0 {
		out.Actions = make([]ctmodel.ActionPublic, 0, len(m.Actions))
		for _, a := range m.Actions {
			out.Actions = append(out.Actions, ctmodel.ActionPublic{
				Name:            a.Name,
				RateLimitPerMin: EffectiveActionRateLimit(m, &a),
				RequiresAI:      a.RequiresAI,
				Description:     a.Description,
			})
		}
	}
	if m.AI != nil {
		out.AI = &ctmodel.AIBlock{FeatureID: m.AI.FeatureID, Required: m.AI.Required}
	}
	if m.Network != nil {
		out.Network = &ctmodel.NetworkBlock{AllowedHosts: append([]string{}, m.Network.AllowedHosts...)}
	}
	return out, nil
}
