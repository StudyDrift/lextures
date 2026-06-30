package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursefeed"
)

const maxFeedMessageBodyLen = 8000

func canvasFetchAnnouncements(ctx context.Context, client *http.Client, canvasBase, accessToken string, canvasCourseID int64) ([]map[string]any, error) {
	path := fmt.Sprintf("courses/%d/discussion_topics", canvasCourseID)
	q := url.Values{"only_announcements": []string{"true"}}
	return canvasGetArrayPaginated(ctx, client, canvasBase, accessToken, path, q)
}

func canvasAnnouncementFeedBody(title, messageHTML string) string {
	title = strings.TrimSpace(title)
	body := strings.TrimSpace(markdownFromHTML(messageHTML))
	switch {
	case title != "" && body != "":
		return title + "\n\n" + body
	case title != "":
		return title
	default:
		return body
	}
}

func truncateFeedMessageBody(body string) string {
	if utf8.RuneCountInString(body) <= maxFeedMessageBodyLen {
		return body
	}
	runes := []rune(body)
	return string(runes[:maxFeedMessageBodyLen-1]) + "…"
}

func canvasAnnouncementAuthorID(topic map[string]any, canvasUserToLocal map[int64]uuid.UUID, fallback uuid.UUID) uuid.UUID {
	canvasUID := canvasCanvasUserIDFromMap(topic)
	if canvasUID > 0 && canvasUserToLocal != nil {
		if local, ok := canvasUserToLocal[canvasUID]; ok && local != uuid.Nil {
			return local
		}
	}
	return fallback
}

func canvasImportAnnouncements(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	importerUserID uuid.UUID,
	canvasUserToLocal map[int64]uuid.UUID,
	mode string,
) (int, error) {
	if pool == nil {
		return 0, errors.New("server misconfiguration")
	}
	topics, err := canvasFetchAnnouncements(ctx, client, canvasBase, accessToken, canvasCourseID)
	if err != nil {
		return 0, err
	}
	channels, err := coursefeed.ListChannels(ctx, pool, courseID, importerUserID)
	if err != nil {
		return 0, err
	}
	var channelID uuid.UUID
	for _, ch := range channels {
		if strings.EqualFold(ch.Name, "announcements") {
			channelID = ch.ID
			break
		}
	}
	if channelID == uuid.Nil {
		return 0, errors.New("announcements channel not found")
	}

	if mode == "erase" || mode == "overwrite" {
		if _, err := pool.Exec(ctx, `
			DELETE FROM course.feed_messages
			WHERE channel_id = $1 AND parent_message_id IS NULL
		`, channelID); err != nil {
			return 0, errors.New("Failed to clear existing announcements.")
		}
	}

	now := time.Now()
	filtered := make([]map[string]any, 0, len(topics))
	for _, topic := range topics {
		if topic == nil {
			continue
		}
		if published, ok := topic["published"].(bool); ok && !published {
			continue
		}
		if delayed := canvasTimeAt(topic, "delayed_post_at"); delayed != nil && delayed.After(now) {
			continue
		}
		body := canvasAnnouncementFeedBody(strAt(topic, "title", ""), strAt(topic, "message", ""))
		if strings.TrimSpace(body) == "" {
			continue
		}
		filtered = append(filtered, topic)
	}

	sort.Slice(filtered, func(i, j int) bool {
		ti := canvasTimeAt(filtered[i], "posted_at")
		tj := canvasTimeAt(filtered[j], "posted_at")
		switch {
		case ti == nil && tj == nil:
			return int64At(filtered[i], "id") < int64At(filtered[j], "id")
		case ti == nil:
			return true
		case tj == nil:
			return false
		default:
			return ti.Before(*tj)
		}
	})

	imported := 0
	for _, topic := range filtered {
		body := truncateFeedMessageBody(canvasAnnouncementFeedBody(strAt(topic, "title", ""), strAt(topic, "message", "")))
		authorID := canvasAnnouncementAuthorID(topic, canvasUserToLocal, importerUserID)
		postedAt := canvasTimeAt(topic, "posted_at")
		pinned := boolAt(topic, "pinned", false)

		var messageID uuid.UUID
		if postedAt != nil {
			err = pool.QueryRow(ctx, `
				INSERT INTO course.feed_messages (channel_id, author_user_id, parent_message_id, body, mentions_everyone, created_at)
				VALUES ($1, $2, NULL, $3, false, $4)
				RETURNING id
			`, channelID, authorID, body, *postedAt).Scan(&messageID)
		} else {
			err = pool.QueryRow(ctx, `
				INSERT INTO course.feed_messages (channel_id, author_user_id, parent_message_id, body, mentions_everyone)
				VALUES ($1, $2, NULL, $3, false)
				RETURNING id
			`, channelID, authorID, body).Scan(&messageID)
		}
		if err != nil {
			return imported, errors.New("Failed to import announcement from Canvas.")
		}
		if pinned {
			if _, err := pool.Exec(ctx, `
				UPDATE course.feed_messages
				SET pinned_at = COALESCE(created_at, NOW()), pinned_by_user_id = $2
				WHERE id = $1
			`, messageID, authorID); err != nil {
				return imported, errors.New("Failed to pin imported announcement from Canvas.")
			}
		}
		imported++
	}
	return imported, nil
}
