package coursestructure

// GradebookMaxPoints is the gradebook "out of" value for an assignment or quiz: explicit points worth
// first, then (for non-adaptive quizzes) the summed question points. nil means the item has no
// gradable maximum. The gradebook, My grades, and the student progress average grade share this rule.
func GradebookMaxPoints(item *ItemResponse) *int {
	if item.PointsWorth != nil {
		v := *item.PointsWorth
		return &v
	}
	if item.Kind == "quiz" && (item.IsAdaptive == nil || !*item.IsAdaptive) {
		if item.PointsPossible != nil {
			v := *item.PointsPossible
			return &v
		}
	}
	return nil
}
