package paymentprovider

import (
	"strings"

	"github.com/lextures/lextures/server/internal/config"
)

// Config bundles credentials for all payment backends.
type Config struct {
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripeMonthlyPriceID string
	StripeAnnualPriceID  string
	PayPalClientID       string
	PayPalClientSecret   string
	PayPalWebhookID      string
	PayPalSandbox        bool
	PublicWebOrigin      string
}

// ConfigFrom merges process config for payment calls.
func ConfigFrom(cfg config.Config) Config {
	return Config{
		StripeSecretKey:      strings.TrimSpace(cfg.StripeSecretKey),
		StripeWebhookSecret:  strings.TrimSpace(cfg.StripeWebhookSecret),
		StripeMonthlyPriceID: strings.TrimSpace(cfg.StripeMonthlyPriceID),
		StripeAnnualPriceID:  strings.TrimSpace(cfg.StripeAnnualPriceID),
		PayPalClientID:       strings.TrimSpace(cfg.PayPalClientID),
		PayPalClientSecret:   strings.TrimSpace(cfg.PayPalClientSecret),
		PayPalWebhookID:      strings.TrimSpace(cfg.PayPalWebhookID),
		PayPalSandbox:        cfg.PayPalSandbox,
		PublicWebOrigin:      strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/"),
	}
}

func (c Config) StripeConfigured() bool {
	return c.StripeSecretKey != ""
}

func (c Config) PayPalConfigured() bool {
	return c.PayPalClientID != "" && c.PayPalClientSecret != ""
}
