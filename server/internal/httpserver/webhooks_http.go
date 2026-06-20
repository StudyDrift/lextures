package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/organization"
	webhooksrepo "github.com/lextures/lextures/server/internal/repos/webhooks"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

type webhookSubscriptionJSON struct {
	ID            string     `json:"id"`
	OrgID         string     `json:"orgId"`
	Label         string     `json:"label"`
	EndpointURL   string     `json:"endpointUrl"`
	EventTypes    []string   `json:"eventTypes"`
	Active        bool       `json:"active"`
	PausedAt      *time.Time `json:"pausedAt,omitempty"`
	TLSSkipVerify bool       `json:"tlsSkipVerify"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Status        string     `json:"status"`
}

type webhookDeliveryJSON struct {
	ID             int64      `json:"id"`
	EventType      string     `json:"eventType"`
	EventID        string     `json:"eventId"`
	AttemptCount   int        `json:"attemptCount"`
	Status         string     `json:"status"`
	LastHTTPStatus *int       `json:"lastHttpStatus,omitempty"`
	LastResponse   *string    `json:"lastResponse,omitempty"`
	LatencyMS      *int       `json:"latencyMs,omitempty"`
	NextRetryAt    *time.Time `json:"nextRetryAt,omitempty"`
	DeliveredAt    *time.Time `json:"deliveredAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Test           bool       `json:"test,omitempty"`
}

func (d Deps) webhooksFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFWebhooks {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Outbound webhooks are not enabled.")
		return true
	}
	return false
}

func subscriptionStatus(s *webhooksrepo.Subscription) string {
	if s == nil {
		return "unknown"
	}
	if s.PausedAt != nil || !s.Active {
		return "paused"
	}
	return "active"
}

func subscriptionToJSON(s *webhooksrepo.Subscription) webhookSubscriptionJSON {
	return webhookSubscriptionJSON{
		ID:            s.ID.String(),
		OrgID:         s.OrgID.String(),
		Label:         s.Label,
		EndpointURL:   s.EndpointURL,
		EventTypes:    s.EventTypes,
		Active:        s.Active,
		PausedAt:      s.PausedAt,
		TLSSkipVerify: s.TLSSkipVerify,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		Status:        subscriptionStatus(s),
	}
}

func deliveryToJSON(row webhooksrepo.Delivery) webhookDeliveryJSON {
	out := webhookDeliveryJSON{
		ID:             row.ID,
		EventType:      row.EventType,
		EventID:        row.EventID.String(),
		AttemptCount:   row.AttemptCount,
		Status:         row.Status,
		LastHTTPStatus: row.LastHTTPStatus,
		LastResponse:   row.LastResponse,
		LatencyMS:      row.LatencyMS,
		NextRetryAt:    row.NextRetryAt,
		DeliveredAt:    row.DeliveredAt,
		CreatedAt:      row.CreatedAt,
	}
	if row.LastResponse != nil && strings.Contains(*row.LastResponse, `"test":true`) {
		out.Test = true
	}
	return out
}

func (d Deps) requireWebhooksManage(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) (actor uuid.UUID, ok bool) {
	if tok, hasTok := auth.APITokenFromContext(r.Context()); hasTok {
		userID, uok := d.meUserID(w, r)
		if !uok {
			return uuid.UUID{}, false
		}
		if !scopeAllowed(tok.Scopes, "webhooks:manage") {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "API token missing webhooks:manage scope.")
			return uuid.UUID{}, false
		}
		uOrg, err := organizationOrgIDForUser(r, d, userID)
		if err != nil || uOrg != orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this organization.")
			return uuid.UUID{}, false
		}
		return userID, true
	}
	return d.orgRoleAccess(w, r, orgID, true)
}

func scopeAllowed(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

func organizationOrgIDForUser(r *http.Request, d Deps, userID uuid.UUID) (uuid.UUID, error) {
	return organizationOrgLookup(r.Context(), d.Pool, userID)
}

func (d Deps) registerWebhookRoutes(r chi.Router) {
	r.Get("/api/v1/webhooks/event-types", d.handleListWebhookEventTypes())
	r.Get("/api/v1/webhooks", d.handleListWebhooks())
	r.Post("/api/v1/webhooks", d.handleCreateWebhook())
	r.Get("/api/v1/webhooks/{id}", d.handleGetWebhook())
	r.Put("/api/v1/webhooks/{id}", d.handleUpdateWebhook())
	r.Delete("/api/v1/webhooks/{id}", d.handleDeleteWebhook())
	r.Post("/api/v1/webhooks/{id}/test", d.handleTestWebhook())
	r.Get("/api/v1/webhooks/{id}/deliveries", d.handleListWebhookDeliveries())

	r.Get("/api/v1/admin/orgs/{orgId}/webhooks/event-types", d.handleListWebhookEventTypes())
	r.Get("/api/v1/admin/orgs/{orgId}/webhooks", d.handleAdminListWebhooks())
	r.Post("/api/v1/admin/orgs/{orgId}/webhooks", d.handleAdminCreateWebhook())
	r.Get("/api/v1/admin/orgs/{orgId}/webhooks/{id}", d.handleAdminGetWebhook())
	r.Put("/api/v1/admin/orgs/{orgId}/webhooks/{id}", d.handleAdminUpdateWebhook())
	r.Delete("/api/v1/admin/orgs/{orgId}/webhooks/{id}", d.handleAdminDeleteWebhook())
	r.Post("/api/v1/admin/orgs/{orgId}/webhooks/{id}/test", d.handleAdminTestWebhook())
	r.Get("/api/v1/admin/orgs/{orgId}/webhooks/{id}/deliveries", d.handleAdminListWebhookDeliveries())
}

func (d Deps) handleListWebhookEventTypes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"eventTypes": webhooks.AllEventTypes(),
			"groups":     webhooks.EventGroups(),
		})
	}
}

func (d Deps) resolveWebhookOrg(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	orgID, err := organizationOrgLookup(r.Context(), d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
		return uuid.UUID{}, false
	}
	if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
		return uuid.UUID{}, false
	}
	return orgID, true
}

func (d Deps) handleListWebhooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		d.writeWebhookList(w, r, orgID)
	}
}

func (d Deps) writeWebhookList(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	list, err := webhooksrepo.ListByOrg(r.Context(), d.Pool, orgID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list webhooks.")
		return
	}
	out := make([]webhookSubscriptionJSON, 0, len(list))
	for i := range list {
		out = append(out, subscriptionToJSON(&list[i]))
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"subscriptions": out})
}

func (d Deps) handleCreateWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		d.createWebhook(w, r, orgID)
	}
}

func (d Deps) createWebhook(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	var body struct {
		Label         string   `json:"label"`
		EndpointURL   string   `json:"endpointUrl"`
		EventTypes    []string `json:"eventTypes"`
		TLSSkipVerify *bool    `json:"tlsSkipVerify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return
	}
	label := strings.TrimSpace(body.Label)
	if label == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "label is required.")
		return
	}
	endpoint := strings.TrimSpace(body.EndpointURL)
	if err := webhooks.ValidateEndpointURL(endpoint); err != nil {
		if errors.Is(err, webhooks.ErrSSRFPolicy) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Endpoint URL blocked by SSRF policy: private and loopback addresses are not allowed.")
			return
		}
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return
	}
	eventTypes, ok := webhooks.NormalizeEventTypes(body.EventTypes)
	if !ok || len(eventTypes) == 0 {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide at least one valid event type.")
		return
	}
	cfg := d.effectiveConfig()
	if len(cfg.PlatformSecretsKey) != 32 {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Platform secrets key not configured.")
		return
	}
	signingKey, err := webhooks.GenerateSigningKey()
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate signing key.")
		return
	}
	keyEnc, err := webhooks.EncryptSigningKey(signingKey, cfg.PlatformSecretsKey)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to encrypt signing key.")
		return
	}
	tlsSkip := false
	if body.TLSSkipVerify != nil {
		tlsSkip = *body.TLSSkipVerify
	}
	var createdBy *uuid.UUID
	if actor, ok := d.meUserID(w, r); ok {
		createdBy = &actor
	}
	sub, err := webhooksrepo.Create(r.Context(), d.Pool, webhooksrepo.CreateInput{
		OrgID: orgID, Label: label, EndpointURL: endpoint, SigningKeyEnc: keyEnc,
		EventTypes: eventTypes, TLSSkipVerify: tlsSkip, CreatedBy: createdBy,
	})
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create webhook subscription.")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"subscription": subscriptionToJSON(sub),
		"signingKey":   string(signingKey),
	})
}

func (d Deps) parseWebhookID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid webhook id.")
		return uuid.UUID{}, false
	}
	return id, true
}

func (d Deps) handleGetWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.writeWebhookGet(w, r, orgID, id)
	}
}

func (d Deps) writeWebhookGet(w http.ResponseWriter, r *http.Request, orgID, id uuid.UUID) {
	sub, err := webhooksrepo.GetByID(r.Context(), d.Pool, orgID, id)
	if err != nil || sub == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Webhook subscription not found.")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"subscription": subscriptionToJSON(sub)})
}

func (d Deps) handleUpdateWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.updateWebhook(w, r, orgID, id)
	}
}

func (d Deps) updateWebhook(w http.ResponseWriter, r *http.Request, orgID, id uuid.UUID) {
	var body struct {
		Label         *string  `json:"label"`
		EndpointURL   *string  `json:"endpointUrl"`
		EventTypes    []string `json:"eventTypes"`
		Active        *bool    `json:"active"`
		Reactivate    *bool    `json:"reactivate"`
		TLSSkipVerify *bool    `json:"tlsSkipVerify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return
	}
	in := webhooksrepo.UpdateInput{TLSSkipVerify: body.TLSSkipVerify}
	if body.Label != nil {
		label := strings.TrimSpace(*body.Label)
		if label == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "label cannot be empty.")
			return
		}
		in.Label = &label
	}
	if body.EndpointURL != nil {
		endpoint := strings.TrimSpace(*body.EndpointURL)
		if err := webhooks.ValidateEndpointURL(endpoint); err != nil {
			if errors.Is(err, webhooks.ErrSSRFPolicy) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Endpoint URL blocked by SSRF policy: private and loopback addresses are not allowed.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		in.EndpointURL = &endpoint
	}
	if body.EventTypes != nil {
		eventTypes, ok := webhooks.NormalizeEventTypes(body.EventTypes)
		if !ok || len(eventTypes) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide at least one valid event type.")
			return
		}
		in.EventTypes = eventTypes
	}
	if body.Active != nil {
		in.Active = body.Active
	}
	if body.Reactivate != nil && *body.Reactivate {
		in.Reactivate = true
	}
	sub, err := webhooksrepo.Update(r.Context(), d.Pool, orgID, id, in)
	if err != nil || sub == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Webhook subscription not found.")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"subscription": subscriptionToJSON(sub)})
}

func (d Deps) handleDeleteWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.deleteWebhook(w, r, orgID, id)
	}
}

func (d Deps) deleteWebhook(w http.ResponseWriter, r *http.Request, orgID, id uuid.UUID) {
	ok, err := webhooksrepo.Delete(r.Context(), d.Pool, orgID, id)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete webhook subscription.")
		return
	}
	if !ok {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Webhook subscription not found.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) handleTestWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.testWebhook(w, r, orgID, id)
	}
}

func (d Deps) testWebhook(w http.ResponseWriter, r *http.Request, orgID, id uuid.UUID) {
	var body struct {
		EventType string `json:"eventType"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	eventType := webhooks.EventType(strings.TrimSpace(body.EventType))
	if eventType == "" {
		eventType = webhooks.EventGradePosted
	}
	if _, ok := webhooks.ValidEventTypes()[string(eventType)]; !ok {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid event type.")
		return
	}
	sub, err := webhooksrepo.GetByID(r.Context(), d.Pool, orgID, id)
	if err != nil || sub == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Webhook subscription not found.")
		return
	}
	delivery, err := webhooksvc.DeliverTest(r.Context(), d.Pool, d.effectiveConfig(), sub, eventType)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Test delivery failed.")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"delivery": deliveryToJSON(*delivery)})
}

func (d Deps) handleListWebhookDeliveries() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.resolveWebhookOrg(w, r)
		if !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.listWebhookDeliveries(w, r, orgID, id)
	}
}

func (d Deps) listWebhookDeliveries(w http.ResponseWriter, r *http.Request, orgID, id uuid.UUID) {
	sub, err := webhooksrepo.GetByID(r.Context(), d.Pool, orgID, id)
	if err != nil || sub == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Webhook subscription not found.")
		return
	}
	rows, err := webhooksrepo.ListDeliveries(r.Context(), d.Pool, id, 100)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load delivery log.")
		return
	}
	out := make([]webhookDeliveryJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, deliveryToJSON(row))
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"deliveries": out})
}

func (d Deps) parseAdminOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
		return uuid.UUID{}, false
	}
	return orgID, true
}

func (d Deps) handleAdminListWebhooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		d.writeWebhookList(w, r, orgID)
	}
}

func (d Deps) handleAdminCreateWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		d.createWebhook(w, r, orgID)
	}
}

func (d Deps) handleAdminGetWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.writeWebhookGet(w, r, orgID, id)
	}
}

func (d Deps) handleAdminUpdateWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.updateWebhook(w, r, orgID, id)
	}
}

func (d Deps) handleAdminDeleteWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.deleteWebhook(w, r, orgID, id)
	}
}

func (d Deps) handleAdminTestWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.testWebhook(w, r, orgID, id)
	}
}

func (d Deps) handleAdminListWebhookDeliveries() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.webhooksFeatureOff(w) {
			return
		}
		orgID, ok := d.parseAdminOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireWebhooksManage(w, r, orgID); !ok {
			return
		}
		id, ok := d.parseWebhookID(w, r)
		if !ok {
			return
		}
		d.listWebhookDeliveries(w, r, orgID, id)
	}
}

func organizationOrgLookup(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (uuid.UUID, error) {
	return organization.OrgIDForUser(ctx, pool, userID)
}
