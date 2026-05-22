package mail

import (
	"fmt"
	"html/template"
)

func renderAtRiskAlert(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	student := vars["studentName"]
	score := vars["score"]
	factor := vars["topFactor"]
	link := vars["link"]
	subject := fmt.Sprintf("At-risk alert: %s in %s", student, course)
	bodyText := fmt.Sprintf(`A student may need support in %s.

Student: %s
At-risk score: %s
Primary concern: %s

Review at-risk students: %s
`, course, student, score, factor, link)
	html, err := renderLayout("At-risk student alert", fmt.Sprintf(
		`<p><strong>%s</strong> may need support in <strong>%s</strong>.</p>
<p>At-risk score: <strong>%s</strong><br/>
Primary concern: %s</p>
<p><a href="%s" style="color:%s;font-weight:600;">Open at-risk dashboard</a></p>`,
		template.HTMLEscapeString(student),
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(score),
		template.HTMLEscapeString(factor),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}
