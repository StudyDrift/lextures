package logging

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strings"
)

// RedactorConfig controls PII redaction behavior.
type RedactorConfig struct {
	Enabled    bool
	Disabled   bool // DISABLE_PII_REDACTION=1
	Registry   *FieldRegistry
	HMACSecret []byte
}

// Redactor replaces sensitive structured log values.
type Redactor struct {
	cfg RedactorConfig
}

// NewRedactor builds a redactor from config.
func NewRedactor(cfg RedactorConfig) *Redactor {
	if cfg.Registry == nil {
		cfg.Registry = NewFieldRegistry()
	}
	return &Redactor{cfg: cfg}
}

// RedactValue returns a safe replacement for a structured field value.
func (r *Redactor) RedactValue(field string, value any) (any, bool) {
	if !r.cfg.Registry.shouldRedact(field, r.cfg.Disabled) {
		return value, false
	}
	field = normalizeFieldName(field)
	GlobalRedactionMetrics.Inc(field)
	if r.cfg.Registry.useHash(field) {
		return r.hashValue(value), true
	}
	s := stringifyValue(value)
	if field == "ip_address" || field == "ip" || field == "client_ip" || field == "remote_addr" {
		return maskIP(s), true
	}
	return fmt.Sprintf("[REDACTED:%s]", field), true
}

func (r *Redactor) hashValue(value any) string {
	s := stringifyValue(value)
	if s == "" {
		return "[REDACTED]"
	}
	key := r.cfg.HMACSecret
	if len(key) == 0 {
		return "[REDACTED]"
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(s))
	sum := mac.Sum(nil)
	return "hmac:" + hex.EncodeToString(sum[:8])
}

func stringifyValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case slog.Value:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func maskIP(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "[REDACTED:ip_address]"
	}
	host := s
	if h, _, err := net.SplitHostPort(s); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "[REDACTED:ip_address]"
	}
	if v4 := ip.To4(); v4 != nil {
		v4[3] = 0
		return v4.String()
	}
	// IPv6: zero last 64 bits (last 8 bytes) for subnet-style masking.
	for i := 8; i < 16; i++ {
		ip[i] = 0
	}
	return ip.String()
}

// RedactAttr returns a possibly redacted slog attribute.
func (r *Redactor) RedactAttr(a slog.Attr) slog.Attr {
	if a.Equal(slog.Attr{}) {
		return a
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		out := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			out[i] = r.RedactAttr(ga)
		}
		return slog.GroupAttrs(a.Key, out...)
	}
	if v, ok := r.RedactValue(a.Key, attrValue(a)); ok {
		return slog.Any(a.Key, v)
	}
	return a
}

func attrValue(a slog.Attr) any {
	if a.Value.Kind() == slog.KindAny {
		return a.Value.Any()
	}
	return a.Value
}
