package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/devicepushtokens"
)

// DeliveryResult describes the outcome of a native push send attempt.
type DeliveryResult struct {
	InvalidToken bool
	Retryable    bool
}

// NativeDispatcher sends push notifications to APNs and FCM device tokens.
type NativeDispatcher struct {
	Config config.Config
	Client *http.Client

	apnsTokenMu sync.Mutex
	apnsJWT     string
	apnsJWTExp  time.Time
}

func NewNativeDispatcher(cfg config.Config) *NativeDispatcher {
	return &NativeDispatcher{
		Config: cfg,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *NativeDispatcher) apnsConfigured() bool {
	return strings.TrimSpace(d.Config.APNSP8Key) != "" &&
		strings.TrimSpace(d.Config.APNSKeyID) != "" &&
		strings.TrimSpace(d.Config.APNSTeamID) != "" &&
		strings.TrimSpace(d.Config.APNSBundleID) != ""
}

func (d *NativeDispatcher) fcmConfigured() bool {
	return strings.TrimSpace(d.Config.FCMServiceAccountJSON) != ""
}

// Send delivers a push to one device token.
func (d *NativeDispatcher) Send(ctx context.Context, row devicepushtokens.Row, title, body, actionURL string) (DeliveryResult, error) {
	switch row.Platform {
	case "apns":
		if !d.apnsConfigured() {
			return DeliveryResult{}, nil
		}
		return d.sendAPNS(ctx, row, title, body, actionURL)
	case "fcm":
		if !d.fcmConfigured() {
			return DeliveryResult{}, nil
		}
		return d.sendFCM(ctx, row, title, body, actionURL)
	default:
		return DeliveryResult{}, fmt.Errorf("unknown platform %q", row.Platform)
	}
}

func (d *NativeDispatcher) sendAPNS(ctx context.Context, row devicepushtokens.Row, title, body, actionURL string) (DeliveryResult, error) {
	host := "api.push.apple.com"
	if strings.EqualFold(d.Config.APNSEnvironment, "development") {
		host = "api.sandbox.push.apple.com"
	}
	bundleID := d.Config.APNSBundleID
	if row.AppBundleID != "" {
		bundleID = row.AppBundleID
	}

	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
		"action_url": actionURL,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return DeliveryResult{}, err
	}

	jwtToken, err := d.apnsAuthToken()
	if err != nil {
		return DeliveryResult{Retryable: true}, err
	}

	url := fmt.Sprintf("https://%s/3/device/%s", host, row.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return DeliveryResult{}, err
	}
	req.Header.Set("authorization", "bearer "+jwtToken)
	req.Header.Set("apns-topic", bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("content-type", "application/json")

	resp, err := d.Client.Do(req)
	if err != nil {
		return DeliveryResult{Retryable: true}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return DeliveryResult{}, nil
	case http.StatusGone, http.StatusBadRequest:
		reason := resp.Header.Get("apns-reason")
		if reason == "BadDeviceToken" || reason == "Unregistered" || resp.StatusCode == http.StatusGone {
			return DeliveryResult{InvalidToken: true}, nil
		}
		return DeliveryResult{}, fmt.Errorf("apns status %d reason %s", resp.StatusCode, reason)
	case http.StatusTooManyRequests:
		return DeliveryResult{Retryable: true}, fmt.Errorf("apns rate limited")
	default:
		reason := resp.Header.Get("apns-reason")
		slog.Warn("apns.send", "status", resp.StatusCode, "reason", reason)
		return DeliveryResult{Retryable: resp.StatusCode >= 500}, fmt.Errorf("apns status %d", resp.StatusCode)
	}
}

func (d *NativeDispatcher) apnsAuthToken() (string, error) {
	d.apnsTokenMu.Lock()
	defer d.apnsTokenMu.Unlock()
	if d.apnsJWT != "" && time.Now().Before(d.apnsJWTExp.Add(-1*time.Minute)) {
		return d.apnsJWT, nil
	}

	key, err := parseAPNSP8Key(d.Config.APNSP8Key)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": d.Config.APNSTeamID,
		"iat": now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = d.Config.APNSKeyID
	signed, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	d.apnsJWT = signed
	d.apnsJWTExp = now.Add(50 * time.Minute)
	return signed, nil
}

func parseAPNSP8Key(raw string) (*ecdsa.PrivateKey, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "BEGIN") {
		raw = "-----BEGIN PRIVATE KEY-----\n" + raw + "\n-----END PRIVATE KEY-----"
	}
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("invalid apns p8 key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("apns p8 key is not ecdsa")
	}
	return ecKey, nil
}
