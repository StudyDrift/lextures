package platformconfig

// Defaults for platform boolean settings when the database column is NULL.
// Secrets and integration endpoints still come from process environment.
type Defaults struct {
	BlindGradingEnabled         bool
	GradePostingPoliciesEnabled bool
	MagicLinkEnabled            bool
	VirtualClassroomEnabled     bool
}

// DefaultDefaults matches prior env defaults (boolEnvDefaultTrue fields).
func DefaultDefaults() Defaults {
	return Defaults{
		BlindGradingEnabled:         true,
		GradePostingPoliciesEnabled: true,
		MagicLinkEnabled:            true,
		VirtualClassroomEnabled:     true,
	}
}

func mergeBool(db *bool, whenUnset bool) bool {
	if db != nil {
		return *db
	}
	return whenUnset
}
