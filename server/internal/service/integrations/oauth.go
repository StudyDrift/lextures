package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/crypto"
	integrationsrepo "github.com/lextures/lextures/server/internal/repos/integrations"
)

// ErrInvalidState is returned when an OAuth callback state fails verification.
var ErrInvalidState = errors.New("integrations: invalid or expired oauth state")

const stateTTL = 10 * time.Minute

// stateClaims is the payload embedded in a signed OAuth state token.
type stateClaims struct {
	OrgID    uuid.UUID `json:"o"`
	UserID   uuid.UUID `json:"u"`
	Provider string    `json:"p"`
	Nonce    string    `json:"n"`
	Exp      int64     `json:"e"`
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) signState(c stateClaims) string {
	payload, _ := json.Marshal(c)
	enc := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(enc))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return enc + "." + sig
}

func (s *Service) verifyState(token string) (stateClaims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return stateClaims{}, ErrInvalidState
	}
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return stateClaims{}, ErrInvalidState
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return stateClaims{}, ErrInvalidState
	}
	var c stateClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return stateClaims{}, ErrInvalidState
	}
	if s.now().Unix() > c.Exp {
		return stateClaims{}, ErrInvalidState
	}
	return c, nil
}

// AuthorizeURL builds the provider authorization URL and signed state for a
// connect request initiated by userID in orgID. Returns ErrNotConfigured when
// the provider has no OAuth client credentials.
func (s *Service) AuthorizeURL(p Provider, orgID, userID uuid.UUID) (string, error) {
	meta, err := s.Meta(p)
	if err != nil {
		return "", err
	}
	if !s.Configured(p) {
		return "", ErrNotConfigured
	}
	state := s.signState(stateClaims{
		OrgID:    orgID,
		UserID:   userID,
		Provider: string(p),
		Nonce:    randomNonce(),
		Exp:      s.now().Add(stateTTL).Unix(),
	})
	q := url.Values{}
	q.Set("client_id", s.Creds[p].ClientID)
	q.Set("redirect_uri", s.RedirectURI(p))
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(meta.Scopes, " "))
	q.Set("state", state)
	q.Set("access_type", "offline") // request a refresh token (Google)
	q.Set("prompt", "consent")
	sep := "?"
	if strings.Contains(meta.AuthorizeURL, "?") {
		sep = "&"
	}
	return meta.AuthorizeURL + sep + q.Encode(), nil
}

// tokenResponse is the OAuth 2 token endpoint response shape.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// exchangeCode swaps an authorization code for tokens at the provider token endpoint.
func (s *Service) exchangeCode(ctx context.Context, p Provider, code string) (Tokens, error) {
	return s.tokenRequest(ctx, p, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {s.RedirectURI(p)},
	})
}

// refreshTokens obtains a fresh access token using a stored refresh token.
func (s *Service) refreshTokens(ctx context.Context, p Provider, refreshToken string) (Tokens, error) {
	return s.tokenRequest(ctx, p, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (s *Service) tokenRequest(ctx context.Context, p Provider, form url.Values) (Tokens, error) {
	meta, err := s.Meta(p)
	if err != nil {
		return Tokens{}, err
	}
	creds := s.Creds[p]
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, meta.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return Tokens{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Tokens{}, fmt.Errorf("integrations: token endpoint status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return Tokens{}, fmt.Errorf("integrations: decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return Tokens{}, errors.New("integrations: token response missing access_token")
	}
	t := Tokens{AccessToken: tr.AccessToken, RefreshToken: tr.RefreshToken}
	if tr.Scope != "" {
		t.Scopes = strings.Fields(tr.Scope)
	}
	if tr.ExpiresIn > 0 {
		exp := s.now().Add(time.Duration(tr.ExpiresIn) * time.Second)
		t.ExpiresAt = &exp
	}
	return t, nil
}

// resolveAccountHTTP fetches the provider account/tenant id for fresh tokens.
func (s *Service) resolveAccountHTTP(ctx context.Context, p Provider, t Tokens) (string, error) {
	switch p {
	case ProviderGoogleClassroom:
		return s.googleUserID(ctx, t.AccessToken)
	default:
		// Microsoft/Canva: fall back to a stable hash of the token subject when
		// no dedicated identity endpoint is wired yet.
		sum := sha256.Sum256([]byte(string(p) + ":" + t.AccessToken))
		return base64.RawURLEncoding.EncodeToString(sum[:12]), nil
	}
}

const googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

func (s *Service) googleUserID(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("integrations: userinfo status %d", resp.StatusCode)
	}
	var info struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", err
	}
	if info.Sub != "" {
		return info.Sub, nil
	}
	if info.Email != "" {
		return info.Email, nil
	}
	return "", errors.New("integrations: userinfo missing subject")
}

// HandleCallback verifies state, exchanges the code, resolves the account id, and
// persists an encrypted connection. Returns the stored connection.
func (s *Service) HandleCallback(ctx context.Context, providerSlug, code, state string) (integrationsrepo.Connection, error) {
	claims, err := s.verifyState(state)
	if err != nil {
		return integrationsrepo.Connection{}, err
	}
	if claims.Provider != providerSlug {
		return integrationsrepo.Connection{}, ErrInvalidState
	}
	p, err := ParseProvider(providerSlug)
	if err != nil {
		return integrationsrepo.Connection{}, err
	}
	if code == "" {
		return integrationsrepo.Connection{}, errors.New("integrations: missing authorization code")
	}
	tokens, err := s.exchangeCode(ctx, p, code)
	if err != nil {
		return integrationsrepo.Connection{}, err
	}
	externalID, err := s.ResolveAccount(ctx, p, tokens)
	if err != nil {
		return integrationsrepo.Connection{}, fmt.Errorf("integrations: resolve account: %w", err)
	}
	accessEnc, err := crypto.EncryptString(tokens.AccessToken)
	if err != nil {
		return integrationsrepo.Connection{}, err
	}
	refreshEnc, err := crypto.EncryptString(tokens.RefreshToken)
	if err != nil {
		return integrationsrepo.Connection{}, err
	}
	scopes := tokens.Scopes
	if len(scopes) == 0 {
		if m, mErr := s.Meta(p); mErr == nil {
			scopes = m.Scopes
		}
	}
	connectedBy := claims.UserID
	return integrationsrepo.Upsert(ctx, s.Pool, integrationsrepo.UpsertParams{
		OrgID:           claims.OrgID,
		Provider:        string(p),
		ExternalID:      externalID,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenExpiresAt:  tokens.ExpiresAt,
		Scopes:          scopes,
		ConnectedBy:     &connectedBy,
	})
}

// freshAccessToken returns a valid access token for a connection, refreshing it
// (and persisting the new tokens) when the stored one is expired or near expiry.
func (s *Service) freshAccessToken(ctx context.Context, c integrationsrepo.Connection) (string, error) {
	access, refresh, err := s.connectionTokens(c)
	if err != nil {
		return "", err
	}
	if c.TokenExpiresAt == nil || s.now().Before(c.TokenExpiresAt.Add(-1*time.Minute)) {
		return access, nil
	}
	p, err := ParseProvider(c.Provider)
	if err != nil {
		return "", err
	}
	tokens, err := s.refreshTokens(ctx, p, refresh)
	if err != nil {
		return "", err
	}
	newAccessEnc, err := crypto.EncryptString(tokens.AccessToken)
	if err != nil {
		return "", err
	}
	newRefresh := refresh
	if tokens.RefreshToken != "" {
		newRefresh = tokens.RefreshToken
	}
	newRefreshEnc, err := crypto.EncryptString(newRefresh)
	if err != nil {
		return "", err
	}
	if _, err := integrationsrepo.Upsert(ctx, s.Pool, integrationsrepo.UpsertParams{
		OrgID:           c.OrgID,
		Provider:        c.Provider,
		ExternalID:      c.ExternalID,
		AccessTokenEnc:  newAccessEnc,
		RefreshTokenEnc: newRefreshEnc,
		TokenExpiresAt:  tokens.ExpiresAt,
		Scopes:          c.Scopes,
		ConnectedBy:     c.ConnectedBy,
	}); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}
