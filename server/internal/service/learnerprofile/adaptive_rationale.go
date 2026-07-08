package learnerprofile

import (
	"fmt"
	"strings"
)

// RationaleForInterest returns a profile rationale when a recommendation matches an interest topic.
func RationaleForInterest(topic string) ProfileRationale {
	text := fmt.Sprintf("Personalised because you're drawn to %s", topic)
	return ProfileRationale{
		Text:       text,
		FacetKey:   "interests",
		InsightKey: "topic",
	}
}

// RationaleForGrowth returns a profile rationale when content matches a growth area.
func RationaleForGrowth(concept string) ProfileRationale {
	text := fmt.Sprintf("Personalised because %s is an area to grow for you", concept)
	return ProfileRationale{
		Text:       text,
		FacetKey:   "strengths_growth",
		InsightKey: "growth_areas",
	}
}

// RationaleForNeedsReview returns a profile rationale when review targets a decayed concept.
func RationaleForNeedsReview(concept string) ProfileRationale {
	text := fmt.Sprintf("Personalised because %s is due for review", concept)
	return ProfileRationale{
		Text:       text,
		FacetKey:   "strengths_growth",
		InsightKey: "needs_review",
	}
}

// RationaleForPeakWindow returns a profile rationale tied to study rhythm timing.
func RationaleForPeakWindow(window PeakWindowFacet, inWindow bool) ProfileRationale {
	var text string
	if inWindow {
		text = fmt.Sprintf("Personalised because this is your usual study time (%s %s)", window.Dow, window.HourBucket)
	} else {
		text = fmt.Sprintf("Personalised because you often study during %s %s — review now or save for then", window.Dow, window.HourBucket)
	}
	return ProfileRationale{
		Text:       text,
		FacetKey:   "study_rhythm",
		InsightKey: "peak_study_window",
	}
}

// RationaleForModality returns a profile rationale when preferring a content format.
func RationaleForModality(modality string) ProfileRationale {
	mod := strings.TrimSpace(modality)
	if mod == "" {
		mod = "this format"
	}
	text := fmt.Sprintf("Personalised because you engage more with %s content", mod)
	return ProfileRationale{
		Text:       text,
		FacetKey:   "content_modality",
		InsightKey: "modality_affinity",
	}
}

// RationaleForHelpSeeking returns a profile rationale for tutor scaffolding adjustments.
func RationaleForHelpSeeking(style string) ProfileRationale {
	var text string
	switch style {
	case "early-reliance":
		text = "Personalised because you often reach for hints early — I'll offer a nudge before a full hint"
	case "independent":
		text = "Personalised because you tend to work independently — I'll wait before offering hints"
	default:
		text = "Personalised because of how you seek help when stuck"
	}
	return ProfileRationale{
		Text:       text,
		FacetKey:   "learning_approach",
		InsightKey: "help_seeking",
	}
}

// MatchConcept returns the first concept name found in haystack (case-insensitive substring).
func MatchConcept(haystack string, concepts []string) string {
	lower := strings.ToLower(haystack)
	for _, c := range concepts {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(c)) {
			return c
		}
	}
	return ""
}

// MatchTopic returns the first interest topic found in haystack.
func MatchTopic(haystack string, topics []string) string {
	return MatchConcept(haystack, topics)
}