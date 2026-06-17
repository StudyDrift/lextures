package credentials

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	credrepo "github.com/lextures/lextures/server/internal/repos/credentials"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

const badgeExportTTL = 24 * time.Hour

// IssueCourseParams controls course-completion credential issuance.
type IssueCourseParams struct {
	RecipientID uuid.UUID
	LearnerName string
	CourseID    uuid.UUID
	Now         time.Time
}

// IssueCourseCompletion issues an Open Badges 3.0 credential for course completion (idempotent).
func IssueCourseCompletion(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.Config,
	p IssueCourseParams,
) (*credrepo.IssuedCredential, error) {
	if p.Now.IsZero() {
		p.Now = time.Now().UTC()
	}
	existing, err := credrepo.GetByRecipientAndSource(ctx, pool, p.RecipientID, credrepo.SourceCourse, p.CourseID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	meta, err := credrepo.CourseMetaByID(ctx, pool, p.CourseID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("course not found")
	}

	institution := issuerName(cfg)
	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, err
	}

	achievementID := fmt.Sprintf("%s/achievements/%s", strings.TrimRight(cfg.PublicWebOrigin, "/"), p.CourseID.String())
	subject := map[string]any{
		"id":   fmt.Sprintf("urn:uuid:user:%s", p.RecipientID.String()),
		"type": []string{"AchievementSubject"},
		"name": strings.TrimSpace(p.LearnerName),
		"achievement": map[string]any{
			"id":          achievementID,
			"type":        []string{"Achievement"},
			"name":        meta.Title,
			"description": fmt.Sprintf("Completed all items in %s.", meta.Title),
			"criteria": map[string]any{
				"narrative": "Learner completed every item in the self-paced course.",
			},
		},
	}

	vc, err := vcsigning.SignAchievementCredential(subject, institution, key, p.Now)
	if err != nil {
		return nil, err
	}
	vcBytes, err := json.Marshal(vc)
	if err != nil {
		return nil, err
	}
	subjectBytes, err := json.Marshal(subject)
	if err != nil {
		return nil, err
	}

	verifyURL := VerificationURL(cfg.PublicWebOrigin, uuid.Nil)
	created, err := credrepo.Create(ctx, pool, credrepo.IssuedCredential{
		RecipientID:    p.RecipientID,
		SourceType:     credrepo.SourceCourse,
		SourceID:       p.CourseID,
		Title:          meta.Title,
		CredentialJSON: subjectBytes,
		Proof:          json.RawMessage(vcBytes),
		IssuedAt:       p.Now,
	})
	if err != nil {
		return nil, err
	}
	verifyURL = VerificationURL(cfg.PublicWebOrigin, created.ID)

	pdfBytes, err := BuildPDF(PDFInput{
		InstitutionName: institution,
		LearnerName:     p.LearnerName,
		CredentialName:  meta.Title,
		IssuedAt:        created.IssuedAt,
		VerificationURL: verifyURL,
	})
	if err == nil && len(pdfBytes) > 0 {
		keyStr := fmt.Sprintf("credential:%s.pdf", created.ID.String())
		created.PDFKey = &keyStr
		_, _ = pool.Exec(ctx, `UPDATE credentials.issued_credentials SET pdf_key = $2 WHERE id = $1`, created.ID, keyStr)
	}

	return created, nil
}

// VerificationURL builds the public verification page URL for a credential.
func VerificationURL(origin string, credentialID uuid.UUID) string {
	return fmt.Sprintf("%s/verify/%s", strings.TrimRight(strings.TrimSpace(origin), "/"), credentialID.String())
}

// LinkedInParamsForCredential returns pre-filled LinkedIn certification parameters.
func LinkedInParamsForCredential(cfg config.Config, cred *credrepo.IssuedCredential) LinkedInParams {
	verifyURL := VerificationURL(cfg.PublicWebOrigin, cred.ID)
	return BuildLinkedInParams(cred.Title, issuerName(cfg), verifyURL, cred.ID.String(), cred.IssuedAt)
}

// BadgeExportToken issues an HMAC token for time-limited badge JSON download.
func BadgeExportToken(cfg config.Config, credentialID uuid.UUID, now time.Time) (string, time.Time, error) {
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" {
		secret = strings.TrimSpace(cfg.CCRSigningSeedB64)
	}
	if secret == "" {
		return "", time.Time{}, fmt.Errorf("signing secret unavailable")
	}
	expires := now.UTC().Add(badgeExportTTL)
	payload := fmt.Sprintf("%s:%d", credentialID.String(), expires.Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + sig))
	return token, expires, nil
}

// VerifyBadgeExportToken validates a badge export token and returns the credential id.
func VerifyBadgeExportToken(cfg config.Config, token string, now time.Time) (uuid.UUID, error) {
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" {
		secret = strings.TrimSpace(cfg.CCRSigningSeedB64)
	}
	if secret == "" {
		return uuid.UUID{}, fmt.Errorf("signing secret unavailable")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}
	parts := strings.SplitN(string(raw), ":", 3)
	if len(parts) != 3 {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}
	var exp int64
	if _, err := fmt.Sscanf(parts[1], "%d", &exp); err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}
	if now.UTC().Unix() > exp {
		return uuid.UUID{}, fmt.Errorf("token expired")
	}
	payload := fmt.Sprintf("%s:%d", parts[0], exp)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}
	return id, nil
}

// VerifyCredential checks the VC proof on a stored credential.
func VerifyCredential(cfg config.Config, proof json.RawMessage) (bool, error) {
	var vc map[string]any
	if err := json.Unmarshal(proof, &vc); err != nil {
		return false, err
	}
	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return false, err
	}
	return vcsigning.VerifyCredential(vc, key.PublicKey)
}

// FullCredentialJSON merges stored subject JSON with signed VC proof for export.
func FullCredentialJSON(cred *credrepo.IssuedCredential) ([]byte, error) {
	return cred.Proof, nil
}

func issuerName(cfg config.Config) string {
	if strings.TrimSpace(cfg.CCRInstitutionName) != "" {
		return strings.TrimSpace(cfg.CCRInstitutionName)
	}
	return "Lextures"
}