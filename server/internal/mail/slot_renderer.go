package mail

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// SlotRenderer resolves a slot through org/system overrides then code default.
// Wired by emailtemplates.WireMailSlotRenderer to avoid import cycles.
type SlotRenderer func(ctx context.Context, orgID *uuid.UUID, slot string, vars map[string]string, branding *BrandingOpts) (RenderedEmail, error)

var slotRenderer SlotRenderer

// SetSlotRenderer registers the delivery-time template resolver.
func SetSlotRenderer(r SlotRenderer) {
	slotRenderer = r
}

// RenderSlot uses the wired resolver when present; otherwise falls back to
// built-in RenderTemplate.
func RenderSlot(ctx context.Context, orgID *uuid.UUID, slot string, vars map[string]string, branding *BrandingOpts) (RenderedEmail, error) {
	if slotRenderer != nil {
		r, err := slotRenderer(ctx, orgID, slot, vars, branding)
		if err == nil && (strings.TrimSpace(r.HTMLBody) != "" || strings.TrimSpace(r.BodyText) != "") {
			return r, nil
		}
	}
	return RenderTemplate(slot, vars, branding)
}
