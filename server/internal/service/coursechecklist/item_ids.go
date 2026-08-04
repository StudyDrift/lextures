package coursechecklist

// Stable checklist item IDs for CC.3 foundation packs (FR-1–FR-33).
const (
	// A1 — Course identity & configuration
	ItemCourseTitleAndDescription ItemID = "course.title-and-description"
	ItemCourseDates               ItemID = "course.dates"
	ItemCourseTimezone            ItemID = "course.timezone"
	ItemCoursePublished           ItemID = "course.published"
	ItemCourseRelativeSchedule    ItemID = "course.relative-schedule"
	ItemCourseVisibilityWindow    ItemID = "course.visibility-window"
	ItemCourseGradingScheme       ItemID = "course.grading-scheme"
	ItemCourseHeroImage           ItemID = "course.hero-image"
	ItemCourseHomeLanding         ItemID = "course.home-landing"
	ItemCourseFeaturesReviewed    ItemID = "course.features-reviewed"
	ItemCourseLanguage            ItemID = "course.language"

	// A2 — Learner orientation
	ItemOrientationWelcomeMessage            ItemID = "orientation.welcome-message"
	ItemOrientationStartHere                 ItemID = "orientation.start-here"
	ItemOrientationInstructorContact         ItemID = "orientation.instructor-contact"
	ItemOrientationResponseTime              ItemID = "orientation.response-time"
	ItemOrientationParticipationExpectations ItemID = "orientation.participation-expectations"
	ItemOrientationNetiquette                ItemID = "orientation.netiquette"
	ItemOrientationTechRequirements          ItemID = "orientation.tech-requirements"
	ItemOrientationSupportResources          ItemID = "orientation.support-resources"
	ItemOrientationInstructorIntroduction    ItemID = "orientation.instructor-introduction"
	ItemOrientationLearnerIntroductions      ItemID = "orientation.learner-introductions"

	// A3 — Syllabus & published policies
	ItemSyllabusExists                 ItemID = "syllabus.exists"
	ItemSyllabusGradingPolicy          ItemID = "syllabus.grading-policy"
	ItemSyllabusLatePolicy             ItemID = "syllabus.late-policy"
	ItemSyllabusAcademicIntegrity      ItemID = "syllabus.academic-integrity"
	ItemSyllabusAccessibilityStatement ItemID = "syllabus.accessibility-statement"
	ItemSyllabusAcceptanceDecision     ItemID = "syllabus.acceptance-decision"
	ItemSyllabusPrintable              ItemID = "syllabus.printable"

	// A4 — People & enrollment
	ItemPeopleStudentsEnrolled ItemID = "people.students-enrolled"
	ItemPeopleStaffRoles       ItemID = "people.staff-roles"
	ItemPeopleSections         ItemID = "people.sections"
	ItemPeopleStaleInvitations ItemID = "people.stale-invitations"
	ItemPeopleGuardianLinks    ItemID = "people.guardian-links"

	// Retired CC.1 reference ID (replaced by people.sections).
	ItemCourseSections ItemID = "course.sections"

	// B1 — Course structure hygiene (CC.4)
	ItemStructureModulesExist        ItemID = "structure.modules-exist"
	ItemStructureEmptyModules        ItemID = "structure.empty-modules"
	ItemStructurePlaceholderTitles   ItemID = "structure.placeholder-titles"
	ItemStructureModuleOverviews     ItemID = "structure.module-overviews"
	ItemStructureUnpublishedItems    ItemID = "structure.unpublished-items"
	ItemStructureOrphanItems         ItemID = "structure.orphan-items"
	ItemStructurePacingSignal        ItemID = "structure.pacing-signal"
	ItemStructureContentVariety      ItemID = "structure.content-variety"
	ItemStructureInteractiveElements ItemID = "structure.interactive-elements"
	ItemStructureAttribution         ItemID = "structure.attribution"
	ItemStructureFileReferences      ItemID = "structure.file-references"
	ItemStructureGatingReview        ItemID = "structure.gating-review"

	// B2 — Learning outcomes (CC.4)
	ItemOutcomesDefined            ItemID = "outcomes.defined"
	ItemOutcomesMeasurable         ItemID = "outcomes.measurable"
	ItemOutcomesDescribed          ItemID = "outcomes.described"
	ItemOutcomesModuleAlignment    ItemID = "outcomes.module-alignment"
	ItemOutcomesAssessmentMapping  ItemID = "outcomes.assessment-mapping"
	ItemOutcomesCoverage           ItemID = "outcomes.coverage"
	ItemOutcomesSummativeCoverage  ItemID = "outcomes.summative-coverage"
	ItemOutcomesMasteryScale       ItemID = "outcomes.mastery-scale"
	ItemOutcomesStandardsAlignment ItemID = "outcomes.standards-alignment"
	ItemOutcomesSyllabusPublished  ItemID = "outcomes.syllabus-published"
)
