// Package transcriptissue orchestrates academic-record assembly, PDF/PESC rendering, and persistence (T01).
package transcriptissue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	"github.com/lextures/lextures/server/internal/service/transcriptpesc"
	"github.com/lextures/lextures/server/internal/service/transcriptpdf"
	"github.com/lextures/lextures/server/internal/service/transcriptverify"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

// GenerateFormats selects which derived artifacts to render.
type GenerateFormats struct {
	PDF bool
	XML bool
}

// GenerateResult is the outcome of a generate/preview operation.
type GenerateResult struct {
	Record          *academicrecord.AcademicRecord
	Hash            string
	CanonicalJSON   []byte
	PDF             []byte
	PESCXML         []byte
	Document        *transcriptsrepo.Document // nil for preview (no persistence)
	VerificationURL string
	VerifyToken     string
}

// GenerateParams configures issuance or preview.
type GenerateParams struct {
	UserID          uuid.UUID
	GeneratedBy     uuid.UUID
	Variant         academicrecord.Variant
	TermIDs         []uuid.UUID
	Formats         GenerateFormats
	Persist         bool
	InstitutionName string
	Scale           academicrecord.ScaleKind
	GeneratedAt     time.Time
	// Config enables VC signing + verify URL embedding for official documents (T08).
	Config *config.Config
}

// Generate assembles the academic record, renders requested formats, and optionally persists.
// Unofficial previews (Persist=false) are never stored as official artifacts.
func Generate(ctx context.Context, pool *pgxpool.Pool, params GenerateParams) (*GenerateResult, error) {
	if pool == nil {
		return nil, fmt.Errorf("transcriptissue: nil pool")
	}
	if params.Variant == "" {
		params.Variant = academicrecord.VariantUnofficial
	}
	if !params.Formats.PDF && !params.Formats.XML {
		params.Formats.PDF = true
		params.Formats.XML = true
	}
	if params.Scale == "" {
		params.Scale = academicrecord.ScaleFourPoint
	}

	u, err := user.FindByID(ctx, pool, params.UserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("transcriptissue: load user: %w", err)
	}
	studentName := displayName(u)
	studentID := ""
	if u.Sid != nil {
		studentID = strings.TrimSpace(*u.Sid)
	}

	inst := strings.TrimSpace(params.InstitutionName)
	var orgID *uuid.UUID
	if oid, err := organization.OrgIDForUser(ctx, pool, params.UserID); err == nil {
		orgID = &oid
		if inst == "" {
			if org, err := organization.GetByID(ctx, pool, oid); err == nil && org != nil {
				inst = org.Name
			}
		}
	}
	if inst == "" {
		inst = "Lextures"
	}

	rec, err := academicrecord.AssembleFromDB(ctx, pool, academicrecord.AssembleParams{
		UserID:          params.UserID,
		Variant:         params.Variant,
		TermIDs:         params.TermIDs,
		InstitutionName: inst,
		StudentName:     studentName,
		StudentID:       studentID,
		Scale:           params.Scale,
		GeneratedAt:     params.GeneratedAt,
	})
	if err != nil {
		return nil, err
	}

	hash, canon, err := academicrecord.ContentHash(rec)
	if err != nil {
		return nil, err
	}

	out := &GenerateResult{Record: rec, Hash: hash, CanonicalJSON: canon}

	verifyToken := ""
	verifyURL := ""
	var vcProof json.RawMessage
	signOfficial := params.Persist && params.Variant != academicrecord.VariantUnofficial && params.Config != nil
	if signOfficial {
		verifyToken = uuid.NewString()
		verifyURL = transcriptverify.VerificationURL(params.Config.PublicWebOrigin, verifyToken)
		out.VerifyToken = verifyToken
		out.VerificationURL = verifyURL

		key, err := ccrsvc.ResolveSigningKey(*params.Config, params.Config.PublicWebOrigin, params.Config.CCRSigningSeedB64)
		if err != nil {
			return nil, fmt.Errorf("transcriptissue: signing key: %w", err)
		}
		issuedAt := params.GeneratedAt
		if issuedAt.IsZero() {
			issuedAt = time.Now().UTC()
		}
		subject := transcriptverify.BuildTranscriptSubject(verifyToken, hash, string(params.Variant))
		vc, err := vcsigning.SignTranscriptCredential(subject, inst, key, issuedAt)
		if err != nil {
			return nil, fmt.Errorf("transcriptissue: sign credential: %w", err)
		}
		raw, err := json.Marshal(vc)
		if err != nil {
			return nil, err
		}
		vcProof = raw
	}

	if params.Formats.PDF {
		pdfOpts := transcriptpdf.Options{
			VerificationURL: verifyURL,
			ContentHash:     hash,
			ShortCode:       shortCode(verifyToken),
		}
		pdf, err := transcriptpdf.BuildPDF(rec, pdfOpts)
		if err != nil {
			return nil, err
		}
		out.PDF = pdf
	}
	if params.Formats.XML {
		xmlBytes, err := transcriptpesc.BuildXML(rec)
		if err != nil {
			return nil, err
		}
		if err := transcriptpesc.ValidateStructure(xmlBytes); err != nil {
			return nil, err
		}
		out.PESCXML = xmlBytes
	}

	if !params.Persist {
		return out, nil
	}
	if params.Variant == academicrecord.VariantUnofficial {
		return out, nil
	}

	var gpa *float64
	if rec.Cumulative.GPA != nil {
		gpa = rec.Cumulative.GPA
	}
	c := rec.Cumulative.CreditsEarned
	credits := &c
	genBy := params.GeneratedBy
	var verifyTokPtr *string
	if verifyToken != "" {
		verifyTokPtr = &verifyToken
	}
	var pdfHashPtr *string
	if len(out.PDF) > 0 {
		h := transcriptverify.PDFHash(out.PDF)
		pdfHashPtr = &h
	}
	doc, err := transcriptsrepo.InsertDocument(ctx, pool, transcriptsrepo.InsertDocumentInput{
		UserID:          params.UserID,
		OrgID:           orgID,
		Variant:         transcriptsrepo.DocumentVariant(params.Variant),
		Canonical:       json.RawMessage(canon),
		SchemaVersion:   rec.SchemaVersion,
		TemplateVersion: rec.TemplateVersion,
		ContentHash:     hash,
		PDFBytes:        out.PDF,
		PESCXMLBytes:    out.PESCXML,
		VCProof:         vcProof,
		GPACumulative:   gpa,
		CreditsEarned:   credits,
		GeneratedBy:     &genBy,
		VerifyToken:     verifyTokPtr,
		PDFHash:         pdfHashPtr,
	})
	if err != nil {
		return nil, err
	}
	out.Document = doc
	return out, nil
}

func shortCode(token string) string {
	token = strings.TrimSpace(token)
	if len(token) < 8 {
		return token
	}
	// First segment of UUID for manual entry fallback.
	if i := strings.IndexByte(token, '-'); i > 0 {
		return strings.ToUpper(token[:i])
	}
	return strings.ToUpper(token[:8])
}

func displayName(u *user.Row) string {
	if u == nil {
		return "Student"
	}
	if u.DisplayName != nil && strings.TrimSpace(*u.DisplayName) != "" {
		return strings.TrimSpace(*u.DisplayName)
	}
	parts := make([]string, 0, 2)
	if u.FirstName != nil && strings.TrimSpace(*u.FirstName) != "" {
		parts = append(parts, strings.TrimSpace(*u.FirstName))
	}
	if u.LastName != nil && strings.TrimSpace(*u.LastName) != "" {
		parts = append(parts, strings.TrimSpace(*u.LastName))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return u.Email
}
