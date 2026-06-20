// Package integrations implements inbound third-party connectors for plan 16.4:
// Google Classroom import/roster sync, Microsoft Teams Education roster sync,
// Canva embeds, and LTI 1.1 external-tool launches. OAuth tokens are stored
// encrypted at rest via internal/crypto.
//
// Each provider is isolated behind the provider registry and an injectable HTTP
// client / account resolver so the orchestration can be unit-tested without
// reaching real Google/Microsoft/Canva endpoints.
package integrations

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/crypto"
	integrationsrepo "github.com/lextures/lextures/server/internal/repos/integrations"
)

// Provider identifies a connectable third-party platform.
type Provider string

const (
	ProviderGoogleClassroom Provider = "google_classroom"
	ProviderMicrosoftTeams  Provider = "microsoft_teams"
	ProviderCanva           Provider = "canva"
)

// ErrUnknownProvider is returned for an unrecognised provider slug.
var ErrUnknownProvider = errors.New("integrations: unknown provider")

// ErrNotConfigured is returned when a provider has no OAuth client credentials.
var ErrNotConfigured = errors.New("integrations: provider not configured")

// ProviderMeta describes a provider's OAuth endpoints and requested scopes.
type ProviderMeta struct {
	Provider     Provider
	DisplayName  string
	AuthorizeURL string
	TokenURL     string
	// Scopes are read-only by policy (plan 16.4 §6 Security).
	Scopes []string
}

// OAuthCredentials holds a provider's registered OAuth client id/secret.
type OAuthCredentials struct {
	ClientID     string
	ClientSecret string
}

// Tokens is the normalised result of an OAuth token exchange/refresh.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	Scopes       []string
}

// Service orchestrates OAuth connect/callback, import, and sync for all providers.
type Service struct {
	Pool *pgxpool.Pool
	HTTP *http.Client
	// PublicBase is the externally reachable base URL used to build OAuth redirect URIs.
	PublicBase string
	// StateSecret signs OAuth state tokens (CSRF protection).
	StateSecret []byte
	// Creds maps a provider to its OAuth client credentials.
	Creds map[Provider]OAuthCredentials
	// Providers is the provider registry (endpoints + scopes).
	Providers map[Provider]ProviderMeta
	// Classroom is the Google Classroom API client; injectable for tests.
	Classroom ClassroomClient
	// ResolveAccount returns the provider account/tenant id for freshly issued tokens.
	// Injectable for tests; defaults to a provider-aware HTTP implementation.
	ResolveAccount func(ctx context.Context, p Provider, t Tokens) (string, error)
	// Now is the clock, overridable in tests.
	Now func() time.Time
}

// DefaultProviders returns the built-in provider registry.
func DefaultProviders() map[Provider]ProviderMeta {
	return map[Provider]ProviderMeta{
		ProviderGoogleClassroom: {
			Provider:     ProviderGoogleClassroom,
			DisplayName:  "Google Classroom",
			AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"openid", "email",
				"https://www.googleapis.com/auth/classroom.courses.readonly",
				"https://www.googleapis.com/auth/classroom.rosters.readonly",
				"https://www.googleapis.com/auth/classroom.coursework.students.readonly",
			},
		},
		ProviderMicrosoftTeams: {
			Provider:     ProviderMicrosoftTeams,
			DisplayName:  "Microsoft Teams Education",
			AuthorizeURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			Scopes:       []string{"openid", "email", "offline_access", "EduRoster.ReadBasic"},
		},
		ProviderCanva: {
			Provider:     ProviderCanva,
			DisplayName:  "Canva for Education",
			AuthorizeURL: "https://www.canva.com/api/oauth/authorize",
			TokenURL:     "https://api.canva.com/rest/v1/oauth/token",
			Scopes:       []string{"design:content:read"},
		},
	}
}

// NewService builds a Service with the default provider registry and credentials
// read from the environment. PublicBase should be the externally reachable URL.
func NewService(pool *pgxpool.Pool, publicBase string, stateSecret []byte) *Service {
	s := &Service{
		Pool:        pool,
		HTTP:        &http.Client{Timeout: 30 * time.Second},
		PublicBase:  strings.TrimRight(publicBase, "/"),
		StateSecret: stateSecret,
		Providers:   DefaultProviders(),
		Creds:       credsFromEnv(),
		Now:         time.Now,
	}
	s.Classroom = &httpClassroomClient{http: s.HTTP}
	s.ResolveAccount = s.resolveAccountHTTP
	return s
}

func credsFromEnv() map[Provider]OAuthCredentials {
	read := func(prefix string) OAuthCredentials {
		return OAuthCredentials{
			ClientID:     strings.TrimSpace(os.Getenv(prefix + "_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv(prefix + "_CLIENT_SECRET")),
		}
	}
	return map[Provider]OAuthCredentials{
		ProviderGoogleClassroom: read("GOOGLE_CLASSROOM"),
		ProviderMicrosoftTeams:  read("MICROSOFT_TEAMS"),
		ProviderCanva:           read("CANVA"),
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Meta returns the provider registry entry, or an error for unknown providers.
func (s *Service) Meta(p Provider) (ProviderMeta, error) {
	m, ok := s.Providers[p]
	if !ok {
		return ProviderMeta{}, ErrUnknownProvider
	}
	return m, nil
}

// Configured reports whether a provider has usable OAuth client credentials.
func (s *Service) Configured(p Provider) bool {
	c, ok := s.Creds[p]
	return ok && c.ClientID != "" && c.ClientSecret != ""
}

// ParseProvider validates a provider slug.
func ParseProvider(s string) (Provider, error) {
	switch Provider(strings.ToLower(strings.TrimSpace(s))) {
	case ProviderGoogleClassroom:
		return ProviderGoogleClassroom, nil
	case ProviderMicrosoftTeams:
		return ProviderMicrosoftTeams, nil
	case ProviderCanva:
		return ProviderCanva, nil
	default:
		return "", ErrUnknownProvider
	}
}

// RedirectURI is the OAuth callback URL for a provider.
func (s *Service) RedirectURI(p Provider) string {
	return s.PublicBase + "/integrations/oauth/" + string(p) + "/callback"
}

// ConnectionView is a redacted connection projection safe to return over the API.
type ConnectionView struct {
	ID            uuid.UUID  `json:"id"`
	Provider      string     `json:"provider"`
	DisplayName   string     `json:"displayName"`
	ExternalID    string     `json:"externalId"`
	Scopes        []string   `json:"scopes"`
	LastSyncedAt  *time.Time `json:"lastSyncedAt,omitempty"`
	LastSyncError *string    `json:"lastSyncError,omitempty"`
	Connected     bool       `json:"connected"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func (s *Service) view(c integrationsrepo.Connection) ConnectionView {
	display := string(c.Provider)
	if m, ok := s.Providers[Provider(c.Provider)]; ok {
		display = m.DisplayName
	}
	scopes := c.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	return ConnectionView{
		ID:            c.ID,
		Provider:      c.Provider,
		DisplayName:   display,
		ExternalID:    c.ExternalID,
		Scopes:        scopes,
		LastSyncedAt:  c.LastSyncedAt,
		LastSyncError: c.LastSyncError,
		Connected:     true,
		CreatedAt:     c.CreatedAt,
	}
}

// List returns redacted connection views for an org, plus catalog entries for
// providers that are configured but not yet connected.
func (s *Service) List(ctx context.Context, orgID uuid.UUID) ([]ConnectionView, error) {
	conns, err := integrationsrepo.ListByOrg(ctx, s.Pool, orgID)
	if err != nil {
		return nil, err
	}
	connected := make(map[string]bool, len(conns))
	out := make([]ConnectionView, 0, len(conns)+len(s.Providers))
	for _, c := range conns {
		connected[c.Provider] = true
		out = append(out, s.view(c))
	}
	// Surface configured-but-unconnected providers so the admin UI can render
	// a "Connect" card for each available connector.
	for p, m := range s.Providers {
		if connected[string(p)] {
			continue
		}
		out = append(out, ConnectionView{
			Provider:    string(p),
			DisplayName: m.DisplayName,
			Scopes:      []string{},
			Connected:   false,
		})
	}
	return out, nil
}

// Disconnect removes a connection. Imported content is retained (plan 16.4 §6
// Backward compatibility): only the OAuth grant and sync schedule are removed.
func (s *Service) Disconnect(ctx context.Context, orgID, id uuid.UUID) error {
	return integrationsrepo.Delete(ctx, s.Pool, orgID, id)
}

// connectionTokens decrypts the stored access/refresh tokens for a connection.
func (s *Service) connectionTokens(c integrationsrepo.Connection) (access, refresh string, err error) {
	access, err = crypto.DecryptString(c.AccessTokenEnc)
	if err != nil {
		return "", "", err
	}
	refresh, err = crypto.DecryptString(c.RefreshTokenEnc)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}
