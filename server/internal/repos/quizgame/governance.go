package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	GuestJoinDisabled         = "disabled"
	GuestJoinTeacherMediated  = "teacher_mediated"
	GuestJoinOpen             = "open"
	DefaultMaxPlayersPerGame  = 300
	DefaultRetentionDays      = 365
	DefaultGuestRetentionDays = 30
	DefaultMaxConcurrentGames = 50
	DefaultMode               = "live_classic"
	DefaultLeaderboardPrivacy = "names"
)

// PlatformSettings are platform-scoped Live Quiz governance settings (IQ.11).
type PlatformSettings struct {
	MaxConcurrentGames        *int   `json:"maxConcurrentGames"`
	MaxPlayersPerGame         int    `json:"maxPlayersPerGame"`
	MaxKitsPerCourse          *int   `json:"maxKitsPerCourse"`
	RetentionDays             int    `json:"retentionDays"`
	GuestJoinPolicy           string `json:"guestJoinPolicy"`
	DefaultMode               string `json:"defaultMode"`
	DefaultLeaderboardPrivacy string `json:"defaultLeaderboardPrivacy"`
	AIGenerationEnabled       bool   `json:"aiGenerationEnabled"`
	AIGenerationsPerDay       *int   `json:"aiGenerationsPerDay"`
}

// EffectiveSettings are resolved platform + org overrides (org bounded by platform).
type EffectiveSettings struct {
	PlatformSettings
	OrgID string `json:"orgId,omitempty"`
}

// OrgSettingsRow is the stored org override blob.
type OrgSettingsRow struct {
	OrgID     string          `json:"orgId"`
	Overrides json.RawMessage `json:"overrides"`
	UpdatedBy *string         `json:"updatedBy,omitempty"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// DefaultPlatformSettings returns safe defaults when the settings row is missing columns.
func DefaultPlatformSettings() PlatformSettings {
	maxConc := DefaultMaxConcurrentGames
	return PlatformSettings{
		MaxConcurrentGames:        &maxConc,
		MaxPlayersPerGame:         DefaultMaxPlayersPerGame,
		MaxKitsPerCourse:          nil,
		RetentionDays:             DefaultRetentionDays,
		GuestJoinPolicy:           GuestJoinTeacherMediated,
		DefaultMode:               DefaultMode,
		DefaultLeaderboardPrivacy: DefaultLeaderboardPrivacy,
		AIGenerationEnabled:       false,
		AIGenerationsPerDay:       nil,
	}
}

// GetPlatformSettings loads iq_* columns from platform_app_settings.
func GetPlatformSettings(ctx context.Context, pool *pgxpool.Pool) (PlatformSettings, error) {
	out := DefaultPlatformSettings()
	var maxConc, maxKits, aiPerDay *int
	var maxPlayers, retention int
	var guestPolicy, mode, privacy string
	var aiEnabled bool
	err := pool.QueryRow(ctx, `
		SELECT iq_max_concurrent_games, iq_max_players_per_game, iq_max_kits_per_course,
		       iq_retention_days, iq_guest_join_policy, iq_default_mode,
		       iq_default_leaderboard_privacy, iq_ai_generation_enabled, iq_ai_generations_per_day
		FROM settings.platform_app_settings
		ORDER BY updated_at DESC NULLS LAST
		LIMIT 1
	`).Scan(&maxConc, &maxPlayers, &maxKits, &retention, &guestPolicy, &mode, &privacy, &aiEnabled, &aiPerDay)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	out.MaxConcurrentGames = maxConc
	if maxPlayers > 0 {
		out.MaxPlayersPerGame = maxPlayers
	}
	out.MaxKitsPerCourse = maxKits
	if retention > 0 {
		out.RetentionDays = retention
	}
	out.GuestJoinPolicy = normalizeGuestJoinPolicy(guestPolicy)
	out.DefaultMode = normalizeDefaultMode(mode)
	out.DefaultLeaderboardPrivacy = normalizeLeaderboardPrivacy(privacy)
	out.AIGenerationEnabled = aiEnabled
	out.AIGenerationsPerDay = aiPerDay
	return out, nil
}

// PatchPlatformSettingsInput is a partial update for platform IQ settings.
type PatchPlatformSettingsInput struct {
	MaxConcurrentGames        *int
	ClearMaxConcurrentGames   bool
	MaxPlayersPerGame         *int
	MaxKitsPerCourse          *int
	ClearMaxKitsPerCourse     bool
	RetentionDays             *int
	GuestJoinPolicy           *string
	DefaultMode               *string
	DefaultLeaderboardPrivacy *string
	AIGenerationEnabled       *bool
	AIGenerationsPerDay       *int
	ClearAIGenerationsPerDay  bool
}

// PatchPlatformSettings upserts iq_* columns on the singleton platform settings row.
func PatchPlatformSettings(ctx context.Context, pool *pgxpool.Pool, in PatchPlatformSettingsInput) (PlatformSettings, error) {
	cur, err := GetPlatformSettings(ctx, pool)
	if err != nil {
		return PlatformSettings{}, err
	}
	if in.ClearMaxConcurrentGames {
		cur.MaxConcurrentGames = nil
	} else if in.MaxConcurrentGames != nil {
		if *in.MaxConcurrentGames < 0 {
			return PlatformSettings{}, fmt.Errorf("quizgame: maxConcurrentGames must be >= 0")
		}
		v := *in.MaxConcurrentGames
		cur.MaxConcurrentGames = &v
	}
	if in.MaxPlayersPerGame != nil {
		if *in.MaxPlayersPerGame <= 0 {
			return PlatformSettings{}, fmt.Errorf("quizgame: maxPlayersPerGame must be > 0")
		}
		cur.MaxPlayersPerGame = *in.MaxPlayersPerGame
	}
	if in.ClearMaxKitsPerCourse {
		cur.MaxKitsPerCourse = nil
	} else if in.MaxKitsPerCourse != nil {
		if *in.MaxKitsPerCourse < 0 {
			return PlatformSettings{}, fmt.Errorf("quizgame: maxKitsPerCourse must be >= 0")
		}
		v := *in.MaxKitsPerCourse
		cur.MaxKitsPerCourse = &v
	}
	if in.RetentionDays != nil {
		if *in.RetentionDays <= 0 {
			return PlatformSettings{}, fmt.Errorf("quizgame: retentionDays must be > 0")
		}
		cur.RetentionDays = *in.RetentionDays
	}
	if in.GuestJoinPolicy != nil {
		cur.GuestJoinPolicy = normalizeGuestJoinPolicy(*in.GuestJoinPolicy)
		if cur.GuestJoinPolicy == "" {
			return PlatformSettings{}, fmt.Errorf("quizgame: invalid guestJoinPolicy")
		}
	}
	if in.DefaultMode != nil {
		cur.DefaultMode = normalizeDefaultMode(*in.DefaultMode)
		if cur.DefaultMode == "" {
			return PlatformSettings{}, fmt.Errorf("quizgame: invalid defaultMode")
		}
	}
	if in.DefaultLeaderboardPrivacy != nil {
		cur.DefaultLeaderboardPrivacy = normalizeLeaderboardPrivacy(*in.DefaultLeaderboardPrivacy)
		if cur.DefaultLeaderboardPrivacy == "" {
			return PlatformSettings{}, fmt.Errorf("quizgame: invalid defaultLeaderboardPrivacy")
		}
	}
	if in.AIGenerationEnabled != nil {
		cur.AIGenerationEnabled = *in.AIGenerationEnabled
	}
	if in.ClearAIGenerationsPerDay {
		cur.AIGenerationsPerDay = nil
	} else if in.AIGenerationsPerDay != nil {
		if *in.AIGenerationsPerDay < 0 {
			return PlatformSettings{}, fmt.Errorf("quizgame: aiGenerationsPerDay must be >= 0")
		}
		v := *in.AIGenerationsPerDay
		cur.AIGenerationsPerDay = &v
	}

	var maxConc, maxKits, aiPerDay any
	if cur.MaxConcurrentGames != nil {
		maxConc = *cur.MaxConcurrentGames
	}
	if cur.MaxKitsPerCourse != nil {
		maxKits = *cur.MaxKitsPerCourse
	}
	if cur.AIGenerationsPerDay != nil {
		aiPerDay = *cur.AIGenerationsPerDay
	}

	tag, err := pool.Exec(ctx, `
		UPDATE settings.platform_app_settings SET
			iq_max_concurrent_games = $1,
			iq_max_players_per_game = $2,
			iq_max_kits_per_course = $3,
			iq_retention_days = $4,
			iq_guest_join_policy = $5,
			iq_default_mode = $6,
			iq_default_leaderboard_privacy = $7,
			iq_ai_generation_enabled = $8,
			iq_ai_generations_per_day = $9,
			updated_at = NOW()
	`, maxConc, cur.MaxPlayersPerGame, maxKits, cur.RetentionDays, cur.GuestJoinPolicy,
		cur.DefaultMode, cur.DefaultLeaderboardPrivacy, cur.AIGenerationEnabled, aiPerDay)
	if err != nil {
		return PlatformSettings{}, err
	}
	if tag.RowsAffected() == 0 {
		return PlatformSettings{}, fmt.Errorf("quizgame: platform settings row missing")
	}
	return GetPlatformSettings(ctx, pool)
}

// OrgOverrides are optional org-scoped tunables (must stay within platform bounds).
type OrgOverrides struct {
	MaxConcurrentGames        *int    `json:"maxConcurrentGames,omitempty"`
	MaxPlayersPerGame         *int    `json:"maxPlayersPerGame,omitempty"`
	MaxKitsPerCourse          *int    `json:"maxKitsPerCourse,omitempty"`
	RetentionDays             *int    `json:"retentionDays,omitempty"`
	GuestJoinPolicy           *string `json:"guestJoinPolicy,omitempty"`
	DefaultMode               *string `json:"defaultMode,omitempty"`
	DefaultLeaderboardPrivacy *string `json:"defaultLeaderboardPrivacy,omitempty"`
	AIGenerationEnabled       *bool   `json:"aiGenerationEnabled,omitempty"`
	AIGenerationsPerDay       *int    `json:"aiGenerationsPerDay,omitempty"`
}

// GetOrgSettings returns stored org overrides or nil.
func GetOrgSettings(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*OrgSettingsRow, error) {
	var row OrgSettingsRow
	var oid uuid.UUID
	var updatedBy *uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT org_id, overrides, updated_by, updated_at
		FROM quizgame.org_settings
		WHERE org_id = $1
	`, orgID).Scan(&oid, &row.Overrides, &updatedBy, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	row.OrgID = oid.String()
	if updatedBy != nil {
		s := updatedBy.String()
		row.UpdatedBy = &s
	}
	if len(row.Overrides) == 0 {
		row.Overrides = json.RawMessage(`{}`)
	}
	return &row, nil
}

// UpsertOrgSettings stores org overrides after bounding against platform settings.
func UpsertOrgSettings(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, actor *uuid.UUID, ov OrgOverrides) (EffectiveSettings, error) {
	plat, err := GetPlatformSettings(ctx, pool)
	if err != nil {
		return EffectiveSettings{}, err
	}
	bounded, err := BoundOrgOverrides(plat, ov)
	if err != nil {
		return EffectiveSettings{}, err
	}
	raw, err := json.Marshal(bounded)
	if err != nil {
		return EffectiveSettings{}, err
	}
	var actorAny any
	if actor != nil {
		actorAny = *actor
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO quizgame.org_settings (org_id, overrides, updated_by, updated_at)
		VALUES ($1, $2::jsonb, $3, NOW())
		ON CONFLICT (org_id) DO UPDATE SET
			overrides = EXCLUDED.overrides,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
	`, orgID, raw, actorAny)
	if err != nil {
		return EffectiveSettings{}, err
	}
	return ResolveEffectiveSettings(ctx, pool, orgID)
}

// BoundOrgOverrides clamps org values so they cannot exceed platform allowances.
func BoundOrgOverrides(plat PlatformSettings, ov OrgOverrides) (OrgOverrides, error) {
	out := OrgOverrides{}
	if ov.MaxConcurrentGames != nil {
		if *ov.MaxConcurrentGames < 0 {
			return out, fmt.Errorf("quizgame: maxConcurrentGames must be >= 0")
		}
		v := *ov.MaxConcurrentGames
		if plat.MaxConcurrentGames != nil && v > *plat.MaxConcurrentGames {
			v = *plat.MaxConcurrentGames
		}
		out.MaxConcurrentGames = &v
	}
	if ov.MaxPlayersPerGame != nil {
		if *ov.MaxPlayersPerGame <= 0 {
			return out, fmt.Errorf("quizgame: maxPlayersPerGame must be > 0")
		}
		v := *ov.MaxPlayersPerGame
		if v > plat.MaxPlayersPerGame {
			v = plat.MaxPlayersPerGame
		}
		out.MaxPlayersPerGame = &v
	}
	if ov.MaxKitsPerCourse != nil {
		if *ov.MaxKitsPerCourse < 0 {
			return out, fmt.Errorf("quizgame: maxKitsPerCourse must be >= 0")
		}
		v := *ov.MaxKitsPerCourse
		if plat.MaxKitsPerCourse != nil && v > *plat.MaxKitsPerCourse {
			v = *plat.MaxKitsPerCourse
		}
		out.MaxKitsPerCourse = &v
	}
	if ov.RetentionDays != nil {
		if *ov.RetentionDays <= 0 {
			return out, fmt.Errorf("quizgame: retentionDays must be > 0")
		}
		v := *ov.RetentionDays
		if v > plat.RetentionDays {
			v = plat.RetentionDays
		}
		out.RetentionDays = &v
	}
	if ov.GuestJoinPolicy != nil {
		p := normalizeGuestJoinPolicy(*ov.GuestJoinPolicy)
		if p == "" {
			return out, fmt.Errorf("quizgame: invalid guestJoinPolicy")
		}
		// Org cannot open guests beyond platform policy.
		if guestPolicyRank(p) > guestPolicyRank(plat.GuestJoinPolicy) {
			p = plat.GuestJoinPolicy
		}
		out.GuestJoinPolicy = &p
	}
	if ov.DefaultMode != nil {
		m := normalizeDefaultMode(*ov.DefaultMode)
		if m == "" {
			return out, fmt.Errorf("quizgame: invalid defaultMode")
		}
		out.DefaultMode = &m
	}
	if ov.DefaultLeaderboardPrivacy != nil {
		p := normalizeLeaderboardPrivacy(*ov.DefaultLeaderboardPrivacy)
		if p == "" {
			return out, fmt.Errorf("quizgame: invalid defaultLeaderboardPrivacy")
		}
		out.DefaultLeaderboardPrivacy = &p
	}
	if ov.AIGenerationEnabled != nil {
		v := *ov.AIGenerationEnabled && plat.AIGenerationEnabled
		out.AIGenerationEnabled = &v
	}
	if ov.AIGenerationsPerDay != nil {
		if *ov.AIGenerationsPerDay < 0 {
			return out, fmt.Errorf("quizgame: aiGenerationsPerDay must be >= 0")
		}
		v := *ov.AIGenerationsPerDay
		if plat.AIGenerationsPerDay != nil && v > *plat.AIGenerationsPerDay {
			v = *plat.AIGenerationsPerDay
		}
		out.AIGenerationsPerDay = &v
	}
	return out, nil
}

// ResolveEffectiveSettings merges platform + org overrides for an org.
func ResolveEffectiveSettings(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (EffectiveSettings, error) {
	plat, err := GetPlatformSettings(ctx, pool)
	if err != nil {
		return EffectiveSettings{}, err
	}
	eff := EffectiveSettings{PlatformSettings: plat, OrgID: orgID.String()}
	row, err := GetOrgSettings(ctx, pool, orgID)
	if err != nil || row == nil {
		return eff, err
	}
	var ov OrgOverrides
	if err := json.Unmarshal(row.Overrides, &ov); err != nil {
		return eff, nil
	}
	if ov.MaxConcurrentGames != nil {
		eff.MaxConcurrentGames = ov.MaxConcurrentGames
	}
	if ov.MaxPlayersPerGame != nil {
		eff.MaxPlayersPerGame = *ov.MaxPlayersPerGame
	}
	if ov.MaxKitsPerCourse != nil {
		eff.MaxKitsPerCourse = ov.MaxKitsPerCourse
	}
	if ov.RetentionDays != nil {
		eff.RetentionDays = *ov.RetentionDays
	}
	if ov.GuestJoinPolicy != nil {
		eff.GuestJoinPolicy = *ov.GuestJoinPolicy
	}
	if ov.DefaultMode != nil {
		eff.DefaultMode = *ov.DefaultMode
	}
	if ov.DefaultLeaderboardPrivacy != nil {
		eff.DefaultLeaderboardPrivacy = *ov.DefaultLeaderboardPrivacy
	}
	if ov.AIGenerationEnabled != nil {
		eff.AIGenerationEnabled = *ov.AIGenerationEnabled
	}
	if ov.AIGenerationsPerDay != nil {
		eff.AIGenerationsPerDay = ov.AIGenerationsPerDay
	}
	return eff, nil
}

// ResolveEffectiveSettingsForCourse resolves settings for the course's owning org.
func ResolveEffectiveSettingsForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (EffectiveSettings, error) {
	orgID, err := OrgIDForCourse(ctx, pool, courseCode)
	if err != nil {
		return EffectiveSettings{}, err
	}
	if orgID == uuid.Nil {
		plat, perr := GetPlatformSettings(ctx, pool)
		return EffectiveSettings{PlatformSettings: plat}, perr
	}
	return ResolveEffectiveSettings(ctx, pool, orgID)
}

// OrgIDForCourse returns the organization that owns the course.
func OrgIDForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT org_id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("quizgame: course not found")
		}
		return uuid.Nil, err
	}
	return orgID, nil
}

func normalizeGuestJoinPolicy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case GuestJoinDisabled, GuestJoinTeacherMediated, GuestJoinOpen:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func guestPolicyRank(p string) int {
	switch p {
	case GuestJoinOpen:
		return 2
	case GuestJoinTeacherMediated:
		return 1
	default:
		return 0
	}
}

func normalizeDefaultMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "live_classic", "team", "student_paced", "homework":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func normalizeLeaderboardPrivacy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "names", "anon_to_peers", "anonymous":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}
