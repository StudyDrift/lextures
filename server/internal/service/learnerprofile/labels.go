package learnerprofile

// ResolveLabel maps a stored i18n key to a display string for the request locale.
// v1 resolves English defaults; LP07 may add full locale catalogs.
func ResolveLabel(locale, i18nKey string) string {
	_ = locale
	if label, ok := defaultLabels[i18nKey]; ok {
		return label
	}
	return i18nKey
}

var defaultLabels = map[string]string{
	"learner_profile.study_rhythm.peak_study_window": "When you study most",
	"learner_profile.study_rhythm.study_consistency": "Study consistency",
	"learner_profile.study_rhythm.study_streak":      "Study streak",
	"learner_profile.study_rhythm.session_shape":     "Typical session length",

	"learner_profile.content_modality.modality_affinity":  "How you engage with each format",
	"learner_profile.content_modality.complexity_comfort": "Reading level comfort band",
	"learner_profile.content_modality.content_pacing":     "How thoroughly you work through content",

	"learner_profile.strengths_growth.top_strengths": "Your top strengths",
	"learner_profile.strengths_growth.growth_areas":  "Areas to grow",
	"learner_profile.strengths_growth.needs_review":  "Concepts to review",

	"learner_profile.interests.topic": "Topic you're drawn to",

	"learner_profile.learning_approach.persistence":   "How you persist through challenges",
	"learner_profile.learning_approach.help_seeking":  "How you seek help",
	"learner_profile.learning_approach.consolidation": "How you consolidate learning",
}