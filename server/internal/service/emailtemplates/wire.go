package emailtemplates

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/mail"
)

// WireMailSlotRenderer registers the delivery resolver used by mail.Send*
// wrappers so system emails consult org/system overrides without importing this
// package from mail (avoids an import cycle).
func WireMailSlotRenderer(pool *pgxpool.Pool, enabled func() bool) {
	if enabled != nil {
		DeliveryOverridesEnabled = enabled
	}
	mail.SetSlotRenderer(func(ctx context.Context, orgID *uuid.UUID, slot string, vars map[string]string, branding *mail.BrandingOpts) (mail.RenderedEmail, error) {
		if orgID != nil && *orgID != uuid.Nil {
			return RenderForDelivery(ctx, pool, *orgID, slot, vars, branding)
		}
		return RenderSystemForDelivery(ctx, pool, slot, vars, branding)
	})
}
