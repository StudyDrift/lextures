package diagnostic

// ActiveForCourse combines platform, course, and configuration flags.
func ActiveForCourse(globalOn, courseFlag, hasConfig bool) bool {
	return globalOn && courseFlag && hasConfig
}
