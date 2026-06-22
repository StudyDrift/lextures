package coursecopy

// Include selects which parts of a source course are copied into a new course.
type Include struct {
	Modules     bool `json:"modules"`
	Assignments bool `json:"assignments"`
	Quizzes     bool `json:"quizzes"`
	Enrollments bool `json:"enrollments"`
	Grades      bool `json:"grades"`
	Settings    bool `json:"settings"`
	Files       bool `json:"files"`
}

// WithDefaults returns all categories enabled when every flag is false (matches Canvas import UX).
func (i Include) WithDefaults() Include {
	if i.Modules || i.Assignments || i.Quizzes || i.Enrollments || i.Grades || i.Settings || i.Files {
		return i
	}
	return Include{
		Modules:     true,
		Assignments: true,
		Quizzes:     true,
		Enrollments: true,
		Grades:      true,
		Settings:    true,
		Files:       true,
	}
}

func (i Include) wantsStructure() bool {
	return i.Modules || i.Assignments || i.Quizzes
}

func (i Include) shouldCopyKind(kind string) bool {
	switch kind {
	case "module", "heading":
		return i.Modules || i.Assignments || i.Quizzes
	case "content_page", "external_link", "lti_link", "survey", "vibe_activity":
		return i.Modules
	case "assignment":
		return i.Assignments
	case "quiz":
		return i.Quizzes
	default:
		return i.Modules
	}
}