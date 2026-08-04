package coursechecklist

// BuildBuiltinRegistry returns the CC.1 registry with reference rules only.
// Real rule packs land in CC.3–CC.6 as rules_*.go files.
func BuildBuiltinRegistry() (*Registry, error) {
	return NewRegistry([]ItemDescriptor{
		referenceCourseDates(),
		referenceCourseSections(),
	})
}
