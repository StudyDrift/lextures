package coursechecklist

// BuildBuiltinRegistry returns the CC.3 foundation rule packs (33 recommended items).
// Reference rules from CC.1 are kept as helpers for engine unit tests only.
func BuildBuiltinRegistry() (*Registry, error) {
	items := []ItemDescriptor{}
	items = append(items, foundationsRules()...)
	items = append(items, orientationRules()...)
	items = append(items, syllabusRules()...)
	items = append(items, peopleRules()...)
	return NewRegistry(items)
}
