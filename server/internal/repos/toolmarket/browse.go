package toolmarket

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BrowseListings returns discoverable tools (public + granted unlisted).
func BrowseListings(ctx context.Context, pool *pgxpool.Pool, f BrowseFilters) ([]Listing, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	q := strings.TrimSpace(f.Query)
	subject := strings.TrimSpace(f.Subject)
	grade := strings.TrimSpace(f.Grade)

	rows, err := pool.Query(ctx, `
		SELECT t.tool_id, t.display_name, t.summary, t.subject_tags, t.grade_tags,
		       t.visibility, t.pricing_model, t.status, t.support_url, t.privacy_url,
		       COALESCE(r.version, ''),
		       COALESCE(r.data_sheet_json->>'wcagLevel', COALESCE(r.data_sheet_json->>'wcag_level', 'AA')),
		       COALESCE(
		         (SELECT array_agg(c) FROM jsonb_array_elements_text(COALESCE(r.manifest_json->'capabilities','[]'::jsonb)) c),
		         '{}'::text[]
		       ),
		       r.sunset_at
		FROM toolmarket.tools t
		LEFT JOIN LATERAL (
			SELECT * FROM toolmarket.tool_releases tr
			WHERE tr.tool_pk = t.id AND tr.review_status = 'approved' AND tr.published_at IS NOT NULL
			ORDER BY tr.published_at DESC
			LIMIT 1
		) r ON TRUE
		WHERE t.status = 'approved'
		  AND (
		    t.visibility = 'public'
		    OR (
		      t.visibility IN ('unlisted','private')
		      AND $1::uuid IS NOT NULL
		      AND EXISTS (
		        SELECT 1 FROM toolmarket.tool_access_grants g
		        WHERE g.tool_pk = t.id AND g.org_id = $1::uuid
		      )
		    )
		  )
		  AND ($2 = '' OR $2 = ANY(t.subject_tags))
		  AND ($3 = '' OR $3 = ANY(t.grade_tags))
		  AND ($4 = '' OR t.display_name ILIKE '%'||$4||'%' OR t.summary ILIKE '%'||$4||'%' OR t.tool_id ILIKE '%'||$4||'%')
		ORDER BY t.display_name ASC
		LIMIT $5 OFFSET $6`,
		f.OrgID, subject, grade, q, f.Limit, f.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Listing{}
	for rows.Next() {
		var l Listing
		if err := rows.Scan(
			&l.ToolID, &l.DisplayName, &l.Summary, &l.SubjectTags, &l.GradeTags,
			&l.Visibility, &l.PricingModel, &l.Status, &l.SupportURL, &l.PrivacyURL,
			&l.Version, &l.WCAGLevel, &l.Capabilities, &l.SunsetAt,
		); err != nil {
			return nil, err
		}
		if l.SubjectTags == nil {
			l.SubjectTags = []string{}
		}
		if l.GradeTags == nil {
			l.GradeTags = []string{}
		}
		if l.Capabilities == nil {
			l.Capabilities = []string{}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetPublicListing returns a listing if the caller may see it.
func GetPublicListing(ctx context.Context, pool *pgxpool.Pool, toolID string, orgID *uuid.UUID) (*Listing, *Tool, *Release, error) {
	tool, err := GetToolByToolID(ctx, pool, toolID)
	if err != nil || tool == nil {
		return nil, tool, nil, err
	}
	if tool.Status != StatusApproved {
		return nil, tool, nil, nil
	}
	visible := tool.Visibility == VisibilityPublic
	if !visible && orgID != nil {
		ok, err := HasAccessGrant(ctx, pool, tool.ID, *orgID)
		if err != nil {
			return nil, tool, nil, err
		}
		visible = ok
	}
	if !visible {
		return nil, tool, nil, nil
	}
	rels, err := ListReleases(ctx, pool, tool.ID)
	if err != nil {
		return nil, tool, nil, err
	}
	var latest *Release
	for i := range rels {
		if rels[i].ReviewStatus == ReviewApproved && rels[i].PublishedAt != nil {
			latest = &rels[i]
			break
		}
	}
	if latest == nil {
		return nil, tool, nil, nil
	}
	caps := []string{}
	var manifest struct {
		Capabilities []string `json:"capabilities"`
	}
	_ = json.Unmarshal(latest.ManifestJSON, &manifest)
	if manifest.Capabilities != nil {
		caps = manifest.Capabilities
	}
	wcag := "AA"
	var sheet struct {
		WCAGLevel string `json:"wcagLevel"`
	}
	_ = json.Unmarshal(latest.DataSheetJSON, &sheet)
	if sheet.WCAGLevel != "" {
		wcag = sheet.WCAGLevel
	}
	listing := &Listing{
		ToolID:       tool.ToolID,
		DisplayName:  tool.DisplayName,
		Summary:      tool.Summary,
		SubjectTags:  tool.SubjectTags,
		GradeTags:    tool.GradeTags,
		Visibility:   tool.Visibility,
		PricingModel: tool.PricingModel,
		Status:       tool.Status,
		Version:      latest.Version,
		WCAGLevel:    wcag,
		Capabilities: caps,
		SupportURL:   tool.SupportURL,
		PrivacyURL:   tool.PrivacyURL,
		SunsetAt:     latest.SunsetAt,
	}
	return listing, tool, latest, nil
}
