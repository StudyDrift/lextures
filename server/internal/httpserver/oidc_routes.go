package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/service/authservice"
	"github.com/lextures/lextures/server/internal/service/oidcauth"
)

// handleOIDCLogin is GET /auth/oidc/{provider}/login — starts the OIDC code+PKCE flow.
func (d Deps) handleOIDCLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database is not configured.")
			return
		}
		if d.OIDC == nil {
			d.OIDC = oidcauth.NewService(d.effectiveConfig())
		}
		prov := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
		q := r.URL.Query()
		var next, linkID, configID *string
		if s := strings.TrimSpace(q.Get("next")); s != "" {
			next = &s
		}
		if s := strings.TrimSpace(q.Get("linkId")); s != "" {
			linkID = &s
		}
		if s := strings.TrimSpace(q.Get("configId")); s != "" {
			configID = &s
		}
		var linkUUID *uuid.UUID
		if linkID != nil && *linkID != "" {
			u, err := uuid.Parse(*linkID)
			if err != nil {
				writeAuthErr(w, authservice.FieldError{Message: "Invalid linkId."})
				return
			}
			linkUUID = &u
		}
		var configUUID *uuid.UUID
		if configID != nil && *configID != "" {
			u, err := uuid.Parse(*configID)
			if err != nil {
				writeAuthErr(w, authservice.FieldError{Message: "Invalid configId."})
				return
			}
			configUUID = &u
		}
		target, err := d.OIDC.BuildAuthorizeRedirectURL(r.Context(), d.Pool, prov, configUUID, linkUUID, next)
		if err != nil {
			writeAuthErr(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	}
}

// handleOIDCCallback is GET /auth/oidc/{provider}/callback — exchanges code, returns HTML to app with fragment token.
func (d Deps) handleOIDCCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database is not configured.")
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "JWT is not configured.")
			return
		}
		if d.OIDC == nil {
			d.OIDC = oidcauth.NewService(d.effectiveConfig())
		}
		prov := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
		q := r.URL.Query()
		if errName := q.Get("error"); errName != "" {
			msg := q.Get("error_description")
			if msg == "" {
				msg = errName
			}
			enc := url.QueryEscape(msg)
			public := d.OIDC.PublicWebBase()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>Sign-in</title></head>
<body><script>location.replace("` + public + `/sso-error?message=` + enc + `");</script>
<p>Redirecting…</p></body></html>`))
			return
		}
		code := q.Get("code")
		if code == "" {
			writeAuthErr(w, authservice.FieldError{Message: "Missing authorization code."})
			return
		}
		state := q.Get("state")
		if state == "" {
			writeAuthErr(w, authservice.FieldError{Message: "Missing state parameter."})
			return
		}
		res, nextPath, err := d.OIDC.CompleteLogin(r.Context(), d.Pool, d.JWTSigner, prov, code, state, authservice.ClientMetaFromRequest(r))
		if err != nil {
			writeAuthErr(w, err)
			return
		}
		frag := "access_token=" + url.QueryEscape(res.AccessToken) + "&token_type=" + url.QueryEscape(res.TokenType)
		if res.RefreshToken != "" {
			frag += "&refresh_token=" + url.QueryEscape(res.RefreshToken) + fmt.Sprintf("&expires_in=%d", res.ExpiresIn)
		}
		if res.MFAPendingToken != "" {
			frag += "&mfa_pending_token=" + url.QueryEscape(res.MFAPendingToken)
			if res.RequiresMFA {
				frag += "&requires_mfa=1"
			}
			if res.MFASetupRequired {
				frag += "&mfa_setup_required=1"
			}
		}
		next := "/"
		if nextPath != nil {
			np := strings.TrimSpace(*nextPath)
			if np != "" && np[0] == '/' {
				next = np
			}
		}
		public := d.OIDC.PublicWebBase()
		nextQ := ""
		if next != "/" {
			nextQ = "&next=" + url.QueryEscape(next)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Signing in</title></head>
<body><script>location.replace("` + public + `/saml-callback#` + frag + nextQ + `");</script>
<p>Redirecting to the app…</p></body></html>`
		_, _ = w.Write([]byte(html))
	}
}

type nativeAppleBody struct {
	IDToken           string  `json:"id_token"`
	RawNonce          string  `json:"raw_nonce"`
	AuthorizationCode *string `json:"authorization_code"`
	FullName          *string `json:"full_name"`
	Email             *string `json:"email"`
}

type nativeGoogleBody struct {
	IDToken  string  `json:"id_token"`
	RawNonce *string `json:"raw_nonce"`
}

// handleOIDCAppleNative is POST /api/v1/auth/oidc/apple/native — verifies an AuthenticationServices ID token.
func (d Deps) handleOIDCAppleNative() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database is not configured.")
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "JWT is not configured.")
			return
		}
		if d.OIDC == nil {
			d.OIDC = oidcauth.NewService(d.effectiveConfig())
		}
		var b nativeAppleBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req := oidcauth.NativeAppleLoginRequest{
			IDToken:  b.IDToken,
			RawNonce: b.RawNonce,
		}
		if b.AuthorizationCode != nil {
			req.AuthorizationCode = strings.TrimSpace(*b.AuthorizationCode)
		}
		if b.FullName != nil {
			req.FullName = strings.TrimSpace(*b.FullName)
		}
		if b.Email != nil {
			req.Email = strings.TrimSpace(*b.Email)
		}
		res, err := d.OIDC.CompleteNativeAppleLogin(r.Context(), d.Pool, d.JWTSigner, req, authservice.ClientMetaFromRequest(r))
		if err != nil {
			writeAuthErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(res)
	}
}

// handleOIDCGoogleNative is POST /api/v1/auth/oidc/google/native — verifies a Credential Manager Google ID token.
func (d Deps) handleOIDCGoogleNative() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database is not configured.")
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "JWT is not configured.")
			return
		}
		if d.OIDC == nil {
			d.OIDC = oidcauth.NewService(d.effectiveConfig())
		}
		var b nativeGoogleBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		req := oidcauth.NativeGoogleLoginRequest{IDToken: b.IDToken}
		if b.RawNonce != nil {
			req.RawNonce = strings.TrimSpace(*b.RawNonce)
		}
		res, err := d.OIDC.CompleteNativeGoogleLogin(r.Context(), d.Pool, d.JWTSigner, req, authservice.ClientMetaFromRequest(r))
		if err != nil {
			writeAuthErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(res)
	}
}
