package bots

import (
	"fmt"
	"strings"
	"time"

	botsrepo "github.com/lextures/lextures/server/internal/repos/bots"
)

// EventPayload is the parsed webhook envelope data used for message rendering.
type EventPayload struct {
	EventType string
	CourseID  string
	CourseCode string
	Title     string
	DueAt     string
	URL       string
	StudentUserID string
	PointsEarned  float64
	Body      string
}

// SlackBlocks renders a Slack Block Kit message for an event.
func SlackBlocks(p EventPayload) map[string]any {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = eventTitle(p.EventType)
	}
	blocks := []map[string]any{
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s*", escapeSlack(title)),
			},
		},
	}
	if p.CourseCode != "" {
		blocks = append(blocks, map[string]any{
			"type": "context",
			"elements": []map[string]any{{
				"type": "mrkdwn",
				"text": fmt.Sprintf("Course: *%s*", escapeSlack(p.CourseCode)),
			}},
		})
	}
	if p.DueAt != "" {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": fmt.Sprintf("Due: %s", escapeSlack(p.DueAt)),
			},
		})
	}
	if p.Body != "" {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": escapeSlack(p.Body),
			},
		})
	}
	if p.URL != "" {
		blocks = append(blocks, map[string]any{
			"type": "actions",
			"elements": []map[string]any{{
				"type": "button",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  "View in Lextures",
					"emoji": true,
				},
				"url":   p.URL,
				"style": "primary",
			}},
		})
	}
	return map[string]any{
		"blocks":       blocks,
		"text":         title,
		"mrkdwn":       true,
		"accessibility": map[string]any{"alt_text": title},
	}
}

// DiscordEmbed renders a Discord rich embed.
func DiscordEmbed(p EventPayload) map[string]any {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = eventTitle(p.EventType)
	}
	fields := []map[string]any{}
	if p.CourseCode != "" {
		fields = append(fields, map[string]any{"name": "Course", "value": p.CourseCode, "inline": true})
	}
	if p.DueAt != "" {
		fields = append(fields, map[string]any{"name": "Due", "value": p.DueAt, "inline": true})
	}
	embed := map[string]any{
		"title":       title,
		"description": p.Body,
		"color":       0x4F46E5,
		"fields":      fields,
	}
	if p.URL != "" {
		embed["url"] = p.URL
	}
	return embed
}

// TeamsAdaptiveCard renders a Teams Adaptive Card body.
func TeamsAdaptiveCard(p EventPayload) map[string]any {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = eventTitle(p.EventType)
	}
	body := []map[string]any{
		{"type": "TextBlock", "text": title, "weight": "Bolder", "size": "Medium"},
	}
	if p.CourseCode != "" {
		body = append(body, map[string]any{"type": "TextBlock", "text": "Course: " + p.CourseCode, "isSubtle": true})
	}
	if p.DueAt != "" {
		body = append(body, map[string]any{"type": "TextBlock", "text": "Due: " + p.DueAt})
	}
	if p.Body != "" {
		body = append(body, map[string]any{"type": "TextBlock", "text": p.Body, "wrap": true})
	}
	card := map[string]any{
		"type":    "AdaptiveCard",
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"version": "1.4",
		"body":    body,
	}
	if p.URL != "" {
		card["actions"] = []map[string]any{{
			"type":  "Action.OpenUrl",
			"title": "View in Lextures",
			"url":   p.URL,
		}}
	}
	return card
}

// UpcomingText formats slash-command upcoming due dates.
func UpcomingText(items []botsrepo.UpcomingItem) string {
	if len(items) == 0 {
		return "You have no upcoming due dates. 🎉"
	}
	var b strings.Builder
	b.WriteString("*Your upcoming due dates:*\n")
	for i, item := range items {
		line := fmt.Sprintf("%d. %s — %s (%s)", i+1, item.Title, item.CourseCode, item.DueAt.UTC().Format(time.RFC822))
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func eventTitle(eventType string) string {
	switch eventType {
	case "assignment.created":
		return "New assignment"
	case "assignment.due_soon":
		return "Assignment due soon"
	case "grade.released":
		return "Grade released"
	case "announcement.created":
		return "New announcement"
	default:
		return "Lextures update"
	}
}

func escapeSlack(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
