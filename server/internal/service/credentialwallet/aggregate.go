// Package credentialwallet aggregates learner credentials into a portable wallet (T09).
package credentialwallet

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	badgesrepo "github.com/lextures/lextures/server/internal/repos/badges"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
	credrepo "github.com/lextures/lextures/server/internal/repos/credentials"
	diplomasrepo "github.com/lextures/lextures/server/internal/repos/diplomas"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	walletrepo "github.com/lextures/lextures/server/internal/repos/wallet"
)

// Enabled reports whether any credential source flag is on (plan T09 rollout).
func Enabled(cfg config.Config) bool {
	return cfg.FFTranscripts ||
		cfg.FFCoCurricularTranscript ||
		cfg.FFCompetencyBadges ||
		cfg.FFCompletionCredentials ||
		cfg.FFCEUTracking ||
		cfg.FFDiplomas
}

func institutionName(cfg config.Config) string {
	name := strings.TrimSpace(cfg.CCRInstitutionName)
	if name == "" {
		return "Lextures"
	}
	return name
}

// CollectSources gathers live credential rows from each registered provider.
func CollectSources(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID) ([]walletrepo.SourceItem, error) {
	if pool == nil {
		return nil, fmt.Errorf("credentialwallet: missing pool")
	}
	issuer := institutionName(cfg)
	var out []walletrepo.SourceItem

	if cfg.FFTranscripts {
		docs, err := transcriptsrepo.ListDocumentsByUser(ctx, pool, userID)
		if err != nil {
			return nil, fmt.Errorf("credentialwallet: transcripts: %w", err)
		}
		for _, d := range docs {
			title := transcriptTitle(d)
			var issued *time.Time
			if !d.GeneratedAt.IsZero() {
				t := d.GeneratedAt
				issued = &t
			}
			verify := ""
			if d.VerifyToken != nil {
				verify = *d.VerifyToken
			}
			out = append(out, walletrepo.SourceItem{
				Kind:        walletrepo.KindTranscript,
				SourceID:    d.ID,
				Title:       title,
				Issuer:      issuer,
				IssuedAt:    issued,
				VerifyToken: verify,
				Revoked:     d.RevokedAt != nil,
				Metadata: map[string]any{
					"variant":      string(d.Variant),
					"version":      d.Version,
					"contentHash":  d.ContentHash,
					"hasPDF":       len(d.PDFBytes) > 0 || (d.PDFKey != nil && *d.PDFKey != ""),
					"hasVC":        len(d.VCProof) > 0,
					"downloadPath": "/api/v1/transcripts/documents/" + d.ID.String() + "/download",
				},
			})
		}
	}

	if cfg.FFCoCurricularTranscript {
		docs, err := ccrrepo.ListDocuments(ctx, pool, userID)
		if err != nil {
			return nil, fmt.Errorf("credentialwallet: clr: %w", err)
		}
		for _, d := range docs {
			var issued *time.Time
			if !d.GeneratedAt.IsZero() {
				t := d.GeneratedAt
				issued = &t
			}
			verify := ""
			if d.ShareToken != nil {
				verify = *d.ShareToken
			}
			out = append(out, walletrepo.SourceItem{
				Kind:        walletrepo.KindCLR,
				SourceID:    d.ID,
				Title:       "Co-curricular record",
				Issuer:      issuer,
				IssuedAt:    issued,
				VerifyToken: verify,
				Revoked:     false,
				Metadata: map[string]any{
					"hasVC":        len(d.VCProof) > 0,
					"downloadPath": "/api/v1/me/ccr/" + d.ID.String() + "/download?format=pdf",
					"jsonPath":     "/api/v1/me/ccr/" + d.ID.String() + "/download?format=json",
				},
			})
		}
	}

	if cfg.FFCompetencyBadges {
		awards, err := badgesrepo.ListAwardsByRecipient(ctx, pool, userID)
		if err != nil {
			return nil, fmt.Errorf("credentialwallet: badges: %w", err)
		}
		for _, a := range awards {
			title := "Competency badge"
			def, err := badgesrepo.GetDefinitionByID(ctx, pool, a.DefinitionID)
			if err != nil {
				return nil, err
			}
			if def != nil && strings.TrimSpace(def.Name) != "" {
				title = def.Name
			}
			var issued *time.Time
			if !a.IssuedAt.IsZero() {
				t := a.IssuedAt
				issued = &t
			}
			out = append(out, walletrepo.SourceItem{
				Kind:        walletrepo.KindBadge,
				SourceID:    a.ID,
				Title:       title,
				Issuer:      issuer,
				IssuedAt:    issued,
				VerifyToken: a.ShareSlug,
				Revoked:     a.Revoked,
				Metadata: map[string]any{
					"shareSlug": a.ShareSlug,
					"isPublic":  a.IsPublic,
					"hasVC":     len(a.CredentialJSON) > 0,
					"verifyPath": "/api/v1/badges/verify/" + a.ShareSlug,
				},
			})
		}
	}

	if cfg.FFCompletionCredentials {
		creds, err := credrepo.ListByRecipient(ctx, pool, userID)
		if err != nil {
			return nil, fmt.Errorf("credentialwallet: certificates: %w", err)
		}
		for _, c := range creds {
			var issued *time.Time
			if !c.IssuedAt.IsZero() {
				t := c.IssuedAt
				issued = &t
			}
			out = append(out, walletrepo.SourceItem{
				Kind:        walletrepo.KindCertificate,
				SourceID:    c.ID,
				Title:       c.Title,
				Issuer:      issuer,
				IssuedAt:    issued,
				VerifyToken: c.ID.String(),
				Revoked:     c.Revoked,
				Metadata: map[string]any{
					"sourceType":   string(c.SourceType),
					"hasVC":        len(c.CredentialJSON) > 0,
					"downloadPath": "/api/v1/credentials/" + c.ID.String() + "/download",
					"jsonPath":     "/api/v1/credentials/" + c.ID.String() + "/json",
					"verifyPath":   "/api/v1/credentials/" + c.ID.String() + "/verify",
				},
			})
		}
	}

	if cfg.FFCEUTracking {
		ceItems, err := listCERecords(ctx, pool, userID, issuer)
		if err != nil {
			return nil, err
		}
		out = append(out, ceItems...)
	}

	if cfg.FFDiplomas {
		dips, err := diplomasrepo.ListByUser(ctx, pool, userID)
		if err != nil {
			return nil, fmt.Errorf("credentialwallet: diplomas: %w", err)
		}
		for _, d := range dips {
			kind := walletrepo.KindDiploma
			if d.Kind == diplomasrepo.KindCertificate {
				// Formal T11 certificates share the diploma wallet bucket so they stay
				// distinct from course completion certificates (KindCertificate / 15.5).
				kind = walletrepo.KindDiploma
			}
			var issued *time.Time
			if !d.IssuedAt.IsZero() {
				t := d.IssuedAt
				issued = &t
			}
			verify := ""
			if d.VerifyToken != nil {
				verify = *d.VerifyToken
			}
			meta := map[string]any{
				"kind":         string(d.Kind),
				"version":      d.Version,
				"contentHash":  d.ContentHash,
				"hasPDF":       len(d.PDFBytes) > 0 || (d.PDFKey != nil && *d.PDFKey != ""),
				"hasVC":        len(d.VCProof) > 0,
				"downloadPath": "/api/v1/me/diplomas/" + d.ID.String() + "/download",
				"conferredAt":  d.ConferredAt.UTC().Format(time.RFC3339),
			}
			if d.Program != nil {
				meta["program"] = *d.Program
			}
			if d.Honors != nil {
				meta["honors"] = *d.Honors
			}
			out = append(out, walletrepo.SourceItem{
				Kind:        kind,
				SourceID:    d.ID,
				Title:       d.CredentialTitle,
				Issuer:      issuer,
				IssuedAt:    issued,
				VerifyToken: verify,
				Revoked:     d.RevokedAt != nil,
				Metadata:    meta,
			})
		}
	}

	return out, nil
}

func transcriptTitle(d transcriptsrepo.Document) string {
	switch d.Variant {
	case transcriptsrepo.DocumentOfficial:
		return fmt.Sprintf("Official transcript (v%d)", d.Version)
	case transcriptsrepo.DocumentUnofficial:
		return "Unofficial transcript"
	case transcriptsrepo.DocumentPartial:
		return "Partial transcript"
	case transcriptsrepo.DocumentInProgress:
		return "In-progress transcript"
	default:
		return "Transcript"
	}
}

func listCERecords(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, issuer string) ([]walletrepo.SourceItem, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id,
       COALESCE(NULLIF(BTRIM(c.title), ''), NULLIF(BTRIM(c.course_code), ''), 'CE record') AS title,
       a.issued_at,
       a.ceu_credit::float8,
       a.contact_hours::float8
FROM seattime.ceu_awards a
LEFT JOIN course.courses c ON c.id = a.course_id
WHERE a.user_id = $1
ORDER BY a.issued_at DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("credentialwallet: ce_record: %w", err)
	}
	defer rows.Close()
	var out []walletrepo.SourceItem
	for rows.Next() {
		var id uuid.UUID
		var title string
		var issuedAt time.Time
		var ceu, hours float64
		if err := rows.Scan(&id, &title, &issuedAt, &ceu, &hours); err != nil {
			return nil, err
		}
		t := issuedAt
		out = append(out, walletrepo.SourceItem{
			Kind:     walletrepo.KindCERecord,
			SourceID: id,
			Title:    title,
			Issuer:   issuer,
			IssuedAt: &t,
			Revoked:  false,
			Metadata: map[string]any{
				"ceuCredit":     ceu,
				"contactHours":  hours,
				"downloadPath":  "/api/v1/me/ce-transcript",
			},
		})
	}
	return out, rows.Err()
}

// Refresh rebuilds the wallet index for a user from live sources.
func Refresh(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID) ([]walletrepo.Item, error) {
	sources, err := CollectSources(ctx, pool, cfg, userID)
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		if _, err := walletrepo.UpsertSource(ctx, pool, userID, src); err != nil {
			return nil, err
		}
	}
	if err := walletrepo.DeleteMissing(ctx, pool, userID, sources); err != nil {
		return nil, err
	}
	return walletrepo.ListItems(ctx, pool, userID)
}

// VerifyStatus maps an item to a learner-facing verification label.
func VerifyStatus(it walletrepo.Item) string {
	if it.Revoked {
		return "revoked"
	}
	if it.VerifyToken != nil && strings.TrimSpace(*it.VerifyToken) != "" {
		return "verified"
	}
	switch it.Kind {
	case walletrepo.KindCERecord:
		return "unavailable"
	default:
		return "unverified"
	}
}

// VerifyURL builds a public verify URL when possible.
func VerifyURL(webOrigin string, it walletrepo.Item) string {
	if it.VerifyToken == nil || strings.TrimSpace(*it.VerifyToken) == "" {
		return ""
	}
	origin := strings.TrimRight(strings.TrimSpace(webOrigin), "/")
	tok := strings.TrimSpace(*it.VerifyToken)
	switch it.Kind {
	case walletrepo.KindBadge:
		return origin + "/api/v1/badges/verify/" + tok
	case walletrepo.KindCertificate:
		return origin + "/verify/" + tok
	case walletrepo.KindTranscript, walletrepo.KindCLR, walletrepo.KindDiploma:
		return origin + "/verify/" + tok
	default:
		return ""
	}
}
