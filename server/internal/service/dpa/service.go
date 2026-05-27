// Package dpa implements the SDPC/NDPA Data Processing Agreement portal:
// DPA versioning, district acceptance recording, data inventory, and SDPC CSV export (plan 10.5).
package dpa

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/dpa"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates DPA compliance admin actions (list all acceptances, manage versions).
const AdminPermission = "compliance:dpa:admin:*"

// ReAcknowledgementWindow is the time a district has to re-acknowledge after a DPA version bump.
const ReAcknowledgementWindow = 30 * 24 * time.Hour

var (
	ErrNotFound  = errors.New("dpa: record not found")
	ErrForbidden = errors.New("dpa: forbidden")
)

// CheckAdmin returns true when the user holds the compliance:dpa:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// GetCurrentVersion returns the most recently effective DPA version, or ErrNotFound.
func GetCurrentVersion(ctx context.Context, pool *pgxpool.Pool) (*repo.DPAVersion, error) {
	v, err := repo.GetCurrentVersion(ctx, pool)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

// ListVersions returns all DPA versions ordered by effective_at descending.
func ListVersions(ctx context.Context, pool *pgxpool.Pool) ([]repo.DPAVersion, error) {
	return repo.ListVersions(ctx, pool)
}

// AcceptDPA records a DPA acceptance for the given org and version.
// Returns the acceptance row ID. Idempotent for the same org+version pair.
func AcceptDPA(ctx context.Context, pool *pgxpool.Pool, orgID, dpaVersionID, acceptedBy uuid.UUID, remoteAddr string) (uuid.UUID, error) {
	ip := parseIP(remoteAddr)
	return repo.InsertAcceptance(ctx, pool, orgID, dpaVersionID, acceptedBy, ip)
}

// HasSignedCurrentVersion reports whether the org has accepted the current DPA version.
func HasSignedCurrentVersion(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (bool, error) {
	v, err := repo.GetCurrentVersion(ctx, pool)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	a, err := repo.GetAcceptanceByOrgVersion(ctx, pool, orgID, v.ID)
	if err != nil {
		return false, err
	}
	return a != nil, nil
}

// GetOrgAcceptance returns the acceptance for a given org+version, or nil.
func GetOrgAcceptance(ctx context.Context, pool *pgxpool.Pool, orgID, dpaVersionID uuid.UUID) (*repo.DPAAcceptance, error) {
	return repo.GetAcceptanceByOrgVersion(ctx, pool, orgID, dpaVersionID)
}

// ListAcceptances returns all DPA acceptance records (for compliance admin).
func ListAcceptances(ctx context.Context, pool *pgxpool.Pool) ([]repo.DPAAcceptance, error) {
	return repo.ListAcceptances(ctx, pool)
}

// ListDataInventory returns all data inventory items.
func ListDataInventory(ctx context.Context, pool *pgxpool.Pool) ([]repo.DataInventoryItem, error) {
	return repo.ListDataInventory(ctx, pool)
}

// SDPCCSVExport generates an SDPC-compatible CSV export of the data inventory.
// Columns follow the SDPC National Data Privacy Agreement exhibit format.
func SDPCCSVExport(ctx context.Context, pool *pgxpool.Pool) ([]byte, error) {
	items, err := repo.ListDataInventory(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("dpa: list inventory: %w", err)
	}
	return buildCSVBytes(items)
}

// buildCSVBytes writes items to SDPC CSV format and returns the bytes.
func buildCSVBytes(items []repo.DataInventoryItem) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"Element Name", "Category", "Purpose", "Legal Basis",
		"Retention Days", "Shared with Sub-Processors", "Sub-Processor Names",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, item := range items {
		retDays := ""
		if item.RetentionDays != nil {
			retDays = strconv.Itoa(*item.RetentionDays)
		}
		shared := "No"
		if item.SharedWithSubProcessors {
			shared = "Yes"
		}
		spNames := strings.Join(item.SubProcessorNames, "; ")
		row := []string{
			item.ElementName,
			item.Category,
			item.Purpose,
			item.LegalBasis,
			retDays,
			shared,
			spNames,
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// NDPATemplateData is the pre-populated fields for the SDPC National DPA document.
type NDPATemplateData struct {
	VendorName          string    `json:"vendorName"`
	VendorAddress       string    `json:"vendorAddress"`
	DPAVersionStr       string    `json:"dpaVersionStr"`
	EffectiveAt         string    `json:"effectiveAt"`
	TemplateURL         string    `json:"templateUrl"`
	SubProcessors       []string  `json:"subProcessors"`
	DataInventorySummary []string `json:"dataInventorySummary"`
	GeneratedAt         string    `json:"generatedAt"`
}

// GenerateNDPATemplate returns a pre-populated NDPA template data structure for a district admin.
func GenerateNDPATemplate(v *repo.DPAVersion, items []repo.DataInventoryItem) *NDPATemplateData {
	subs := ndpaSubProcessors(items)
	summary := make([]string, 0, len(items))
	for _, item := range items {
		summary = append(summary, fmt.Sprintf("%s (%s)", item.ElementName, item.Category))
	}
	return &NDPATemplateData{
		VendorName:           "Lextures, Inc.",
		VendorAddress:        "United States",
		DPAVersionStr:        v.VersionStr,
		EffectiveAt:          v.EffectiveAt.UTC().Format(time.RFC3339),
		TemplateURL:          v.TemplateURL,
		SubProcessors:        subs,
		DataInventorySummary: summary,
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
	}
}

func ndpaSubProcessors(items []repo.DataInventoryItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		for _, sp := range item.SubProcessorNames {
			if sp != "" && !seen[sp] {
				seen[sp] = true
				out = append(out, sp)
			}
		}
	}
	if len(out) == 0 {
		return []string{"OpenRouter (AI model routing)", "AWS (cloud infrastructure)", "SendGrid (transactional email)"}
	}
	return out
}

// parseIP extracts the IP from remoteAddr (host:port format).
func parseIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(host)
}
