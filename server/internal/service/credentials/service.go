// Package credentials issues Open Badges 3.0 completion credentials (plan 15.5).
package credentials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/logging"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
	repocred "github.com/lextures/lextures/server/internal/repos/credentials"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	"github.com/lextures/lextures/server/internal/service/filestorage"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/service/pdfrender"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
	"github.com/lextures/lextures/server/internal/notificationevents"
)

// IssueInput describes a credential issuance request.
type IssueInput struct {
	RecipientID uuid.UUID
	LearnerName string
	SourceType  repocred.SourceType
	SourceID    uuid.UUID
	Title       string
	Description string
	IssuedAt    time.Time
}

// IssueDeps bundles runtime dependencies for issuance.
type IssueDeps struct {
	Pool        *pgxpool.Pool
	Cfg         config.Config
	Storage     filestorage.Driver
	Notify      *notifications.Service
}

// Issue creates or returns an existing signed credential (idempotent).
func Issue(ctx context.Context, deps IssueDeps, in IssueInput) (*repocred.IssuedCredential, bool, error) {
	if deps.Pool == nil {
		return nil, false, fmt.Errorf("credentials: missing database pool")
	}
	if !deps.Cfg.FFCompletionCredentials {
		return nil, false, nil
	}
	issuedAt := IssuanceTimestamp(in.IssuedAt)
	achievementName := DefaultAchievementName(string(in.SourceType), in.Title)
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = fmt.Sprintf("Certificate of completion for %s.", achievementName)
	}

	key, err := ccrsvc.ResolveSigningKey(deps.Cfg, deps.Cfg.PublicWebOrigin, deps.Cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, false, err
	}
	institution := institutionName(deps.Cfg)
	subject := BuildAchievementSubject(
		in.RecipientID,
		in.LearnerName,
		achievementName,
		description,
		CriteriaNarrativeForSource(string(in.SourceType)),
	)
	vc, err := vcsigning.SignAchievementCredential(subject, institution, key, issuedAt)
	if err != nil {
		return nil, false, err
	}
	vcBytes, err := json.Marshal(vc)
	if err != nil {
		return nil, false, err
	}
	proofRaw, _ := json.Marshal(vc["proof"])

	created, isNew, err := repocred.InsertIssued(ctx, deps.Pool, repocred.IssuedCredential{
		RecipientID:    in.RecipientID,
		SourceType:     in.SourceType,
		SourceID:       in.SourceID,
		CredentialJSON: vcBytes,
		Proof:          proofRaw,
	})
	if err != nil {
		return nil, false, err
	}
	if isNew {
		logging.GlobalCredentialMetrics.IncIssued(string(in.SourceType))
		syncCCR(ctx, deps.Pool, in, achievementName, issuedAt, created.ID, deps.Cfg)
		go finalizeCredential(context.WithoutCancel(ctx), deps, created, in, achievementName, institution, issuedAt)
	}
	return created, isNew, nil
}

// IssueForCourseCompletion issues a course credential when self-paced completion fires.
func IssueForCourseCompletion(ctx context.Context, deps IssueDeps, courseID, recipientID uuid.UUID, learnerName string) (*repocred.IssuedCredential, bool, error) {
	title, err := courseTitle(ctx, deps.Pool, courseID)
	if err != nil {
		return nil, false, err
	}
	return Issue(ctx, deps, IssueInput{
		RecipientID: recipientID,
		LearnerName: learnerName,
		SourceType:  repocred.SourceCourse,
		SourceID:    courseID,
		Title:       title,
	})
}

// IssueForPathCompletion issues a path-level credential.
func IssueForPathCompletion(ctx context.Context, deps IssueDeps, pathID, recipientID uuid.UUID, learnerName, pathTitle string) (*repocred.IssuedCredential, bool, error) {
	return Issue(ctx, deps, IssueInput{
		RecipientID: recipientID,
		LearnerName: learnerName,
		SourceType:  repocred.SourcePath,
		SourceID:    pathID,
		Title:       pathTitle,
	})
}

// VerifyResult is returned by public verification.
type VerifyResult struct {
	Valid        bool
	Status       string
	Revoked      bool
	IssuerName   string
	LearnerName  string
	Achievement  string
	IssuedAt     time.Time
	Credential   map[string]any
}

// Verify checks signature and revocation for a stored credential.
func Verify(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, id uuid.UUID) (*VerifyResult, error) {
	row, err := repocred.GetByID(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	var vc map[string]any
	if err := json.Unmarshal(row.CredentialJSON, &vc); err != nil {
		return nil, err
	}
	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, err
	}
	valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil {
		return nil, err
	}
	logging.GlobalCredentialMetrics.IncVerifications()

	out := &VerifyResult{
		Valid:       valid && !row.Revoked,
		IssuerName:  institutionName(cfg),
		LearnerName: extractLearnerName(vc),
		Achievement: extractAchievementName(vc),
		IssuedAt:    row.IssuedAt,
		Credential:  vc,
	}
	if row.Revoked {
		out.Status = "Revoked"
		out.Valid = false
	} else if valid {
		out.Status = "Valid"
	} else {
		out.Status = "Invalid credential — signature mismatch."
	}
	return out, nil
}

// BuildPDFBytes renders the certificate PDF for download.
func BuildPDFBytes(cfg config.Config, row *repocred.IssuedCredential, learnerName string) ([]byte, error) {
	var vc map[string]any
	if err := json.Unmarshal(row.CredentialJSON, &vc); err != nil {
		return nil, err
	}
	verifyURL := verificationURL(cfg.PublicWebOrigin, row.ID)
	return pdfrender.BuildCertificate(pdfrender.CertificateInput{
		InstitutionName: institutionName(cfg),
		LearnerName:     coalesce(learnerName, extractLearnerName(vc)),
		AchievementName: extractAchievementName(vc),
		Description:     extractDescription(vc),
		IssuedAt:        row.IssuedAt,
		VerificationURL: verifyURL,
	})
}

func finalizeCredential(ctx context.Context, deps IssueDeps, row *repocred.IssuedCredential, in IssueInput, achievementName, institution string, issuedAt time.Time) {
	pdfBytes, err := pdfrender.BuildCertificate(pdfrender.CertificateInput{
		InstitutionName: institution,
		LearnerName:     in.LearnerName,
		AchievementName: achievementName,
		Description:     in.Description,
		IssuedAt:        issuedAt,
		VerificationURL: verificationURL(deps.Cfg.PublicWebOrigin, row.ID),
	})
	if err == nil && deps.Storage != nil {
		key := pdfObjectKey(row.ID)
		_ = deps.Storage.PutObject(ctx, key, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf")
		_ = repocred.UpdatePDFKey(ctx, deps.Pool, row.ID, key)
	}
	if deps.Notify != nil {
		verifyURL := verificationURL(deps.Cfg.PublicWebOrigin, row.ID)
		_ = deps.Notify.EnqueueEmail(ctx, in.RecipientID, notificationevents.CertificateAwarded, "certificate_awarded", map[string]string{
			"subject":         fmt.Sprintf("Your certificate: %s", achievementName),
			"achievementName": achievementName,
			"verificationUrl": verifyURL,
			"link":            verifyURL,
			"digestLine":      fmt.Sprintf("Certificate earned: %s", achievementName),
		}, nil)
	}
}

func syncCCR(ctx context.Context, pool *pgxpool.Pool, in IssueInput, title string, issuedAt time.Time, credentialID uuid.UUID, cfg config.Config) {
	if !cfg.FFCoCurricularTranscript {
		return
	}
	evidence := verificationURL(cfg.PublicWebOrigin, credentialID)
	sourceID := in.SourceID
	_, _ = ccrrepo.CreateAchievement(ctx, pool, ccrrepo.Achievement{
		UserID:          in.RecipientID,
		AchievementType: ccrrepo.TypeCertificate,
		SourceID:        &sourceID,
		Title:           title,
		IssuedAt:        issuedAt,
		EvidenceURL:     &evidence,
	})
}

func courseTitle(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var title string
	err := pool.QueryRow(ctx, `SELECT title FROM course.courses WHERE id = $1`, courseID).Scan(&title)
	return title, err
}

func institutionName(cfg config.Config) string {
	if n := strings.TrimSpace(cfg.CCRInstitutionName); n != "" {
		return n
	}
	return "Lextures"
}

func verificationURL(origin string, id uuid.UUID) string {
	return fmt.Sprintf("%s/verify/%s", strings.TrimRight(strings.TrimSpace(origin), "/"), id.String())
}

func pdfObjectKey(id uuid.UUID) string {
	return "platform/credentials/" + id.String() + ".pdf"
}

func coalesce(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func extractLearnerName(vc map[string]any) string {
	subject, _ := vc["credentialSubject"].(map[string]any)
	if name, _ := subject["name"].(string); strings.TrimSpace(name) != "" {
		return name
	}
	return ""
}

func extractAchievementName(vc map[string]any) string {
	subject, _ := vc["credentialSubject"].(map[string]any)
	achievement, _ := subject["achievement"].(map[string]any)
	if name, _ := achievement["name"].(string); strings.TrimSpace(name) != "" {
		return name
	}
	return "Certificate of Completion"
}

func extractDescription(vc map[string]any) string {
	subject, _ := vc["credentialSubject"].(map[string]any)
	achievement, _ := subject["achievement"].(map[string]any)
	if desc, _ := achievement["description"].(string); strings.TrimSpace(desc) != "" {
		return desc
	}
	return ""
}

// ReadStoredPDF returns PDF bytes from object storage when available.
func ReadStoredPDF(ctx context.Context, storage filestorage.Driver, pdfKey string) ([]byte, error) {
	if storage == nil || strings.TrimSpace(pdfKey) == "" {
		return nil, io.EOF
	}
	rc, err := storage.GetObject(ctx, pdfKey)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}