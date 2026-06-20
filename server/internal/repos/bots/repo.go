package botsrepo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("bots: not found")

// Platform identifies a messaging integration.
type Platform string

const (
	PlatformSlack   Platform = "slack"
	PlatformTeams   Platform = "teams"
	PlatformDiscord Platform = "discord"
)

// ConnectionSettings holds per-workspace bot configuration.
type ConnectionSettings struct {
	DueSoonHours        int  `json:"dueSoonHours"`
	GradeChannelEnabled bool `json:"gradeChannelEnabled"`
}

func DefaultSettings() ConnectionSettings {
	return ConnectionSettings{DueSoonHours: 24, GradeChannelEnabled: false}
}

// Connection is a connected workspace bot installation.
type Connection struct {
	ID                    uuid.UUID
	OrgID                 uuid.UUID
	Platform              Platform
	WorkspaceID           string
	WorkspaceName         string
	BotTokenEnc           string
	SigningSecretEnc      string
	WebhookSubscriptionID *uuid.UUID
	Settings              ConnectionSettings
	ConnectedBy           *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ChannelMapping routes course events to a platform channel.
type ChannelMapping struct {
	ID           uuid.UUID
	ConnectionID uuid.UUID
	CourseID     *uuid.UUID
	ChannelID    string
	ChannelName  string
	EventTypes   []string
	CreatedAt    time.Time
}

// UserLink binds a Lextures user to a platform user id.
type UserLink struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Platform       Platform
	PlatformUserID string
	CreatedAt      time.Time
}

// UpcomingItem is one due assignment for slash-command responses.
type UpcomingItem struct {
	Title      string
	CourseCode string
	CourseName string
	DueAt      time.Time
	URL        string
}

func scanSettings(raw []byte) ConnectionSettings {
	def := DefaultSettings()
	if len(raw) == 0 {
		return def
	}
	var s ConnectionSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return def
	}
	if s.DueSoonHours <= 0 {
		s.DueSoonHours = 24
	}
	return s
}

func settingsJSON(s ConnectionSettings) ([]byte, error) {
	if s.DueSoonHours <= 0 {
		s.DueSoonHours = 24
	}
	return json.Marshal(s)
}

// CreateConnectionInput holds a new workspace connection.
type CreateConnectionInput struct {
	OrgID                 uuid.UUID
	Platform              Platform
	WorkspaceID           string
	WorkspaceName         string
	BotTokenEnc           string
	SigningSecretEnc      string
	WebhookSubscriptionID *uuid.UUID
	Settings              ConnectionSettings
	ConnectedBy           *uuid.UUID
}

// CreateConnection inserts a bot connection row.
func CreateConnection(ctx context.Context, pool *pgxpool.Pool, in CreateConnectionInput) (*Connection, error) {
	settingsRaw, err := settingsJSON(in.Settings)
	if err != nil {
		return nil, err
	}
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.bot_connections (
    org_id, platform, workspace_id, workspace_name, bot_token_enc, signing_secret_enc,
    webhook_subscription_id, settings, connected_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, org_id, platform, workspace_id, COALESCE(workspace_name, ''), bot_token_enc,
          signing_secret_enc, webhook_subscription_id, settings, connected_by, created_at, updated_at
`, in.OrgID, string(in.Platform), in.WorkspaceID, nullIfEmpty(in.WorkspaceName), in.BotTokenEnc,
		in.SigningSecretEnc, in.WebhookSubscriptionID, settingsRaw, in.ConnectedBy)
	return scanConnection(row)
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func scanConnection(row pgx.Row) (*Connection, error) {
	var c Connection
	var platform string
	var settingsRaw []byte
	var wsName *string
	err := row.Scan(&c.ID, &c.OrgID, &platform, &c.WorkspaceID, &wsName, &c.BotTokenEnc,
		&c.SigningSecretEnc, &c.WebhookSubscriptionID, &settingsRaw, &c.ConnectedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Platform = Platform(platform)
	if wsName != nil {
		c.WorkspaceName = *wsName
	}
	c.Settings = scanSettings(settingsRaw)
	return &c, nil
}

// GetConnection returns a connection scoped to org.
func GetConnection(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (*Connection, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, platform, workspace_id, workspace_name, bot_token_enc, signing_secret_enc,
       webhook_subscription_id, settings, connected_by, created_at, updated_at
FROM integrations.bot_connections
WHERE id = $1 AND org_id = $2
`, id, orgID)
	return scanConnection(row)
}

// GetConnectionByID loads a connection without org scope (delivery worker).
func GetConnectionByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Connection, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, platform, workspace_id, workspace_name, bot_token_enc, signing_secret_enc,
       webhook_subscription_id, settings, connected_by, created_at, updated_at
FROM integrations.bot_connections
WHERE id = $1
`, id)
	return scanConnection(row)
}

// ConnectionForSubscription finds the bot connection tied to a webhook subscription.
func ConnectionForSubscription(ctx context.Context, pool *pgxpool.Pool, subID uuid.UUID) (*Connection, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, platform, workspace_id, workspace_name, bot_token_enc, signing_secret_enc,
       webhook_subscription_id, settings, connected_by, created_at, updated_at
FROM integrations.bot_connections
WHERE webhook_subscription_id = $1
`, subID)
	return scanConnection(row)
}

// ListConnectionsByOrg returns all bot connections for an org.
func ListConnectionsByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Connection, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, platform, workspace_id, workspace_name, bot_token_enc, signing_secret_enc,
       webhook_subscription_id, settings, connected_by, created_at, updated_at
FROM integrations.bot_connections
WHERE org_id = $1
ORDER BY platform, created_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		var c Connection
		var platform string
		var settingsRaw []byte
		var wsName *string
		if err := rows.Scan(&c.ID, &c.OrgID, &platform, &c.WorkspaceID, &wsName, &c.BotTokenEnc,
			&c.SigningSecretEnc, &c.WebhookSubscriptionID, &settingsRaw, &c.ConnectedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Platform = Platform(platform)
		if wsName != nil {
			c.WorkspaceName = *wsName
		}
		c.Settings = scanSettings(settingsRaw)
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteConnection removes a connection and cascades mappings.
func DeleteConnection(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) error {
	tag, err := pool.Exec(ctx, `DELETE FROM integrations.bot_connections WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertChannelMapping creates or replaces a channel mapping.
func UpsertChannelMapping(ctx context.Context, pool *pgxpool.Pool, connectionID uuid.UUID, courseID *uuid.UUID, channelID, channelName string, eventTypes []string) (*ChannelMapping, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.bot_channel_mappings (connection_id, course_id, channel_id, channel_name, event_types)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (connection_id, course_id, channel_id) DO UPDATE SET
    channel_name = EXCLUDED.channel_name,
    event_types = EXCLUDED.event_types
RETURNING id, connection_id, course_id, channel_id, COALESCE(channel_name, ''), event_types, created_at
`, connectionID, courseID, channelID, nullIfEmpty(channelName), eventTypes)
	var m ChannelMapping
	var chName string
	if err := row.Scan(&m.ID, &m.ConnectionID, &m.CourseID, &m.ChannelID, &chName, &m.EventTypes, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.ChannelName = chName
	return &m, nil
}

// ListChannelMappings returns mappings for a connection.
func ListChannelMappings(ctx context.Context, pool *pgxpool.Pool, connectionID uuid.UUID) ([]ChannelMapping, error) {
	rows, err := pool.Query(ctx, `
SELECT id, connection_id, course_id, channel_id, COALESCE(channel_name, ''), event_types, created_at
FROM integrations.bot_channel_mappings
WHERE connection_id = $1
ORDER BY created_at
`, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChannelMapping
	for rows.Next() {
		var m ChannelMapping
		if err := rows.Scan(&m.ID, &m.ConnectionID, &m.CourseID, &m.ChannelID, &m.ChannelName, &m.EventTypes, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteChannelMapping removes one mapping row.
func DeleteChannelMapping(ctx context.Context, pool *pgxpool.Pool, connectionID, mappingID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM integrations.bot_channel_mappings WHERE id = $1 AND connection_id = $2
`, mappingID, connectionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MappingsForEvent returns channel mappings subscribed to eventType for org/platform course.
func MappingsForEvent(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, platform Platform, eventType string, courseID *uuid.UUID) ([]struct {
	Connection Connection
	Mapping    ChannelMapping
}, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.org_id, c.platform, c.workspace_id, c.workspace_name, c.bot_token_enc,
       c.signing_secret_enc, c.webhook_subscription_id, c.settings, c.connected_by, c.created_at, c.updated_at,
       m.id, m.connection_id, m.course_id, m.channel_id, COALESCE(m.channel_name, ''), m.event_types, m.created_at
FROM integrations.bot_connections c
INNER JOIN integrations.bot_channel_mappings m ON m.connection_id = c.id
WHERE c.org_id = $1 AND c.platform = $2
  AND ($3::uuid IS NULL OR m.course_id = $3 OR m.course_id IS NULL)
  AND $4 = ANY (m.event_types)
`, orgID, string(platform), courseID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		Connection Connection
		Mapping    ChannelMapping
	}
	for rows.Next() {
		var c Connection
		var m ChannelMapping
		var plat string
		var settingsRaw []byte
		var wsName *string
		if err := rows.Scan(&c.ID, &c.OrgID, &plat, &c.WorkspaceID, &wsName, &c.BotTokenEnc,
			&c.SigningSecretEnc, &c.WebhookSubscriptionID, &settingsRaw, &c.ConnectedBy, &c.CreatedAt, &c.UpdatedAt,
			&m.ID, &m.ConnectionID, &m.CourseID, &m.ChannelID, &m.ChannelName, &m.EventTypes, &m.CreatedAt); err != nil {
			return nil, err
		}
		c.Platform = Platform(plat)
		if wsName != nil {
			c.WorkspaceName = *wsName
		}
		c.Settings = scanSettings(settingsRaw)
		out = append(out, struct {
			Connection Connection
			Mapping    ChannelMapping
		}{Connection: c, Mapping: m})
	}
	return out, rows.Err()
}

// UpsertUserLink binds a platform user to a Lextures account.
func UpsertUserLink(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, platform Platform, platformUserID string) (*UserLink, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.bot_user_links (user_id, platform, platform_user_id)
VALUES ($1, $2, $3)
ON CONFLICT (platform, platform_user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING id, user_id, platform, platform_user_id, created_at
`, userID, string(platform), platformUserID)
	var l UserLink
	var plat string
	if err := row.Scan(&l.ID, &l.UserID, &plat, &l.PlatformUserID, &l.CreatedAt); err != nil {
		return nil, err
	}
	l.Platform = Platform(plat)
	return &l, nil
}

// DeleteUserLink removes a user's platform link.
func DeleteUserLink(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, platform Platform) error {
	_, err := pool.Exec(ctx, `DELETE FROM integrations.bot_user_links WHERE user_id = $1 AND platform = $2`, userID, string(platform))
	return err
}

// UserLinkByPlatformUser finds a link for slash commands.
func UserLinkByPlatformUser(ctx context.Context, pool *pgxpool.Pool, platform Platform, platformUserID string) (*UserLink, error) {
	row := pool.QueryRow(ctx, `
SELECT id, user_id, platform, platform_user_id, created_at
FROM integrations.bot_user_links
WHERE platform = $1 AND platform_user_id = $2
`, string(platform), platformUserID)
	var l UserLink
	var plat string
	err := row.Scan(&l.ID, &l.UserID, &plat, &l.PlatformUserID, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l.Platform = Platform(plat)
	return &l, nil
}

// ListUserLinks returns all platform links for a user.
func ListUserLinks(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]UserLink, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, platform, platform_user_id, created_at
FROM integrations.bot_user_links
WHERE user_id = $1
ORDER BY platform
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserLink
	for rows.Next() {
		var l UserLink
		var plat string
		if err := rows.Scan(&l.ID, &l.UserID, &plat, &l.PlatformUserID, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Platform = Platform(plat)
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListUpcomingDueItems returns the next limit due assignments for a user.
func ListUpcomingDueItems(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int, webOrigin string) ([]UpcomingItem, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := pool.Query(ctx, `
SELECT csi.title, COALESCE(c.course_code, ''), COALESCE(c.name, ''), csi.due_at, c.id
FROM course.course_structure_items csi
INNER JOIN course.courses c ON c.id = csi.course_id
INNER JOIN course.course_enrollments ce ON ce.course_id = c.id AND ce.user_id = $1
WHERE csi.due_at IS NOT NULL
  AND csi.due_at > now()
  AND csi.item_type IN ('content_page', 'assignment', 'quiz')
  AND ce.status = 'active'
ORDER BY csi.due_at ASC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UpcomingItem
	for rows.Next() {
		var item UpcomingItem
		var courseID uuid.UUID
		if err := rows.Scan(&item.Title, &item.CourseCode, &item.CourseName, &item.DueAt, &courseID); err != nil {
			return nil, err
		}
		item.URL = webOrigin + "/courses/" + courseID.String()
		out = append(out, item)
	}
	return out, rows.Err()
}

// MarkDueSoonSent records that a due-soon DM was delivered.
func MarkDueSoonSent(ctx context.Context, pool *pgxpool.Pool, structureItemID, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO integrations.bot_due_soon_sent (structure_item_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, structureItemID, userID)
	return err
}

// WasDueSoonSent reports whether a due-soon notification was already sent.
func WasDueSoonSent(ctx context.Context, pool *pgxpool.Pool, structureItemID, userID uuid.UUID) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT 1 FROM integrations.bot_due_soon_sent WHERE structure_item_id = $1 AND user_id = $2
`, structureItemID, userID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
