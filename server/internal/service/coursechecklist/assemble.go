package coursechecklist

import (
	"strings"
	"time"

	"github.com/google/uuid"
	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
)

// AssembleOptions controls response assembly from an evaluation Result.
type AssembleOptions struct {
	CourseCode           string
	ComputedAt           time.Time
	Stale                bool
	EvidenceTruncated    bool
	IncludeNotApplicable bool
	Dismissed            []ccrepo.ItemState
	DisplayNames         map[uuid.UUID]string
}

// AssembleChecklist builds the API response from Result + dismissals (FR-12/13).
func AssembleChecklist(res Result, opt AssembleOptions) ChecklistResponse {
	dismissedByID := make(map[string]ccrepo.ItemState, len(opt.Dismissed))
	for _, st := range opt.Dismissed {
		if st.DismissedAt != nil {
			dismissedByID[st.ItemID] = st
		}
	}

	byCat := make(map[CategoryID][]ChecklistItem, len(CategoryOrder))
	var dismissedPile []ChecklistItem
	for _, fr := range res.Findings {
		item := itemFromFinding(fr, dismissedByID[string(fr.ID)], opt.DisplayNames)
		if _, isDismissed := dismissedByID[string(fr.ID)]; isDismissed {
			dismissedPile = append(dismissedPile, item)
			continue
		}
		if fr.Finding.Status == StatusNotApplicable && !opt.IncludeNotApplicable {
			continue
		}
		byCat[fr.Category] = append(byCat[fr.Category], item)
	}

	categories := make([]ChecklistCategory, 0, len(CategoryOrder))
	for _, catID := range CategoryOrder {
		items := byCat[catID]
		if len(items) == 0 {
			continue
		}
		meta := CategoryTitle(catID)
		categories = append(categories, ChecklistCategory{
			ID:       string(catID),
			TitleKey: meta.TitleKey,
			Title:    meta.Title,
			Items:    items,
		})
	}
	if dismissedPile == nil {
		dismissedPile = []ChecklistItem{}
	}

	summary := DeriveSummary(res, dismissedByID, opt.ComputedAt, opt.Stale)
	return ChecklistResponse{
		CourseCode:        opt.CourseCode,
		EngineVersion:     res.EngineVersion,
		CatalogVersion:    res.CatalogVersion,
		ComputedAt:        opt.ComputedAt,
		Stale:             opt.Stale,
		EvidenceTruncated: opt.EvidenceTruncated,
		Summary:           summary,
		Categories:        categories,
		Dismissed:         dismissedPile,
	}
}

// DeriveSummary computes badge counters excluding dismissed items (FR-14).
func DeriveSummary(res Result, dismissedByID map[string]ccrepo.ItemState, computedAt time.Time, stale bool) ChecklistSummary {
	var done, total, outstandingEssential, outstandingTotal int
	for _, fr := range res.Findings {
		if _, ok := dismissedByID[string(fr.ID)]; ok {
			continue
		}
		switch fr.Finding.Status {
		case StatusDone:
			done++
			total++
		case StatusTodo, StatusInProgress:
			total++
			outstandingTotal++
			if fr.Tier == TierEssential {
				outstandingEssential++
			}
		case StatusUnknown:
			// unknown excluded from total (same as Counts aggregation)
		case StatusNotApplicable:
			// N/A excluded from progress denominator
		}
	}
	return ChecklistSummary{
		OutstandingEssential: outstandingEssential,
		OutstandingTotal:     outstandingTotal,
		Done:                 done,
		Total:                total,
		Dismissed:            len(dismissedByID),
		ComputedAt:           computedAt,
		Stale:                stale,
	}
}

func itemFromFinding(fr ItemResult, st ccrepo.ItemState, names map[uuid.UUID]string) ChecklistItem {
	var detail *string
	if strings.TrimSpace(fr.Finding.DetailDefault) != "" {
		d := fr.Finding.DetailDefault
		detail = &d
	}
	var help *string
	if strings.TrimSpace(fr.HelpRef) != "" {
		h := fr.HelpRef
		help = &h
	}
	sources := fr.Sources
	if sources == nil {
		sources = []string{}
	}
	item := ChecklistItem{
		ID:       string(fr.ID),
		TitleKey: fr.TitleKey,
		Title:    fr.TitleDefault,
		WhyKey:   fr.WhyKey,
		Why:      fr.WhyDefault,
		Tier:     fr.Tier,
		Status:   fr.Finding.Status,
		Detail:   detail,
		Progress: fr.Finding.Progress,
		Sources:  sources,
		HelpRef:  help,
		Target:   navTargetPtr(fr.Target),
		Evidence: evidenceFromFinding(fr),
		Action:   actionFromFinding(fr),
	}
	if st.DismissedAt != nil {
		byID := ""
		byName := ""
		if st.DismissedByUserID != nil {
			byID = st.DismissedByUserID.String()
			if names != nil {
				byName = names[*st.DismissedByUserID]
			}
		}
		item.Dismissal = &ChecklistDismissal{
			DismissedAt:   st.DismissedAt.UTC(),
			ByUserID:      byID,
			ByDisplayName: byName,
			Reason:        st.DismissReason,
			Note:          st.DismissNote,
		}
	}
	return item
}

func actionFromFinding(fr ItemResult) *ChecklistAction {
	if fr.Action == nil {
		return nil
	}
	a := fr.Action
	if strings.TrimSpace(string(a.Kind)) == "" || strings.TrimSpace(a.Endpoint) == "" {
		return nil
	}
	label := a.Label
	if strings.TrimSpace(label) == "" {
		label = string(a.Kind)
	}
	return &ChecklistAction{
		Kind:       string(a.Kind),
		LabelKey:   a.LabelKey,
		Label:      label,
		Endpoint:   a.Endpoint,
		RequiresAI: a.RequiresAI,
	}
}

func navTargetPtr(t NavTarget) *ChecklistNavTarget {
	if strings.TrimSpace(t.Route) == "" {
		return nil
	}
	route := t.Route
	// Substitute per-row entity keys into {itemId} (CC.4 / CC.8 graceful path).
	if strings.TrimSpace(t.EntityKey) != "" {
		route = strings.ReplaceAll(route, "{itemId}", t.EntityKey)
	}
	out := &ChecklistNavTarget{Route: route}
	if strings.TrimSpace(t.Anchor) != "" {
		a := t.Anchor
		out.Anchor = &a
	}
	if strings.TrimSpace(t.EntityKey) != "" {
		ek := t.EntityKey
		out.EntityKey = &ek
	}
	return out
}

func evidenceFromFinding(fr ItemResult) *ChecklistEvidence {
	if len(fr.Finding.Evidence) == 0 && fr.EvidenceShape == nil {
		return nil
	}
	cols := []string{}
	if fr.EvidenceShape != nil {
		cols = append(cols, fr.EvidenceShape.Columns...)
	}
	rows := make([]ChecklistEvidenceRow, 0, len(fr.Finding.Evidence))
	for _, er := range fr.Finding.Evidence {
		var sub *string
		if strings.TrimSpace(er.Sublabel) != "" {
			s := er.Sublabel
			sub = &s
		}
		var tgt *ChecklistNavTarget
		if er.TargetOverride != nil {
			tgt = navTargetPtr(*er.TargetOverride)
		}
		rows = append(rows, ChecklistEvidenceRow{
			Label:    er.Label,
			Sublabel: sub,
			Status:   string(er.Status),
			Target:   tgt,
		})
	}
	return &ChecklistEvidence{
		Columns:     cols,
		Rows:        rows,
		TruncatedAt: fr.Finding.TruncatedAt,
	}
}

// DropEvidence strips evidence from all findings (payload size fallback, AC-13).
func DropEvidence(res Result) Result {
	out := res
	out.Findings = make([]ItemResult, len(res.Findings))
	copy(out.Findings, res.Findings)
	for i := range out.Findings {
		out.Findings[i].Finding.Evidence = nil
		out.Findings[i].Finding.TruncatedAt = nil
	}
	return out
}
