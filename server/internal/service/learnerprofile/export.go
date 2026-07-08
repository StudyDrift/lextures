package learnerprofile

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
)

// ExportDocument is a portable GDPR Art. 20 export of the full learner profile.
type ExportDocument struct {
	UserID         string                   `json:"userId"`
	Status         string                   `json:"status"`
	LastComputedAt *string                  `json:"lastComputedAt,omitempty"`
	Facets         []ExportFacet            `json:"facets"`
	Disclosure     ExportDisclosure         `json:"disclosure"`
	ExportedAt     string                   `json:"exportedAt"`
	ExportKind     string                   `json:"exportKind"`
}

// ExportDisclosure states the Art. 22 posture for the derived profile.
type ExportDisclosure struct {
	ProfilingNotice string `json:"profilingNotice"`
	Art22Posture    string `json:"art22Posture"`
}

// ExportFacet is one facet with insights and full provenance.
type ExportFacet struct {
	FacetKey        string          `json:"facetKey"`
	State           string          `json:"state"`
	Summary         any             `json:"summary"`
	Confidence      float64         `json:"confidence"`
	ComputedVersion int             `json:"computedVersion"`
	UpdatedAt       string          `json:"updatedAt"`
	Insights        []ExportInsight `json:"insights"`
}

// ExportInsight is one insight with evidence provenance.
type ExportInsight struct {
	InsightKey   string         `json:"insightKey"`
	Label        string         `json:"label"`
	LabelI18nKey string         `json:"labelI18nKey"`
	Value        any            `json:"value"`
	Confidence   float64        `json:"confidence"`
	Salience     int            `json:"salience"`
	Evidence     []ExportEvidence `json:"evidence"`
}

// ExportEvidence is one provenance row in a portable export.
type ExportEvidence struct {
	SourceKind       string  `json:"sourceKind"`
	SourceTable      string  `json:"sourceTable"`
	ObservationCount int     `json:"observationCount"`
	CourseID         *string `json:"courseId,omitempty"`
	WindowStart      *string `json:"windowStart,omitempty"`
	WindowEnd        *string `json:"windowEnd,omitempty"`
	Contribution     *float64 `json:"contribution,omitempty"`
	SampleRefs       any     `json:"sampleRefs,omitempty"`
}

// Export builds a portable export document for userID including full provenance.
func (s *Service) Export(ctx context.Context, userID uuid.UUID) (ExportDocument, error) {
	profileID, err := lprepo.EnsureProfile(ctx, s.Pool, userID)
	if err != nil {
		return ExportDocument{}, err
	}
	p, err := lprepo.GetProfileByUserID(ctx, s.Pool, userID)
	if err != nil {
		return ExportDocument{}, err
	}
	status := "active"
	var lastComputed *string
	if p != nil {
		status = p.Status
		if p.LastComputedAt != nil {
			s := p.LastComputedAt.UTC().Format(time.RFC3339)
			lastComputed = &s
		}
	}
	facets, err := lprepo.ListFacets(ctx, s.Pool, profileID)
	if err != nil {
		return ExportDocument{}, err
	}
	exportFacets := make([]ExportFacet, 0, len(facets))
	for _, f := range facets {
		insights, err := lprepo.ListInsights(ctx, s.Pool, f.ID)
		if err != nil {
			return ExportDocument{}, err
		}
		ids := make([]uuid.UUID, len(insights))
		for i, ins := range insights {
			ids[i] = ins.ID
		}
		evMap, err := lprepo.ListEvidenceForInsights(ctx, s.Pool, ids)
		if err != nil {
			return ExportDocument{}, err
		}
		exportInsights := make([]ExportInsight, 0, len(insights))
		for _, ins := range insights {
			evRows := evMap[ins.ID]
			if len(evRows) == 0 {
				continue
			}
			exportInsights = append(exportInsights, exportInsightToExport(ins, evRows))
		}
		var summary any
		_ = json.Unmarshal(f.Summary, &summary)
		if summary == nil {
			summary = map[string]any{}
		}
		exportFacets = append(exportFacets, ExportFacet{
			FacetKey:        f.FacetKey,
			State:           f.State,
			Summary:         summary,
			Confidence:      f.Confidence,
			ComputedVersion: f.ComputedVersion,
			UpdatedAt:       f.UpdatedAt.UTC().Format(time.RFC3339),
			Insights:        exportInsights,
		})
	}
	return ExportDocument{
		UserID:         userID.String(),
		Status:         status,
		LastComputedAt: lastComputed,
		Facets:         exportFacets,
		Disclosure: ExportDisclosure{
			ProfilingNotice: "This learner profile is derived automatically from your learning activity across courses.",
			Art22Posture:    "The learner profile and its consumers are assistive and advisory only. Lextures does not use the profile for consequential automated decisions without meaningful human oversight.",
		},
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		ExportKind: "learner-profile",
	}, nil
}

func exportInsightToExport(ins lprepo.Insight, evRows []lprepo.Evidence) ExportInsight {
	var value any
	_ = json.Unmarshal(ins.Value, &value)
	if value == nil {
		value = map[string]any{}
	}
	evidence := make([]ExportEvidence, 0, len(evRows))
	for _, ev := range evRows {
		item := ExportEvidence{
			SourceKind:       ev.SourceKind,
			SourceTable:      ev.SourceTable,
			ObservationCount: ev.ObservationCount,
			Contribution:     ev.Contribution,
		}
		if ev.CourseID != nil {
			s := ev.CourseID.String()
			item.CourseID = &s
		}
		if ev.WindowStart != nil {
			s := ev.WindowStart.UTC().Format(time.RFC3339)
			item.WindowStart = &s
		}
		if ev.WindowEnd != nil {
			s := ev.WindowEnd.UTC().Format(time.RFC3339)
			item.WindowEnd = &s
		}
		if len(ev.SampleRefs) > 0 {
			var refs any
			_ = json.Unmarshal(ev.SampleRefs, &refs)
			item.SampleRefs = refs
		}
		evidence = append(evidence, item)
	}
	return ExportInsight{
		InsightKey:   ins.InsightKey,
		Label:        ResolveLabel("en", ins.LabelI18nKey),
		LabelI18nKey: ins.LabelI18nKey,
		Value:        value,
		Confidence:   ins.Confidence,
		Salience:     ins.Salience,
		Evidence:     evidence,
	}
}