// Package transcriptverify implements unified credential verification for transcripts and CLRs (T08).
package transcriptverify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
	diplomasrepo "github.com/lextures/lextures/server/internal/repos/diplomas"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

// Result codes returned to verifiers.
const (
	ResultGenuine  = transcriptsrepo.VerifyResultGenuine
	ResultTampered = transcriptsrepo.VerifyResultTampered
	ResultRevoked  = transcriptsrepo.VerifyResultRevoked
	ResultNotFound = transcriptsrepo.VerifyResultNotFound
)

// Outcome is a minimal-disclosure verification response.
type Outcome struct {
	Result       string         `json:"result"`
	Valid        bool           `json:"valid"`
	Status       string         `json:"status"`
	DocumentType string         `json:"documentType"`
	DocumentID   *uuid.UUID     `json:"documentId,omitempty"`
	IssuerName   string         `json:"issuerName"`
	IssuerDID    string         `json:"issuerDid,omitempty"`
	IssuedAt     string         `json:"issuedAt,omitempty"`
	RevokedAt    *string        `json:"revokedAt,omitempty"`
	Credential   map[string]any `json:"credential,omitempty"`
}

// Context carries requester metadata for the audit log.
type Context struct {
	Method string
	IP     string
	UA     string
}

// VerifyByToken resolves a verify/share token across transcript, CLR, and diploma documents.
func VerifyByToken(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, token string, vctx Context) (*Outcome, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return notFound(ctx, pool, "", vctx), nil
	}

	if doc, err := transcriptsrepo.GetDocumentByVerifyToken(ctx, pool, token); err != nil {
		return nil, err
	} else if doc != nil {
		return verifyTranscriptDoc(ctx, pool, cfg, doc, vctx)
	}

	if doc, err := ccrrepo.GetDocumentByShareToken(ctx, pool, token); err != nil {
		return nil, err
	} else if doc != nil {
		return verifyCLRDoc(ctx, pool, cfg, doc, vctx)
	}

	if cfg.FFDiplomas {
		if dip, err := diplomasrepo.GetByVerifyToken(ctx, pool, token); err != nil {
			return nil, err
		} else if dip != nil {
			return verifyDiplomaDoc(ctx, pool, cfg, dip, vctx)
		}
	}

	return notFound(ctx, pool, "", vctx), nil
}

// VerifyByPDFHash looks up an issued transcript by SHA-256 of uploaded PDF bytes.
func VerifyByPDFHash(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, pdfBytes []byte, vctx Context) (*Outcome, error) {
	if len(pdfBytes) == 0 {
		return notFound(ctx, pool, transcriptsrepo.VerifyDocTranscript, vctx), nil
	}
	sum := sha256.Sum256(pdfBytes)
	hash := hex.EncodeToString(sum[:])
	doc, err := transcriptsrepo.GetDocumentByPDFHash(ctx, pool, hash)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return notFound(ctx, pool, transcriptsrepo.VerifyDocTranscript, vctx), nil
	}
	// Byte match against stored PDF confirms integrity of the uploaded file.
	if len(doc.PDFBytes) > 0 && !bytesEqual(doc.PDFBytes, pdfBytes) {
		out := &Outcome{
			Result:       ResultTampered,
			Valid:        false,
			Status:       "Tampered",
			DocumentType: transcriptsrepo.VerifyDocTranscript,
			DocumentID:   &doc.ID,
			IssuerName:   issuerName(cfg),
			IssuedAt:     doc.GeneratedAt.UTC().Format(time.RFC3339),
		}
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultTampered, vctx)
		return out, nil
	}
	return verifyTranscriptDoc(ctx, pool, cfg, doc, vctx)
}

func verifyTranscriptDoc(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, doc *transcriptsrepo.Document, vctx Context) (*Outcome, error) {
	out := &Outcome{
		DocumentType: transcriptsrepo.VerifyDocTranscript,
		DocumentID:   &doc.ID,
		IssuerName:   issuerName(cfg),
		IssuedAt:     doc.GeneratedAt.UTC().Format(time.RFC3339),
	}

	if doc.RevokedAt != nil {
		s := doc.RevokedAt.UTC().Format(time.RFC3339)
		out.RevokedAt = &s
		out.Result = ResultRevoked
		out.Valid = false
		out.Status = "Revoked"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultRevoked, vctx)
		return out, nil
	}

	if !transcriptsrepo.VerifyDocumentHash(doc) {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultTampered, vctx)
		return out, nil
	}

	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, fmt.Errorf("resolve signing key: %w", err)
	}
	out.IssuerDID = key.DID

	if len(doc.VCProof) == 0 {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultTampered, vctx)
		return out, nil
	}

	var vc map[string]any
	if err := json.Unmarshal(doc.VCProof, &vc); err != nil {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultTampered, vctx)
		return out, nil
	}

	valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultTampered, vctx)
		return out, nil
	}

	if name, ok := issuerNameFromVC(vc); ok {
		out.IssuerName = name
	}
	if issued, ok := vc["issuanceDate"].(string); ok && strings.TrimSpace(issued) != "" {
		out.IssuedAt = issued
	}

	out.Result = ResultGenuine
	out.Valid = true
	out.Status = "Genuine"
	if doc.DisclosePublicly {
		out.Credential = vc
	}
	_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocTranscript, ResultGenuine, vctx)
	return out, nil
}

func verifyDiplomaDoc(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, dip *diplomasrepo.Diploma, vctx Context) (*Outcome, error) {
	out := &Outcome{
		DocumentType: transcriptsrepo.VerifyDocDiploma,
		DocumentID:   &dip.ID,
		IssuerName:   issuerName(cfg),
		IssuedAt:     dip.IssuedAt.UTC().Format(time.RFC3339),
	}

	if dip.RevokedAt != nil {
		s := dip.RevokedAt.UTC().Format(time.RFC3339)
		out.RevokedAt = &s
		out.Result = ResultRevoked
		out.Valid = false
		out.Status = "Revoked"
		_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultRevoked, vctx)
		return out, nil
	}

	if !diplomasrepo.VerifyContentHash(dip) {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultTampered, vctx)
		return out, nil
	}

	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, fmt.Errorf("resolve signing key: %w", err)
	}
	out.IssuerDID = key.DID

	if len(dip.VCProof) == 0 {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultTampered, vctx)
		return out, nil
	}

	var vc map[string]any
	if err := json.Unmarshal(dip.VCProof, &vc); err != nil {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultTampered, vctx)
		return out, nil
	}

	valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultTampered, vctx)
		return out, nil
	}

	if name, ok := issuerNameFromVC(vc); ok {
		out.IssuerName = name
	}
	if issued, ok := vc["issuanceDate"].(string); ok && strings.TrimSpace(issued) != "" {
		out.IssuedAt = issued
	}

	out.Result = ResultGenuine
	out.Valid = true
	out.Status = "Genuine"
	out.Credential = vc
	_ = logVerification(ctx, pool, &dip.ID, transcriptsrepo.VerifyDocDiploma, ResultGenuine, vctx)
	return out, nil
}

func verifyCLRDoc(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, doc *ccrrepo.Document, vctx Context) (*Outcome, error) {
	out := &Outcome{
		DocumentType: transcriptsrepo.VerifyDocCLR,
		DocumentID:   &doc.ID,
		IssuerName:   issuerName(cfg),
		IssuedAt:     doc.GeneratedAt.UTC().Format(time.RFC3339),
	}

	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, fmt.Errorf("resolve signing key: %w", err)
	}
	out.IssuerDID = key.DID

	var vc map[string]any
	if err := json.Unmarshal(doc.VCProof, &vc); err != nil {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Tampered"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocCLR, ResultTampered, vctx)
		return out, nil
	}

	valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		out.Result = ResultTampered
		out.Valid = false
		out.Status = "Invalid"
		_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocCLR, ResultTampered, vctx)
		return out, nil
	}

	if name, ok := issuerNameFromVC(vc); ok {
		out.IssuerName = name
	}
	if issued, ok := vc["issuanceDate"].(string); ok && strings.TrimSpace(issued) != "" {
		out.IssuedAt = issued
	}

	out.Result = ResultGenuine
	out.Valid = true
	out.Status = "Valid"
	// CLR public share implies holder opted into disclosure (existing behavior).
	out.Credential = vc
	_ = logVerification(ctx, pool, &doc.ID, transcriptsrepo.VerifyDocCLR, ResultGenuine, vctx)
	return out, nil
}

func notFound(ctx context.Context, pool *pgxpool.Pool, docType string, vctx Context) *Outcome {
	if docType == "" {
		docType = transcriptsrepo.VerifyDocTranscript
	}
	_ = logVerification(ctx, pool, nil, docType, ResultNotFound, vctx)
	return &Outcome{
		Result:       ResultNotFound,
		Valid:        false,
		Status:       "Not found",
		DocumentType: docType,
	}
}

func logVerification(ctx context.Context, pool *pgxpool.Pool, docID *uuid.UUID, docType, result string, vctx Context) error {
	if pool == nil {
		return nil
	}
	method := strings.TrimSpace(vctx.Method)
	if method == "" {
		method = transcriptsrepo.VerifyMethodLink
	}
	var ip *string
	if s := strings.TrimSpace(vctx.IP); s != "" {
		ip = &s
	}
	var ua *string
	if s := strings.TrimSpace(vctx.UA); s != "" {
		if len(s) > 512 {
			s = s[:512]
		}
		ua = &s
	}
	_, err := transcriptsrepo.InsertVerification(ctx, pool, transcriptsrepo.InsertVerificationInput{
		DocumentID:   docID,
		DocumentType: docType,
		Result:       result,
		Method:       method,
		RequesterIP:  ip,
		RequesterUA:  ua,
	})
	return err
}

func issuerName(cfg config.Config) string {
	if s := strings.TrimSpace(cfg.CCRInstitutionName); s != "" {
		return s
	}
	return "Lextures"
}

func issuerNameFromVC(vc map[string]any) (string, bool) {
	raw, ok := vc["issuer"]
	if !ok {
		return "", false
	}
	switch v := raw.(type) {
	case map[string]any:
		if name, ok := v["name"].(string); ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name), true
		}
	case string:
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PDFHash returns the SHA-256 hex digest of PDF bytes.
func PDFHash(pdf []byte) string {
	sum := sha256.Sum256(pdf)
	return hex.EncodeToString(sum[:])
}

// BuildTranscriptSubject builds the VC credentialSubject for an official transcript.
func BuildTranscriptSubject(verifyToken, contentHash, variant string) map[string]any {
	return map[string]any{
		"id":          "urn:lextures:transcript:" + verifyToken,
		"type":        "OfficialTranscript",
		"contentHash": contentHash,
		"variant":     variant,
	}
}

// VerificationURL builds the public SPA verify URL for a token.
func VerificationURL(webOrigin, token string) string {
	base := strings.TrimRight(strings.TrimSpace(webOrigin), "/")
	if base == "" || strings.TrimSpace(token) == "" {
		return ""
	}
	return base + "/verify/" + strings.TrimSpace(token)
}
