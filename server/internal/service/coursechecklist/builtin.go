package coursechecklist

// BuildBuiltinRegistry returns the CC.3 foundation packs plus CC.4 structure/outcomes
// packs (55 recommended items). Reference rules from CC.1 remain helpers for engine tests only.
func BuildBuiltinRegistry() (*Registry, error) {
	items := []ItemDescriptor{}
	items = append(items, foundationsRules()...)
	items = append(items, orientationRules()...)
	items = append(items, syllabusRules()...)
	items = append(items, peopleRules()...)
	items = append(items, structureRules()...)
	items = append(items, outcomesRules()...)
	return NewRegistry(items)
}
