package mail

import "html/template"

func renderCoachingTip(vars map[string]string, logo, color string) (RenderedEmail, error) {
	tip := vars["tipText"]
	link := vars["link"]
	subject := vars["subject"]
	if subject == "" {
		subject = "Your weekly study coaching tip"
	}
	plain := tip + "\n\nView your study insights: " + link
	body := template.HTMLEscapeString(tip)
	linkHTML := template.HTMLEscapeString(link)
	html, err := renderLayout("Weekly study tip", `
<p style="font-size:16px;line-height:1.5;">`+body+`</p>
<p style="margin-top:24px;"><a href="`+linkHTML+`" style="color:`+template.HTMLEscapeString(color)+`;font-weight:600;">Open study insights</a></p>
`, logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: plain, HTMLBody: html}, nil
}
