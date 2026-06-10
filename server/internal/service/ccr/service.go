package ccr

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
	"github.com/lextures/lextures/server/internal/service/vc_signing"
)

// GenerateParams controls CCR generation.
type GenerateParams struct {
	UserID          uuid.UUID
	LearnerName     string
	SharePublicly   bool
	InstitutionName string
	APIOrigin       string
	SigningSeedB64  string
	Now             time.Time
}

// GenerateResult is a newly created CCR document.
type GenerateResult struct {
	Document     *ccrrepo.Document
	Achievements []AggregatedAchievement
	Verification string
}

// Generate builds, signs, and stores a CLR document for the learner.
func Generate(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, p GenerateParams) (*GenerateResult, error) {
	if p.Now.IsZero() {
		p.Now = time.Now().UTC()
	}
	achievements, err := AggregateAchievements(ctx, pool, p.UserID)
	if err != nil {
		return nil, err
	}

	key, err := ResolveSigningKey(cfg, p.APIOrigin, p.SigningSeedB64)
	if err != nil {
		return nil, err
	}

	institution := strings.TrimSpace(p.InstitutionName)
	if institution == "" {
		institution = "Lextures"
	}
	learnerDID := fmt.Sprintf("urn:uuid:user:%s", p.UserID.String())
	subject := BuildCLRSubject(learnerDID, p.LearnerName, key.DID, institution, achievements, p.Now)
	clrJSON, err := MarshalCLRJSON(subject)
	if err != nil {
		return nil, err
	}

	var subjectMap map[string]any
	if err := json.Unmarshal(clrJSON, &subjectMap); err != nil {
		return nil, err
	}
	vc, err := vcsigning.SignCredential(subjectMap, institution, key, p.Now)
	if err != nil {
		return nil, err
	}
	vcBytes, err := json.Marshal(vc)
	if err != nil {
		return nil, err
	}

	var shareToken *string
	var verificationURL string
	if p.SharePublicly {
		token := uuid.NewString()
		shareToken = &token
		verificationURL = strings.TrimRight(strings.TrimSpace(p.APIOrigin), "/") + "/verify/" + token
	}

	doc := ccrrepo.Document{
		UserID:  p.UserID,
		CLRJSON: clrJSON,
		VCProof: json.RawMessage(vcBytes),
	}
	if shareToken != nil {
		keyStr := "share:" + *shareToken
		doc.PDFKey = &keyStr
		doc.ShareToken = shareToken
	}

	created, err := ccrrepo.CreateDocument(ctx, pool, doc)
	if err != nil {
		return nil, err
	}
	return &GenerateResult{
		Document:     created,
		Achievements: achievements,
		Verification: verificationURL,
	}, nil
}

// ResolveSigningKey loads or derives the institution Ed25519 signing key.
func ResolveSigningKey(cfg config.Config, apiOrigin, seedB64 string) (vcsigning.KeyMaterial, error) {
	origin := strings.TrimSpace(apiOrigin)
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/")
	}
	seed := strings.TrimSpace(seedB64)
	if seed == "" {
		seed = strings.TrimSpace(cfg.CCRSigningSeedB64)
	}
	if seed != "" {
		return vcsigning.KeyFromPrivateSeed(seed, origin)
	}
	if strings.TrimSpace(cfg.JWTSecret) != "" {
		h := sha256.Sum256([]byte("ccr-signing-v1:" + cfg.JWTSecret))
		return vcsigning.KeyFromPrivateSeed(base64.StdEncoding.EncodeToString(h[:ed25519SeedSize()]), origin)
	}
	return vcsigning.GenerateKey(origin)
}

func ed25519SeedSize() int {
	return 32
}
