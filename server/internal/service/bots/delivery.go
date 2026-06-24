package bots

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	botsrepo "github.com/lextures/lextures/server/internal/repos/bots"
	"github.com/lextures/lextures/server/internal/webhooks"
)

var botMessagesTotal atomic.Uint64

// BotMessagesTotal returns the delivery counter (for observability hooks).
func BotMessagesTotal() uint64 {
	return botMessagesTotal.Load()
}

// DeliverWebhookJob delivers one webhook envelope to platform channels via bot APIs.
func DeliverWebhookJob(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, svc *Service, subID uuid.UUID, eventType string, payload []byte) (int, string, time.Duration, error) {
	conn, err := botsrepo.ConnectionForSubscription(ctx, pool, subID)
	if err != nil || conn == nil {
		return 0, "", 0, fmt.Errorf("bot connection not found for subscription")
	}
	if !svc.platformEnabled(cfg, conn.Platform) {
		return 0, "", 0, fmt.Errorf("bot platform disabled")
	}
	ep, err := parseEnvelope(eventType, payload, svc.WebOrigin)
	if err != nil {
		return 0, "", 0, err
	}
	token, err := svc.decryptToken(conn.BotTokenEnc, cfg)
	if err != nil {
		return 0, "", 0, err
	}

	// grade.released must not post to channels unless explicitly enabled (FERPA).
	channelBlocked := gradeChannelBlocked(ep.EventType, conn.Settings.GradeChannelEnabled)

	var courseID *uuid.UUID
	if ep.CourseID != "" {
		if id, perr := uuid.Parse(ep.CourseID); perr == nil {
			courseID = &id
		}
	}

	mappings, err := botsrepo.MappingsForEvent(ctx, pool, conn.OrgID, conn.Platform, ep.EventType, courseID)
	if err != nil {
		return 0, "", 0, err
	}

	// Personal grade DMs when student id is present.
	if ep.EventType == string(webhooks.EventGradeReleased) && ep.StudentUserID != "" {
		if derr := svc.deliverGradeDM(ctx, cfg, conn, token, ep); derr != nil {
			slog.Warn("bots.grade_dm", "err", derr)
		}
	}

	if channelBlocked {
		recordMetric(conn.Platform, ep.EventType, "blocked")
		return 200, "grade channel blocked", 0, nil
	}

	if len(mappings) == 0 {
		recordMetric(conn.Platform, ep.EventType, "no_mapping")
		return 200, "no channel mappings", 0, nil
	}

	var lastStatus int
	var lastBody string
	var totalLatency time.Duration
	for _, row := range mappings {
		status, body, latency, derr := svc.postToChannel(ctx, conn.Platform, token, row.Mapping.ChannelID, ep)
		totalLatency += latency
		lastStatus = status
		lastBody = body
		if derr != nil {
			recordMetric(conn.Platform, ep.EventType, "error")
			return status, body, totalLatency, derr
		}
		recordMetric(conn.Platform, ep.EventType, "ok")
	}
	return lastStatus, lastBody, totalLatency, nil
}

func (s *Service) deliverGradeDM(ctx context.Context, cfg config.Config, conn *botsrepo.Connection, token string, ep EventPayload) error {
	studentID, err := uuid.Parse(ep.StudentUserID)
	if err != nil {
		return err
	}
	links, err := botsrepo.ListUserLinks(ctx, s.Pool, studentID)
	if err != nil {
		return err
	}
	var link *botsrepo.UserLink
	for i := range links {
		if links[i].Platform == conn.Platform {
			link = &links[i]
			break
		}
	}
	if link == nil {
		return fmt.Errorf("student not linked on %s", conn.Platform)
	}
	switch conn.Platform {
	case botsrepo.PlatformSlack:
		dm, err := slackOpenDM(ctx, s.HTTP, token, link.PlatformUserID)
		if err != nil {
			return err
		}
		_, _, _, err = s.postToChannel(ctx, conn.Platform, token, dm, ep)
		return err
	default:
		_, _, _, err := s.postToChannel(ctx, conn.Platform, token, link.PlatformUserID, ep)
		return err
	}
}

func (s *Service) postToChannel(ctx context.Context, platform botsrepo.Platform, token, channelID string, ep EventPayload) (int, string, time.Duration, error) {
	switch platform {
	case botsrepo.PlatformSlack:
		payload := SlackBlocks(ep)
		payload["channel"] = channelID
		return (&slackClient{http: s.HTTP}).postMessage(ctx, token, channelID, payload)
	case botsrepo.PlatformDiscord:
		return (&discordClient{http: s.HTTP}).postMessage(ctx, token, channelID, map[string]any{
			"embeds": []map[string]any{DiscordEmbed(ep)},
		})
	case botsrepo.PlatformTeams:
		return (&teamsClient{http: s.HTTP}).postActivity(ctx, s.TeamsServiceURL, channelID, token, TeamsAdaptiveCard(ep))
	default:
		return 0, "", 0, fmt.Errorf("unknown platform %q", platform)
	}
}

func parseEnvelope(eventType string, payload []byte, webOrigin string) (EventPayload, error) {
	var env struct {
		EventType string          `json:"event_type"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return EventPayload{}, err
	}
	if env.EventType == "" {
		env.EventType = eventType
	}
	ep := EventPayload{EventType: env.EventType}
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return ep, nil
	}
	ep.CourseID = strField(data, "courseId")
	ep.CourseCode = strField(data, "courseCode")
	ep.Title = strField(data, "title")
	ep.DueAt = strField(data, "dueAt")
	ep.URL = strField(data, "url")
	ep.StudentUserID = strField(data, "studentUserId")
	ep.Body = strField(data, "body")
	if ep.URL == "" && ep.CourseID != "" {
		ep.URL = webOrigin + "/courses/" + ep.CourseID
	}
	if pts, ok := data["pointsEarned"].(float64); ok {
		ep.PointsEarned = pts
	}
	return ep, nil
}

func strField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func gradeChannelBlocked(eventType string, gradeChannelEnabled bool) bool {
	return eventType == string(webhooks.EventGradeReleased) && !gradeChannelEnabled
}

func recordMetric(platform botsrepo.Platform, eventType, status string) {
	botMessagesTotal.Add(1)
	slog.Debug("bot_messages_total", "platform", platform, "event_type", eventType, "status", status)
}
