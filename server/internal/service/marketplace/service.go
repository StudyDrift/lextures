// Package marketplace implements OAuth 2.1 app authorization for the plugin marketplace (plan 16.9).
package marketplace

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidState is returned when an OAuth callback state fails verification.
var ErrInvalidState = errors.New("marketplace: invalid or expired oauth state")

// ErrInvalidCodeChallenge is returned when PKCE code_verifier doesn't match challenge.
var ErrInvalidCodeChallenge = errors.New("marketplace: invalid pkce code_verifier")

// ErrInvalidRedirectURI is returned when the redirect_uri is not registered.
var ErrInvalidRedirectURI = errors.New("marketplace: redirect_uri not registered for this app")

// ErrInvalidClientCredentials is returned when client_id/secret are wrong.
var ErrInvalidClientCredentials = errors.New("marketplace: invalid client credentials")

const stateTTL = 10 * time.Minute

// OAuthStateClaims is the signed payload stored in the OAuth state parameter.
type OAuthStateClaims struct {
	ClientID      string    `json:"c"`
	OrgID         uuid.UUID `json:"o"`
	UserID        uuid.UUID `json:"u"`
	RedirectURI   string    `json:"r"`
	Scopes        []string  `json:"s"`
	CodeChallenge string    `json:"cc"` // PKCE S256 challenge
	Nonce         string    `json:"n"`
	Exp           int64     `json:"e"`
}

// Service holds dependencies for the marketplace OAuth flow.
type Service struct {
	StateSecret []byte
	now         func() time.Time
}

// New returns a ready Service.
func New(stateSecret []byte) *Service {
	return &Service{StateSecret: stateSecret, now: time.Now}
}

// BuildConsentURL creates a signed state token for the OAuth consent redirect.
// Returns the state string that the frontend uses to build /oauth/authorize.
func (s *Service) BuildConsentURL(orgID, userID uuid.UUID, clientID, redirectURI string, scopes []string, codeChallenge string) (string, error) {
	state := s.signState(OAuthStateClaims{
		ClientID:      clientID,
		OrgID:         orgID,
		UserID:        userID,
		RedirectURI:   redirectURI,
		Scopes:        scopes,
		CodeChallenge: codeChallenge,
		Nonce:         randomNonce(),
		Exp:           s.now().Add(stateTTL).Unix(),
	})
	return state, nil
}

// VerifyConsentState verifies the signed state and returns the embedded claims.
func (s *Service) VerifyConsentState(state string) (OAuthStateClaims, error) {
	return s.verifyState(state)
}

// VerifyPKCE checks that SHA-256(verifier) == base64url(challenge).
func VerifyPKCE(challenge, verifier string) bool {
	return verifyPKCE(challenge, verifier)
}

func verifyPKCE(challenge, verifier string) bool {
	if challenge == "" || verifier == "" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return computed == challenge
}

// ScopeLabel returns a human-readable label for a known scope.
func ScopeLabel(scope string) string {
	labels := map[string]string{
		"courses:read":     "Read course information",
		"courses:write":    "Create and modify courses",
		"enrollments:read": "Read enrollment records",
		"users:read":       "Read user profiles",
		"grades:read":      "Read grade data",
		"grades:write":     "This app can modify grades in your organization.",
		"assignments:read": "Read assignment details",
		"pii:read":         "Access personally identifiable information",
	}
	if l, ok := labels[scope]; ok {
		return l
	}
	return scope
}

// ScopeIsWrite returns true if the scope grants write access (used for consent screen warnings).
func ScopeIsWrite(scope string) bool {
	return strings.HasSuffix(scope, ":write")
}

// ValidateRedirectURI checks that redirectURI is in the registered list.
func ValidateRedirectURI(registeredURIs []string, redirectURI string) bool {
	for _, u := range registeredURIs {
		if u == redirectURI {
			return true
		}
	}
	return false
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) signState(c OAuthStateClaims) string {
	payload, _ := json.Marshal(c)
	enc := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(enc))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return enc + "." + sig
}

func (s *Service) verifyState(token string) (OAuthStateClaims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return OAuthStateClaims{}, ErrInvalidState
	}
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return OAuthStateClaims{}, ErrInvalidState
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return OAuthStateClaims{}, ErrInvalidState
	}
	var c OAuthStateClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return OAuthStateClaims{}, ErrInvalidState
	}
	if s.now().Unix() > c.Exp {
		return OAuthStateClaims{}, ErrInvalidState
	}
	return c, nil
}
