package oidcauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgxpool"

	pauth "github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/service/authservice"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// NativeAppleLoginRequest is the body for POST /api/v1/auth/oidc/apple/native.
type NativeAppleLoginRequest struct {
	IDToken           string
	RawNonce          string
	AuthorizationCode string // optional; not exchanged in v1
	FullName          string // first authorization only
	Email             string // first authorization only (or private relay)
}

// NativeGoogleLoginRequest is the body for POST /api/v1/auth/oidc/google/native.
type NativeGoogleLoginRequest struct {
	IDToken  string
	RawNonce string // optional; verified when present in the ID token
}

// HashNonceSHA256 returns the hex-encoded SHA-256 of rawNonce (Apple/Google nonce claim).
func HashNonceSHA256(rawNonce string) string {
	sum := sha256.Sum256([]byte(rawNonce))
	return hex.EncodeToString(sum[:])
}

// audienceAllowed reports whether aud is in the allow-list (exact match, case-sensitive).
func audienceAllowed(aud string, allowed []string) bool {
	aud = strings.TrimSpace(aud)
	if aud == "" {
		return false
	}
	for _, a := range allowed {
		if strings.TrimSpace(a) == aud {
			return true
		}
	}
	return false
}

// CompleteNativeAppleLogin verifies an Apple identity token from AuthenticationServices
// and issues Lextures tokens (MOB.9).
func (s *Service) CompleteNativeAppleLogin(
	ctx context.Context, pool *pgxpool.Pool, jwt *pauth.JWTSigner,
	req NativeAppleLoginRequest, meta *authservice.ClientMeta,
) (authservice.AuthResponse, error) {
	telemetry.RecordBusinessEvent("auth_native_signin_start")
	if !s.Cfg.OIDCAppleNativeAvailable() {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Native Sign in with Apple is not enabled."}
	}
	idToken := strings.TrimSpace(req.IDToken)
	rawNonce := strings.TrimSpace(req.RawNonce)
	if idToken == "" || rawNonce == "" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "id_token and raw_nonce are required."}
	}

	prov, err := s.providerForIssuer(ctx, "https://appleid.apple.com")
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Could not contact Apple identity services."}
	}
	// Multi-audience (bundle IDs): skip built-in single ClientID check and validate aud ourselves.
	ver := prov.Verifier(&oidc.Config{SkipClientIDCheck: true})
	ctxO := oidc.ClientContext(ctx, s.HTTP)
	tok, err := ver.Verify(ctxO, idToken)
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Apple identity token."}
	}
	if iss := strings.TrimSpace(tok.Issuer); iss != "https://appleid.apple.com" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Apple identity token issuer."}
	}
	if !audienceAllowed(firstAudience(tok), s.Cfg.OIDCAppleNativeAudiences()) {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Apple identity token audience."}
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Nonce string `json:"nonce"`
	}
	if err := tok.Claims(&claims); err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Apple identity token claims."}
	}
	wantNonce := HashNonceSHA256(rawNonce)
	// Apple may put the SHA256 of the raw nonce (hex) in the claim; some SDKs send the raw value hashed differently.
	// Compare case-insensitively on hex, and also accept raw if the client already hashed.
	gotNonce := strings.ToLower(strings.TrimSpace(claims.Nonce))
	if gotNonce == "" || (gotNonce != strings.ToLower(wantNonce) && gotNonce != strings.ToLower(rawNonce)) {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Apple sign-in (nonce)."}
	}
	if strings.TrimSpace(claims.Sub) == "" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "The identity provider did not return a subject."}
	}

	email := strings.TrimSpace(claims.Email)
	if email == "" {
		email = strings.TrimSpace(req.Email)
	}
	fullName := strings.TrimSpace(req.FullName)

	res, err := s.finishProviderIdentityLogin(ctx, pool, jwt, finishIdentityInput{
		Provider:        "apple",
		Subject:         claims.Sub,
		Email:           email,
		FullName:        fullName,
		Meta:            meta,
		AllowEmptyEmail: true,
	})
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, err
	}
	telemetry.RecordBusinessEvent("auth_native_signin_success")
	return res, nil
}

// CompleteNativeGoogleLogin verifies a Google ID token from Credential Manager and issues tokens.
func (s *Service) CompleteNativeGoogleLogin(
	ctx context.Context, pool *pgxpool.Pool, jwt *pauth.JWTSigner,
	req NativeGoogleLoginRequest, meta *authservice.ClientMeta,
) (authservice.AuthResponse, error) {
	telemetry.RecordBusinessEvent("auth_native_signin_start")
	if !s.Cfg.OIDCGoogleNativeAvailable() {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Native Google sign-in is not enabled."}
	}
	idToken := strings.TrimSpace(req.IDToken)
	if idToken == "" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "id_token is required."}
	}
	aud := s.Cfg.OIDCGoogleNativeAudienceResolved()
	if aud == "" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Google native audience is not configured."}
	}

	prov, err := s.providerForIssuer(ctx, "https://accounts.google.com")
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Could not contact Google identity services."}
	}
	ver := prov.Verifier(&oidc.Config{ClientID: aud})
	ctxO := oidc.ClientContext(ctx, s.HTTP)
	tok, err := ver.Verify(ctxO, idToken)
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Google identity token."}
	}
	iss := strings.TrimSpace(tok.Issuer)
	if iss != "https://accounts.google.com" && iss != "accounts.google.com" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Google identity token issuer."}
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Nonce string `json:"nonce"`
	}
	if err := tok.Claims(&claims); err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Google identity token claims."}
	}
	if raw := strings.TrimSpace(req.RawNonce); raw != "" {
		want := strings.ToLower(HashNonceSHA256(raw))
		got := strings.ToLower(strings.TrimSpace(claims.Nonce))
		if got == "" || (got != want && got != strings.ToLower(raw)) {
			telemetry.RecordBusinessEvent("auth_native_signin_error")
			return authservice.AuthResponse{}, authservice.FieldError{Message: "Invalid Google sign-in (nonce)."}
		}
	}
	if strings.TrimSpace(claims.Sub) == "" {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "The identity provider did not return a subject."}
	}
	email := strings.TrimSpace(claims.Email)
	if email == "" || !strings.Contains(email, "@") {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, authservice.FieldError{Message: "The identity provider did not return a usable email address."}
	}
	if err := checkHostedDomain("google", s.Cfg, nil, userNormalizeEmail(email)); err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, err
	}

	res, err := s.finishProviderIdentityLogin(ctx, pool, jwt, finishIdentityInput{
		Provider: "google",
		Subject:  claims.Sub,
		Email:    email,
		FullName: strings.TrimSpace(claims.Name),
		Meta:     meta,
	})
	if err != nil {
		telemetry.RecordBusinessEvent("auth_native_signin_error")
		return authservice.AuthResponse{}, err
	}
	telemetry.RecordBusinessEvent("auth_native_signin_success")
	return res, nil
}

func firstAudience(tok *oidc.IDToken) string {
	if tok == nil {
		return ""
	}
	// go-oidc stores audience on the verified token; Claims also has aud as string or []string.
	var c struct {
		Aud any `json:"aud"`
	}
	if err := tok.Claims(&c); err != nil {
		return ""
	}
	switch v := c.Aud.(type) {
	case string:
		return v
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	}
	return fmt.Sprint(c.Aud)
}

// userNormalizeEmail avoids an import cycle pattern; delegates to the same rules as user.NormalizeEmail.
func userNormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
