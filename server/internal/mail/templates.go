package mail

import (
	"fmt"
	"html/template"
	"strings"
)

// RenderedEmail is plain and HTML bodies for a template.
type RenderedEmail struct {
	Subject  string
	BodyText string
	HTMLBody string
}

// RenderTemplate renders a named notification template with vars.
func RenderTemplate(name string, vars map[string]string, branding *BrandingOpts) (RenderedEmail, error) {
	color := "#4F46E5"
	logo := ""
	display := "StudyDrift"
	if branding != nil {
		if strings.TrimSpace(branding.PrimaryColor) != "" {
			color = strings.TrimSpace(branding.PrimaryColor)
		}
		if branding.LogoURL != nil && strings.TrimSpace(*branding.LogoURL) != "" {
			logo = strings.TrimSpace(*branding.LogoURL)
		}
		if branding.FromDisplayName != nil && strings.TrimSpace(*branding.FromDisplayName) != "" {
			display = strings.TrimSpace(*branding.FromDisplayName)
		}
	}
	_ = display

	switch name {
	case "grade_posted":
		return renderGradePosted(vars, logo, color)
	case "assignment_created":
		return renderAssignmentCreated(vars, logo, color)
	case "discussion_reply":
		return renderDiscussionReply(vars, logo, color)
	case "password_reset":
		return renderPasswordResetTemplate(vars, logo, color)
	case "daily_digest":
		return renderDailyDigest(vars, logo, color)
	case "at_risk_alert":
		return renderAtRiskAlert(vars, logo, color)
	case "coaching_tip":
		return renderCoachingTip(vars, logo, color)
	case "conference_confirmed":
		return renderConferenceConfirmed(vars, logo, color)
	case "conference_cancelled":
		return renderConferenceCancelled(vars, logo, color)
	case "conference_reminder":
		return renderConferenceReminder(vars, logo, color)
	case "incomplete_granted":
		return renderIncompleteGranted(vars, logo, color)
	case "incomplete_reminder":
		return renderIncompleteReminder(vars, logo, color)
	case "payment_failed":
		return renderPaymentFailed(vars, logo, color)
	default:
		subject := vars["subject"]
		if subject == "" {
			subject = "Notification from StudyDrift"
		}
		body := vars["body"]
		return RenderedEmail{Subject: subject, BodyText: body, HTMLBody: ""}, nil
	}
}

func renderGradePosted(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	assignment := vars["assignmentName"]
	link := vars["link"]
	subject := fmt.Sprintf("Grade posted: %s — %s", course, assignment)
	bodyText := fmt.Sprintf(`Your grade has been posted for "%s" in %s.

View your grade: %s
`, assignment, course, link)
	html, err := renderLayout("Grade posted", fmt.Sprintf(
		`<p>Your grade has been posted for <strong>%s</strong> in <strong>%s</strong>.</p>
<p><a href="%s" style="color:%s;font-weight:600;">View your grade</a></p>`,
		template.HTMLEscapeString(assignment),
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderAssignmentCreated(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	assignment := vars["assignmentName"]
	link := vars["link"]
	subject := fmt.Sprintf("New assignment: %s — %s", assignment, course)
	bodyText := fmt.Sprintf(`A new assignment "%s" was added to %s.

Open assignment: %s
`, assignment, course, link)
	html, err := renderLayout("New assignment", fmt.Sprintf(
		`<p>A new assignment <strong>%s</strong> was added to <strong>%s</strong>.</p>
<p><a href="%s" style="color:%s;font-weight:600;">Open assignment</a></p>`,
		template.HTMLEscapeString(assignment),
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderDiscussionReply(vars map[string]string, logo, color string) (RenderedEmail, error) {
	thread := vars["threadTitle"]
	course := vars["courseName"]
	link := vars["link"]
	subject := fmt.Sprintf("New reply in %s — %s", thread, course)
	bodyText := fmt.Sprintf(`Someone replied in the discussion "%s" in %s.

View discussion: %s
`, thread, course, link)
	html, err := renderLayout("New discussion reply", fmt.Sprintf(
		`<p>Someone replied in <strong>%s</strong> in <strong>%s</strong>.</p>
<p><a href="%s" style="color:%s;font-weight:600;">View discussion</a></p>`,
		template.HTMLEscapeString(thread),
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderPasswordResetTemplate(vars map[string]string, logo, color string) (RenderedEmail, error) {
	resetURL := vars["resetUrl"]
	subject := "Reset your StudyDrift password"
	bodyText := fmt.Sprintf(`You requested a password reset.

Open this link to choose a new password (expires in one hour):

%s

If you did not request this, you can ignore this message.
`, resetURL)
	html, err := renderLayout("Password reset", fmt.Sprintf(
		`<p>You requested a password reset.</p>
<p><a href="%s" style="color:%s;font-weight:600;">Choose a new password</a> (expires in one hour).</p>
<p style="font-size:13px;color:#6b7280;">If you did not request this, you can ignore this message.</p>`,
		template.HTMLEscapeString(resetURL),
		color,
	), logo, "")
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderDailyDigest(vars map[string]string, logo, color string) (RenderedEmail, error) {
	lines := vars["lines"]
	subject := "Your daily StudyDrift summary"
	bodyText := "Here is your daily summary:\n\n" + lines + "\n"
	html, err := renderLayout("Daily summary", fmt.Sprintf(
		`<p>Here is your daily summary:</p><ul style="padding-left:20px;">%s</ul>`,
		vars["linesHtml"],
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	_ = color
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderConferenceConfirmed(vars map[string]string, logo, color string) (RenderedEmail, error) {
	when := vars["when"]
	summary := vars["summary"]
	location := vars["location"]
	subject := "Parent-teacher conference confirmed"
	bodyText := fmt.Sprintf(`Your parent-teacher conference is confirmed.

%s
When: %s
Location: %s

A calendar invite is attached to this email.
`, summary, when, location)
	html, err := renderLayout("Conference confirmed", fmt.Sprintf(
		`<p>Your parent-teacher conference is confirmed.</p>
<p><strong>%s</strong></p>
<p>When: %s<br/>Location: %s</p>
<p style="font-size:13px;color:#6b7280;">A calendar invite is attached to this email.</p>`,
		template.HTMLEscapeString(summary),
		template.HTMLEscapeString(when),
		template.HTMLEscapeString(location),
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	_ = color
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderConferenceCancelled(vars map[string]string, logo, color string) (RenderedEmail, error) {
	when := vars["when"]
	summary := vars["summary"]
	subject := "Parent-teacher conference cancelled"
	bodyText := fmt.Sprintf(`Your parent-teacher conference has been cancelled.

%s
Was scheduled for: %s
`, summary, when)
	html, err := renderLayout("Conference cancelled", fmt.Sprintf(
		`<p>Your parent-teacher conference has been cancelled.</p>
<p><strong>%s</strong></p>
<p>Was scheduled for: %s</p>`,
		template.HTMLEscapeString(summary),
		template.HTMLEscapeString(when),
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	_ = color
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderConferenceReminder(vars map[string]string, logo, color string) (RenderedEmail, error) {
	when := vars["when"]
	summary := vars["summary"]
	location := vars["location"]
	subject := "Reminder: parent-teacher conference tomorrow"
	bodyText := fmt.Sprintf(`Reminder: you have a parent-teacher conference tomorrow.

%s
When: %s
Location: %s
`, summary, when, location)
	html, err := renderLayout("Conference reminder", fmt.Sprintf(
		`<p>Reminder: you have a parent-teacher conference tomorrow.</p>
<p><strong>%s</strong></p>
<p>When: %s<br/>Location: %s</p>`,
		template.HTMLEscapeString(summary),
		template.HTMLEscapeString(when),
		template.HTMLEscapeString(location),
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	_ = color
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderIncompleteGranted(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	deadline := vars["extensionDeadline"]
	link := vars["link"]
	subject := fmt.Sprintf("Incomplete grade granted — %s", course)
	bodyText := fmt.Sprintf(`You have been granted an Incomplete grade for %s.

Extension deadline: %s

Open your course: %s
`, course, deadline, link)
	html, err := renderLayout("Incomplete grade granted", fmt.Sprintf(
		`<p>You have been granted an <strong>Incomplete</strong> grade for <strong>%s</strong>.</p>
<p>Extension deadline: <strong>%s</strong></p>
<p><a href="%s" style="color:%s;font-weight:600;">Open your course</a></p>`,
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(deadline),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderIncompleteReminder(vars map[string]string, logo, color string) (RenderedEmail, error) {
	course := vars["courseName"]
	student := vars["studentName"]
	deadline := vars["extensionDeadline"]
	days := vars["daysRemaining"]
	link := vars["link"]
	subject := fmt.Sprintf("Incomplete deadline in %s days — %s", days, course)
	bodyText := fmt.Sprintf(`Reminder: the Incomplete grade deadline is approaching.

Course: %s
Student: %s
Deadline: %s (%s days remaining)

View gradebook: %s
`, course, student, deadline, days, link)
	html, err := renderLayout("Incomplete deadline reminder", fmt.Sprintf(
		`<p>Reminder: the <strong>Incomplete</strong> grade deadline is approaching.</p>
<p>Course: <strong>%s</strong><br/>
Student: <strong>%s</strong><br/>
Deadline: <strong>%s</strong> (%s days remaining)</p>
<p><a href="%s" style="color:%s;font-weight:600;">View gradebook</a></p>`,
		template.HTMLEscapeString(course),
		template.HTMLEscapeString(student),
		template.HTMLEscapeString(deadline),
		template.HTMLEscapeString(days),
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderPaymentFailed(vars map[string]string, logo, color string) (RenderedEmail, error) {
	link := vars["billingUrl"]
	subject := "Payment failed — update your billing details"
	bodyText := fmt.Sprintf(`We could not process your latest subscription payment.

Update your payment method: %s
`, link)
	html, err := renderLayout("Payment failed", fmt.Sprintf(
		`<p>We could not process your latest subscription payment. Your access may be limited until billing is updated.</p>
<p><a href="%s" style="color:%s;font-weight:600;">Manage billing</a></p>`,
		template.HTMLEscapeString(link),
		color,
	), logo, vars["unsubscribeUrl"])
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderLayout(title, bodyHTML, logoURL, unsubscribeURL string) (string, error) {
	logoBlock := ""
	if logoURL != "" {
		logoBlock = fmt.Sprintf(`<div style="margin-bottom:16px;"><img src="%s" alt="" width="180" style="max-width:100%%;height:auto;" /></div>`, template.HTMLEscapeString(logoURL))
	}
	footer := ""
	if unsubscribeURL != "" {
		footer = fmt.Sprintf(`<p style="margin-top:24px;font-size:12px;color:#6b7280;">
<a href="%s" style="color:#6b7280;">Unsubscribe from this notification type</a>
</p>`, template.HTMLEscapeString(unsubscribeURL))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"/></head>
<body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;font-size:14px;">
%s
<h1 style="font-size:18px;margin:0 0 12px;">%s</h1>
%s
%s
</body></html>`, logoBlock, template.HTMLEscapeString(title), bodyHTML, footer), nil
}
