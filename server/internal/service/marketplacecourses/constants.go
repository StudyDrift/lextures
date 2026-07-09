// Package marketplacecourses provisions first-party free marketplace courses from embedded content (MC0).
package marketplacecourses

import mcrepo "github.com/lextures/lextures/server/internal/repos/marketplacecourses"

const (
	GradePolicyQuizAutoscore  = "quiz_autoscore"
	GradePolicyCompletionFull = "completion_full"
	GradePolicyGraderAgent    = "grader_agent"

	defaultLocale = "en"
)

// SystemPublisherID is the platform publisher for official courses.
var SystemPublisherID = mcrepo.SystemPublisherID
