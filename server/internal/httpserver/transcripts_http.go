package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/lextures/lextures/server/internal/repos/user"
)

func (d Deps) transcriptsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFTranscripts {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Transcripts is not enabled.")
		return true
	}
	return false
}

type transcriptStudentJSON struct {
	UserID      string  `json:"userId"`
	Email       string  `json:"email"`
	FirstName   *string `json:"firstName,omitempty"`
	LastName    *string `json:"lastName,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	StudentID   *string `json:"studentId,omitempty"`
}

type transcriptDeliveryJSON struct {
	Type            string  `json:"type"`
	Email           *string `json:"email,omitempty"`
	Address         *string `json:"address,omitempty"`
	UrgencyDays     int     `json:"urgencyDays,omitempty"`
	UrgencyDaysMin  *int    `json:"urgencyDaysMin,omitempty"`
	UrgencyUnit     string  `json:"urgencyUnit,omitempty"`
}

type transcriptWebhookPayload struct {
	RequestID   string                 `json:"requestId"`
	RequestedAt string                 `json:"requestedAt"`
	Delivery    transcriptDeliveryJSON `json:"delivery"`
	Student     transcriptStudentJSON  `json:"student"`
}

type transcriptRequestJSON struct {
	ID                  string  `json:"id"`
	Status              string  `json:"status"`
	DeliveryType        string  `json:"deliveryType"`
	DeliveryEmail       *string `json:"deliveryEmail,omitempty"`
	DeliveryAddress     *string `json:"deliveryAddress,omitempty"`
	UrgencyDays         int     `json:"urgencyDays,omitempty"`
	UrgencyDaysMin      *int    `json:"urgencyDaysMin,omitempty"`
	UrgencyUnit         string  `json:"urgencyUnit,omitempty"`
	RequestedAt         string  `json:"requestedAt"`
	SubmittedAt         *string `json:"submittedAt,omitempty"`
	ErrorMessage        *string `json:"errorMessage,omitempty"`
	WebhookResponseCode *int    `json:"webhookResponseCode,omitempty"`
}

func requestToJSON(r transcriptsrepo.Request) transcriptRequestJSON {
	out := transcriptRequestJSON{
		ID:                  r.ID.String(),
		Status:              string(r.Status),
		DeliveryType:        string(r.DeliveryType),
		DeliveryEmail:       r.DeliveryEmail,
		DeliveryAddress:     r.DeliveryAddress,
		RequestedAt:         r.RequestedAt.UTC().Format(time.RFC3339),
		ErrorMessage:        r.ErrorMessage,
		WebhookResponseCode: r.WebhookResponseCode,
	}
	if r.DeliveryType != transcriptsrepo.DeliveryEmail {
		out.UrgencyDays = r.UrgencyDays
		out.UrgencyDaysMin = r.UrgencyDaysMin
		out.UrgencyUnit = string(r.UrgencyUnit)
	}
	if r.SubmittedAt != nil {
		s := r.SubmittedAt.UTC().Format(time.RFC3339)
		out.SubmittedAt = &s
	}
	return out
}

type transcriptsConfigJSON struct {
	WebhookURL              string  `json:"webhookUrl"`
	WebhookSecret           string  `json:"webhookSecret"`
	HasWebhookSecret        bool    `json:"hasWebhookSecret"`
	PickupInstructions      *string `json:"pickupInstructions,omitempty"`
	OfficialEnabled         bool    `json:"officialEnabled"`
	OrdersUIEnabled         bool    `json:"ordersUiEnabled"`
	AutoApprovalEnabled     bool    `json:"autoApprovalEnabled"`
	RegistrarConsoleEnabled bool    `json:"registrarConsoleEnabled"`
	ConsentRequired         bool    `json:"consentRequired"`
}

func configToJSON(c *transcriptsrepo.Config) transcriptsConfigJSON {
	out := transcriptsConfigJSON{
		OfficialEnabled:         c.OfficialEnabled,
		OrdersUIEnabled:         c.OrdersUIEnabled,
		AutoApprovalEnabled:     c.AutoApprovalEnabled,
		RegistrarConsoleEnabled: c.RegistrarConsoleEnabled,
		ConsentRequired:         c.ConsentRequired,
	}
	if c.WebhookURL != nil {
		out.WebhookURL = *c.WebhookURL
	}
	if c.WebhookSecret != nil && strings.TrimSpace(*c.WebhookSecret) != "" {
		out.HasWebhookSecret = true
		out.WebhookSecret = placeholderSecretResponse
	}
	if c.PickupInstructions != nil && strings.TrimSpace(*c.PickupInstructions) != "" {
		s := strings.TrimSpace(*c.PickupInstructions)
		out.PickupInstructions = &s
	}
	return out
}

type transcriptsStudentConfigJSON struct {
	PickupInstructions *string `json:"pickupInstructions,omitempty"`
	PickupAvailable    bool    `json:"pickupAvailable"`
	OfficialEnabled    bool    `json:"officialEnabled"`
	OrdersUIEnabled    bool    `json:"ordersUiEnabled"`
	ConsentRequired    bool    `json:"consentRequired"`
}

func studentConfigToJSON(c *transcriptsrepo.Config) transcriptsStudentConfigJSON {
	out := transcriptsStudentConfigJSON{
		OfficialEnabled: c.OfficialEnabled,
		OrdersUIEnabled: c.OrdersUIEnabled,
		ConsentRequired: c.ConsentRequired,
	}
	if c.PickupInstructions != nil && strings.TrimSpace(*c.PickupInstructions) != "" {
		s := strings.TrimSpace(*c.PickupInstructions)
		out.PickupInstructions = &s
		out.PickupAvailable = true
	}
	return out
}

func (d Deps) registerTranscriptsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/transcripts/config", d.handleGetAdminTranscriptsConfig())
	r.Put("/api/v1/admin/transcripts/config", d.handlePutAdminTranscriptsConfig())
	r.Get("/api/v1/admin/transcripts/requests", d.handleGetAdminTranscriptRequests())
	r.Get("/api/v1/transcripts/config", d.handleGetTranscriptsConfig())
	r.Post("/api/v1/transcripts/requests", d.handlePostTranscriptRequest())
	r.Get("/api/v1/transcripts/requests", d.handleGetTranscriptRequests())
	d.registerTranscriptDocumentRoutes(r)
	d.registerTranscriptOrderRoutes(r)
}

// GET /api/v1/admin/transcripts/config
func (d Deps) handleGetAdminTranscriptsConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToJSON(cfg))
	}
}

type putTranscriptsConfigBody struct {
	WebhookURL              string  `json:"webhookUrl"`
	WebhookSecret           *string `json:"webhookSecret"`
	PickupInstructions      *string `json:"pickupInstructions"`
	OfficialEnabled         *bool   `json:"officialEnabled"`
	OrdersUIEnabled         *bool   `json:"ordersUiEnabled"`
	AutoApprovalEnabled     *bool   `json:"autoApprovalEnabled"`
	RegistrarConsoleEnabled *bool   `json:"registrarConsoleEnabled"`
	ConsentRequired         *bool   `json:"consentRequired"`
}

// PUT /api/v1/admin/transcripts/config
func (d Deps) handlePutAdminTranscriptsConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putTranscriptsConfigBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		url := strings.TrimSpace(body.WebhookURL)
		if url == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "webhookUrl is required.")
			return
		}
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "webhookUrl must be an http or https URL.")
			return
		}
		var secret *string
		if body.WebhookSecret != nil {
			s := strings.TrimSpace(*body.WebhookSecret)
			if s != "" && s != placeholderSecretResponse {
				secret = &s
			}
		}
		cfg, err := transcriptsrepo.UpsertConfig(r.Context(), d.Pool, transcriptsrepo.UpsertConfigInput{
			WebhookURL:              url,
			WebhookSecret:           secret,
			PickupInstructions:      body.PickupInstructions,
			OfficialEnabled:         body.OfficialEnabled,
			OrdersUIEnabled:         body.OrdersUIEnabled,
			AutoApprovalEnabled:     body.AutoApprovalEnabled,
			RegistrarConsoleEnabled: body.RegistrarConsoleEnabled,
			ConsentRequired:         body.ConsentRequired,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save transcripts config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToJSON(cfg))
	}
}

// GET /api/v1/admin/transcripts/requests
func (d Deps) handleGetAdminTranscriptRequests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
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
		list, err := transcriptsrepo.ListFailed(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load transcript requests.")
			return
		}
		out := make([]transcriptRequestJSON, 0, len(list))
		for _, item := range list {
			out = append(out, requestToJSON(item))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": out})
	}
}

// GET /api/v1/transcripts/config
func (d Deps) handleGetTranscriptsConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(studentConfigToJSON(cfg))
	}
}

type postTranscriptRequestBody struct {
	DeliveryType    string `json:"deliveryType"`
	DeliveryEmail   string `json:"deliveryEmail"`
	DeliveryAddress string `json:"deliveryAddress"`
	MailUrgency     string `json:"mailUrgency"`
	UrgencyDays     int    `json:"urgencyDays"`
}

func parseTranscriptRequestBody(body postTranscriptRequestBody, cfg *transcriptsrepo.Config) (transcriptsrepo.InsertRequestInput, string) {
	deliveryType := strings.TrimSpace(strings.ToLower(body.DeliveryType))
	switch transcriptsrepo.DeliveryType(deliveryType) {
	case transcriptsrepo.DeliveryEmail, transcriptsrepo.DeliveryMail, transcriptsrepo.DeliveryPickup:
	default:
		return transcriptsrepo.InsertRequestInput{}, "deliveryType must be email, mail, or pickup."
	}

	input := transcriptsrepo.InsertRequestInput{
		DeliveryType: transcriptsrepo.DeliveryType(deliveryType),
	}

	switch transcriptsrepo.DeliveryType(deliveryType) {
	case transcriptsrepo.DeliveryEmail:
		input.UrgencyDays = 1
		input.UrgencyUnit = transcriptsrepo.UrgencyDays
	case transcriptsrepo.DeliveryMail:
		input.UrgencyUnit = transcriptsrepo.UrgencyBusinessDays
		mailUrgency := strings.TrimSpace(strings.ToLower(body.MailUrgency))
		if mailUrgency == "" {
			mailUrgency = "standard"
		}
		switch mailUrgency {
		case "standard":
			min := 3
			input.UrgencyDaysMin = &min
			input.UrgencyDays = 5
		case "rush":
			min := 1
			input.UrgencyDaysMin = &min
			input.UrgencyDays = 2
		default:
			return transcriptsrepo.InsertRequestInput{}, "mailUrgency must be standard or rush."
		}
	case transcriptsrepo.DeliveryPickup:
		input.UrgencyUnit = transcriptsrepo.UrgencyBusinessDays
		if body.UrgencyDays != 1 && body.UrgencyDays != 2 && body.UrgencyDays != 3 {
			return transcriptsrepo.InsertRequestInput{}, "urgencyDays must be 1, 2, or 3 business days for pickup."
		}
		input.UrgencyDays = body.UrgencyDays
		if cfg.PickupInstructions == nil || strings.TrimSpace(*cfg.PickupInstructions) == "" {
			return transcriptsrepo.InsertRequestInput{}, "Pickup is not available. Choose another delivery method."
		}
	}

	switch transcriptsrepo.DeliveryType(deliveryType) {
	case transcriptsrepo.DeliveryEmail:
		email := strings.TrimSpace(body.DeliveryEmail)
		if email == "" {
			return transcriptsrepo.InsertRequestInput{}, "deliveryEmail is required for email delivery."
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return transcriptsrepo.InsertRequestInput{}, "deliveryEmail must be a valid email address."
		}
		input.DeliveryEmail = &email
	case transcriptsrepo.DeliveryMail:
		address := strings.TrimSpace(body.DeliveryAddress)
		if len(address) < 10 {
			return transcriptsrepo.InsertRequestInput{}, "deliveryAddress must be a complete mailing address."
		}
		input.DeliveryAddress = &address
	}

	return input, ""
}

// POST /api/v1/transcripts/requests
// Deprecated: proxies to a one-item transcript order, then continues legacy webhook delivery.
func (d Deps) handlePostTranscriptRequest() http.HandlerFunc {
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
		var body postTranscriptRequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		if cfg.WebhookURL == nil || strings.TrimSpace(*cfg.WebhookURL) == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Transcript requests are not configured yet. Contact your institution.")
			return
		}
		input, msg := parseTranscriptRequestBody(body, cfg)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		req, err := transcriptsrepo.InsertRequest(r.Context(), d.Pool, userID, &orgID, input)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create transcript request.")
			return
		}
		// Mirror into the order model (best-effort; legacy request remains source of webhook delivery).
		_ = d.proxyLegacyRequestToOrder(r.Context(), userID, &orgID, cfg, input, req.ID)

		go d.deliverTranscriptWebhook(context.Background(), *req, userID, cfg)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", "Sat, 01 Nov 2026 00:00:00 GMT")
		w.Header().Set("Link", "</api/v1/transcripts/orders>; rel=\"successor-version\"")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"request": requestToJSON(*req)})
	}
}

func (d Deps) proxyLegacyRequestToOrder(
	ctx context.Context,
	userID uuid.UUID,
	orgID *uuid.UUID,
	cfg *transcriptsrepo.Config,
	input transcriptsrepo.InsertRequestInput,
	legacyID uuid.UUID,
) error {
	item := transcriptsrepo.CreateOrderItemInput{
		Urgency: transcriptsrepo.UrgencyStandard,
	}
	if input.DeliveryType == transcriptsrepo.DeliveryMail {
		if input.UrgencyDaysMin != nil && *input.UrgencyDaysMin <= 2 {
			item.Urgency = transcriptsrepo.UrgencyRush
		}
		item.DeliveryMethod = transcriptsrepo.DeliveryPostalMail
		addr := map[string]string{}
		if input.DeliveryAddress != nil {
			addr["raw"] = *input.DeliveryAddress
		}
		raw, _ := json.Marshal(addr)
		name := "Mail recipient"
		item.AdHoc = &transcriptsrepo.AdHocRecipientInput{
			Type:         transcriptsrepo.RecipientOther,
			Name:         name,
			CanonicalKey: transcriptsStrPtr("adhoc:mail:" + strings.ToLower(strings.TrimSpace(transcriptsDerefStr(input.DeliveryAddress)))),
			Capabilities: []string{string(transcriptsrepo.DeliveryPostalMail)},
			Address:      raw,
		}
	} else {
		item.DeliveryMethod = transcriptsrepo.DeliverySecureLink
		selfID := transcriptsrepo.GlobalSelfRecipientID
		item.RecipientID = &selfID
		if input.DeliveryType == transcriptsrepo.DeliveryPickup {
			item.AdHoc = &transcriptsrepo.AdHocRecipientInput{
				Type:         transcriptsrepo.RecipientOther,
				Name:         "Pickup",
				CanonicalKey: transcriptsStrPtr("adhoc:pickup:" + userID.String()),
				Capabilities: []string{string(transcriptsrepo.DeliverySecureLink)},
			}
			item.RecipientID = nil
			if input.UrgencyDays <= 1 {
				item.Urgency = transcriptsrepo.UrgencyRush
			}
		}
	}
	order, err := transcriptsrepo.CreateOrder(ctx, d.Pool, cfg, transcriptsrepo.CreateOrderInput{
		UserID: userID,
		OrgID:  orgID,
		Items:  []transcriptsrepo.CreateOrderItemInput{item},
	})
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `
UPDATE transcripts.orders
SET legacy_request_id = $2, status = 'in_review', submitted_at = NOW()
WHERE id = $1
`, order.ID, legacyID)
	return err
}

func transcriptsStrPtr(s string) *string { return &s }

func transcriptsDerefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// GET /api/v1/transcripts/requests
func (d Deps) handleGetTranscriptRequests() http.HandlerFunc {
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
		list, err := transcriptsrepo.ListByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load transcript requests.")
			return
		}
		out := make([]transcriptRequestJSON, 0, len(list))
		for _, item := range list {
			out = append(out, requestToJSON(item))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": out})
	}
}

func (d Deps) deliverTranscriptWebhook(ctx context.Context, req transcriptsrepo.Request, userID uuid.UUID, cfg *transcriptsrepo.Config) {
	if d.Pool == nil || cfg.WebhookURL == nil {
		return
	}
	u, err := user.FindByID(ctx, d.Pool, userID)
	if err != nil || u == nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, req.ID, "Could not load student profile.", nil)
		return
	}
	delivery := transcriptDeliveryJSON{
		Type:    string(req.DeliveryType),
		Email:   req.DeliveryEmail,
		Address: req.DeliveryAddress,
	}
	if req.DeliveryType != transcriptsrepo.DeliveryEmail {
		delivery.UrgencyDays = req.UrgencyDays
		delivery.UrgencyDaysMin = req.UrgencyDaysMin
		delivery.UrgencyUnit = string(req.UrgencyUnit)
	}
	payload := transcriptWebhookPayload{
		RequestID:   req.ID.String(),
		RequestedAt: req.RequestedAt.UTC().Format(time.RFC3339),
		Delivery:    delivery,
		Student: transcriptStudentJSON{
			UserID:      u.ID,
			Email:       u.Email,
			FirstName:   u.FirstName,
			LastName:    u.LastName,
			DisplayName: u.DisplayName,
			StudentID:   u.Sid,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, req.ID, "Failed to encode webhook payload.", nil)
		return
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(*cfg.WebhookURL), bytes.NewReader(body))
	if err != nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, req.ID, "Invalid webhook URL.", nil)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Lextures-Transcripts/1.0")
	if cfg.WebhookSecret != nil && strings.TrimSpace(*cfg.WebhookSecret) != "" {
		mac := hmac.New(sha256.New, []byte(strings.TrimSpace(*cfg.WebhookSecret)))
		_, _ = mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		httpReq.Header.Set("X-Lextures-Signature", "sha256="+sig)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		msg := "Webhook delivery failed: " + err.Error()
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, req.ID, msg, nil)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	code := resp.StatusCode
	if code >= 200 && code < 300 {
		_ = transcriptsrepo.MarkSubmitted(ctx, d.Pool, req.ID, code)
		return
	}
	msg := "Institution webhook returned an error."
	_ = transcriptsrepo.MarkFailed(ctx, d.Pool, req.ID, msg, &code)
}
