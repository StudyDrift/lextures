package mail

import (
	"fmt"
	"html/template"
)

func renderSubmissionComment(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	assignment := vars["assignmentName"]
	excerpt := vars["excerpt"]
	link := vars["link"]
	subject := vars["subject"]
	if subject == "" {
		subject = fmt.Sprintf("New comment on %s", assignment)
	}
	plain := fmt.Sprintf(`New feedback on "%s" in %s.

%s

Open the feedback: %s
`, assignment, course, excerpt, link)
	excerptHTML := ""
	if excerpt != "" {
		excerptHTML = fmt.Sprintf(`<p style="margin-top:12px;">%s</p>`, template.HTMLEscapeString(excerpt))
	}
	html, err := renderLayout("New comment on your submission", fmt.Sprintf(
		`<p>New feedback on <strong>%s</strong> in <strong>%s</strong>.</p>
%s
<p><a href="%s" style="color:%s;font-weight:600;">Open the feedback</a></p>`,
		template.HTMLEscapeString(assignment),
		template.HTMLEscapeString(course),
		excerptHTML,
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: plain, HTMLBody: html}, nil
}

func renderParentWeeklyProgress(vars map[string]string, logo, color string) (RenderedEmail, error) {
	subject := vars["subject"]
	if subject == "" {
		subject = "Weekly progress"
	}
	bodyText := vars["bodyText"]
	bodyHTML := vars["bodyHtml"]
	link := vars["link"]
	if link != "" {
		bodyHTML += fmt.Sprintf(`<p style="margin-top:24px;"><a href="%s" style="color:%s;font-weight:600;">Open Lextures</a></p>`,
			template.HTMLEscapeString(link), color)
		if bodyText != "" {
			bodyText += "\n\nOpen Lextures: " + link + "\n"
		}
	}
	html, err := renderLayout("Weekly progress", bodyHTML, logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}
