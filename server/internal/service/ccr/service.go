// Package ccr implements co-curricular transcript generation (plan 14.13).
package ccr

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skip2/go-qrcode"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
	repo "github.com/lextures/lextures/server/internal/repos/ccr"
	"github.com/lextures/lextures/server/internal/service/reportpdf"
	"github.com/lextures/lextures/server/internal/service/vc_signing"
)

var (
	ErrFeatureDisabled = errors.New("co-curricular transcript is not enabled")
	ErrNoAchievements  = errors.New("no achievements available for CCR generation")
)

// Service generates CLR documents and manages achievement aggregation.
type Service struct {
	Pool           *pgxpool.Pool
	Config         config.Config
	SecretsKey     []byte
	PublicWebOrigin string
}

// GenerateParams controls on-demand CCR generation.
type GenerateParams struct {
	UserID         uuid.UUID
	StudentName    string
	InstitutionName string
	ConsentToShare bool
	VerificationBaseURL string
}

// GenerateResult is the output of a successful generation run.
type GenerateResult struct {
	Document *repo.Document
	Achievements []repo.Achievement
}

// ListAchievements returns synced achievements for a student.
func (s *Service) ListAchievements(ctx context.Context, userID uuid.UUID) ([]repo.Achievement, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if err := s.syncDerivedAchievements(ctx, userID); err != nil {
		return nil, err
	}
	return repo.ListAchievementsByUser(ctx, s.Pool, userID)
}

// ListDocuments returns generated documents for a student.
func (s *Service) ListDocuments(ctx context.Context, userID uuid.UUID) ([]repo.Document, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	return repo.ListDocumentsByUser(ctx, s.Pool, userID)
}

// GetDocument loads one owned document.
func (s *Service) GetDocument(ctx context.Context, userID, docID uuid.UUID) (*repo.Document, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	return repo.GetDocumentByIDForUser(ctx, s.Pool, userID, docID)
}

// GetDocumentByShareToken loads a publicly shareable document.
func (s *Service) GetDocumentByShareToken(ctx context.Context, token string) (*repo.Document, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	return repo.GetDocumentByShareToken(ctx, s.Pool, token)
}

// AddManualAchievement inserts an admin-authored extracurricular record.
func (s *Service) AddManualAchievement(ctx context.Context, p repo.UpsertAchievementParams) (*repo.Achievement, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	p.AchievementType = repo.TypeExtracurricular
	return repo.InsertManualAchievement(ctx, s.Pool, p)
}

// Generate builds, signs, and stores a CLR document for a student.
func (s *Service) Generate(ctx context.Context, p GenerateParams) (*GenerateResult, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if s.Pool == nil {
		return nil, fmt.Errorf("ccr: pool required")
	}
	if err := s.syncDerivedAchievements(ctx, p.UserID); err != nil {
		return nil, err
	}
	achievements, err := repo.ListAchievementsByUser(ctx, s.Pool, p.UserID)
	if err != nil {
		return nil, err
	}
	if len(achievements) == 0 {
		return nil, ErrNoAchievements
	}

	km, err := s.loadOrCreateSigningKey(ctx)
	if err != nil {
		return nil, err
	}

	issuedAt := time.Now().UTC()
	clr := BuildCLR(BuildCLRInput{
		DocumentID:      uuid.New(),
		InstitutionName: p.InstitutionName,
		StudentName:     p.StudentName,
		StudentDID:      studentDID(p.UserID),
		IssuerDID:       km.IssuerDID,
		IssuedAt:        issuedAt,
		Achievements:    achievements,
	})

	signed, err := vc_signing.SignCLR(km, clr, studentDID(p.UserID))
	if err != nil {
		return nil, err
	}
	vcProof, err := json.Marshal(signed.Proof)
	if err != nil {
		return nil, err
	}
	clrJSON, err := json.Marshal(clr)
	if err != nil {
		return nil, err
	}

	var shareToken *string
	var consentedAt *time.Time
	if p.ConsentToShare {
		t := uuid.NewString()
		shareToken = &t
		now := issuedAt
		consentedAt = &now
	}

	verifyURL := ""
	if shareToken != nil && strings.TrimSpace(p.VerificationBaseURL) != "" {
		verifyURL = strings.TrimRight(p.VerificationBaseURL, "/") + "/verify/" + *shareToken
	}
	pdfBytes, err := BuildPDF(BuildPDFInput{
		InstitutionName: p.InstitutionName,
		StudentName:     p.StudentName,
		GeneratedAt:     issuedAt,
		Achievements:    achievements,
		VerificationURL: verifyURL,
	})
	if err != nil {
		return nil, err
	}
	pdfKey := fmt.Sprintf("ccr/%s/%s.pdf", p.UserID, uuid.NewString())
	if err := s.storePDF(ctx, pdfKey, pdfBytes); err != nil {
		return nil, err
	}

	doc, err := repo.InsertDocument(ctx, s.Pool, repo.InsertDocumentParams{
		UserID:      p.UserID,
		ConsentedAt: consentedAt,
		CLRJSON:     clrJSON,
		VCProof:     vcProof,
		PDFKey:      &pdfKey,
		ShareToken:  shareToken,
	})
	if err != nil {
		return nil, err
	}
	recordGenerated()
	return &GenerateResult{Document: doc, Achievements: achievements}, nil
}

// VerifyShareToken validates the VC proof for a shared document.
func (s *Service) VerifyShareToken(ctx context.Context, token string) (valid bool, issuerName string, issuedAt time.Time, achievements []repo.Achievement, err error) {
	if err := s.requireEnabled(); err != nil {
		return false, "", time.Time{}, nil, err
	}
	doc, err := repo.GetDocumentByShareToken(ctx, s.Pool, token)
	if err != nil {
		return false, "", time.Time{}, nil, err
	}
	if doc == nil {
		return false, "", time.Time{}, nil, nil
	}
	cfg, err := repo.GetSigningConfig(ctx, s.Pool)
	if err != nil || cfg == nil {
		return false, "", time.Time{}, nil, err
	}
	pub, err := vc_signing.PublicKeyFromJWK(cfg.PublicKeyJWK)
	if err != nil {
		return false, "", time.Time{}, nil, err
	}
	var proof struct {
		JWT string `json:"jwt"`
	}
	if err := json.Unmarshal(doc.VCProof, &proof); err != nil {
		return false, "", time.Time{}, nil, err
	}
	ok, verr := vc_signing.VerifyJWT(pub, proof.JWT)
	if verr != nil || !ok {
		recordVerification(false)
		return false, "", time.Time{}, nil, verr
	}
	if err := s.syncDerivedAchievements(ctx, doc.UserID); err != nil {
		return false, "", time.Time{}, nil, err
	}
	achievements, err = repo.ListAchievementsByUser(ctx, s.Pool, doc.UserID)
	if err != nil {
		return false, "", time.Time{}, nil, err
	}
	recordVerification(true)
	return true, s.Config.PublicWebOrigin, doc.GeneratedAt, achievements, nil
}

// SigningKeyMaterial returns the active institutional signing key for DID document serving.
func (s *Service) SigningKeyMaterial(ctx context.Context) (*vc_signing.KeyMaterial, error) {
	return s.loadOrCreateSigningKey(ctx)
}

func (s *Service) requireEnabled() error {
	if !s.Config.FFCoCurricularTranscript {
		return ErrFeatureDisabled
	}
	return nil
}

func (s *Service) syncDerivedAchievements(ctx context.Context, userID uuid.UUID) error {
	completions, err := repo.ListCourseCompletions(ctx, s.Pool, userID)
	if err != nil {
		return err
	}
	for _, row := range completions {
		desc := fmt.Sprintf("Final grade: %s", row.FinalGrade)
		sourceID := row.SubmissionID
		if err := repo.UpsertAchievement(ctx, s.Pool, repo.UpsertAchievementParams{
			UserID:          userID,
			AchievementType: repo.TypeCourseCompletion,
			SourceID:        &sourceID,
			Title:           row.CourseTitle,
			Description:     &desc,
			IssuedAt:        row.SubmittedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadOrCreateSigningKey(ctx context.Context) (*vc_signing.KeyMaterial, error) {
	cfg, err := repo.GetSigningConfig(ctx, s.Pool)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		privBytes, err := appsecrets.Decrypt(cfg.PrivateKeyCipher, s.SecretsKey)
		if err != nil {
			return nil, err
		}
		if len(privBytes) != ed25519.PrivateKeySize {
			return nil, errors.New("ccr: invalid stored private key")
		}
		return vc_signing.KeyMaterialFromPrivate(cfg.IssuerDID, ed25519.PrivateKey(privBytes), cfg.PublicKeyJWK), nil
	}
	km, err := vc_signing.GenerateKeyMaterial(s.PublicWebOrigin)
	if err != nil {
		return nil, err
	}
	cipher, err := appsecrets.Encrypt(km.PrivateKey, s.SecretsKey)
	if err != nil {
		return nil, err
	}
	if _, err := repo.UpsertSigningConfig(ctx, s.Pool, km.IssuerDID, km.PublicKeyJWK, cipher); err != nil {
		return nil, err
	}
	return km, nil
}

func (s *Service) storePDF(ctx context.Context, key string, pdf []byte) error {
	_ = ctx
	_ = key
	_ = pdf
	// PDF bytes are returned on download directly from generation; object storage hook reserved for plan 8.1.
	return nil
}

func studentDID(userID uuid.UUID) string {
	return "urn:uuid:" + userID.String()
}

// BuildPDFInput describes a CCR PDF export.
type BuildPDFInput struct {
	InstitutionName string
	StudentName     string
	GeneratedAt     time.Time
	Achievements    []repo.Achievement
	VerificationURL string
}

// BuildPDF renders an accessible achievement summary PDF with optional QR code.
func BuildPDF(in BuildPDFInput) ([]byte, error) {
	rows := make([]reportpdf.CCRAchievementRow, 0, len(in.Achievements))
	for _, a := range in.Achievements {
		rows = append(rows, reportpdf.CCRAchievementRow{
			Type:    string(a.AchievementType),
			Title:   a.Title,
			Issued:  a.IssuedAt.UTC().Format("2006-01-02"),
			Details: achievementDetails(a),
		})
	}
	var qrPNG []byte
	if strings.TrimSpace(in.VerificationURL) != "" {
		code, err := qrcode.New(in.VerificationURL, qrcode.Medium)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, code.Image(256)); err != nil {
			return nil, err
		}
		qrPNG = buf.Bytes()
	}
	return reportpdf.BuildCCRPDF(reportpdf.CCRInput{
		InstitutionName: in.InstitutionName,
		StudentName:     in.StudentName,
		GeneratedAt:     in.GeneratedAt,
		Achievements:    rows,
		VerificationURL: in.VerificationURL,
		QRCodePNG:       qrPNG,
	})
}

func achievementDetails(a repo.Achievement) string {
	if a.Description != nil && strings.TrimSpace(*a.Description) != "" {
		return strings.TrimSpace(*a.Description)
	}
	if len(a.OutcomeTags) > 0 {
		return strings.Join(a.OutcomeTags, ", ")
	}
	return ""
}
