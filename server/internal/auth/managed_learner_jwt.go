package auth

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const managedLearnerTTL = 8 * time.Hour

// ManagedLearnerUser is the identity encoded in a parent Learn-as JWT (#640).
type ManagedLearnerUser struct {
	ActorID      string // parent / managing adult
	TargetUserID string // managed child
	TargetEmail  string
	OrgID        string
	OrgSlug      string
	JTI          string
	ExpiresAt    time.Time
}

// SignManagedLearner issues a Learn-as JWT for a parent switching into a managed learner (#640).
func (s *JWTSigner) SignManagedLearner(actorID, targetUserID, targetEmail, orgID, orgSlug string) (string, ManagedLearnerUser, error) {
	if !isUUID(actorID) || !isUUID(targetUserID) || strings.TrimSpace(targetEmail) == "" {
		return "", ManagedLearnerUser{}, ErrInvalidToken
	}
	if orgID != "" && !isUUID(orgID) {
		return "", ManagedLearnerUser{}, ErrInvalidToken
	}
	jti := uuid.NewString()
	exp := s.now().Add(managedLearnerTTL)
	claims := managedLearnerClaims{
		Typ:     "managed_learner",
		ActorID: actorID,
		Subject: targetUserID,
		Email:   targetEmail,
		OrgID:   orgID,
		OrgSlug: orgSlug,
		JTI:     jti,
		Expires: unixSeconds(exp),
	}
	tok, err := s.sign(claims)
	if err != nil {
		return "", ManagedLearnerUser{}, err
	}
	return tok, ManagedLearnerUser{
		ActorID:      actorID,
		TargetUserID: targetUserID,
		TargetEmail:  targetEmail,
		OrgID:        strings.TrimSpace(orgID),
		OrgSlug:      strings.TrimSpace(orgSlug),
		JTI:          jti,
		ExpiresAt:    exp.UTC(),
	}, nil
}

// VerifyManagedLearner validates a Learn-as managed_learner JWT.
func (s *JWTSigner) VerifyManagedLearner(token string) (ManagedLearnerUser, error) {
	var claims managedLearnerClaims
	if err := s.verify(token, &claims); err != nil {
		return ManagedLearnerUser{}, err
	}
	if claims.Typ != "managed_learner" {
		return ManagedLearnerUser{}, ErrInvalidToken
	}
	if !isUUID(claims.ActorID) || !isUUID(claims.Subject) || strings.TrimSpace(claims.Email) == "" || !isUUID(claims.JTI) {
		return ManagedLearnerUser{}, ErrInvalidToken
	}
	if claims.OrgID != "" && !isUUID(claims.OrgID) {
		return ManagedLearnerUser{}, ErrInvalidToken
	}
	if isExpired(claims.Expires, s.now()) {
		return ManagedLearnerUser{}, ErrExpiredToken
	}
	return ManagedLearnerUser{
		ActorID:      claims.ActorID,
		TargetUserID: claims.Subject,
		TargetEmail:  claims.Email,
		OrgID:        strings.TrimSpace(claims.OrgID),
		OrgSlug:      strings.TrimSpace(claims.OrgSlug),
		JTI:          claims.JTI,
		ExpiresAt:    time.Unix(claims.Expires, 0).UTC(),
	}, nil
}

type managedLearnerClaims struct {
	Typ     string `json:"typ"`
	ActorID string `json:"actor_id"`
	Subject string `json:"sub"`
	Email   string `json:"email"`
	OrgID   string `json:"org_id,omitempty"`
	OrgSlug string `json:"org_slug,omitempty"`
	JTI     string `json:"jti"`
	Expires int64  `json:"exp"`
}
