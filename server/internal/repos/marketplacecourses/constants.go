// Package marketplacecourses provides persistence helpers for official marketplace course provisioning (MC0).
package marketplacecourses

import "github.com/google/uuid"

const (
	// SystemPublisherEmail is the non-login publisher identity for official courses.
	SystemPublisherEmail = "publisher@system.lextures.invalid"
	// SystemPublisherName is the display name for the platform publisher.
	SystemPublisherName = "Lextures Official"
)

// SystemPublisherID is the dedicated non-login instructor for official marketplace courses (MC0).
var SystemPublisherID = uuid.MustParse("a0000000-0000-4000-8000-000000000003")
