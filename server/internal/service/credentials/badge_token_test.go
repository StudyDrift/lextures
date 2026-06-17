package credentials

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

func TestBadgeExportTokenRoundTrip(t *testing.T) {
	cfg := config.Config{JWTSecret: "01234567890123456789012345678901"}
	id := uuid.New()
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	token, expires, err := BadgeExportToken(cfg, id, now)
	if err != nil {
		t.Fatalf("BadgeExportToken: %v", err)
	}
	if !expires.After(now) {
		t.Fatalf("expires should be in the future")
	}
	parsed, err := VerifyBadgeExportToken(cfg, token, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("VerifyBadgeExportToken: %v", err)
	}
	if parsed != id {
		t.Fatalf("id mismatch: %s vs %s", parsed, id)
	}
	_, err = VerifyBadgeExportToken(cfg, token, now.Add(48*time.Hour))
	if err == nil {
		t.Fatal("expected expired token error")
	}
}