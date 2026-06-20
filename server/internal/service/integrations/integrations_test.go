package integrations

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/crypto"
	integrationsrepo "github.com/lextures/lextures/server/internal/repos/integrations"
)

func encryptForTest(plain string) (string, error) {
	return crypto.EncryptString(plain)
}

func connForTest(accessEnc, refreshEnc string, expires *time.Time) integrationsrepo.Connection {
	return integrationsrepo.Connection{
		ID:              uuid.New(),
		OrgID:           uuid.New(),
		Provider:        string(ProviderGoogleClassroom),
		ExternalID:      "ext-1",
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenExpiresAt:  expires,
	}
}

func testService() *Service {
	return &Service{
		PublicBase:  "https://lms.example.edu",
		StateSecret: []byte("test-state-secret-0123456789"),
		Providers:   DefaultProviders(),
		Creds: map[Provider]OAuthCredentials{
			ProviderGoogleClassroom: {ClientID: "cid", ClientSecret: "secret"},
		},
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	}
}

func TestParseProvider(t *testing.T) {
	for _, ok := range []string{"google_classroom", "Microsoft_Teams ", "canva"} {
		if _, err := ParseProvider(ok); err != nil {
			t.Errorf("ParseProvider(%q) unexpected error: %v", ok, err)
		}
	}
	if _, err := ParseProvider("dropbox"); err == nil {
		t.Error("ParseProvider(dropbox) expected error")
	}
}

func TestConfigured(t *testing.T) {
	s := testService()
	if !s.Configured(ProviderGoogleClassroom) {
		t.Error("google_classroom should be configured")
	}
	if s.Configured(ProviderCanva) {
		t.Error("canva should not be configured without creds")
	}
}

func TestRedirectURI(t *testing.T) {
	s := testService()
	got := s.RedirectURI(ProviderGoogleClassroom)
	want := "https://lms.example.edu/integrations/oauth/google_classroom/callback"
	if got != want {
		t.Errorf("RedirectURI = %q, want %q", got, want)
	}
}

func TestAuthorizeURLAndState(t *testing.T) {
	s := testService()
	org, user := uuid.New(), uuid.New()
	u, err := s.AuthorizeURL(ProviderGoogleClassroom, org, user)
	if err != nil {
		t.Fatalf("AuthorizeURL error: %v", err)
	}
	if got := substringAfter(u, "state="); got == "" {
		t.Fatal("authorize URL missing state")
	}
	// Unconfigured provider must be rejected.
	if _, err := s.AuthorizeURL(ProviderCanva, org, user); err != ErrNotConfigured {
		t.Errorf("AuthorizeURL(canva) = %v, want ErrNotConfigured", err)
	}
}

func TestStateRoundTripAndTamper(t *testing.T) {
	s := testService()
	org, user := uuid.New(), uuid.New()
	state := s.signState(stateClaims{
		OrgID:    org,
		UserID:   user,
		Provider: "google_classroom",
		Nonce:    "n",
		Exp:      s.now().Add(stateTTL).Unix(),
	})
	claims, err := s.verifyState(state)
	if err != nil {
		t.Fatalf("verifyState error: %v", err)
	}
	if claims.OrgID != org || claims.UserID != user {
		t.Error("verifyState lost claim data")
	}
	// Tampered signature must fail.
	if _, err := s.verifyState(state + "x"); err == nil {
		t.Error("tampered state should fail verification")
	}
	// Expired state must fail.
	expired := s.signState(stateClaims{Provider: "google_classroom", Exp: s.now().Add(-time.Minute).Unix()})
	if _, err := s.verifyState(expired); err != ErrInvalidState {
		t.Errorf("expired state = %v, want ErrInvalidState", err)
	}
}

func substringAfter(s, sep string) string {
	i := indexOf(s, sep)
	if i < 0 {
		return ""
	}
	return s[i+len(sep):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
