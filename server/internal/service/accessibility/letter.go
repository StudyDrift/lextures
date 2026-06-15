package accessibility

import (
	"fmt"
	"sort"
	"strings"
)

// typeLabels are the externalised, instructor-facing accommodation names. They never
// reference the underlying disability or diagnosis (FR-4, ADA/Section 504 need-to-know).
var typeLabels = map[string]string{
	TypeExtendedTime15:  "1.5x Extended Time",
	TypeExtendedTime2:   "2.0x Extended Time",
	TypeSeparateTesting: "Separate Testing Environment",
	TypeAlternateFormat: "Alternate Format Materials",
	TypeScreenReader:    "Screen Reader Support",
	TypeSpeechToText:    "Speech-to-Text",
	TypeReducedDistract: "Reduced-Distraction Setting",
	TypeOther:           "Additional Accommodation",
}

// Label returns the instructor-facing label for an accommodation type.
func Label(t string) string {
	if l, ok := typeLabels[t]; ok {
		return l
	}
	return t
}

// Labels returns instructor-facing labels for a list of types, de-duplicated and sorted.
func Labels(types []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(types))
	for _, t := range types {
		l := Label(t)
		if _, dup := seen[l]; dup {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// LetterSubject is the instructor notification subject line.
func LetterSubject(studentName string) string {
	return fmt.Sprintf("Accommodation notice for %s", studentName)
}

// RenderLetter produces the FR-7 instructor notification letter body. It discloses only
// the accommodation labels and effective date — never the disability (AC-2). effectiveDate
// is a display string (e.g. "2026-06-14"); pass "" to omit.
func RenderLetter(studentName, effectiveDate string, types []string) string {
	labels := Labels(types)
	var b strings.Builder
	fmt.Fprintf(&b, "Dear Instructor,\n\n")
	fmt.Fprintf(&b, "The accessibility services office has approved the following accommodation(s) "+
		"for %s, a student enrolled in your course:\n\n", studentName)
	for _, l := range labels {
		fmt.Fprintf(&b, "  • %s\n", l)
	}
	if effectiveDate != "" {
		fmt.Fprintf(&b, "\nEffective date: %s\n", effectiveDate)
	}
	fmt.Fprintf(&b, "\nThese accommodations are applied automatically by the learning platform; no "+
		"action is required on your part. This notice intentionally does not disclose the nature of "+
		"the student's disability, in accordance with ADA, Section 504, and FERPA.\n\n")
	fmt.Fprintf(&b, "Please contact the accessibility services office with any questions.\n")
	return b.String()
}
