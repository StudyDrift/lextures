// Package diplomaissue issues signed diplomas and certificates into the wallet (T11).
package diplomaissue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/logging"
	diplomasrepo "github.com/lextures/lextures/server/internal/repos/diplomas"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	"github.com/lextures/lextures/server/internal/service/credentialwallet"
	"github.com/lextures/lextures/server/internal/service/diplomapdf"
	"github.com/lextures/lextures/server/internal/service/transcriptverify"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
	"github.com/lextures/lextures/server/internal/telemetry"
)

var (
	ErrFeatureDisabled = errors.New("diplomas are not enabled")
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrTemplateInactive = errors.New("template is inactive")
)

// IssueParams controls single-learner issuance.
type IssueParams struct {
	OrgID       uuid.UUID
	TemplateID  uuid.UUID
	UserID      uuid.UUID
	LearnerName string
	Program     *string
	Honors      *string
	ConferredAt time.Time
	ProgramRef  *uuid.UUID
	IssuedBy    *uuid.UUID
	// CorrectPrior when true revokes an existing active credential and issues a new version.
	CorrectPrior bool
}

// IssueResult is the outcome of one issuance attempt.
type IssueResult struct {
	Diploma *diplomasrepo.Diploma `json:"diploma,omitempty"`
	Skipped bool                  `json:"skipped"`
	Reason  string                `json:"reason,omitempty"`
}

func issuerName(cfg config.Config) string {
	name := strings.TrimSpace(cfg.CCRInstitutionName)
	if name == "" {
		return "Lextures"
	}
	return name
}

// Issue creates a signed diploma/certificate for one learner (idempotent unless CorrectPrior).
func Issue(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, p IssueParams) (*IssueResult, error) {
	if !cfg.FFDiplomas {
		return nil, ErrFeatureDisabled
	}
	if pool == nil || p.OrgID == uuid.Nil || p.TemplateID == uuid.Nil || p.UserID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if p.ConferredAt.IsZero() {
		p.ConferredAt = time.Now().UTC()
	}

	tmpl, err := diplomasrepo.GetTemplateByID(ctx, pool, p.TemplateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil || tmpl.OrgID != p.OrgID {
		return nil, ErrNotFound
	}
	if !tmpl.Active {
		return nil, ErrTemplateInactive
	}

	existing, err := diplomasrepo.FindExisting(ctx, pool, p.UserID, &p.TemplateID, p.ProgramRef)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.RevokedAt == nil && !p.CorrectPrior {
		return &IssueResult{Diploma: existing, Skipped: true, Reason: "already_issued"}, nil
	}

	var replacesID *uuid.UUID
	version := 1
	if existing != nil {
		if p.CorrectPrior {
			if existing.RevokedAt == nil {
				if _, err := diplomasrepo.Revoke(ctx, pool, existing.ID, "superseded by correction"); err != nil {
					return nil, err
				}
				logging.GlobalDiplomaMetrics.IncRevoked()
				telemetry.RecordBusinessEvent("credential.diploma.revoked")
			}
			replacesID = &existing.ID
			version = existing.Version + 1
			// Clear unique key on prior row by leaving it revoked; new insert needs unique slot.
			// Unique is on (user, template, program_ref) — prior row still occupies it.
			// For corrections we update program_ref uniqueness by revoking isn't enough.
			// Strategy: delete unique occupancy by nulling program_ref on superseded rows is wrong.
			// Better: on correction, update the existing row's program_ref to a sentinel? No.
			// Plan: UNIQUE (user_id, template_id, program_ref) — corrections create new version
			// and revoke prior. To allow insert, we must free the unique key.
			// Approach used: set program_ref of revoked prior to a dedicated replaces UUID attached
			// only for uniqueness release — actually simplest is to delete uniqueness via
			// updating prior program_ref to NULL when already NULL conflicts...
			// Use: when correcting, change prior's program_ref to its own id (still unique across rows).
			if err := releaseIdempotencyKey(ctx, pool, existing); err != nil {
				return nil, err
			}
		} else if existing.RevokedAt != nil {
			// Re-issue after revoke without CorrectPrior: treat as new version.
			replacesID = &existing.ID
			version = existing.Version + 1
			if err := releaseIdempotencyKey(ctx, pool, existing); err != nil {
				return nil, err
			}
		}
	}

	learner := strings.TrimSpace(p.LearnerName)
	if learner == "" {
		learner = "Learner"
	}
	program := p.Program
	if program == nil {
		program = tmpl.Program
	}
	honors := p.Honors
	title := strings.TrimSpace(tmpl.Title)
	if title == "" {
		title = tmpl.Name
	}
	conferral := ""
	if tmpl.ConferralText != nil {
		conferral = *tmpl.ConferralText
	}
	progStr := ""
	if program != nil {
		progStr = *program
	}
	honorsStr := ""
	if honors != nil {
		honorsStr = *honors
	}

	verifyToken := uuid.NewString()
	verifyURL := transcriptverify.VerificationURL(cfg.PublicWebOrigin, verifyToken)
	now := time.Now().UTC()

	canonicalObj := map[string]any{
		"kind":            string(tmpl.Kind),
		"credentialTitle": title,
		"program":         progStr,
		"honors":          honorsStr,
		"conferredAt":     p.ConferredAt.UTC().Format(time.RFC3339),
		"learnerName":     learner,
		"templateId":      tmpl.ID.String(),
		"orgId":           p.OrgID.String(),
		"userId":          p.UserID.String(),
		"version":         version,
		"verifyToken":     verifyToken,
	}
	if p.ProgramRef != nil {
		canonicalObj["programRef"] = p.ProgramRef.String()
	}
	canonical, contentHash, err := diplomasrepo.MustMarshalCanonical(canonicalObj)
	if err != nil {
		return nil, err
	}

	pdfBytes, err := diplomapdf.Build(diplomapdf.Input{
		Kind:            string(tmpl.Kind),
		InstitutionName: issuerName(cfg),
		LearnerName:     learner,
		CredentialTitle: title,
		Program:         progStr,
		Honors:          honorsStr,
		ConferralText:   conferral,
		ConferredAt:     p.ConferredAt,
		VerificationURL: verifyURL,
	})
	if err != nil {
		return nil, fmt.Errorf("diplomaissue: pdf: %w", err)
	}

	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, fmt.Errorf("diplomaissue: signing key: %w", err)
	}
	subject := map[string]any{
		"id":              "urn:lextures:diploma:" + verifyToken,
		"type":            string(tmpl.Kind),
		"credentialTitle": title,
		"program":         progStr,
		"honors":          honorsStr,
		"conferredAt":     p.ConferredAt.UTC().Format(time.RFC3339),
		"contentHash":     contentHash,
		"verifyToken":     verifyToken,
		"version":         version,
	}
	vc, err := vcsigning.SignDiplomaCredential(subject, issuerName(cfg), key, now)
	if err != nil {
		return nil, fmt.Errorf("diplomaissue: sign: %w", err)
	}
	vcRaw, err := json.Marshal(vc)
	if err != nil {
		return nil, err
	}

	tmplID := tmpl.ID
	dip, err := diplomasrepo.InsertDiploma(ctx, pool, diplomasrepo.InsertDiplomaInput{
		UserID:          p.UserID,
		OrgID:           p.OrgID,
		TemplateID:      &tmplID,
		Kind:            tmpl.Kind,
		CredentialTitle: title,
		Program:         program,
		Honors:          honors,
		ConferredAt:     p.ConferredAt,
		Version:         version,
		ReplacesID:      replacesID,
		Canonical:       canonical,
		ContentHash:     contentHash,
		PDFBytes:        pdfBytes,
		VCProof:         vcRaw,
		VerifyToken:     verifyToken,
		IssuedBy:        p.IssuedBy,
		ProgramRef:      p.ProgramRef,
	})
	if err != nil {
		if errors.Is(err, diplomasrepo.ErrDuplicate) {
			existing2, ferr := diplomasrepo.FindExisting(ctx, pool, p.UserID, &p.TemplateID, p.ProgramRef)
			if ferr != nil {
				return nil, ferr
			}
			if existing2 != nil {
				return &IssueResult{Diploma: existing2, Skipped: true, Reason: "already_issued"}, nil
			}
		}
		return nil, err
	}

	logging.GlobalDiplomaMetrics.IncIssued(string(tmpl.Kind))
	telemetry.RecordBusinessEvent("credential.diploma.issued")

	// Best-effort wallet refresh so the learner sees the credential immediately.
	_, _ = credentialwallet.Refresh(ctx, pool, cfg, p.UserID)

	return &IssueResult{Diploma: dip}, nil
}

func releaseIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, d *diplomasrepo.Diploma) error {
	// Point the superseded row's program_ref at its own id so (user, template, program_ref)
	// no longer collides with the new issuance using the original program_ref (including NULL).
	sentinel := d.ID
	_, err := pool.Exec(ctx, `
UPDATE credentials.diplomas
SET program_ref = $2
WHERE id = $1
`, d.ID, sentinel)
	return err
}

// ProcessBatch issues pending/failed items in a batch (resumable).
func ProcessBatch(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, batchID uuid.UUID) error {
	if !cfg.FFDiplomas {
		return ErrFeatureDisabled
	}
	batch, err := diplomasrepo.GetBatch(ctx, pool, batchID)
	if err != nil {
		return err
	}
	if batch == nil {
		return ErrNotFound
	}
	if err := diplomasrepo.MarkBatchRunning(ctx, pool, batchID); err != nil {
		return err
	}
	items, err := diplomasrepo.ListPendingBatchItems(ctx, pool, batchID)
	if err != nil {
		return err
	}
	var lastErr error
	failN := 0
	for _, it := range items {
		name, _ := lookupDisplayName(ctx, pool, it.UserID)
		res, err := Issue(ctx, pool, cfg, IssueParams{
			OrgID:       batch.OrgID,
			TemplateID:  batch.TemplateID,
			UserID:      it.UserID,
			LearnerName: name,
			Program:     batch.Program,
			Honors:      batch.Honors,
			ConferredAt: batch.ConferredAt,
			ProgramRef:  batch.ProgramRef,
			IssuedBy:    batch.CreatedBy,
		})
		if err != nil {
			failN++
			lastErr = err
			_ = diplomasrepo.UpdateBatchItemResult(ctx, pool, it.ID, "failed", nil, err.Error())
			logging.GlobalDiplomaMetrics.IncBatchFail()
			continue
		}
		if res.Skipped {
			_ = diplomasrepo.UpdateBatchItemResult(ctx, pool, it.ID, "skipped", &res.Diploma.ID, res.Reason)
			logging.GlobalDiplomaMetrics.IncBatchSkip()
			continue
		}
		_ = diplomasrepo.UpdateBatchItemResult(ctx, pool, it.ID, "issued", &res.Diploma.ID, "")
		logging.GlobalDiplomaMetrics.IncBatchOK()
	}
	status := "completed"
	summary := ""
	if failN > 0 && failN == len(items) {
		status = "failed"
		if lastErr != nil {
			summary = lastErr.Error()
		}
	} else if failN > 0 {
		summary = fmt.Sprintf("%d item(s) failed", failN)
	}
	return diplomasrepo.FinishBatch(ctx, pool, batchID, status, summary)
}

func lookupDisplayName(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var name *string
	err := pool.QueryRow(ctx, `
SELECT NULLIF(BTRIM(COALESCE(display_name, '')), '')
FROM "user".users WHERE id = $1
`, userID).Scan(&name)
	if err != nil {
		return "", err
	}
	if name != nil {
		return *name, nil
	}
	return "", nil
}

// Revoke revokes an issued diploma (RBAC at HTTP).
func Revoke(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, id uuid.UUID, reason string) (*diplomasrepo.Diploma, error) {
	if !cfg.FFDiplomas {
		return nil, ErrFeatureDisabled
	}
	d, err := diplomasrepo.Revoke(ctx, pool, id, reason)
	if err != nil {
		if errors.Is(err, diplomasrepo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	logging.GlobalDiplomaMetrics.IncRevoked()
	telemetry.RecordBusinessEvent("credential.diploma.revoked")
	_, _ = credentialwallet.Refresh(ctx, pool, cfg, d.UserID)
	return d, nil
}

// Unrevoke clears revocation.
func Unrevoke(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, id uuid.UUID) (*diplomasrepo.Diploma, error) {
	if !cfg.FFDiplomas {
		return nil, ErrFeatureDisabled
	}
	d, err := diplomasrepo.Unrevoke(ctx, pool, id)
	if err != nil {
		if errors.Is(err, diplomasrepo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	_, _ = credentialwallet.Refresh(ctx, pool, cfg, d.UserID)
	return d, nil
}
