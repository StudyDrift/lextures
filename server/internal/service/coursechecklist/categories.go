package coursechecklist

// CategoryMeta provides i18n keys and English defaults for category headings.
type CategoryMeta struct {
	TitleKey string
	Title    string
}

// categoryMetaByID is the display metadata for CategoryOrder entries.
var categoryMetaByID = map[CategoryID]CategoryMeta{
	CategoryFoundations:   {TitleKey: "coursechecklist.category.foundations", Title: "Foundations"},
	CategoryOrientation:   {TitleKey: "coursechecklist.category.orientation", Title: "Orientation"},
	CategoryStructure:     {TitleKey: "coursechecklist.category.structure", Title: "Structure"},
	CategoryOutcomes:      {TitleKey: "coursechecklist.category.outcomes", Title: "Outcomes"},
	CategoryAssessment:    {TitleKey: "coursechecklist.category.assessment", Title: "Assessment"},
	CategoryFeedback:      {TitleKey: "coursechecklist.category.feedback", Title: "Feedback"},
	CategoryAccessibility: {TitleKey: "coursechecklist.category.accessibility", Title: "Accessibility"},
	CategoryLaunch:        {TitleKey: "coursechecklist.category.launch", Title: "Launch readiness"},
	CategoryReference:     {TitleKey: "coursechecklist.category.reference", Title: "Reference"},
}

// CategoryTitle returns title key + English default for a category.
func CategoryTitle(id CategoryID) CategoryMeta {
	if m, ok := categoryMetaByID[id]; ok {
		return m
	}
	return CategoryMeta{
		TitleKey: "coursechecklist.category." + string(id),
		Title:    string(id),
	}
}
