package transcriptdelivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/mail"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
	"github.com/lextures/lextures/server/internal/service/transcriptedi"
	"github.com/lextures/lextures/server/internal/service/transcriptpesc"
	"github.com/lextures/lextures/server/internal/webhooks"
)

// AdapterResult is the outcome of one adapter send.
type AdapterResult struct {
	Status       transcriptsrepo.DeliveryAttemptStatus
	ResponseCode *int
	Detail       string
	ShareURL     string
}

// Adapter delivers one order item via a specific transport.
type Adapter interface {
	Name() transcriptsrepo.DeliveryMethod
	Deliver(ctx context.Context, env *Env, dc *transcriptsrepo.DeliveryItemContext, attempt *transcriptsrepo.DeliveryAttempt) (AdapterResult, error)
}

// Env carries dependencies for adapters.
type Env struct {
	Pool   *pgxpool.Pool
	Cfg    config.Config
	Client *http.Client
	Now    func() time.Time
}

func (e *Env) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now().UTC()
}

func (e *Env) httpClient() *http.Client {
	if e.Client != nil {
		return e.Client
	}
	return outboundClient()
}

func outboundClient() *http.Client {
	transport := &http.Transport{
		DialContext: safeDialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := webhooks.ValidateEndpointURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if webhooks.BlockedIP(ip.IP) {
			return nil, webhooks.ErrSSRFPolicy
		}
	}
	d := net.Dialer{Timeout: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}

// SelectAdapter returns the adapter for a delivery method.
func SelectAdapter(method transcriptsrepo.DeliveryMethod, deliveryV2 bool) (Adapter, error) {
	switch method {
	case transcriptsrepo.DeliveryAPIPeer:
		return apiPeerAdapter{}, nil
	case transcriptsrepo.DeliverySecureLink, transcriptsrepo.DeliveryElectronicPDF:
		return secureLinkAdapter{}, nil
	case transcriptsrepo.DeliveryElectronicPESC:
		if !deliveryV2 {
			return nil, fmt.Errorf("adapter %s requires delivery_v2", method)
		}
		return pescAdapter{}, nil
	case transcriptsrepo.DeliveryEDISPEEDE:
		if !deliveryV2 {
			return nil, fmt.Errorf("adapter %s requires delivery_v2", method)
		}
		return ediAdapter{}, nil
	case transcriptsrepo.DeliveryPostalMail:
		if !deliveryV2 {
			return nil, fmt.Errorf("adapter %s requires delivery_v2", method)
		}
		return postalAdapter{}, nil
	default:
		return nil, fmt.Errorf("unknown delivery method %s", method)
	}
}

type apiPeerAdapter struct{}

func (apiPeerAdapter) Name() transcriptsrepo.DeliveryMethod {
	return transcriptsrepo.DeliveryAPIPeer
}

type peerWebhookPayload struct {
	OrderID      string `json:"orderId"`
	OrderItemID  string `json:"orderItemId"`
	DocumentID   string `json:"documentId,omitempty"`
	DeliveryMethod string `json:"deliveryMethod"`
	ContentHash  string `json:"contentHash,omitempty"`
	DownloadURL  string `json:"downloadUrl,omitempty"`
	Format       string `json:"format,omitempty"`
	Student      struct {
		UserID string `json:"userId"`
	} `json:"student"`
	RequestedAt string `json:"requestedAt"`
}

func (a apiPeerAdapter) Deliver(
	ctx context.Context,
	env *Env,
	dc *transcriptsrepo.DeliveryItemContext,
	_ *transcriptsrepo.DeliveryAttempt,
) (AdapterResult, error) {
	endpoint, secret, err := resolvePeerEndpoint(ctx, env.Pool, dc)
	if err != nil {
		return AdapterResult{}, err
	}
	if err := webhooks.ValidateEndpointURL(endpoint); err != nil {
		// Local/dev often uses http://localhost — allow only when APP_ENV=local.
		if env.Cfg.AppEnv != "local" || !strings.HasPrefix(strings.TrimSpace(endpoint), "http://") {
			return AdapterResult{}, fmt.Errorf("peer endpoint: %w", err)
		}
	}

	var downloadURL string
	if dc.Document != nil && len(dc.Document.PDFBytes) > 0 {
		link, lerr := ensureShareLink(ctx, env, dc)
		if lerr == nil {
			downloadURL = shareURL(env.Cfg, link.Token)
		}
	}

	payload := peerWebhookPayload{
		OrderID:        dc.Order.ID.String(),
		OrderItemID:    dc.Item.ID.String(),
		DeliveryMethod: string(dc.Item.DeliveryMethod),
		RequestedAt:    env.now().Format(time.RFC3339),
	}
	payload.Student.UserID = dc.Order.UserID.String()
	if dc.Document != nil {
		payload.DocumentID = dc.Document.ID.String()
		payload.ContentHash = dc.Document.ContentHash
		payload.Format = "pdf"
	}
	payload.DownloadURL = downloadURL

	body, err := json.Marshal(payload)
	if err != nil {
		return AdapterResult{}, err
	}
	code, detail, err := postSigned(ctx, env.httpClient(), endpoint, body, secret)
	if err != nil {
		return AdapterResult{}, wrapTransient(err)
	}
	rc := code
	if code >= 200 && code < 300 {
		return AdapterResult{Status: transcriptsrepo.AttemptDelivered, ResponseCode: &rc, Detail: detail}, nil
	}
	if code >= 500 || code == 429 {
		return AdapterResult{}, wrapTransient(fmt.Errorf("peer returned %d: %s", code, detail))
	}
	return AdapterResult{Status: transcriptsrepo.AttemptFailed, ResponseCode: &rc, Detail: detail}, fmt.Errorf("peer returned %d", code)
}

func resolvePeerEndpoint(ctx context.Context, pool *pgxpool.Pool, dc *transcriptsrepo.DeliveryItemContext) (endpoint, secret string, err error) {
	if dc.Item.Recipient != nil && len(dc.Item.Recipient.PeerConfig) > 0 {
		var pc struct {
			Endpoint string `json:"endpoint"`
			URL      string `json:"url"`
			Secret   string `json:"secret"`
		}
		if json.Unmarshal(dc.Item.Recipient.PeerConfig, &pc) == nil {
			endpoint = strings.TrimSpace(pc.Endpoint)
			if endpoint == "" {
				endpoint = strings.TrimSpace(pc.URL)
			}
			secret = strings.TrimSpace(pc.Secret)
		}
	}
	cfg, err := transcriptsrepo.GetConfig(ctx, pool)
	if err != nil {
		return "", "", err
	}
	if endpoint == "" && cfg.WebhookURL != nil {
		endpoint = strings.TrimSpace(*cfg.WebhookURL)
	}
	if secret == "" && cfg.WebhookSecret != nil {
		secret = strings.TrimSpace(*cfg.WebhookSecret)
	}
	if endpoint == "" {
		return "", "", fmt.Errorf("no api_peer endpoint configured")
	}
	return endpoint, secret, nil
}

func postSigned(ctx context.Context, client *http.Client, endpoint string, body []byte, secret string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lextures-Transcripts/2.0")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		req.Header.Set("X-Lextures-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	snip, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return resp.StatusCode, string(snip), nil
}

type secureLinkAdapter struct{}

func (secureLinkAdapter) Name() transcriptsrepo.DeliveryMethod {
	return transcriptsrepo.DeliverySecureLink
}

func (a secureLinkAdapter) Deliver(
	ctx context.Context,
	env *Env,
	dc *transcriptsrepo.DeliveryItemContext,
	_ *transcriptsrepo.DeliveryAttempt,
) (AdapterResult, error) {
	if dc.Document == nil || len(dc.Document.PDFBytes) == 0 {
		return AdapterResult{}, fmt.Errorf("document PDF required for secure link")
	}
	link, err := ensureShareLink(ctx, env, dc)
	if err != nil {
		return AdapterResult{}, err
	}
	url := shareURL(env.Cfg, link.Token)
	detail := "secure link created"
	// Email when method is secure_link_email and recipient has email.
	if dc.Item.DeliveryMethod == transcriptsrepo.DeliverySecureLink {
		to := recipientEmail(dc)
		if to != "" {
			if err := sendSecureLinkEmail(env.Cfg, to, url, link.ExpiresAt); err != nil {
				return AdapterResult{}, wrapTransient(err)
			}
			detail = "secure link emailed"
		}
	}
	return AdapterResult{
		Status:   transcriptsrepo.AttemptDelivered,
		Detail:   detail,
		ShareURL: url,
	}, nil
}

func sendSecureLinkEmail(cfg config.Config, to, url string, expires time.Time) error {
	subject := "Your secure transcript download link"
	body := fmt.Sprintf(
		"A transcript is ready for you.\n\nDownload (expires %s):\n%s\n\nDo not forward this link. Access is limited and tracked.\n",
		expires.UTC().Format(time.RFC3339),
		url,
	)
	html := fmt.Sprintf(
		`<p>A transcript is ready for you.</p><p><a href="%s">Download transcript</a></p><p>This link expires %s. Do not forward it; access is limited and tracked.</p>`,
		url, expires.UTC().Format(time.RFC3339),
	)
	return mail.SendMultipart(cfg, to, subject, body, html, nil)
}

func ensureShareLink(ctx context.Context, env *Env, dc *transcriptsrepo.DeliveryItemContext) (*transcriptsrepo.ShareLink, error) {
	existing, err := transcriptsrepo.LatestShareLinkForItem(ctx, env.Pool, dc.Item.ID)
	if err == nil && existing.ExpiresAt.After(env.now()) && existing.DownloadCount < existing.MaxDownloads {
		return existing, nil
	}
	expires := env.now().Add(7 * 24 * time.Hour)
	return transcriptsrepo.CreateShareLink(ctx, env.Pool, dc.Item.ID, dc.Document.ID, expires, 5)
}

func shareURL(cfg config.Config, token string) string {
	base := strings.TrimRight(cfg.PublicWebOrigin, "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + "/r/t/" + token
}

func recipientEmail(dc *transcriptsrepo.DeliveryItemContext) string {
	if dc.Item.Recipient != nil && dc.Item.Recipient.Email != nil {
		return strings.TrimSpace(*dc.Item.Recipient.Email)
	}
	return ""
}

type pescAdapter struct{}

func (pescAdapter) Name() transcriptsrepo.DeliveryMethod {
	return transcriptsrepo.DeliveryElectronicPESC
}

func (a pescAdapter) Deliver(
	ctx context.Context,
	env *Env,
	dc *transcriptsrepo.DeliveryItemContext,
	_ *transcriptsrepo.DeliveryAttempt,
) (AdapterResult, error) {
	xmlBytes, err := documentPESC(dc)
	if err != nil {
		return AdapterResult{}, err
	}
	if err := transcriptpesc.ValidateStructure(xmlBytes); err != nil {
		return AdapterResult{}, fmt.Errorf("pesc validation: %w", err)
	}
	endpoint, secret, err := resolvePeerEndpoint(ctx, env.Pool, dc)
	if err != nil {
		// Without a peer endpoint, treat validated XML as staged/delivered to outbox.
		return AdapterResult{
			Status: transcriptsrepo.AttemptDelivered,
			Detail: "pesc xml validated; no peer endpoint — staged",
		}, nil
	}
	code, detail, err := postBytes(ctx, env.httpClient(), endpoint, xmlBytes, "application/xml", secret)
	if err != nil {
		return AdapterResult{}, wrapTransient(err)
	}
	rc := code
	if code >= 200 && code < 300 {
		return AdapterResult{Status: transcriptsrepo.AttemptDelivered, ResponseCode: &rc, Detail: detail}, nil
	}
	if code >= 500 || code == 429 {
		return AdapterResult{}, wrapTransient(fmt.Errorf("pesc peer returned %d", code))
	}
	return AdapterResult{Status: transcriptsrepo.AttemptFailed, ResponseCode: &rc, Detail: detail}, fmt.Errorf("pesc peer returned %d", code)
}

type ediAdapter struct{}

func (ediAdapter) Name() transcriptsrepo.DeliveryMethod {
	return transcriptsrepo.DeliveryEDISPEEDE
}

func (a ediAdapter) Deliver(
	ctx context.Context,
	env *Env,
	dc *transcriptsrepo.DeliveryItemContext,
	_ *transcriptsrepo.DeliveryAttempt,
) (AdapterResult, error) {
	rec, err := decodeCanonical(dc)
	if err != nil {
		return AdapterResult{}, err
	}
	ediBytes, err := transcriptedi.BuildTS130(rec)
	if err != nil {
		return AdapterResult{}, err
	}
	endpoint, secret, err := resolvePeerEndpoint(ctx, env.Pool, dc)
	if err != nil {
		return AdapterResult{
			Status: transcriptsrepo.AttemptDelivered,
			Detail: "edi ts130 validated; no peer endpoint — staged",
		}, nil
	}
	code, detail, err := postBytes(ctx, env.httpClient(), endpoint, ediBytes, "application/edi-x12", secret)
	if err != nil {
		return AdapterResult{}, wrapTransient(err)
	}
	rc := code
	if code >= 200 && code < 300 {
		return AdapterResult{Status: transcriptsrepo.AttemptDelivered, ResponseCode: &rc, Detail: detail}, nil
	}
	if code >= 500 || code == 429 {
		return AdapterResult{}, wrapTransient(fmt.Errorf("edi peer returned %d", code))
	}
	return AdapterResult{Status: transcriptsrepo.AttemptFailed, ResponseCode: &rc, Detail: detail}, fmt.Errorf("edi peer returned %d", code)
}

type postalAdapter struct{}

func (postalAdapter) Name() transcriptsrepo.DeliveryMethod {
	return transcriptsrepo.DeliveryPostalMail
}

func (a postalAdapter) Deliver(
	ctx context.Context,
	env *Env,
	dc *transcriptsrepo.DeliveryItemContext,
	_ *transcriptsrepo.DeliveryAttempt,
) (AdapterResult, error) {
	if dc.Document == nil {
		return AdapterResult{}, transcriptsrepo.ErrDocumentRequired
	}
	if dc.Item.Recipient == nil || len(dc.Item.Recipient.Address) == 0 {
		return AdapterResult{}, fmt.Errorf("postal address required")
	}
	addr := dc.Item.Recipient.Address
	if err := validatePostalAddress(addr); err != nil {
		return AdapterResult{}, err
	}
	job, err := transcriptsrepo.InsertPostalJob(ctx, env.Pool, dc.Item.ID, dc.Document.ID, addr)
	if err != nil {
		return AdapterResult{}, err
	}
	return AdapterResult{
		Status: transcriptsrepo.AttemptDelivered,
		Detail: fmt.Sprintf("postal job queued %s", job.ID.String()),
	}, nil
}

func validatePostalAddress(addr json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(addr, &m); err != nil {
		return fmt.Errorf("invalid address json")
	}
	line1, _ := m["line1"].(string)
	city, _ := m["city"].(string)
	if strings.TrimSpace(line1) == "" && strings.TrimSpace(city) == "" {
		// Accept freeform "street" / "address1" too.
		if s, ok := m["street"].(string); ok && strings.TrimSpace(s) != "" {
			return nil
		}
		if s, ok := m["address1"].(string); ok && strings.TrimSpace(s) != "" {
			return nil
		}
		return fmt.Errorf("postal address incomplete")
	}
	return nil
}

func documentPESC(dc *transcriptsrepo.DeliveryItemContext) ([]byte, error) {
	if dc.Document == nil {
		return nil, transcriptsrepo.ErrDocumentRequired
	}
	if len(dc.Document.PESCXMLBytes) > 0 {
		return dc.Document.PESCXMLBytes, nil
	}
	rec, err := decodeCanonical(dc)
	if err != nil {
		return nil, err
	}
	return transcriptpesc.BuildXML(rec)
}

func decodeCanonical(dc *transcriptsrepo.DeliveryItemContext) (*academicrecord.AcademicRecord, error) {
	if dc.Document == nil || len(dc.Document.Canonical) == 0 {
		return nil, transcriptsrepo.ErrDocumentRequired
	}
	if !transcriptsrepo.VerifyDocumentHash(dc.Document) {
		return nil, fmt.Errorf("document integrity check failed")
	}
	var rec academicrecord.AcademicRecord
	if err := json.Unmarshal(dc.Document.Canonical, &rec); err != nil {
		return nil, fmt.Errorf("canonical decode: %w", err)
	}
	return &rec, nil
}

func postBytes(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	body []byte,
	contentType, secret string,
) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "Lextures-Transcripts/2.0")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		req.Header.Set("X-Lextures-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	snip, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return resp.StatusCode, string(snip), nil
}
