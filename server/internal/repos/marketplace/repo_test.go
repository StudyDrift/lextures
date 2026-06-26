package marketplace_test

import (
	"testing"

	repoMarket "github.com/lextures/lextures/server/internal/repos/marketplace"
)

func TestGenerateClientSecret_Format(t *testing.T) {
	secret, hash, prefix, err := repoMarket.GenerateClientSecret()
	if err != nil {
		t.Fatalf("GenerateClientSecret error: %v", err)
	}
	if len(secret) < 10 {
		t.Errorf("secret too short: %q", secret)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if prefix == "" || len(prefix) != 8 {
		t.Errorf("prefix should be 8 chars, got %q", prefix)
	}
	if secret[:8] != prefix {
		t.Errorf("prefix mismatch: secret[:8]=%q prefix=%q", secret[:8], prefix)
	}
}

func TestGenerateAccessToken_Format(t *testing.T) {
	token, hash, prefix, err := repoMarket.GenerateAccessToken()
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	if len(token) < 10 {
		t.Error("access token too short")
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if len(prefix) != 8 {
		t.Errorf("prefix should be 8 chars, got %q", prefix)
	}
}

func TestGenerateRefreshToken_Format(t *testing.T) {
	token, hash, prefix, err := repoMarket.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if len(token) < 10 {
		t.Error("refresh token too short")
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if len(prefix) != 8 {
		t.Errorf("prefix should be 8 chars, got %q", prefix)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "mkt_abc123"
	h1 := repoMarket.HashToken(token)
	h2 := repoMarket.HashToken(token)
	if h1 != h2 {
		t.Error("HashToken should be deterministic")
	}
	if h1 == "" {
		t.Error("HashToken should not return empty string")
	}
}

func TestHashToken_DifferentForDifferentTokens(t *testing.T) {
	h1 := repoMarket.HashToken("mkt_token1")
	h2 := repoMarket.HashToken("mkt_token2")
	if h1 == h2 {
		t.Error("different tokens should have different hashes")
	}
}

func TestTokensAreUnique(t *testing.T) {
	t1, _, _, _ := repoMarket.GenerateAccessToken()
	t2, _, _, _ := repoMarket.GenerateAccessToken()
	if t1 == t2 {
		t.Error("two generated tokens should be unique")
	}
}
