package platformconfig

// Defaults for platform boolean settings when the database column is NULL.
// Secrets and integration endpoints still come from process environment.
type Defaults struct {
	BlindGradingEnabled         bool
	GradePostingPoliciesEnabled bool
	MagicLinkEnabled            bool
	VirtualClassroomEnabled     bool
	AdminAuditLogEnabled        bool
	AiDisclosureEnabled         bool
	LearnerModelEMAAlpha        float64
}

// DefaultDefaults matches prior product defaults for unset platform boolean columns.
func DefaultDefaults() Defaults {
	return Defaults{
		BlindGradingEnabled:         true,
		GradePostingPoliciesEnabled: true,
		MagicLinkEnabled:            true,
		VirtualClassroomEnabled:     true,
		AdminAuditLogEnabled:        true, // plan 10.11 default on; disable via platform settings
		AiDisclosureEnabled:         true, // default on; disable via platform settings
		LearnerModelEMAAlpha:        0.3,  // matches prior LEARNER_MODEL_EMA_ALPHA default
	}
}

func mergeBool(db *bool, whenUnset bool) bool {
	if db != nil {
		return *db
	}
	return whenUnset
}

// mergeFloat64 returns the DB value when set and valid (in (0,1]); otherwise the default.
func mergeFloat64(db *float64, whenUnset float64) float64 {
	if db != nil && *db > 0 && *db <= 1 {
		return *db
	}
	return whenUnset
}
