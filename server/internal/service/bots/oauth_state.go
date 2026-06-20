package bots

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type oauthState struct {
	OrgID    uuid.UUID `json:"o,omitempty"`
	UserID   uuid.UUID `json:"u"`
	Platform string    `json:"p"`
	Exp      int64     `json:"e"`
}

func (s *Service) signOAuthState(st oauthState) string {
	st.Exp = s.now().Add(10 * time.Minute).Unix()
	payload, _ := json.Marshal(st)
	enc := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(enc))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return enc + "." + sig
}

func (s *Service) verifyOAuthState(token string) (oauthState, error) {
	parts := split2(token, ".")
	if len(parts) != 2 {
		return oauthState{}, ErrInvalidState
	}
	mac := hmac.New(sha256.New, s.StateSecret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return oauthState{}, ErrInvalidState
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return oauthState{}, ErrInvalidState
	}
	var st oauthState
	if err := json.Unmarshal(raw, &st); err != nil {
		return oauthState{}, ErrInvalidState
	}
	if s.now().Unix() > st.Exp {
		return oauthState{}, ErrInvalidState
	}
	return st, nil
}

func split2(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] && i+1 < len(s) {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
