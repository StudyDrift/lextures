package starterlibrary

import "fmt"

// seed is the curated fact sheet for one starter course.
type seed struct {
	slug      string
	title     string
	band      string
	subject   string
	minutes   int
	standards []string
	summary   string
	goal      string
	detail    string
	talk      string
	try       string
	check     string
	unit      string
	practice  string
}

func (s seed) course() Course {
	return Course{
		Slug:             s.slug,
		Title:            s.title,
		Summary:          s.summary,
		GradeBand:        s.band,
		Subject:          s.subject,
		EstimatedMinutes: s.minutes,
		Standards:        append([]string(nil), s.standards...),
		Modules: []Module{
			{
				Title: s.unit,
				Pages: []Page{
					{Title: "Start here", Markdown: startHere(s)},
					{Title: "A closer look", Markdown: closerLook(s)},
				},
			},
			{
				Title: s.practice,
				Pages: []Page{
					{Title: "Try it", Markdown: tryIt(s)},
					{Title: "Check what stuck", Markdown: checkIt(s)},
				},
			},
		},
	}
}

func startHere(s seed) string {
	return fmt.Sprintf(`%s

**In this unit you will**
- %s
- Edit any example so it matches your learner

This is a starter copy. Change the wording, add a page, or skip a part that you already taught.`, s.summary, s.goal)
}

func closerLook(s seed) string {
	return fmt.Sprintf(`%s

**Talk about it**

%s`, s.detail, s.talk)
}

func tryIt(s seed) string {
	return fmt.Sprintf(`**Do this together**

%s

Write what you notice in a sentence or two. A drawing or a short list is enough.`, s.try)
}

func checkIt(s seed) string {
	return fmt.Sprintf(`Answer without looking back, then check with a parent.

1. %s

If it was hard, redo the try-it with a smaller example. That is part of learning, not a failed day.`, s.check)
}
