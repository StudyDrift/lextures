package coursechecklist

// DataNeed declares which snapshot slices a rule depends on. LoadSnapshot uses
// the union of DataNeeds across selected rules to prune queries in Only mode (FR-8 / AC-5).
type DataNeed string

const (
	DataNeedCourse              DataNeed = "course"               // course row + feature columns + dates
	DataNeedStructure           DataNeed = "structure"            // modules / headings / items
	DataNeedItemMeta            DataNeed = "item_meta"            // content/assignment/quiz/survey/link metadata
	DataNeedSyllabus            DataNeed = "syllabus"             // syllabus sections
	DataNeedOutcomes            DataNeed = "outcomes"             // learning outcomes + links
	DataNeedGrading             DataNeed = "grading"              // assignment groups + grading scheme
	DataNeedEnrollments         DataNeed = "enrollments"          // role aggregates (+ privacy-safe roster stubs)
	DataNeedFeed                DataNeed = "feed"                 // channels + latest message per channel
	DataNeedFiles               DataNeed = "files"                // course file metadata
	DataNeedSections            DataNeed = "sections"             // course sections
	DataNeedAccommodations      DataNeed = "accommodations"       // accommodation counts (+ type aggregates, CC.5)
	DataNeedStandards           DataNeed = "standards"            // K-12 standards + item alignments (optional)
	DataNeedContentTools        DataNeed = "content_tool_counts"  // content tool instance item IDs
	DataNeedModulePrerequisites DataNeed = "module_prerequisites" // gating prerequisite edges
	// CC.5 assessment / interaction slices
	DataNeedAssessmentItems   DataNeed = "assessment_items"    // assignment/quiz availability, rubrics, quiz settings
	DataNeedPeerReview        DataNeed = "peer_review_configs" // peer-review configs
	DataNeedDiscussions       DataNeed = "discussions"         // forums / structured discussions
	DataNeedOfficeHours       DataNeed = "office_hours"        // future appointment slots
	DataNeedEnrollmentGroups  DataNeed = "enrollment_groups"   // group sets + unassigned student count
	DataNeedAnnouncementCadence DataNeed = "announcement_cadence" // announcement timestamps for presence plan
)

// AllDataNeeds is the full set loaded for a complete evaluation.
var AllDataNeeds = []DataNeed{
	DataNeedCourse,
	DataNeedStructure,
	DataNeedItemMeta,
	DataNeedSyllabus,
	DataNeedOutcomes,
	DataNeedGrading,
	DataNeedEnrollments,
	DataNeedFeed,
	DataNeedFiles,
	DataNeedSections,
	DataNeedAccommodations,
	DataNeedStandards,
	DataNeedContentTools,
	DataNeedModulePrerequisites,
	DataNeedAssessmentItems,
	DataNeedPeerReview,
	DataNeedDiscussions,
	DataNeedOfficeHours,
	DataNeedEnrollmentGroups,
	DataNeedAnnouncementCadence,
}

func unionDataNeeds(items []ItemDescriptor) []DataNeed {
	seen := make(map[DataNeed]struct{}, len(AllDataNeeds))
	var out []DataNeed
	for _, it := range items {
		for _, n := range it.DataNeeds {
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	// Course row is always required for Applies predicates.
	if _, ok := seen[DataNeedCourse]; !ok {
		out = append([]DataNeed{DataNeedCourse}, out...)
	}
	return out
}

func hasDataNeed(needs []DataNeed, want DataNeed) bool {
	for _, n := range needs {
		if n == want {
			return true
		}
	}
	return false
}
