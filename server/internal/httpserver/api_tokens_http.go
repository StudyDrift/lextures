package httpserver

import (
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
)

func (d Deps) apiTokensEnabled() bool {
	return d.effectiveConfig().FFAPITokens
}

func (d Deps) apiTokenIPHashKey() string {
	cfg := d.effectiveConfig()
	if len(cfg.PlatformSecretsKey) > 0 {
		return string(cfg.PlatformSecretsKey)
	}
	return cfg.JWTSecret
}

// requireAPITokensEnabled gates access-key management routes (plan 16.2).
func (d Deps) requireAPITokensEnabled(w http.ResponseWriter) bool {
	if !d.apiTokensEnabled() {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "API access keys are not enabled on this server.")
		return false
	}
	return true
}
