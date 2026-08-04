package coursechecklist

// computeGradableItems builds the shared gradable set (assignment/quiz, not archived,
// points > 0 when ItemMeta is present). Callers without ItemMeta include all assignment/quiz rows.
func computeGradableItems(snap CourseSnapshot) []GradableItem {
	var out []GradableItem
	for _, it := range snap.StructureItems {
		if it.Archived || !isGradableKind(it.Kind) {
			continue
		}
		meta, hasMeta := snap.ItemMeta[it.ID]
		var points *int
		if hasMeta {
			points = meta.PointsWorth
			// Open Q1: ungraded practice (points ≤ 0) is not "gradable" for mapping rules.
			if points != nil && *points <= 0 {
				continue
			}
		}
		out = append(out, GradableItem{
			ID:        it.ID,
			Title:     it.Title,
			Kind:      it.Kind,
			ParentID:  it.ParentID,
			SortOrder: it.SortOrder,
			Points:    points,
		})
	}
	return out
}

// GradableItemsFor returns the shared gradable set, computing it on demand for unit tests
// that build snapshots by hand without LoadSnapshot.
func GradableItemsFor(snap CourseSnapshot) []GradableItem {
	if snap.GradableComputed {
		return snap.GradableItems
	}
	return computeGradableItems(snap)
}
