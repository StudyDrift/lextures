package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	repoMarket "github.com/lextures/lextures/server/internal/repos/marketplace"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	svcMarket "github.com/lextures/lextures/server/internal/service/marketplace"
)

func (d Deps) marketplaceEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFMarketplace {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Marketplace is not enabled on this server.")
		return false
	}
	return true
}

func (d Deps) marketplaceService() *svcMarket.Service {
	cfg := d.effectiveConfig()
	secret := cfg.JWTSecret
	if len(cfg.PlatformSecretsKey) > 0 {
		secret = string(cfg.PlatformSecretsKey)
	}
	return svcMarket.New([]byte(secret))
}

func (d Deps) registerMarketplaceRoutes(r chi.Router) {
	// Public marketplace listing (no auth required)
	r.Get("/api/v1/marketplace/apps", d.handleMarketplaceList())
	r.Get("/api/v1/marketplace/apps/{slug}", d.handleMarketplaceAppDetail())

	// OAuth 2.1 flow
	r.Get("/oauth/authorize", d.handleOAuthAuthorize())
	r.Post("/oauth/token", d.handleOAuthToken())
	r.Post("/oauth/revoke", d.handleOAuthRevoke())

	// Developer portal
	r.Get("/api/v1/developer/apps", d.handleDeveloperListApps())
	r.Post("/api/v1/developer/apps", d.handleDeveloperCreateApp())

	// Admin — installed apps
	r.Get("/api/v1/admin/marketplace/installed", d.handleAdminListInstalled())
	r.Delete("/api/v1/admin/marketplace/installed/{id}", d.handleAdminRevokeInstalled())
}

// --- Public marketplace ---

type marketplaceAppJSON struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Description     string   `json:"description"`
	LogoURL         *string  `json:"logoUrl"`
	RequestedScopes []string `json:"requestedScopes"`
}

func appToJSON(a repoMarket.App) marketplaceAppJSON {
	return marketplaceAppJSON{
		ID:              a.ID.String(),
		Name:            a.Name,
		Slug:            a.Slug,
		Description:     a.Description,
		LogoURL:         a.LogoURL,
		RequestedScopes: a.RequestedScopes,
	}
}

func (d Deps) handleMarketplaceList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		apps, err := repoMarket.ListPublishedApps(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list apps.")
			return
		}
		out := make([]marketplaceAppJSON, 0, len(apps))
		for _, a := range apps {
			out = append(out, appToJSON(a))
		}
		writeJSON(w, http.StatusOK, map[string]any{"apps": out})
	}
}

func (d Deps) handleMarketplaceAppDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		slug := chi.URLParam(r, "slug")
		a, err := repoMarket.GetAppBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to fetch app.")
			return
		}
		if a == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "App not found.")
			return
		}
		writeJSON(w, http.StatusOK, appToJSON(*a))
	}
}

// --- OAuth 2.1 authorize ---

type oauthAuthorizeResponse struct {
	State      string          `json:"state"`
	AppName    string          `json:"appName"`
	AppLogoURL *string         `json:"appLogoUrl"`
	Scopes     []scopeInfoJSON `json:"scopes"`
}

type scopeInfoJSON struct {
	Scope   string `json:"scope"`
	Label   string `json:"label"`
	IsWrite bool   `json:"isWrite"`
}

// GET /oauth/authorize?client_id=&redirect_uri=&scope=&code_challenge=&code_challenge_method=S256
// Returns JSON for the SPA to render the consent screen.
func (d Deps) handleOAuthAuthorize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		clientID := strings.TrimSpace(q.Get("client_id"))
		redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
		scopeStr := strings.TrimSpace(q.Get("scope"))
		codeChallenge := strings.TrimSpace(q.Get("code_challenge"))
		challengeMethod := strings.TrimSpace(q.Get("code_challenge_method"))

		if clientID == "" || redirectURI == "" || codeChallenge == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing required parameters: client_id, redirect_uri, code_challenge.")
			return
		}
		if challengeMethod != "S256" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Only code_challenge_method=S256 is supported.")
			return
		}

		app, err := repoMarket.GetAppByClientID(r.Context(), d.Pool, clientID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to look up app.")
			return
		}
		if app == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown client_id.")
			return
		}
		if !svcMarket.ValidateRedirectURI(app.RedirectURIs, redirectURI) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "redirect_uri not registered for this app.")
			return
		}

		scopes := splitScopes(scopeStr)
		grantedScopes := filterScopes(scopes, app.RequestedScopes)

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve org.")
			return
		}

		svc := d.marketplaceService()
		state, err := svc.BuildConsentURL(orgID, userID, clientID, redirectURI, grantedScopes, codeChallenge)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build consent state.")
			return
		}

		scopeInfos := make([]scopeInfoJSON, 0, len(grantedScopes))
		for _, s := range grantedScopes {
			scopeInfos = append(scopeInfos, scopeInfoJSON{
				Scope:   s,
				Label:   svcMarket.ScopeLabel(s),
				IsWrite: svcMarket.ScopeIsWrite(s),
			})
		}

		writeJSON(w, http.StatusOK, oauthAuthorizeResponse{
			State:      state,
			AppName:    app.Name,
			AppLogoURL: app.LogoURL,
			Scopes:     scopeInfos,
		})
	}
}

// POST /oauth/token — authorization_code or refresh_token grant
func (d Deps) handleOAuthToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		if err := r.ParseForm(); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form body.")
			return
		}
		grantType := r.FormValue("grant_type")

		switch grantType {
		case "authorization_code":
			d.handleOAuthTokenAuthCode(w, r)
		case "refresh_token":
			d.handleOAuthTokenRefresh(w, r)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported grant_type.")
		}
	}
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (d Deps) handleOAuthTokenAuthCode(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	code := r.FormValue("code") // the signed state returned from /oauth/authorize
	codeVerifier := r.FormValue("code_verifier")

	if clientID == "" || clientSecret == "" || code == "" || codeVerifier == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing required fields: client_id, client_secret, code, code_verifier.")
		return
	}

	app, valid, err := repoMarket.ValidateClientSecret(r.Context(), d.Pool, clientID, clientSecret)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate credentials.")
		return
	}
	if !valid || app == nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeInvalidCredentials, "Invalid client credentials.")
		return
	}

	svc := d.marketplaceService()
	claims, err := svc.VerifyConsentState(code)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid or expired authorization code.")
		return
	}
	if claims.ClientID != clientID {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "client_id mismatch.")
		return
	}
	if !svcMarket.VerifyPKCE(claims.CodeChallenge, codeVerifier) {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid code_verifier.")
		return
	}

	accessToken, accessHash, accessPrefix, err := repoMarket.GenerateAccessToken()
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate access token.")
		return
	}
	refreshToken, refreshHash, refreshPrefix, err := repoMarket.GenerateRefreshToken()
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate refresh token.")
		return
	}

	_, err = repoMarket.CreateInstallation(r.Context(), d.Pool, repoMarket.CreateInstallationParams{
		AppID:              app.ID,
		OrgID:              claims.OrgID,
		AccessTokenHash:    accessHash,
		AccessTokenPrefix:  accessPrefix,
		RefreshTokenHash:   refreshHash,
		RefreshTokenPrefix: refreshPrefix,
		GrantedScopes:      claims.Scopes,
		InstalledBy:        claims.UserID,
	})
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record installation.")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, oauthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	})
}

func (d Deps) handleOAuthTokenRefresh(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	rawRefresh := r.FormValue("refresh_token")

	if clientID == "" || clientSecret == "" || rawRefresh == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing required fields: client_id, client_secret, refresh_token.")
		return
	}

	app, valid, err := repoMarket.ValidateClientSecret(r.Context(), d.Pool, clientID, clientSecret)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate credentials.")
		return
	}
	if !valid || app == nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeInvalidCredentials, "Invalid client credentials.")
		return
	}

	ins, err := repoMarket.GetInstallationByRefreshToken(r.Context(), d.Pool, rawRefresh)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to look up token.")
		return
	}
	if ins == nil || ins.AppID != app.ID {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeInvalidCredentials, "Invalid refresh token.")
		return
	}

	newAccess, newAccessHash, newAccessPrefix, err := repoMarket.GenerateAccessToken()
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate access token.")
		return
	}
	newRefresh, newRefreshHash, newRefreshPrefix, err := repoMarket.GenerateRefreshToken()
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate refresh token.")
		return
	}

	if err := repoMarket.RotateTokens(r.Context(), d.Pool, ins.ID,
		newAccessHash, newAccessPrefix, newRefreshHash, newRefreshPrefix); err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to rotate tokens.")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, oauthTokenResponse{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	})
}

// POST /oauth/revoke
func (d Deps) handleOAuthRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		if err := r.ParseForm(); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid form body.")
			return
		}
		rawToken := r.FormValue("token")
		if rawToken == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing token parameter.")
			return
		}
		ins, err := repoMarket.GetInstallationByAccessToken(r.Context(), d.Pool, rawToken)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to look up token.")
			return
		}
		if ins == nil {
			// Per RFC 7009: already revoked or not found → 200 OK
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := repoMarket.RevokeInstallation(r.Context(), d.Pool, ins.ID, ins.OrgID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke token.")
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// --- Developer portal ---

type developerAppJSON struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Slug               string   `json:"slug"`
	Description        string   `json:"description"`
	LogoURL            *string  `json:"logoUrl"`
	ClientID           string   `json:"clientId"`
	ClientSecretPrefix string   `json:"clientSecretPrefix"`
	RedirectURIs       []string `json:"redirectUris"`
	RequestedScopes    []string `json:"requestedScopes"`
	Published          bool     `json:"published"`
	CreatedAt          string   `json:"createdAt"`
}

func devAppToJSON(a repoMarket.App) developerAppJSON {
	return developerAppJSON{
		ID:                 a.ID.String(),
		Name:               a.Name,
		Slug:               a.Slug,
		Description:        a.Description,
		LogoURL:            a.LogoURL,
		ClientID:           a.ClientID,
		ClientSecretPrefix: a.ClientSecretPrefix,
		RedirectURIs:       a.RedirectURIs,
		RequestedScopes:    a.RequestedScopes,
		Published:          a.Published,
		CreatedAt:          a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (d Deps) handleDeveloperListApps() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		apps, err := repoMarket.ListAppsByDeveloper(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list apps.")
			return
		}
		out := make([]developerAppJSON, 0, len(apps))
		for _, a := range apps {
			out = append(out, devAppToJSON(a))
		}
		writeJSON(w, http.StatusOK, map[string]any{"apps": out})
	}
}

type createAppInput struct {
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Description     string   `json:"description"`
	LogoURL         *string  `json:"logoUrl"`
	RedirectURIs    []string `json:"redirectUris"`
	RequestedScopes []string `json:"requestedScopes"`
}

type createAppResponse struct {
	developerAppJSON
	ClientSecret string `json:"clientSecret"`
}

func (d Deps) handleDeveloperCreateApp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var in createAppInput
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Slug) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name and slug are required.")
			return
		}
		if len(in.RedirectURIs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "At least one redirect_uri is required.")
			return
		}

		rawSecret, secretHash, secretPrefix, err := repoMarket.GenerateClientSecret()
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate credentials.")
			return
		}

		app, err := repoMarket.CreateApp(r.Context(), d.Pool, repoMarket.CreateAppParams{
			DeveloperUserID:    userID,
			Name:               in.Name,
			Slug:               in.Slug,
			Description:        in.Description,
			LogoURL:            in.LogoURL,
			RedirectURIs:       in.RedirectURIs,
			RequestedScopes:    in.RequestedScopes,
			ClientSecretHash:   secretHash,
			ClientSecretPrefix: secretPrefix,
		})
		if err != nil {
			if err == repoMarket.ErrDuplicateSlug {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "App slug is already taken.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create app.")
			return
		}

		writeJSON(w, http.StatusCreated, createAppResponse{
			developerAppJSON: devAppToJSON(app),
			ClientSecret:     rawSecret,
		})
	}
}

// --- Admin installed apps ---

type installedAppJSON struct {
	ID            string   `json:"id"`
	AppID         string   `json:"appId"`
	AppName       string   `json:"appName"`
	AppSlug       string   `json:"appSlug"`
	AppLogoURL    *string  `json:"appLogoUrl"`
	GrantedScopes []string `json:"grantedScopes"`
	InstalledAt   string   `json:"installedAt"`
	InstalledBy   *string  `json:"installedBy"`
	LastUsedAt    *string  `json:"lastUsedAt"`
}

func installationToJSON(ins repoMarket.Installation) installedAppJSON {
	out := installedAppJSON{
		ID:            ins.ID.String(),
		AppID:         ins.AppID.String(),
		AppName:       ins.AppName,
		AppSlug:       ins.AppSlug,
		AppLogoURL:    ins.AppLogoURL,
		GrantedScopes: ins.GrantedScopes,
		InstalledAt:   ins.InstalledAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if ins.InstalledBy != nil {
		s := ins.InstalledBy.String()
		out.InstalledBy = &s
	}
	if ins.LastUsedAt != nil {
		s := ins.LastUsedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		out.LastUsedAt = &s
	}
	return out
}

func (d Deps) handleAdminListInstalled() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		actorID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, actorID, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve org.")
			return
		}
		installations, err := repoMarket.ListInstallationsByOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list installed apps.")
			return
		}
		out := make([]installedAppJSON, 0, len(installations))
		for _, ins := range installations {
			out = append(out, installationToJSON(ins))
		}
		writeJSON(w, http.StatusOK, map[string]any{"installations": out})
	}
}

func (d Deps) handleAdminRevokeInstalled() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.marketplaceEnabled(w) {
			return
		}
		actorID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, actorID, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve org.")
			return
		}
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid installation id.")
			return
		}
		if err := repoMarket.RevokeInstallation(r.Context(), d.Pool, id, orgID); err != nil {
			if err == repoMarket.ErrNotFound {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Installation not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke installation.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- helpers ---

func splitScopes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func filterScopes(requested, allowed []string) []string {
	allowedSet := make(map[string]bool, len(allowed))
	for _, s := range allowed {
		allowedSet[s] = true
	}
	var out []string
	for _, s := range requested {
		if allowedSet[s] {
			out = append(out, s)
		}
	}
	return out
}
