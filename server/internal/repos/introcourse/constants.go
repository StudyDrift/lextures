package introcourse

import "github.com/google/uuid"

const (
	// ShortCode is the immutable idempotency key for the canonical intro course.
	ShortCode = "LEX-WELCOME"
	// CourseCode is the stable URL identifier (must match courses_course_code_format).
	CourseCode = "C-WLCOME"
	// Title is the default English course title seeded at provision time.
	Title = "Welcome to Lextures"
	// Description is the default English course description seeded at provision time.
	Description = "A guided introduction to Lextures: navigation, learning activities, and where to get help."
)

// SystemUserID is the dedicated non-login instructor for the intro course (IC01).
var SystemUserID = uuid.MustParse("a0000000-0000-4000-8000-000000000002")