package coursechecklist

import "strings"

// hideCampusRubricSources is true for Homeschool / K–12 audiences where QM and
// OSCQR codes are opaque to parents and teachers.
func hideCampusRubricSources(snap CourseSnapshot) bool {
	return snap.HomeschoolMode || snap.OrgIsK12
}

// isCampusRubricSource reports whether a checklist source tag is a college
// quality-rubric code (Quality Matters or OSCQR).
func isCampusRubricSource(src string) bool {
	s := strings.TrimSpace(src)
	if s == "" {
		return false
	}
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "QM ") || upper == "QM" || strings.HasPrefix(upper, "QM.") {
		return true
	}
	if strings.HasPrefix(upper, "OSCQR ") || upper == "OSCQR" || strings.HasPrefix(upper, "OSCQR.") {
		return true
	}
	return false
}

// sourcesForAudience returns checklist source tags appropriate for the course
// audience. Campus QM/OSCQR codes are omitted for Homeschool / K–12; WCAG, UDL,
// NSQ, and Product tags are kept.
func sourcesForAudience(sources []string, snap CourseSnapshot) []string {
	if len(sources) == 0 || !hideCampusRubricSources(snap) {
		if sources == nil {
			return []string{}
		}
		return append([]string(nil), sources...)
	}
	out := make([]string, 0, len(sources))
	for _, src := range sources {
		if isCampusRubricSource(src) {
			continue
		}
		out = append(out, src)
	}
	return out
}
