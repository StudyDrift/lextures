package introcourse

import (
	"strings"

	"github.com/lextures/lextures/server/internal/config"
)

// KnownRequiresFlags lists valid requires_flag front-matter values for curriculum fixtures.
var KnownRequiresFlags = []string{
	"learner_profile_enabled",
	"canvas_import_enabled",
	"push_notifications_enabled",
	"adaptive_learner_model_enabled",
	"srs_practice_enabled",
	"diagnostic_assessments_enabled",
	"self_reflection_enabled",
	"ai_disclosure_enabled",
}

var knownRequiresFlagSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(KnownRequiresFlags))
	for _, f := range KnownRequiresFlags {
		m[f] = struct{}{}
	}
	return m
}()

// FlagEnabled reports whether a requires_flag value is active for cfg.
func FlagEnabled(cfg config.Config, flag string) bool {
	switch strings.TrimSpace(flag) {
	case "":
		return true
	case "learner_profile_enabled":
		return cfg.LearnerProfileEnabled
	case "canvas_import_enabled":
		return CanvasImportEnabled(cfg)
	case "push_notifications_enabled":
		return cfg.PushNotificationsEnabled
	case "adaptive_learner_model_enabled":
		return cfg.AdaptiveLearnerModelEnabled
	case "srs_practice_enabled":
		return cfg.SRSPracticeEnabled
	case "diagnostic_assessments_enabled":
		return cfg.DiagnosticAssessmentsEnabled
	case "self_reflection_enabled":
		return cfg.SelfReflectionEnabled
	case "ai_disclosure_enabled":
		return cfg.AiDisclosureEnabled
	default:
		return false
	}
}

// CanvasImportEnabled reports whether Canvas course import is available in this deployment.
// Import works with an in-process queue when RabbitMQ is unset (see AGENTS.md).
func CanvasImportEnabled(cfg config.Config) bool {
	_ = cfg
	return true
}

// IsKnownRequiresFlag reports whether flag is a recognized curriculum gate name.
func IsKnownRequiresFlag(flag string) bool {
	if strings.TrimSpace(flag) == "" {
		return true
	}
	_, ok := knownRequiresFlagSet[strings.TrimSpace(flag)]
	return ok
}