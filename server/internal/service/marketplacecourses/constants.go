// Package marketplacecourses provisions first-party free marketplace courses from embedded content (MC0).
package marketplacecourses

import mcrepo "github.com/lextures/lextures/server/internal/repos/marketplacecourses"

const (
	GradePolicyQuizAutoscore  = "quiz_autoscore"
	GradePolicyCompletionFull = "completion_full"
	GradePolicyGraderAgent    = "grader_agent"

	defaultLocale = "en"

	// HarnessSmokeDir is the MC0 pipeline smoke course; validated in CI but not
	// provisioned on deploy (not real catalog inventory).
	HarnessSmokeDir = "harness-smoke"
)

// SystemPublisherID is the platform publisher for official courses.
var SystemPublisherID = mcrepo.SystemPublisherID

// IsDeployCourse reports whether a content directory should be provisioned on API startup / deploy.
func IsDeployCourse(dirSlug string) bool {
	return dirSlug != HarnessSmokeDir
}
