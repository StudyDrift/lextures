package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func (d Deps) registerTranscriptOrderRoutes(r chi.Router) {
	r.Get("/api/v1/transcripts/recipients", d.handleSearchTranscriptRecipients())
	r.Post("/api/v1/transcripts/orders", d.handlePostTranscriptOrder())
	r.Get("/api/v1/transcripts/orders", d.handleListTranscriptOrders())
	r.Get("/api/v1/transcripts/orders/{id}", d.handleGetTranscriptOrder())
	r.Post("/api/v1/transcripts/orders/{id}/items", d.handlePostTranscriptOrderItem())
	r.Delete("/api/v1/transcripts/orders/{id}/items/{itemId}", d.handleDeleteTranscriptOrderItem())
	r.Post("/api/v1/transcripts/orders/{id}/submit", d.handleSubmitTranscriptOrder())
	d.registerTranscriptTrackingRoutes(r)
	d.registerTranscriptConsentRoutes(r)

	r.Get("/api/v1/admin/transcripts/recipients", d.handleAdminListTranscriptRecipients())
	r.Post("/api/v1/admin/transcripts/recipients", d.handleAdminCreateTranscriptRecipient())
	r.Put("/api/v1/admin/transcripts/recipients/{id}", d.handleAdminUpdateTranscriptRecipient())
	d.registerTranscriptLifecycleRoutes(r)
}

type recipientJSON struct {
	ID           string          `json:"id"`
	OrgID        *string         `json:"orgId,omitempty"`
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	CanonicalKey *string         `json:"canonicalKey,omitempty"`
	Capabilities []string        `json:"capabilities"`
	Email        *string         `json:"email,omitempty"`
	Address      json.RawMessage `json:"address,omitempty"`
	Verified     bool            `json:"verified"`
	Active       bool            `json:"active"`
	CreatedAt    string          `json:"createdAt"`
}

func recipientToJSON(r transcriptsrepo.Recipient) recipientJSON {
	out := recipientJSON{
		ID:           r.ID.String(),
		Type:         string(r.Type),
		Name:         r.Name,
		CanonicalKey: r.CanonicalKey,
		Capabilities: r.Capabilities,
		Email:        r.Email,
		Address:      r.Address,
		Verified:     r.Verified,
		Active:       r.Active,
		CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if out.Capabilities == nil {
		out.Capabilities = []string{}
	}
	if r.OrgID != nil {
		s := r.OrgID.String()
		out.OrgID = &s
	}
	return out
}

type orderItemJSON struct {
	ID             string         `json:"id"`
	RecipientID    *string        `json:"recipientId,omitempty"`
	DocumentID     *string        `json:"documentId,omitempty"`
	DeliveryMethod string         `json:"deliveryMethod"`
	Urgency        string         `json:"urgency"`
	Status         string         `json:"status"`
	CreatedAt      string         `json:"createdAt"`
	Recipient      *recipientJSON `json:"recipient,omitempty"`
}

type orderHoldJSON struct {
	Type           string `json:"type"`
	StudentMessage string `json:"studentMessage"`
	Active         bool   `json:"active"`
}

type orderEventJSON struct {
	ID        string  `json:"id"`
	ItemID    *string `json:"itemId,omitempty"`
	FromState *string `json:"fromState,omitempty"`
	ToState   string  `json:"toState"`
	ActorID   *string `json:"actorId,omitempty"`
	Reason    *string `json:"reason,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

type orderJSON struct {
	ID               string              `json:"id"`
	Status           string              `json:"status"`
	LegacyRequestID  *string             `json:"legacyRequestId,omitempty"`
	ConsentID        *string             `json:"consentId,omitempty"`
	Consent          *consentSummaryJSON `json:"consent,omitempty"`
	RequiresGuardian bool                `json:"requiresGuardian,omitempty"`
	PaymentStatus    string              `json:"paymentStatus,omitempty"`
	PaymentRef       *string             `json:"paymentRef,omitempty"`
	TotalAmount      *int                `json:"totalAmount,omitempty"`
	Currency         *string             `json:"currency,omitempty"`
	AmountRefunded   int                 `json:"amountRefunded,omitempty"`
	CreatedAt        string              `json:"createdAt"`
	SubmittedAt      *string             `json:"submittedAt,omitempty"`
	Items            []orderItemJSON     `json:"items"`
	OnHold           bool                `json:"onHold"`
	Holds            []orderHoldJSON     `json:"holds,omitempty"`
	StudentMessage   *string             `json:"studentMessage,omitempty"`
	RejectionReason  *string             `json:"rejectionReason,omitempty"`
	Events           []orderEventJSON    `json:"events,omitempty"`
}

func orderToJSON(o *transcriptsrepo.Order) orderJSON {
	return orderToJSONExt(o, nil, nil, nil)
}

func orderToJSONExt(
	o *transcriptsrepo.Order,
	holds []transcriptsrepo.Hold,
	events []transcriptsrepo.OrderEvent,
	rejectionReason *string,
) orderJSON {
	out := orderJSON{
		ID:             o.ID.String(),
		Status:         string(o.Status),
		PaymentStatus:  string(o.PaymentStatus),
		TotalAmount:    o.TotalAmount,
		Currency:       o.Currency,
		AmountRefunded: o.AmountRefunded,
		PaymentRef:     o.PaymentRef,
		CreatedAt:      o.CreatedAt.UTC().Format(time.RFC3339),
		Items:          make([]orderItemJSON, 0, len(o.Items)),
		OnHold:         o.Status == transcriptsrepo.OrderOnHold,
	}
	if o.ConsentID != nil {
		s := o.ConsentID.String()
		out.ConsentID = &s
	}
	if o.LegacyRequestID != nil {
		s := o.LegacyRequestID.String()
		out.LegacyRequestID = &s
	}
	if o.SubmittedAt != nil {
		s := o.SubmittedAt.UTC().Format(time.RFC3339)
		out.SubmittedAt = &s
	}
	for _, it := range o.Items {
		item := orderItemJSON{
			ID:             it.ID.String(),
			DeliveryMethod: string(it.DeliveryMethod),
			Urgency:        string(it.Urgency),
			Status:         string(it.Status),
			CreatedAt:      it.CreatedAt.UTC().Format(time.RFC3339),
		}
		if it.RecipientID != nil {
			s := it.RecipientID.String()
			item.RecipientID = &s
		}
		if it.DocumentID != nil {
			s := it.DocumentID.String()
			item.DocumentID = &s
		}
		if it.Recipient != nil {
			rj := recipientToJSON(*it.Recipient)
			item.Recipient = &rj
		}
		out.Items = append(out.Items, item)
	}
	if len(holds) > 0 {
		out.Holds = make([]orderHoldJSON, 0, len(holds))
		for _, h := range holds {
			if !h.Active() {
				continue
			}
			msg := h.StudentMessageSafe()
			out.Holds = append(out.Holds, orderHoldJSON{
				Type:           string(h.Type),
				StudentMessage: msg,
				Active:         true,
			})
			if out.StudentMessage == nil {
				out.StudentMessage = &msg
			}
		}
		out.OnHold = out.OnHold || len(out.Holds) > 0
	}
	if rejectionReason != nil {
		out.RejectionReason = rejectionReason
	}
	if len(events) > 0 {
		out.Events = make([]orderEventJSON, 0, len(events))
		for _, e := range events {
			ej := orderEventJSON{
				ID:        e.ID.String(),
				FromState: e.FromState,
				ToState:   e.ToState,
				Reason:    e.Reason,
				CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
			}
			if e.ItemID != nil {
				s := e.ItemID.String()
				ej.ItemID = &s
			}
			if e.ActorID != nil {
				s := e.ActorID.String()
				ej.ActorID = &s
			}
			out.Events = append(out.Events, ej)
		}
	}
	return out
}

type adHocRecipientBody struct {
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	CanonicalKey *string         `json:"canonicalKey"`
	Capabilities []string        `json:"capabilities"`
	Email        *string         `json:"email"`
	Address      json.RawMessage `json:"address"`
}

type orderItemBody struct {
	RecipientID     *string             `json:"recipientId"`
	AdHocRecipient  *adHocRecipientBody `json:"adHocRecipient"`
	DocumentID      *string             `json:"documentId"`
	DeliveryMethod  string              `json:"deliveryMethod"`
	Urgency         string              `json:"urgency"`
}

type postOrderBody struct {
	Items []orderItemBody `json:"items"`
}

func parseOrderItemBody(body orderItemBody) (transcriptsrepo.CreateOrderItemInput, string) {
	method, ok := transcriptsrepo.ParseDeliveryMethod(body.DeliveryMethod)
	if !ok {
		return transcriptsrepo.CreateOrderItemInput{}, "deliveryMethod must be electronic_pesc, edi_speede, electronic_pdf, secure_link_email, postal_mail, or api_peer."
	}
	urgency, ok := transcriptsrepo.ParseOrderUrgency(body.Urgency)
	if !ok {
		return transcriptsrepo.CreateOrderItemInput{}, "urgency must be standard or rush."
	}
	in := transcriptsrepo.CreateOrderItemInput{
		DeliveryMethod: method,
		Urgency:        urgency,
	}
	if body.DocumentID != nil && strings.TrimSpace(*body.DocumentID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*body.DocumentID))
		if err != nil {
			return transcriptsrepo.CreateOrderItemInput{}, "documentId must be a valid UUID."
		}
		in.DocumentID = &id
	}
	if body.RecipientID != nil && strings.TrimSpace(*body.RecipientID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*body.RecipientID))
		if err != nil {
			return transcriptsrepo.CreateOrderItemInput{}, "recipientId must be a valid UUID."
		}
		in.RecipientID = &id
		return in, ""
	}
	if body.AdHocRecipient == nil {
		return transcriptsrepo.CreateOrderItemInput{}, "recipientId or adHocRecipient is required."
	}
	name := strings.TrimSpace(body.AdHocRecipient.Name)
	if name == "" {
		return transcriptsrepo.CreateOrderItemInput{}, "adHocRecipient.name is required."
	}
	if len(name) > 200 {
		return transcriptsrepo.CreateOrderItemInput{}, "adHocRecipient.name is too long."
	}
	typ := transcriptsrepo.RecipientOther
	if body.AdHocRecipient.Type != "" {
		parsed, ok := transcriptsrepo.ParseRecipientType(body.AdHocRecipient.Type)
		if !ok {
			return transcriptsrepo.CreateOrderItemInput{}, "adHocRecipient.type is invalid."
		}
		typ = parsed
	}
	adhoc := &transcriptsrepo.AdHocRecipientInput{
		Type:         typ,
		Name:         name,
		CanonicalKey: body.AdHocRecipient.CanonicalKey,
		Capabilities: body.AdHocRecipient.Capabilities,
		Address:      body.AdHocRecipient.Address,
	}
	if body.AdHocRecipient.Email != nil {
		email := strings.TrimSpace(*body.AdHocRecipient.Email)
		if email != "" {
			if _, err := mail.ParseAddress(email); err != nil {
				return transcriptsrepo.CreateOrderItemInput{}, "adHocRecipient.email must be a valid email address."
			}
			adhoc.Email = &email
		}
	}
	in.AdHoc = adhoc
	return in, ""
}

func writeOrderRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, transcriptsrepo.ErrOrderNotFound),
		errors.Is(err, transcriptsrepo.ErrOrderItemNotFound),
		errors.Is(err, transcriptsrepo.ErrRecipientNotFound),
		errors.Is(err, transcriptsrepo.ErrDocumentNotOwned):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
	case errors.Is(err, transcriptsrepo.ErrInvalidDeliveryMethod),
		errors.Is(err, transcriptsrepo.ErrDeliveryNotOrgEnabled),
		errors.Is(err, transcriptsrepo.ErrOrderEmpty),
		errors.Is(err, transcriptsrepo.ErrOrderNotDraft),
		errors.Is(err, transcriptsrepo.ErrRecipientDuplicateKey),
		errors.Is(err, transcriptsrepo.ErrIllegalOrderTransition),
		errors.Is(err, transcriptsrepo.ErrTransitionReasonRequired),
		errors.Is(err, transcriptsrepo.ErrHoldAlreadyReleased):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, transcriptsrepo.ErrHoldNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
	default:
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not process transcript order.")
	}
}

// GET /api/v1/transcripts/recipients?q=&type=
func (d Deps) handleSearchTranscriptRecipients() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		var typ *transcriptsrepo.RecipientType
		if raw := strings.TrimSpace(r.URL.Query().Get("type")); raw != "" {
			parsed, ok := transcriptsrepo.ParseRecipientType(raw)
			if !ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type is invalid.")
				return
			}
			typ = &parsed
		}
		list, err := transcriptsrepo.SearchRecipients(r.Context(), d.Pool, &orgID, q, typ, 20)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not search recipients.")
			return
		}
		out := make([]recipientJSON, 0, len(list))
		for _, item := range list {
			out = append(out, recipientToJSON(item))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"recipients": out})
	}
}

// POST /api/v1/transcripts/orders
func (d Deps) handlePostTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postOrderBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Items) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "items must contain at least one recipient.")
			return
		}
		items := make([]transcriptsrepo.CreateOrderItemInput, 0, len(body.Items))
		for _, raw := range body.Items {
			in, msg := parseOrderItemBody(raw)
			if msg != "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			items = append(items, in)
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		order, err := transcriptsrepo.CreateOrder(r.Context(), d.Pool, cfg, transcriptsrepo.CreateOrderInput{
			UserID: userID,
			OrgID:  &orgID,
			Items:  items,
		})
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"order": orderToJSON(order)})
	}
}

// GET /api/v1/transcripts/orders
func (d Deps) handleListTranscriptOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		list, err := transcriptsrepo.ListOrdersByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load transcript orders.")
			return
		}
		out := make([]orderJSON, 0, len(list))
		for i := range list {
			holds, _ := transcriptsrepo.ListActiveHoldsForUser(r.Context(), d.Pool, list[i].UserID, list[i].OrgID)
			events, _ := transcriptsrepo.ListOrderEvents(r.Context(), d.Pool, list[i].ID)
			rej := transcriptsrepo.RejectionReasonFromEvents(events)
			out = append(out, orderToJSONExt(&list[i], holds, events, rej))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"orders": out})
	}
}

// GET /api/v1/transcripts/orders/{id}
func (d Deps) handleGetTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		order, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		holds, _ := transcriptsrepo.ListActiveHoldsForUser(r.Context(), d.Pool, order.UserID, order.OrgID)
		events, _ := transcriptsrepo.ListOrderEvents(r.Context(), d.Pool, order.ID)
		rej := transcriptsrepo.RejectionReasonFromEvents(events)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": orderToJSONExt(order, holds, events, rej),
		})
	}
}

// POST /api/v1/transcripts/orders/{id}/items
func (d Deps) handlePostTranscriptOrderItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body orderItemBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in, msg := parseOrderItemBody(body)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		order, err := transcriptsrepo.AddOrderItem(r.Context(), d.Pool, cfg, orderID, userID, in)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"order": orderToJSON(order)})
	}
}

// DELETE /api/v1/transcripts/orders/{id}/items/{itemId}
func (d Deps) handleDeleteTranscriptOrderItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		order, err := transcriptsrepo.DeleteOrderItem(r.Context(), d.Pool, orderID, itemID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"order": orderToJSON(order)})
	}
}

// POST /api/v1/transcripts/orders/{id}/submit
func (d Deps) handleSubmitTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		order, err := transcriptsrepo.SubmitOrder(r.Context(), d.Pool, cfg, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"order": orderToJSON(order)})
	}
}

type adminRecipientBody struct {
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	CanonicalKey *string         `json:"canonicalKey"`
	Capabilities []string        `json:"capabilities"`
	Email        *string         `json:"email"`
	Address      json.RawMessage `json:"address"`
	Verified     *bool           `json:"verified"`
	Active       *bool           `json:"active"`
}

// GET /api/v1/admin/transcripts/recipients
func (d Deps) handleAdminListTranscriptRecipients() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		includeInactive := strings.EqualFold(r.URL.Query().Get("includeInactive"), "true")
		list, err := transcriptsrepo.ListAdminRecipients(r.Context(), d.Pool, orgID, includeInactive)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load recipients.")
			return
		}
		out := make([]recipientJSON, 0, len(list))
		for _, item := range list {
			out = append(out, recipientToJSON(item))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"recipients": out})
	}
}

// POST /api/v1/admin/transcripts/recipients
func (d Deps) handleAdminCreateTranscriptRecipient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body adminRecipientBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		typ, okType := transcriptsrepo.ParseRecipientType(body.Type)
		if !okType {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type is invalid.")
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name is required.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		rec, err := transcriptsrepo.InsertRecipient(r.Context(), d.Pool, transcriptsrepo.UpsertRecipientInput{
			OrgID:        &orgID,
			Type:         typ,
			Name:         name,
			CanonicalKey: body.CanonicalKey,
			Capabilities: body.Capabilities,
			Email:        body.Email,
			Address:      body.Address,
			Verified:     body.Verified,
			Active:       body.Active,
		})
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"recipient": recipientToJSON(*rec)})
	}
}

// PUT /api/v1/admin/transcripts/recipients/{id}
func (d Deps) handleAdminUpdateTranscriptRecipient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid recipient id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body adminRecipientBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := transcriptsrepo.UpsertRecipientInput{
			Name:         body.Name,
			CanonicalKey: body.CanonicalKey,
			Capabilities: body.Capabilities,
			Email:        body.Email,
			Address:      body.Address,
			Verified:     body.Verified,
			Active:       body.Active,
		}
		if body.Type != "" {
			typ, ok := transcriptsrepo.ParseRecipientType(body.Type)
			if !ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type is invalid.")
				return
			}
			in.Type = typ
		}
		rec, err := transcriptsrepo.UpdateRecipient(r.Context(), d.Pool, id, in)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"recipient": recipientToJSON(*rec)})
	}
}
