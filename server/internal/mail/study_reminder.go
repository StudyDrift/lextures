package mail

import (
	"fmt"
	"html/template"
)

func renderStudyReminderDaily(vars map[string]string, logo, color string) (RenderedEmail, error) {
	goal := vars["dailyGoal"]
	link := vars["link"]
	subject := vars["subject"]
	if subject == "" {
		subject = "Time for your daily study session"
	}
	plain := fmt.Sprintf(`You haven't studied yet today. Your daily goal is %s minutes.

Open your dashboard: %s`, goal, link)
	html, err := renderLayout("Daily study reminder", `
<p style="font-size:16px;line-height:1.5;">You haven&apos;t studied yet today. Your daily goal is <strong>`+template.HTMLEscapeString(goal)+` minutes</strong>.</p>
<p style="margin-top:24px;"><a href="`+template.HTMLEscapeString(link)+`" style="color:`+template.HTMLEscapeString(color)+`;font-weight:600;">Start studying</a></p>
`, logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: plain, HTMLBody: html}, nil
}

func renderStudyReminderStreakAtRisk(vars map[string]string, logo, color string) (RenderedEmail, error) {
	streak := vars["streak"]
	link := vars["link"]
	subject := vars["subject"]
	if subject == "" {
		subject = fmt.Sprintf("Your %s-day streak is at risk", streak)
	}
	plain := fmt.Sprintf(`Your %s-day learning streak will end if you don't study today.

Keep your streak: %s`, streak, link)
	html, err := renderLayout("Streak at risk", `
<p style="font-size:16px;line-height:1.5;">Your <strong>`+template.HTMLEscapeString(streak)+`-day</strong> learning streak will end if you don&apos;t study today.</p>
<p style="margin-top:24px;"><a href="`+template.HTMLEscapeString(link)+`" style="color:`+template.HTMLEscapeString(color)+`;font-weight:600;">Study now</a></p>
`, logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: plain, HTMLBody: html}, nil
}

func renderStudyReminderWeeklySummary(vars map[string]string, logo, color string) (RenderedEmail, error) {
	streak := vars["streak"]
	xp := vars["xpEarned"]
	coursesHTML := vars["coursesHtml"]
	coursesText := vars["coursesText"]
	link := vars["link"]
	subject := vars["subject"]
	if subject == "" {
		subject = "Your weekly learning summary"
	}
	plain := fmt.Sprintf(`Your week in review:

Streak: %s days
XP earned: %s

Courses in progress:
%s

View dashboard: %s`, streak, xp, coursesText, link)
	body := fmt.Sprintf(`
<p style="font-size:16px;line-height:1.5;">Your week in review:</p>
<ul style="font-size:16px;line-height:1.6;padding-left:20px;">
<li><strong>Streak:</strong> %s days</li>
<li><strong>XP earned:</strong> %s</li>
</ul>
<h2 style="font-size:18px;margin-top:24px;">Courses in progress</h2>
%s
<p style="margin-top:24px;"><a href="%s" style="color:%s;font-weight:600;">Open dashboard</a></p>
`, template.HTMLEscapeString(streak), template.HTMLEscapeString(xp), coursesHTML, template.HTMLEscapeString(link), template.HTMLEscapeString(color))
	html, err := renderLayout("Weekly learning summary", body, logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: plain, HTMLBody: html}, nil
}
