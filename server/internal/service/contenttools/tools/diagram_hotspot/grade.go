package diagram_hotspot

import (
	"math"
	"sort"
	"strings"
)

// GradeAssignments grades learner assignments against the answer key.
func GradeAssignments(cfg Config, assignments map[string]*string) GradeResult {
	itemIDs := ItemIDs(cfg)
	perItem := make(map[string]PerItemResult, len(itemIDs))
	correctIDs := make([]string, 0, len(itemIDs))

	correctMap := cfg.CorrectRegionByLabel
	if cfg.Mode == ModeHotspot {
		correctMap = cfg.CorrectRegionByPrompt
	}

	for _, id := range itemIDs {
		okItem := false
		gotRegion := ""
		if ptr, has := assignments[id]; has && ptr != nil {
			gotRegion = strings.TrimSpace(*ptr)
			want := ""
			if correctMap != nil {
				want = strings.TrimSpace(correctMap[id])
			}
			okItem = want != "" && gotRegion == want
		}
		fb := ""
		if cfg.FeedbackByRegion != nil && gotRegion != "" {
			fb = cfg.FeedbackByRegion[gotRegion]
		}
		perItem[id] = PerItemResult{Correct: okItem, Feedback: fb}
		if okItem {
			correctIDs = append(correctIDs, id)
		}
	}

	sort.Strings(correctIDs)
	n := float64(len(itemIDs))
	if n == 0 {
		return GradeResult{
			PerItem:    perItem,
			CorrectIDs: correctIDs,
			ScorePct:   0,
			ScoreRaw:   0,
			ScoreMax:   0,
		}
	}
	raw := float64(len(correctIDs))
	pct := math.Round((raw/n)*10000) / 100
	return GradeResult{
		PerItem:    perItem,
		CorrectIDs: correctIDs,
		ScorePct:   pct,
		ScoreRaw:   raw,
		ScoreMax:   n,
	}
}

// AllItemsAssigned reports whether every configured item has a region.
func AllItemsAssigned(cfg Config, assignments map[string]*string) bool {
	ids := ItemIDs(cfg)
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		v, has := assignments[id]
		if !has || v == nil || strings.TrimSpace(*v) == "" {
			return false
		}
	}
	return true
}

// ValidateAssignmentIDs ensures assignments only reference known items/regions.
func ValidateAssignmentIDs(cfg Config, assignments map[string]*string) error {
	knownItems := map[string]struct{}{}
	for _, id := range ItemIDs(cfg) {
		knownItems[id] = struct{}{}
	}
	regions := RegionIDs(cfg)
	for itemID, region := range assignments {
		if _, ok := knownItems[itemID]; !ok {
			return errUnknownItem
		}
		if region == nil {
			continue
		}
		if _, ok := regions[*region]; !ok {
			return errUnknownRegion
		}
	}
	return nil
}

// ValidateConfigForAuthoring checks required descriptions and alt text.
func ValidateConfigForAuthoring(cfg Config) error {
	if strings.TrimSpace(cfg.Image.URL) == "" {
		return errMissingImage
	}
	if strings.TrimSpace(cfg.Image.Alt) == "" {
		return errMissingAlt
	}
	if len(cfg.Regions) == 0 {
		return errNoRegions
	}
	for _, r := range cfg.Regions {
		if strings.TrimSpace(r.Label) == "" {
			return errMissingRegionLabel
		}
		if strings.TrimSpace(r.Description) == "" {
			return errMissingDescription
		}
		if !validShape(r.Shape) {
			return errInvalidShape
		}
	}
	switch cfg.Mode {
	case ModeHotspot:
		if len(cfg.Prompts) == 0 {
			return errNoPrompts
		}
	default:
		if len(cfg.Labels) == 0 {
			return errNoLabels
		}
	}
	return nil
}

func validShape(s Shape) bool {
	switch s.Kind {
	case "rect":
		return s.W > 0 && s.H > 0
	case "circle":
		return s.R > 0
	case "polygon":
		return len(s.Points) >= 3
	default:
		return false
	}
}

var (
	errUnknownItem         = &assignmentError{msg: "unknown item id"}
	errUnknownRegion       = &assignmentError{msg: "unknown region id"}
	errMissingImage        = &assignmentError{msg: "image url is required"}
	errMissingAlt          = &assignmentError{msg: "image alt text is required"}
	errNoRegions           = &assignmentError{msg: "at least one region is required"}
	errMissingRegionLabel  = &assignmentError{msg: "region label is required"}
	errMissingDescription  = &assignmentError{msg: "region description is required"}
	errInvalidShape        = &assignmentError{msg: "invalid region shape"}
	errNoLabels            = &assignmentError{msg: "at least one label is required"}
	errNoPrompts           = &assignmentError{msg: "at least one prompt is required"}
)

type assignmentError struct{ msg string }

func (e *assignmentError) Error() string { return e.msg }
