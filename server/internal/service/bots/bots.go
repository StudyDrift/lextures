package bots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/lextures/lextures/server/internal/config"
	botsrepo "github.com/lextures/lextures/server/internal/repos/bots"
	webhooksrepo "github.com/lextures/lextures/server/internal/repos/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

var (
	ErrNotConfigured = errors.New("bots: platform not configured")
	ErrInvalidState  = errors.New("bots: invalid oauth state")
)

// Service orchestrates Slack/Teams/Discord bot integrations (plan 16.6).
type Service struct {
	Pool            *pgxpool.Pool
	HTTP            *http.Client
	PublicBase      string
	WebOrigin       string
	StateSecret     []byte
	SecretsKey      []byte
	SlackClientID   string
	SlackClientSecret string
	DiscordClientID string
	DiscordPublicKey  string
	TeamsAppID      string
	TeamsAppPassword string
	TeamsServiceURL string
	Now             func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Service) platformEnabled(cfg config.Config, p botsrepo.Platform) bool {
	switch p {
	case botsrepo.PlatformSlack:
		return cfg.FFBotSlack
	case botsrepo.PlatformTeams:
		return cfg.FFBotTeams
	case botsrepo.PlatformDiscord:
		return cfg.FFBotDiscord
	default:
		return false
	}
}

func (s *Service) AnyEnabled(cfg config.Config) bool {
	return cfg.FFBotSlack || cfg.FFBotTeams || cfg.FFBotDiscord
}

func (s *Service) decryptToken(enc string, cfg config.Config) (string, error) {
	key := s.SecretsKey
	if len(key) != 32 {
		key = cfg.PlatformSecretsKey
	}
	if len(key) != 32 {
		return "", fmt.Errorf("platform secrets key not configured")
	}
	raw, err := webhooks.DecryptSigningKey(enc, key)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Service) encryptSecret(plain string, cfg config.Config) (string, error) {
	key := s.SecretsKey
	if len(key) != 32 {
		key = cfg.PlatformSecretsKey
	}
	return webhooks.EncryptSigningKey([]byte(plain), key)
}

// ConnectionView is the redacted API representation of a bot connection.
type ConnectionView struct {
	ID            string   `json:"id"`
	Platform      string   `json:"platform"`
	WorkspaceID   string   `json:"workspaceId"`
	WorkspaceName string   `json:"workspaceName"`
	Settings      botsrepo.ConnectionSettings `json:"settings"`
	CreatedAt     time.Time `json:"createdAt"`
	Mappings      []MappingView `json:"mappings,omitempty"`
}

type MappingView struct {
	ID          string   `json:"id"`
	CourseID    *string  `json:"courseId,omitempty"`
	ChannelID   string   `json:"channelId"`
	ChannelName string   `json:"channelName,omitempty"`
	EventTypes  []string `json:"eventTypes"`
}

func connectionView(c botsrepo.Connection, mappings []botsrepo.ChannelMapping) ConnectionView {
	v := ConnectionView{
		ID:            c.ID.String(),
		Platform:      string(c.Platform),
		WorkspaceID:   c.WorkspaceID,
		WorkspaceName: c.WorkspaceName,
		Settings:      c.Settings,
		CreatedAt:     c.CreatedAt,
	}
	for _, m := range mappings {
		mv := MappingView{
			ID:          m.ID.String(),
			ChannelID:   m.ChannelID,
			ChannelName: m.ChannelName,
			EventTypes:  m.EventTypes,
		}
		if m.CourseID != nil {
			s := m.CourseID.String()
			mv.CourseID = &s
		}
		v.Mappings = append(v.Mappings, mv)
	}
	return v
}

// List returns org bot connections with mappings.
func (s *Service) List(ctx context.Context, orgID uuid.UUID) ([]ConnectionView, error) {
	conns, err := botsrepo.ListConnectionsByOrg(ctx, s.Pool, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]ConnectionView, 0, len(conns))
	for _, c := range conns {
		mappings, _ := botsrepo.ListChannelMappings(ctx, s.Pool, c.ID)
		out = append(out, connectionView(c, mappings))
	}
	return out, nil
}

// Disconnect removes a bot connection and its webhook subscription.
func (s *Service) Disconnect(ctx context.Context, cfg config.Config, orgID, id uuid.UUID) error {
	conn, err := botsrepo.GetConnection(ctx, s.Pool, orgID, id)
	if err != nil {
		return err
	}
	if conn.WebhookSubscriptionID != nil {
		_, _ = webhooksrepo.Delete(ctx, s.Pool, orgID, *conn.WebhookSubscriptionID)
	}
	return botsrepo.DeleteConnection(ctx, s.Pool, orgID, id)
}

// SlackAuthorizeURL starts Slack OAuth for workspace install.
func (s *Service) SlackAuthorizeURL(orgID, userID uuid.UUID) (string, error) {
	if s.SlackClientID == "" || s.SlackClientSecret == "" {
		return "", ErrNotConfigured
	}
	state := s.signOAuthState(oauthState{OrgID: orgID, UserID: userID, Platform: "slack"})
	q := url.Values{}
	q.Set("client_id", s.SlackClientID)
	q.Set("scope", "chat:write,commands,channels:read,groups:read,im:write,users:read")
	q.Set("redirect_uri", s.slackRedirectURI())
	q.Set("state", state)
	return "https://slack.com/oauth/v2/authorize?" + q.Encode(), nil
}

func (s *Service) slackRedirectURI() string {
	return strings.TrimRight(s.PublicBase, "/") + "/integrations/slack/oauth_redirect"
}

// CompleteSlackOAuth exchanges the code and stores the connection.
func (s *Service) CompleteSlackOAuth(ctx context.Context, cfg config.Config, code, state string) (uuid.UUID, error) {
	st, err := s.verifyOAuthState(state)
	if err != nil || st.Platform != "slack" {
		return uuid.Nil, ErrInvalidState
	}
	if s.SlackClientID == "" || s.SlackClientSecret == "" {
		return uuid.Nil, ErrNotConfigured
	}
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.SlackClientID)
	form.Set("client_secret", s.SlackClientSecret)
	form.Set("redirect_uri", s.slackRedirectURI())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/oauth.v2.access", strings.NewReader(form.Encode()))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		Team        struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.OK {
		return uuid.Nil, fmt.Errorf("slack oauth: %s", out.Error)
	}
	return s.storeConnection(ctx, cfg, st, botsrepo.PlatformSlack, out.Team.ID, out.Team.Name, out.AccessToken, s.SlackClientSecret)
}

// DiscordInviteURL returns the bot invite URL for admins.
func (s *Service) DiscordInviteURL() (string, error) {
	if s.DiscordClientID == "" {
		return "", ErrNotConfigured
	}
	q := url.Values{}
	q.Set("client_id", s.DiscordClientID)
	q.Set("scope", "bot applications.commands")
	q.Set("permissions", "68608")
	return "https://discord.com/api/oauth2/authorize?" + q.Encode(), nil
}

// CompleteDiscordInstall stores a Discord guild connection after admin provides guild id + token.
func (s *Service) CompleteDiscordInstall(ctx context.Context, cfg config.Config, orgID, userID uuid.UUID, guildID, guildName, botToken string) (*ConnectionView, error) {
	if botToken == "" {
		return nil, ErrNotConfigured
	}
	st := oauthState{OrgID: orgID, UserID: userID, Platform: "discord"}
	id, err := s.storeConnection(ctx, cfg, st, botsrepo.PlatformDiscord, guildID, guildName, botToken, s.DiscordPublicKey)
	if err != nil {
		return nil, err
	}
	conn, err := botsrepo.GetConnection(ctx, s.Pool, orgID, id)
	if err != nil {
		return nil, err
	}
	v := connectionView(*conn, nil)
	return &v, nil
}

func (s *Service) storeConnection(ctx context.Context, cfg config.Config, st oauthState, platform botsrepo.Platform, workspaceID, workspaceName, botToken, signingSecret string) (uuid.UUID, error) {
	if !cfg.FFWebhooks {
		return uuid.Nil, fmt.Errorf("webhooks must be enabled for bot delivery")
	}
	if len(cfg.PlatformSecretsKey) != 32 {
		return uuid.Nil, fmt.Errorf("platform secrets key not configured")
	}
	tokenEnc, err := s.encryptSecret(botToken, cfg)
	if err != nil {
		return uuid.Nil, err
	}
	secretEnc, err := s.encryptSecret(signingSecret, cfg)
	if err != nil {
		return uuid.Nil, err
	}
	signingKey, err := webhooks.GenerateSigningKey()
	if err != nil {
		return uuid.Nil, err
	}
	keyEnc, err := webhooks.EncryptSigningKey(signingKey, cfg.PlatformSecretsKey)
	if err != nil {
		return uuid.Nil, err
	}
	eventTypes := []string{
		string(webhooks.EventAssignmentCreated),
		string(webhooks.EventAssignmentDueSoon),
		string(webhooks.EventGradeReleased),
		string(webhooks.EventAnnouncementCreated),
	}
	sub, err := webhooksrepo.Create(ctx, s.Pool, webhooksrepo.CreateInput{
		OrgID:         st.OrgID,
		Label:         fmt.Sprintf("%s bot (%s)", platform, workspaceName),
		EndpointURL:   "https://bots.internal.lextures/delivery",
		SigningKeyEnc: keyEnc,
		EventTypes:    eventTypes,
		CreatedBy:     &st.UserID,
	})
	if err != nil {
		return uuid.Nil, err
	}
	conn, err := botsrepo.CreateConnection(ctx, s.Pool, botsrepo.CreateConnectionInput{
		OrgID:                 st.OrgID,
		Platform:              platform,
		WorkspaceID:           workspaceID,
		WorkspaceName:         workspaceName,
		BotTokenEnc:           tokenEnc,
		SigningSecretEnc:      secretEnc,
		WebhookSubscriptionID: &sub.ID,
		Settings:              botsrepo.DefaultSettings(),
		ConnectedBy:           &st.UserID,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return conn.ID, nil
}

// UpsertMapping creates or updates a channel mapping.
func (s *Service) UpsertMapping(ctx context.Context, orgID, connectionID uuid.UUID, courseID *uuid.UUID, channelID, channelName string, eventTypes []string) (*MappingView, error) {
	conn, err := botsrepo.GetConnection(ctx, s.Pool, orgID, connectionID)
	if err != nil {
		return nil, err
	}
	normalized, ok := webhooks.NormalizeEventTypes(eventTypes)
	if !ok || len(normalized) == 0 {
		return nil, fmt.Errorf("invalid event types")
	}
	m, err := botsrepo.UpsertChannelMapping(ctx, s.Pool, conn.ID, courseID, channelID, channelName, normalized)
	if err != nil {
		return nil, err
	}
	mv := MappingView{ID: m.ID.String(), ChannelID: m.ChannelID, ChannelName: m.ChannelName, EventTypes: m.EventTypes}
	if m.CourseID != nil {
		s := m.CourseID.String()
		mv.CourseID = &s
	}
	return &mv, nil
}

// DeleteMapping removes a channel mapping.
func (s *Service) DeleteMapping(ctx context.Context, orgID, connectionID, mappingID uuid.UUID) error {
	if _, err := botsrepo.GetConnection(ctx, s.Pool, orgID, connectionID); err != nil {
		return err
	}
	return botsrepo.DeleteChannelMapping(ctx, s.Pool, connectionID, mappingID)
}

// LinkUserOAuth starts user account linking OAuth for a platform.
func (s *Service) LinkUserOAuth(platform botsrepo.Platform, userID uuid.UUID) (string, error) {
	switch platform {
	case botsrepo.PlatformSlack:
		if s.SlackClientID == "" {
			return "", ErrNotConfigured
		}
		state := s.signOAuthState(oauthState{UserID: userID, Platform: "user_link_slack"})
		q := url.Values{}
		q.Set("client_id", s.SlackClientID)
		q.Set("user_scope", "identity.basic")
		q.Set("redirect_uri", strings.TrimRight(s.PublicBase, "/")+"/api/v1/me/bot-link/slack/callback")
		q.Set("state", state)
		return "https://slack.com/oauth/v2/authorize?" + q.Encode(), nil
	case botsrepo.PlatformDiscord:
		if s.DiscordClientID == "" {
			return "", ErrNotConfigured
		}
		state := s.signOAuthState(oauthState{UserID: userID, Platform: "user_link_discord"})
		conf := &oauth2.Config{
			ClientID:    s.DiscordClientID,
			RedirectURL: strings.TrimRight(s.PublicBase, "/") + "/api/v1/me/bot-link/discord/callback",
			Scopes:      []string{"identify"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		}
		return conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
	default:
		return "", fmt.Errorf("user linking not supported for %s", platform)
	}
}

// CompleteUserLinkSlack finishes Slack user identity linking.
func (s *Service) CompleteUserLinkSlack(ctx context.Context, code, state string) error {
	st, err := s.verifyOAuthState(state)
	if err != nil || st.Platform != "user_link_slack" {
		return ErrInvalidState
	}
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.SlackClientID)
	form.Set("client_secret", s.SlackClientSecret)
	form.Set("redirect_uri", strings.TrimRight(s.PublicBase, "/")+"/api/v1/me/bot-link/slack/callback")
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/oauth.v2.access", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		OK         bool `json:"ok"`
		AuthedUser struct {
			ID string `json:"id"`
		} `json:"authed_user"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.OK || out.AuthedUser.ID == "" {
		return fmt.Errorf("slack user link failed")
	}
	_, err = botsrepo.UpsertUserLink(ctx, s.Pool, st.UserID, botsrepo.PlatformSlack, out.AuthedUser.ID)
	return err
}

// CompleteUserLinkDiscord finishes Discord user identity linking.
func (s *Service) CompleteUserLinkDiscord(ctx context.Context, code, state string) error {
	st, err := s.verifyOAuthState(state)
	if err != nil || st.Platform != "user_link_discord" {
		return ErrInvalidState
	}
	conf := &oauth2.Config{
		ClientID:     s.DiscordClientID,
		ClientSecret: "",
		RedirectURL:  strings.TrimRight(s.PublicBase, "/") + "/api/v1/me/bot-link/discord/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var user struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &user); err != nil || user.ID == "" {
		return fmt.Errorf("discord user link failed")
	}
	_, err = botsrepo.UpsertUserLink(ctx, s.Pool, st.UserID, botsrepo.PlatformDiscord, user.ID)
	return err
}

// UnlinkUser removes a platform link for the current user.
func (s *Service) UnlinkUser(ctx context.Context, userID uuid.UUID, platform botsrepo.Platform) error {
	return botsrepo.DeleteUserLink(ctx, s.Pool, userID, platform)
}

// ListUserLinks returns linked platforms for a user.
func (s *Service) ListUserLinks(ctx context.Context, userID uuid.UUID) ([]botsrepo.UserLink, error) {
	return botsrepo.ListUserLinks(ctx, s.Pool, userID)
}

// HandleSlackSlashCommand serves /lextures upcoming.
func (s *Service) HandleSlackSlashCommand(ctx context.Context, cfg config.Config, _, platformUserID, _ /* channelID */, commandText string) (string, error) {
	if !strings.HasPrefix(strings.TrimSpace(commandText), "/lextures") {
		return "Unknown command.", nil
	}
	link, err := botsrepo.UserLinkByPlatformUser(ctx, s.Pool, botsrepo.PlatformSlack, platformUserID)
	if err != nil {
		return "Link your Lextures account in Settings → Connected Accounts to use this command.", nil
	}
	items, err := botsrepo.ListUpcomingDueItems(ctx, s.Pool, link.UserID, 5, s.WebOrigin)
	if err != nil {
		return "Could not load upcoming due dates.", nil
	}
	text := UpcomingText(items)
	return text, nil
}

// DecryptSigningSecret returns the workspace signing secret for inbound verification.
func (s *Service) DecryptSigningSecret(enc string, cfg config.Config) (string, error) {
	return s.decryptToken(enc, cfg)
}

// SigningSecretForWorkspace finds signing secret for a Slack workspace.
func (s *Service) SigningSecretForWorkspace(ctx context.Context, cfg config.Config, workspaceID string) (string, error) {
	row := s.Pool.QueryRow(ctx, `
SELECT signing_secret_enc FROM integrations.bot_connections
WHERE platform = 'slack' AND workspace_id = $1
LIMIT 1
`, workspaceID)
	var enc string
	if err := row.Scan(&enc); err != nil {
		return "", err
	}
	return s.DecryptSigningSecret(enc, cfg)
}

// NewFromConfig builds a bots service from process config.
func NewFromConfig(cfg config.Config, pool *pgxpool.Pool, publicBase string) *Service {
	base := publicBase
	if base == "" {
		base = cfg.OIDCPublicBaseURL
	}
	if base == "" {
		base = cfg.SAMLPublicBaseURL
	}
	webOrigin := cfg.PublicWebOrigin
	if webOrigin == "" {
		webOrigin = "http://localhost:5173"
	}
	return &Service{
		Pool:              pool,
		PublicBase:        base,
		WebOrigin:         webOrigin,
		StateSecret:       []byte(cfg.JWTSecret),
		SecretsKey:        cfg.PlatformSecretsKey,
		SlackClientID:     cfg.SlackBotClientID,
		SlackClientSecret: cfg.SlackBotClientSecret,
		DiscordClientID:   cfg.DiscordBotClientID,
		DiscordPublicKey:  cfg.DiscordBotPublicKey,
		TeamsAppID:        cfg.TeamsBotAppID,
		TeamsAppPassword:  cfg.TeamsBotAppPassword,
	}
}
