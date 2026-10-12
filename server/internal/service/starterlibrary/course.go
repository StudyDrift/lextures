// Package starterlibrary is the curated starter course catalog (issue 773).
// Families browse by grade band and subject, then copy a course into their account.
package starterlibrary

// Grade bands match course.grade_levels tokens.
const (
	BandK2  = "K-2"
	Band35  = "3-5"
	Band68  = "6-8"
	Band912 = "9-12"
)

// Subjects are the library filter values.
const (
	SubjectMath     = "math"
	SubjectELA      = "ela"
	SubjectScience  = "science"
	SubjectHistory  = "history"
	SubjectElective = "elective"
)

// GradeBands is the launch set, in display order.
var GradeBands = []string{BandK2, Band35, Band68, Band912}

// Subject is a filter facet.
type Subject struct {
	ID    string
	Label string
}

// Subjects is the launch set, in display order.
var Subjects = []Subject{
	{ID: SubjectMath, Label: "Math"},
	{ID: SubjectELA, Label: "Reading and writing"},
	{ID: SubjectScience, Label: "Science"},
	{ID: SubjectHistory, Label: "History"},
	{ID: SubjectElective, Label: "Electives"},
}

// Page is one editable content page inside a starter module.
type Page struct {
	Title    string
	Markdown string
}

// Module is a starter unit.
type Module struct {
	Title string
	Pages []Page
}

// Course is a curated starter outline.
type Course struct {
	Slug             string
	Title            string
	Summary          string
	GradeBand        string
	Subject          string
	EstimatedMinutes int
	Standards        []string
	Modules          []Module
}

// SubjectLabel returns the display name for a subject id.
func SubjectLabel(id string) string {
	for _, s := range Subjects {
		if s.ID == id {
			return s.Label
		}
	}
	return id
}

// ValidBand reports whether band is a launch grade band.
func ValidBand(band string) bool {
	for _, b := range GradeBands {
		if b == band {
			return true
		}
	}
	return false
}

// ValidSubject reports whether subject is a launch subject id.
func ValidSubject(subject string) bool {
	for _, s := range Subjects {
		if s.ID == subject {
			return true
		}
	}
	return false
}
