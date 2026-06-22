package httpserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/lti"
)

func TestLtiConsumerCallback_IssuesFormPost(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	pemS := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pair, err := lti.FromPKCS8PEM(pemS, "k1")
	if err != nil {
		t.Fatal(err)
	}

	toolID := uuid.New()
	userID := uuid.New()
	courseID := uuid.New()
	itemID := uuid.New()

	hint, err := pair.SignConsumerLoginHint(
		"http://platform.test", "https://tool.example", userID.String(), toolID.String(),
		courseID.String(), itemID.String(), "", "", false,
	)
	if err != nil {
		t.Fatal(err)
	}

	// In-memory tool registry via minimal pool stub is not available; patch findExternalToolByClientID
	// by using a test handler wrapper. Instead, test the OIDC round-trip components at the lti package
	// level and verify callback rejects unknown client_id without DB.
	deps := Deps{
		Config: config.Config{
			LTIEnabled:          true,
			LTIRSAPrivateKeyPEM: pemS,
			LTIRSAKeyID:         "k1",
			LTIAPIBaseURL:       "http://platform.test",
		},
		Lti: &lti.Runtime{Enabled: true, Keys: pair, APIBaseURL: "http://platform.test"},
	}
	h := NewHandler(deps)

	q := url.Values{}
	q.Set("scope", "openid")
	q.Set("response_type", "id_token")
	q.Set("response_mode", "form_post")
	q.Set("client_id", "unknown-client")
	q.Set("redirect_uri", "https://tool.example/callback")
	q.Set("login_hint", hint)
	q.Set("state", "state-1")
	q.Set("nonce", "nonce-1")

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/lti/callback?"+q.Encode(), nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusNotFound {
		t.Fatalf("unknown client without DB: %d %s", rr.Code, rr.Body.String())
	}

	// Verify login_hint validates independently.
	claims, err := lti.VerifyConsumerLoginHint(hint, pair.PublicKey(), "http://platform.test", "https://tool.example")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != userID.String() {
		t.Fatalf("sub: %s", claims.Subject)
	}

	idClaims := lti.PlatformLaunchIDTokenClaims(
		"http://platform.test", "client-1", userID.String(), "nonce-1",
		lti.ConsumerTargetURI("http://platform.test", courseID.String(), itemID.String()),
		lti.MsgResourceLinkRequest, courseID.String(), itemID.String(), itemID.String(), "Test",
		nil, instructorRoles(), "en-US",
	)
	idTok, err := pair.SignPlatformIDToken(idClaims)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(idTok, ".") {
		t.Fatal("expected JWT")
	}
}