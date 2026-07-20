package turn

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultTTL is the ephemeral TURN credential lifetime (FR-11: ≤ 12h).
const DefaultTTL = 12 * time.Hour

// Creds are coturn REST time-limited credentials.
type Creds struct {
	Username   string
	Credential string
	TTLSeconds int64
	ExpiresAt  time.Time
}

// Mint generates coturn REST credentials: username = "<expiryUnix>:<userID>",
// credential = base64(HMAC-SHA1(secret, username)).
func Mint(secret, userID string, now time.Time, ttl time.Duration) (Creds, error) {
	secret = strings.TrimSpace(secret)
	userID = strings.TrimSpace(userID)
	if secret == "" {
		return Creds{}, fmt.Errorf("screenshare/turn: empty shared secret")
	}
	if userID == "" {
		return Creds{}, fmt.Errorf("screenshare/turn: empty user id")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > 12*time.Hour {
		ttl = 12 * time.Hour
	}
	exp := now.UTC().Add(ttl)
	username := fmt.Sprintf("%d:%s", exp.Unix(), userID)
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(username))
	cred := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return Creds{
		Username:   username,
		Credential: cred,
		TTLSeconds: int64(ttl.Seconds()),
		ExpiresAt:  exp,
	}, nil
}

// ValidateExpiry checks that a coturn REST username has not expired.
func ValidateExpiry(username string, now time.Time) bool {
	parts := strings.SplitN(username, ":", 2)
	if len(parts) != 2 {
		return false
	}
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	return now.UTC().Unix() < sec
}
