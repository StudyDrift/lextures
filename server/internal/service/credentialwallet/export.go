package credentialwallet

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
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

// Manifest describes the contents of a wallet export ZIP.
type Manifest struct {
	ExportedAt string           `json:"exportedAt"`
	UserID     string           `json:"userId"`
	Items      []ManifestEntry  `json:"items"`
}

// ManifestEntry is one credential file set in the bundle.
type ManifestEntry struct {
	Kind       string  `json:"kind"`
	SourceID   string  `json:"sourceId"`
	Title      string  `json:"title"`
	Issuer     string  `json:"issuer,omitempty"`
	IssuedAt   *string `json:"issuedAt,omitempty"`
	Revoked    bool    `json:"revoked"`
	PDFPath    string  `json:"pdfPath,omitempty"`
	VCPath     string  `json:"vcPath,omitempty"`
	JSONPath   string  `json:"jsonPath,omitempty"`
}

// BuildExportBundle creates a ZIP of PDFs + VC JSON + manifest for the learner.
func BuildExportBundle(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID) ([]byte, json.RawMessage, error) {
	items, err := Refresh(ctx, pool, cfg, userID)
	if err != nil {
		return nil, nil, err
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	manifest := Manifest{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		UserID:     userID.String(),
		Items:      make([]ManifestEntry, 0, len(items)),
	}

	for i, it := range items {
		prefix := fmt.Sprintf("%02d-%s-%s", i+1, sanitizePath(string(it.Kind)), it.SourceID.String()[:8])
		entry := ManifestEntry{
			Kind:     string(it.Kind),
			SourceID: it.SourceID.String(),
			Title:    it.Title,
			Revoked:  it.Revoked,
		}
		if it.Issuer != nil {
			entry.Issuer = *it.Issuer
		}
		if it.IssuedAt != nil {
			s := it.IssuedAt.UTC().Format(time.RFC3339)
			entry.IssuedAt = &s
		}

		switch it.Kind {
		case walletrepo.KindTranscript:
			doc, err := transcriptsrepo.GetDocumentByID(ctx, pool, userID, it.SourceID)
			if err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			if doc != nil {
				if len(doc.PDFBytes) > 0 {
					path := prefix + "/transcript.pdf"
					if err := writeZipFile(zw, path, doc.PDFBytes); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.PDFPath = path
				}
				if len(doc.VCProof) > 0 {
					path := prefix + "/credential.vc.json"
					if err := writeZipFile(zw, path, doc.VCProof); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.VCPath = path
				}
				if len(doc.Canonical) > 0 {
					path := prefix + "/record.json"
					if err := writeZipFile(zw, path, doc.Canonical); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.JSONPath = path
				}
			}
		case walletrepo.KindCLR:
			doc, err := ccrrepo.GetDocumentByID(ctx, pool, userID, it.SourceID)
			if err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			if doc != nil {
				if len(doc.CLRJSON) > 0 {
					path := prefix + "/clr.json"
					if err := writeZipFile(zw, path, doc.CLRJSON); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.JSONPath = path
				}
				if len(doc.VCProof) > 0 {
					path := prefix + "/credential.vc.json"
					if err := writeZipFile(zw, path, doc.VCProof); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.VCPath = path
				}
			}
		case walletrepo.KindBadge:
			award, err := badgesrepo.GetAwardByID(ctx, pool, it.SourceID)
			if err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			if award != nil && award.RecipientID == userID {
				if len(award.CredentialJSON) > 0 {
					path := prefix + "/open-badge.json"
					if err := writeZipFile(zw, path, award.CredentialJSON); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.JSONPath = path
				}
				if len(award.Proof) > 0 {
					path := prefix + "/credential.vc.json"
					if err := writeZipFile(zw, path, award.Proof); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.VCPath = path
				}
			}
		case walletrepo.KindCertificate:
			cred, err := credrepo.GetByID(ctx, pool, it.SourceID)
			if err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			if cred != nil && cred.RecipientID == userID {
				if len(cred.CredentialJSON) > 0 {
					path := prefix + "/open-badge.json"
					if err := writeZipFile(zw, path, cred.CredentialJSON); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.JSONPath = path
				}
				if len(cred.Proof) > 0 {
					path := prefix + "/credential.vc.json"
					if err := writeZipFile(zw, path, cred.Proof); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.VCPath = path
				}
			}
		case walletrepo.KindCERecord:
			summary := map[string]any{
				"kind":  "ce_record",
				"title": it.Title,
				"id":    it.SourceID.String(),
			}
			if it.IssuedAt != nil {
				summary["issuedAt"] = it.IssuedAt.UTC().Format(time.RFC3339)
			}
			raw, _ := json.MarshalIndent(summary, "", "  ")
			path := prefix + "/ce-record.json"
			if err := writeZipFile(zw, path, raw); err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			entry.JSONPath = path
		case walletrepo.KindDiploma:
			dip, err := diplomasrepo.GetByID(ctx, pool, it.SourceID)
			if err != nil {
				_ = zw.Close()
				return nil, nil, err
			}
			if dip != nil && dip.UserID == userID {
				if len(dip.PDFBytes) > 0 {
					path := prefix + "/diploma.pdf"
					if err := writeZipFile(zw, path, dip.PDFBytes); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.PDFPath = path
				}
				if len(dip.VCProof) > 0 {
					path := prefix + "/credential.vc.json"
					if err := writeZipFile(zw, path, dip.VCProof); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.VCPath = path
				}
				if len(dip.Canonical) > 0 {
					path := prefix + "/record.json"
					if err := writeZipFile(zw, path, dip.Canonical); err != nil {
						_ = zw.Close()
						return nil, nil, err
					}
					entry.JSONPath = path
				}
			}
		}

		manifest.Items = append(manifest.Items, entry)
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = zw.Close()
		return nil, nil, err
	}
	if err := writeZipFile(zw, "manifest.json", manifestBytes); err != nil {
		_ = zw.Close()
		return nil, nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), json.RawMessage(manifestBytes), nil
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func sanitizePath(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "item"
	}
	return out
}

// ProcessExport builds and stores a pending export (background worker entrypoint).
func ProcessExport(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, exportID uuid.UUID) error {
	exp, err := walletrepo.GetExportByID(ctx, pool, exportID)
	if err != nil {
		return err
	}
	if exp == nil {
		return walletrepo.ErrExportNotFound
	}
	if exp.Status == walletrepo.ExportReady {
		return nil
	}
	zipBytes, manifest, err := BuildExportBundle(ctx, pool, cfg, exp.UserID)
	if err != nil {
		_ = walletrepo.FailExport(ctx, pool, exportID, err.Error())
		return err
	}
	return walletrepo.CompleteExport(ctx, pool, exportID, zipBytes, manifest)
}
