package coursechecklist

// BuildBuiltinRegistry returns the CC.3–CC.6 rule packs (101 recommended items).
// Reference rules from CC.1 remain helpers for engine tests only.
func BuildBuiltinRegistry() (*Registry, error) {
	items := []ItemDescriptor{}
	items = append(items, foundationsRules()...)
	items = append(items, orientationRules()...)
	items = append(items, syllabusRules()...)
	items = append(items, peopleRules()...)
	items = append(items, structureRules()...)
	items = append(items, outcomesRules()...)
	items = append(items, assessmentRules()...)
	items = append(items, gradingRules()...)
	items = append(items, feedbackRules()...)
	items = append(items, interactionRules()...)
	items = append(items, a11yRules()...)
	items = append(items, udlRules()...)
	items = append(items, linkAndLaunchRules()...)
	return NewRegistry(items)
}
