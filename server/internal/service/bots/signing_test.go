package bots

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestVerifySlackSignature_Valid(t *testing.T) {
	secret := "test-signing-secret"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := []byte(`command=/lextures&text=upcoming`)
	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	if !VerifySlackSignature(secret, ts, body, sig) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifySlackSignature_TamperedReturnsFalse(t *testing.T) {
	if VerifySlackSignature("secret", "123", []byte("x"), "v0=deadbeef") {
		t.Fatal("expected invalid signature")
	}
}

func TestUpcomingText_Empty(t *testing.T) {
	if UpcomingText(nil) == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestSlackBlocks_IncludesTitle(t *testing.T) {
	blocks := SlackBlocks(EventPayload{EventType: "assignment.created", Title: "Essay 1", CourseCode: "ENG-101"})
	if blocks["blocks"] == nil {
		t.Fatal("expected blocks")
	}
}
