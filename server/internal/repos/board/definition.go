package board

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Template scopes (VC.8).
const (
	TemplateScopeBuiltin = "builtin"
	TemplateScopeCourse  = "course"
	TemplateScopeOrg     = "org"
)

// Copy modes (VC.8).
const (
	CopyModeStructure = "structure"
	CopyModeFull      = "full"
)

// TemplateDefinition is the portable board→template→board payload (FR-1 / maintainability).
type TemplateDefinition struct {
	Layout         string              `json:"layout"`
	Settings       json.RawMessage     `json:"settings,omitempty"`
	ReactionMode   string              `json:"reactionMode,omitempty"`
	Attribution    string              `json:"attribution,omitempty"`
	ModerationMode string              `json:"moderationMode,omitempty"`
	FilterAction   string              `json:"filterAction,omitempty"`
	CanPost        *bool               `json:"canPost,omitempty"`
	CanInteract    *bool               `json:"canInteract,omitempty"`
	CanArrange     *bool               `json:"canArrange,omitempty"`
	Sections       []DefinitionSection `json:"sections"`
	SeedPosts      []DefinitionPost    `json:"seedPosts"`
}

// DefinitionSection is a named section in a template (keyed for seed-post binding).
type DefinitionSection struct {
	Key       string  `json:"key"`
	Title     string  `json:"title"`
	SortIndex float64 `json:"sortIndex"`
}

// DefinitionPost is a seed/copied card without reactions, comments, or share state.
type DefinitionPost struct {
	Key         string          `json:"key,omitempty"`
	ContentType string          `json:"contentType"`
	Title       string          `json:"title,omitempty"`
	Body        json.RawMessage `json:"body,omitempty"`
	LinkURL     string          `json:"linkUrl,omitempty"`
	LinkPreview json.RawMessage `json:"linkPreview,omitempty"`
	DrawingData json.RawMessage `json:"drawingData,omitempty"`
	SectionKey  string          `json:"sectionKey,omitempty"`
	SortIndex   float64         `json:"sortIndex"`
	Position    json.RawMessage `json:"position,omitempty"`
	EventDate   *time.Time      `json:"eventDate,omitempty"`
	Lat         *float64        `json:"lat,omitempty"`
	Lng         *float64        `json:"lng,omitempty"`
	// Attachment is only populated for full-copy snapshots; built-ins use text prompts.
	Attachment *DefinitionAttachment `json:"attachment,omitempty"`
}

// DefinitionAttachment captures attachment metadata for independent re-materialization.
type DefinitionAttachment struct {
	StorageKey string `json:"storageKey"`
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	SizeBytes  int64  `json:"sizeBytes"`
	AltText    string `json:"altText"`
	ScanStatus string `json:"scanStatus"`
}

// ParseDefinition unmarshals and lightly validates a template definition.
func ParseDefinition(raw json.RawMessage) (TemplateDefinition, error) {
	var def TemplateDefinition
	if len(raw) == 0 {
		return def, fmt.Errorf("board: definition is required")
	}
	if err := json.Unmarshal(raw, &def); err != nil {
		return def, fmt.Errorf("board: invalid definition: %w", err)
	}
	layout, err := NormalizeLayout(def.Layout)
	if err != nil {
		return def, err
	}
	def.Layout = layout
	if def.ReactionMode != "" {
		mode, err := NormalizeReactionMode(def.ReactionMode)
		if err != nil {
			return def, err
		}
		def.ReactionMode = mode
	}
	if def.Attribution != "" {
		attr, err := NormalizeAttribution(def.Attribution)
		if err != nil {
			return def, err
		}
		def.Attribution = attr
	}
	if def.ModerationMode != "" {
		m, err := NormalizeModerationMode(def.ModerationMode)
		if err != nil {
			return def, err
		}
		def.ModerationMode = m
	}
	if def.FilterAction != "" {
		f, err := NormalizeFilterAction(def.FilterAction)
		if err != nil {
			return def, err
		}
		def.FilterAction = f
	}
	if def.Sections == nil {
		def.Sections = []DefinitionSection{}
	}
	if def.SeedPosts == nil {
		def.SeedPosts = []DefinitionPost{}
	}
	sectionKeys := map[string]struct{}{}
	for i := range def.Sections {
		key := strings.TrimSpace(def.Sections[i].Key)
		if key == "" {
			key = fmt.Sprintf("section-%d", i+1)
		}
		def.Sections[i].Key = key
		def.Sections[i].Title = strings.TrimSpace(def.Sections[i].Title)
		if def.Sections[i].Title == "" {
			return def, fmt.Errorf("board: section title is required")
		}
		sectionKeys[key] = struct{}{}
	}
	for i := range def.SeedPosts {
		p := &def.SeedPosts[i]
		p.ContentType = strings.TrimSpace(strings.ToLower(p.ContentType))
		if !ValidContentType(p.ContentType) {
			return def, fmt.Errorf("board: invalid seed post content_type %q", p.ContentType)
		}
		if sk := strings.TrimSpace(p.SectionKey); sk != "" {
			if _, ok := sectionKeys[sk]; !ok {
				return def, fmt.Errorf("board: seed post references unknown sectionKey %q", sk)
			}
			p.SectionKey = sk
		}
	}
	return def, nil
}

// MarshalDefinition encodes a definition to JSON.
func MarshalDefinition(def TemplateDefinition) (json.RawMessage, error) {
	if def.Sections == nil {
		def.Sections = []DefinitionSection{}
	}
	if def.SeedPosts == nil {
		def.SeedPosts = []DefinitionPost{}
	}
	if len(def.Settings) == 0 {
		def.Settings = json.RawMessage(`{}`)
	}
	b, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// BoardToDefinition snapshots a board's structure (and optionally posts) into a definition.
// Never includes reactions, comments, reports, moderation log, or share links (FR-8).
func BoardToDefinition(b Board, sections []Section, posts []Post, includePosts bool) TemplateDefinition {
	canPost := b.CanPost
	canInteract := b.CanInteract
	canArrange := b.CanArrange
	def := TemplateDefinition{
		Layout:         b.Layout,
		Settings:       b.Settings,
		ReactionMode:   b.ReactionMode,
		Attribution:    b.Attribution,
		ModerationMode: b.ModerationMode,
		FilterAction:   b.FilterAction,
		CanPost:        &canPost,
		CanInteract:    &canInteract,
		CanArrange:     &canArrange,
		Sections:       make([]DefinitionSection, 0, len(sections)),
		SeedPosts:      []DefinitionPost{},
	}
	if len(def.Settings) == 0 {
		def.Settings = json.RawMessage(`{}`)
	}
	secKeyByID := map[string]string{}
	for i, s := range sections {
		key := fmt.Sprintf("s%d", i+1)
		secKeyByID[s.ID] = key
		def.Sections = append(def.Sections, DefinitionSection{
			Key:       key,
			Title:     s.Title,
			SortIndex: s.SortIndex,
		})
	}
	if !includePosts {
		return def
	}
	for i, p := range posts {
		if p.Removed || p.Hidden {
			continue
		}
		// Skip pending/rejected student content for structure-leaning snapshots.
		if p.Status != "" && p.Status != PostStatusApproved {
			continue
		}
		dp := DefinitionPost{
			Key:         fmt.Sprintf("p%d", i+1),
			ContentType: p.ContentType,
			Title:       p.Title,
			Body:        p.Body,
			SortIndex:   p.SortIndex,
			Position:    p.Position,
			EventDate:   p.EventDate,
			Lat:         p.Lat,
			Lng:         p.Lng,
			DrawingData: p.DrawingData,
			LinkPreview: p.LinkPreview,
		}
		if p.LinkURL != nil {
			dp.LinkURL = *p.LinkURL
		}
		if p.SectionID != nil {
			if key, ok := secKeyByID[*p.SectionID]; ok {
				dp.SectionKey = key
			}
		}
		if p.Attachment != nil {
			dp.Attachment = &DefinitionAttachment{
				StorageKey: p.Attachment.StorageKey,
				FileName:   p.Attachment.FileName,
				MimeType:   p.Attachment.MimeType,
				SizeBytes:  p.Attachment.SizeBytes,
				AltText:    p.Attachment.AltText,
				ScanStatus: p.Attachment.ScanStatus,
			}
		}
		def.SeedPosts = append(def.SeedPosts, dp)
	}
	return def
}

// NormalizeCopyMode returns structure|full.
func NormalizeCopyMode(raw string) (string, error) {
	m := strings.TrimSpace(strings.ToLower(raw))
	switch m {
	case "", CopyModeStructure:
		return CopyModeStructure, nil
	case CopyModeFull:
		return CopyModeFull, nil
	default:
		return "", fmt.Errorf("board: mode must be structure or full")
	}
}

// NormalizeTemplateScope returns builtin|course|org.
func NormalizeTemplateScope(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case TemplateScopeBuiltin, TemplateScopeCourse, TemplateScopeOrg:
		return s, nil
	default:
		return "", fmt.Errorf("board: scope must be builtin, course, or org")
	}
}
