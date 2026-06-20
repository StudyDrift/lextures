package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	botsrepo "github.com/lextures/lextures/server/internal/repos/bots"
	"github.com/lextures/lextures/server/internal/repos/organization"
	bots "github.com/lextures/lextures/server/internal/service/bots"
)

func (d Deps) registerBotRoutes(r chi.Router) {
	r.Get("/api/v1/bots", d.handleListBots())
	r.Delete("/api/v1/bots/{id}", d.handleDisconnectBot())
	r.Post("/api/v1/bots/{id}/mappings", d.handleUpsertBotMapping())
	r.Delete("/api/v1/bots/{id}/mappings/{mappingId}", d.handleDeleteBotMapping())
	r.Post("/api/v1/bots/discord/connect", d.handleDiscordConnect())

	r.Get("/integrations/slack/install", d.handleSlackInstall())
	r.Get("/integrations/slack/oauth_redirect", d.handleSlackOAuthRedirect())
	r.Post("/integrations/slack/events", d.handleSlackEvents())

	r.Get("/integrations/discord/invite", d.handleDiscordInvite())
	r.Post("/integrations/discord/interactions", d.handleDiscordInteractions())

	r.Post("/integrations/teams/messages", d.handleTeamsMessages())

	r.Get("/api/v1/me/bot-links", d.handleListBotLinks())
	r.Post("/api/v1/me/bot-link/{platform}", d.handleStartBotLink())
	r.Delete("/api/v1/me/bot-link/{platform}", d.handleUnlinkBot())
	r.Get("/api/v1/me/bot-link/slack/callback", d.handleBotLinkSlackCallback())
	r.Get("/api/v1/me/bot-link/discord/callback", d.handleBotLinkDiscordCallback())
}

func (d Deps) botsEnabled(w http.ResponseWriter) (*bots.Service, bool) {
	if d.Bots == nil {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Classroom bots are not enabled on this environment.")
		return nil, false
	}
	return d.Bots, true
}

func (d Deps) botPlatformEnabled(w http.ResponseWriter, platform botsrepo.Platform) bool {
	cfg := d.effectiveConfig()
	switch platform {
	case botsrepo.PlatformSlack:
		if !cfg.FFBotSlack {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Slack bot is not enabled.")
			return false
		}
	case botsrepo.PlatformTeams:
		if !cfg.FFBotTeams {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Teams bot is not enabled.")
			return false
		}
	case botsrepo.PlatformDiscord:
		if !cfg.FFBotDiscord {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Discord bot is not enabled.")
			return false
		}
	default:
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown platform.")
		return false
	}
	if !cfg.FFWebhooks {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Outbound webhooks must be enabled for bot delivery.")
		return false
	}
	return true
}

func parseBotPlatform(raw string) (botsrepo.Platform, error) {
	switch strings.TrimSpace(raw) {
	case "slack":
		return botsrepo.PlatformSlack, nil
	case "teams":
		return botsrepo.PlatformTeams, nil
	case "discord":
		return botsrepo.PlatformDiscord, nil
	default:
		return "", errors.New("unknown platform")
	}
}

func (d Deps) handleListBots() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		list, err := svc.List(r.Context(), orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load bot connections.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"connections": list})
	}
}

func (d Deps) handleDisconnectBot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}
		if err := svc.Disconnect(r.Context(), d.effectiveConfig(), orgID, id); err != nil {
			if errors.Is(err, botsrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Bot connection not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to disconnect bot.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleUpsertBotMapping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		connID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}
		var body struct {
			CourseID    *string  `json:"courseId"`
			ChannelID   string   `json:"channelId"`
			ChannelName string   `json:"channelName"`
			EventTypes  []string `json:"eventTypes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.ChannelID) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "channelId is required.")
			return
		}
		var courseID *uuid.UUID
		if body.CourseID != nil && strings.TrimSpace(*body.CourseID) != "" {
			id, perr := uuid.Parse(*body.CourseID)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			courseID = &id
		}
		mv, err := svc.UpsertMapping(r.Context(), orgID, connID, courseID, body.ChannelID, body.ChannelName, body.EventTypes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mapping": mv})
	}
}

func (d Deps) handleDeleteBotMapping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		connID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid connection id.")
			return
		}
		mappingID, err := uuid.Parse(chi.URLParam(r, "mappingId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid mapping id.")
			return
		}
		if err := svc.DeleteMapping(r.Context(), orgID, connID, mappingID); err != nil {
			if errors.Is(err, botsrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Mapping not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete mapping.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleSlackInstall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformSlack) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		authURL, err := svc.SlackAuthorizeURL(orgID, userID)
		if err != nil {
			if errors.Is(err, bots.ErrNotConfigured) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Slack bot OAuth is not configured.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start Slack install.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authorizeUrl": authURL})
	}
}

func (d Deps) handleSlackOAuthRedirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformSlack) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Redirect(w, r, adminIntegrationsRedirect("error", "missing_code"), http.StatusFound)
			return
		}
		if _, err := svc.CompleteSlackOAuth(r.Context(), d.effectiveConfig(), code, state); err != nil {
			http.Redirect(w, r, adminIntegrationsRedirect("error", "slack_oauth_failed"), http.StatusFound)
			return
		}
		http.Redirect(w, r, adminIntegrationsRedirect("connected", "slack"), http.StatusFound)
	}
}

func adminIntegrationsRedirect(kind, value string) string {
	q := url.Values{}
	q.Set(kind, value)
	return "/admin/integrations?" + q.Encode()
}

func (d Deps) handleSlackEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformSlack) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var envelope struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"`
		}
		_ = json.Unmarshal(body, &envelope)
		if envelope.Type == "url_verification" {
			writeJSON(w, http.StatusOK, map[string]any{"challenge": envelope.Challenge})
			return
		}
		ts := r.Header.Get("X-Slack-Request-Timestamp")
		sig := r.Header.Get("X-Slack-Signature")
		teamID := ""
		var cmd struct {
			TeamID      string `json:"team_id"`
			UserID      string `json:"user_id"`
			ChannelID   string `json:"channel_id"`
			Command     string `json:"command"`
			Text        string `json:"text"`
		}
		_ = json.Unmarshal(body, &cmd)
		teamID = cmd.TeamID
		secret, serr := svc.SigningSecretForWorkspace(r.Context(), d.effectiveConfig(), teamID)
		if serr != nil || !bots.VerifySlackSignature(secret, ts, body, sig) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Slack signature.")
			return
		}
		if cmd.Command == "/lextures" || strings.HasPrefix(cmd.Text, "upcoming") {
			text, _ := svc.HandleSlackSlashCommand(r.Context(), d.effectiveConfig(), teamID, cmd.UserID, cmd.ChannelID, cmd.Command+" "+cmd.Text)
			writeJSON(w, http.StatusOK, map[string]any{
				"response_type": "ephemeral",
				"text":          text,
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (d Deps) handleDiscordInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformDiscord) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		inviteURL, err := svc.DiscordInviteURL()
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Discord bot is not configured.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"inviteUrl": inviteURL})
	}
}

func (d Deps) handleDiscordConnect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformDiscord) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Your account is not associated with an organization.")
			return
		}
		var body struct {
			GuildID   string `json:"guildId"`
			GuildName string `json:"guildName"`
			BotToken  string `json:"botToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		conn, err := svc.CompleteDiscordInstall(r.Context(), d.effectiveConfig(), orgID, userID, body.GuildID, body.GuildName, body.BotToken)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"connection": conn})
	}
}

func (d Deps) handleDiscordInteractions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformDiscord) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		ts := r.Header.Get("X-Signature-Timestamp")
		sig := r.Header.Get("X-Signature-Ed25519")
		if !bots.VerifyDiscordSignature(svc.DiscordPublicKey, ts, body, sig) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Discord signature.")
			return
		}
		var interaction struct {
			Type int `json:"type"`
			Data struct {
				Name string `json:"name"`
			} `json:"data"`
			Member struct {
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"member"`
		}
		_ = json.Unmarshal(body, &interaction)
		if interaction.Data.Name == "lextures" || interaction.Data.Name == "upcoming" {
			link, lerr := botsrepo.UserLinkByPlatformUser(r.Context(), d.Pool, botsrepo.PlatformDiscord, interaction.Member.User.ID)
			text := "Link your Lextures account in Settings → Connected Accounts to use this command."
			if lerr == nil && link != nil {
				items, _ := botsrepo.ListUpcomingDueItems(r.Context(), d.Pool, link.UserID, 5, svc.WebOrigin)
				text = bots.UpcomingText(items)
			}
			writeJSON(w, http.StatusOK, discordInteractionResponse(text, true))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (d Deps) handleTeamsMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.botPlatformEnabled(w, botsrepo.PlatformTeams) {
			return
		}
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		if !bots.VerifyTeamsSignature(svc.TeamsAppPassword, r.Header.Get("Authorization")) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid Teams authorization.")
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (d Deps) handleListBotLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		links, err := svc.ListUserLinks(r.Context(), userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load bot links.")
			return
		}
		out := make([]map[string]any, 0, len(links))
		for _, l := range links {
			out = append(out, map[string]any{
				"platform":       l.Platform,
				"platformUserId": l.PlatformUserID,
				"linkedAt":       l.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"links": out})
	}
}

func (d Deps) handleStartBotLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		platform, err := parseBotPlatform(chi.URLParam(r, "platform"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown platform.")
			return
		}
		if !d.botPlatformEnabled(w, platform) {
			return
		}
		authURL, err := svc.LinkUserOAuth(platform, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authorizeUrl": authURL})
	}
}

func (d Deps) handleUnlinkBot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		platform, err := parseBotPlatform(chi.URLParam(r, "platform"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown platform.")
			return
		}
		if err := svc.UnlinkUser(r.Context(), userID, platform); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to unlink account.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleBotLinkSlackCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		if err := svc.CompleteUserLinkSlack(r.Context(), r.URL.Query().Get("code"), r.URL.Query().Get("state")); err != nil {
			http.Redirect(w, r, "/settings/account?error=slack_link_failed", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/settings/account?linked=slack", http.StatusFound)
	}
}

func (d Deps) handleBotLinkDiscordCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc, ok := d.botsEnabled(w)
		if !ok {
			return
		}
		if err := svc.CompleteUserLinkDiscord(r.Context(), r.URL.Query().Get("code"), r.URL.Query().Get("state")); err != nil {
			http.Redirect(w, r, "/settings/account?error=discord_link_failed", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/settings/account?linked=discord", http.StatusFound)
	}
}

func discordInteractionResponse(text string, ephemeral bool) map[string]any {
	flags := 0
	if ephemeral {
		flags = 1 << 6
	}
	return map[string]any{
		"type": 4,
		"data": map[string]any{"content": text, "flags": flags},
	}
}
