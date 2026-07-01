package coursefeed

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessagePublic struct {
	ID                uuid.UUID       `json:"id"`
	ChannelID         uuid.UUID       `json:"channelId"`
	AuthorUserID      uuid.UUID       `json:"authorUserId"`
	AuthorEmail       string          `json:"authorEmail"`
	AuthorDisplayName *string         `json:"authorDisplayName"`
	ParentMessageID   *uuid.UUID      `json:"parentMessageId"`
	Body              string          `json:"body"`
	MentionsEveryone  bool            `json:"mentionsEveryone"`
	MentionUserIDs    []uuid.UUID     `json:"mentionUserIds"`
	PinnedAt          *time.Time      `json:"pinnedAt"`
	CreatedAt         time.Time       `json:"createdAt"`
	EditedAt          *time.Time      `json:"editedAt"`
	LikeCount         int64           `json:"likeCount"`
	ViewerHasLiked    bool            `json:"viewerHasLiked"`
	Replies           []MessagePublic `json:"replies"`
}

type msgRow struct {
	ID                uuid.UUID
	ChannelID         uuid.UUID
	AuthorUserID      uuid.UUID
	AuthorEmail       string
	AuthorDisplayName *string
	ParentMessageID   *uuid.UUID
	Body              string
	MentionsEveryone  bool
	PinnedAt          *time.Time
	CreatedAt         time.Time
	EditedAt          *time.Time
}

func ChannelBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, channelID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM course.feed_channels WHERE id = $1 AND course_id = $2)
	`, channelID, courseID).Scan(&ok)
	return ok, err
}

func ParentIsRootInChannel(ctx context.Context, pool *pgxpool.Pool, channelID, parentID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM course.feed_messages
			WHERE id = $1 AND channel_id = $2 AND parent_message_id IS NULL
		)
	`, parentID, channelID).Scan(&ok)
	return ok, err
}

func ListMessagesThreaded(ctx context.Context, pool *pgxpool.Pool, channelID, viewerID uuid.UUID, limitRoots int64) ([]MessagePublic, error) {
	rows, err := pool.Query(ctx, `
		SELECT m.id, m.channel_id, m.author_user_id, u.email, u.display_name,
		       m.parent_message_id, m.body, m.mentions_everyone, m.pinned_at, m.created_at, m.edited_at
		FROM course.feed_messages m
		INNER JOIN "user".users u ON u.id = m.author_user_id
		WHERE m.channel_id = $1 AND m.parent_message_id IS NULL
		ORDER BY (m.pinned_at IS NOT NULL) DESC, m.pinned_at DESC NULLS LAST, m.created_at ASC
		LIMIT $2
	`, channelID, limitRoots)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roots []msgRow
	for rows.Next() {
		var r msgRow
		if err := rows.Scan(&r.ID, &r.ChannelID, &r.AuthorUserID, &r.AuthorEmail, &r.AuthorDisplayName, &r.ParentMessageID, &r.Body, &r.MentionsEveryone, &r.PinnedAt, &r.CreatedAt, &r.EditedAt); err != nil {
			return nil, err
		}
		roots = append(roots, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return []MessagePublic{}, nil
	}
	rootIDs := make([]uuid.UUID, 0, len(roots))
	for _, r := range roots {
		rootIDs = append(rootIDs, r.ID)
	}
	repRows, err := pool.Query(ctx, `
		SELECT m.id, m.channel_id, m.author_user_id, u.email, u.display_name,
		       m.parent_message_id, m.body, m.mentions_everyone, m.pinned_at, m.created_at, m.edited_at
		FROM course.feed_messages m
		INNER JOIN "user".users u ON u.id = m.author_user_id
		WHERE m.parent_message_id = ANY($1)
		ORDER BY m.created_at ASC
	`, rootIDs)
	if err != nil {
		return nil, err
	}
	defer repRows.Close()
	var replies []msgRow
	for repRows.Next() {
		var r msgRow
		if err := repRows.Scan(&r.ID, &r.ChannelID, &r.AuthorUserID, &r.AuthorEmail, &r.AuthorDisplayName, &r.ParentMessageID, &r.Body, &r.MentionsEveryone, &r.PinnedAt, &r.CreatedAt, &r.EditedAt); err != nil {
			return nil, err
		}
		replies = append(replies, r)
	}
	if err := repRows.Err(); err != nil {
		return nil, err
	}

	allIDs := make([]uuid.UUID, 0, len(roots)+len(replies))
	for _, r := range roots {
		allIDs = append(allIDs, r.ID)
	}
	for _, r := range replies {
		allIDs = append(allIDs, r.ID)
	}
	mentions := map[uuid.UUID][]uuid.UUID{}
	if len(allIDs) > 0 {
		mrows, err := pool.Query(ctx, `SELECT message_id, mentioned_user_id FROM course.feed_message_mentions WHERE message_id = ANY($1)`, allIDs)
		if err != nil {
			return nil, err
		}
		for mrows.Next() {
			var mid, uid uuid.UUID
			if err := mrows.Scan(&mid, &uid); err != nil {
				mrows.Close()
				return nil, err
			}
			mentions[mid] = append(mentions[mid], uid)
		}
		mrows.Close()
	}
	likeCounts := map[uuid.UUID]int64{}
	viewerLikes := map[uuid.UUID]bool{}
	if len(allIDs) > 0 {
		lrows, err := pool.Query(ctx, `SELECT message_id, COUNT(*)::bigint FROM course.feed_message_likes WHERE message_id = ANY($1) GROUP BY message_id`, allIDs)
		if err != nil {
			return nil, err
		}
		for lrows.Next() {
			var mid uuid.UUID
			var c int64
			if err := lrows.Scan(&mid, &c); err != nil {
				lrows.Close()
				return nil, err
			}
			likeCounts[mid] = c
		}
		lrows.Close()
		vrows, err := pool.Query(ctx, `SELECT message_id FROM course.feed_message_likes WHERE message_id = ANY($1) AND user_id = $2`, allIDs, viewerID)
		if err != nil {
			return nil, err
		}
		for vrows.Next() {
			var mid uuid.UUID
			if err := vrows.Scan(&mid); err != nil {
				vrows.Close()
				return nil, err
			}
			viewerLikes[mid] = true
		}
		vrows.Close()
	}
	toPublic := func(r msgRow) MessagePublic {
		mentionIDs := mentions[r.ID]
		if mentionIDs == nil {
			mentionIDs = []uuid.UUID{}
		}
		return MessagePublic{
			ID:                r.ID,
			ChannelID:         r.ChannelID,
			AuthorUserID:      r.AuthorUserID,
			AuthorEmail:       r.AuthorEmail,
			AuthorDisplayName: r.AuthorDisplayName,
			ParentMessageID:   r.ParentMessageID,
			Body:              r.Body,
			MentionsEveryone:  r.MentionsEveryone,
			MentionUserIDs:    mentionIDs,
			PinnedAt:          r.PinnedAt,
			CreatedAt:         r.CreatedAt,
			EditedAt:          r.EditedAt,
			LikeCount:         likeCounts[r.ID],
			ViewerHasLiked:    viewerLikes[r.ID],
			Replies:           []MessagePublic{},
		}
	}
	out := make([]MessagePublic, 0, len(roots))
	byParent := map[uuid.UUID][]MessagePublic{}
	for _, r := range replies {
		if r.ParentMessageID != nil {
			byParent[*r.ParentMessageID] = append(byParent[*r.ParentMessageID], toPublic(r))
		}
	}
	for _, r := range roots {
		p := toPublic(r)
		if rep := byParent[r.ID]; rep != nil {
			p.Replies = rep
		}
		out = append(out, p)
	}
	return out, nil
}

func CreateMessage(ctx context.Context, pool *pgxpool.Pool, channelID, authorID uuid.UUID, body string, parentMessageID *uuid.UUID, mentionUserIDs []uuid.UUID, mentionsEveryone bool) (uuid.UUID, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO course.feed_messages (channel_id, author_user_id, parent_message_id, body, mentions_everyone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, channelID, authorID, parentMessageID, body, mentionsEveryone).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	for _, uid := range mentionUserIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO course.feed_message_mentions (message_id, mentioned_user_id)
			VALUES ($1, $2)
			ON CONFLICT (message_id, mentioned_user_id) DO NOTHING
		`, id, uid); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// MessageBelongsToCourse returns whether the message exists in any channel of the given course.
func MessageBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, messageID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM course.feed_messages m
			JOIN course.feed_channels ch ON ch.id = m.channel_id
			WHERE m.id = $1 AND ch.course_id = $2
		)
	`, messageID, courseID).Scan(&ok)
	return ok, err
}

// GetMessageAuthorAndIsRoot returns the author and whether the message is a root (top-level) post.
func GetMessageAuthorAndIsRoot(ctx context.Context, pool *pgxpool.Pool, messageID uuid.UUID) (author uuid.UUID, isRoot bool, err error) {
	err = pool.QueryRow(ctx, `
		SELECT author_user_id, (parent_message_id IS NULL)
		FROM course.feed_messages WHERE id = $1
	`, messageID).Scan(&author, &isRoot)
	return author, isRoot, err
}

// UpdateMessageBody edits a message body (and sets edited_at) iff the caller is the original author.
func UpdateMessageBody(ctx context.Context, pool *pgxpool.Pool, messageID, authorID uuid.UUID, body string) error {
	_, err := pool.Exec(ctx, `
		UPDATE course.feed_messages SET body = $1, edited_at = NOW()
		WHERE id = $2 AND author_user_id = $3
	`, body, messageID, authorID)
	return err
}

// SetMessagePinned pins or unpins a message (only roots should be pinned per UI).
func SetMessagePinned(ctx context.Context, pool *pgxpool.Pool, messageID uuid.UUID, pinned bool, actorID uuid.UUID) error {
	if pinned {
		_, err := pool.Exec(ctx, `
			UPDATE course.feed_messages
			SET pinned_at = NOW(), pinned_by_user_id = $2
			WHERE id = $1
		`, messageID, actorID)
		return err
	}
	_, err := pool.Exec(ctx, `
		UPDATE course.feed_messages SET pinned_at = NULL, pinned_by_user_id = NULL WHERE id = $1
	`, messageID)
	return err
}

// AddLike inserts a like idempotently (no-op if already liked).
func AddLike(ctx context.Context, pool *pgxpool.Pool, messageID, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO course.feed_message_likes (message_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (message_id, user_id) DO NOTHING
	`, messageID, userID)
	return err
}

// DeleteMessage removes a message when the caller is the author.
func DeleteMessage(ctx context.Context, pool *pgxpool.Pool, messageID, authorID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
		DELETE FROM course.feed_messages
		WHERE id = $1 AND author_user_id = $2
	`, messageID, authorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// RemoveLike deletes a like idempotently.
func RemoveLike(ctx context.Context, pool *pgxpool.Pool, messageID, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM course.feed_message_likes WHERE message_id = $1 AND user_id = $2
	`, messageID, userID)
	return err
}

