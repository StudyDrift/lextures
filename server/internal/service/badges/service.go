// Package badges implements competency micro-badge issuance, handles, and verification (plan B1).
package badges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	badgerepo "github.com/lextures/lextures/server/internal/repos/badges"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

const (
	handleMinLen       = 3
	handleMaxLen       = 32
	handleChangeLimit  = 5
	handleChangeWindow = 30 * 24 * time.Hour
)

// ReservedHandles is the in-app fallback list (also seeded in DB).
var ReservedHandles = map[string]struct{}{
	"admin": {}, "api": {}, "verify": {}, "settings": {}, "me": {}, "badges": {},
	"www": {}, "self": {}, "support": {}, "help": {}, "login": {}, "signup": {},
	"null": {}, "undefined": {}, "lextures": {}, "system": {}, "root": {},
	"static": {}, "assets": {}, "achievements": {},
}

var handleRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,30})[a-z0-9]$`)
var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Validation errors returned to HTTP layer.
var (
	ErrInvalidHandle      = errors.New("invalid handle")
	ErrHandleReserved     = errors.New("handle is reserved")
	ErrHandleTaken        = errors.New("handle is taken")
	ErrHandleRateLimited  = errors.New("handle change rate limited")
	ErrInvalidSlug        = errors.New("invalid slug")
	ErrSlugTaken          = errors.New("slug already used in course")
	ErrFeatureDisabled    = errors.New("competency badges are not enabled")
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrMinorNeedsConsent  = errors.New("guardian consent required to make badge page public")
	ErrInvalidInput       = errors.New("invalid input")
)

// ValidateHandleFormat checks charset/length rules (FR-11).
func ValidateHandleFormat(handle string) error {
	h := strings.ToLower(strings.TrimSpace(handle))
	if len(h) < handleMinLen || len(h) > handleMaxLen {
		return ErrInvalidHandle
	}
	if !handleRe.MatchString(h) {
		return ErrInvalidHandle
	}
	if strings.Contains(h, "--") {
		return ErrInvalidHandle
	}
	if _, ok := ReservedHandles[h]; ok {
		return ErrHandleReserved
	}
	return nil
}

// Slugify converts a name into a URL-safe slug.
func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '_' || r == '-':
			if b.Len() > 0 && !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "badge"
	}
	if len(out) > 64 {
		out = out[:64]
		out = strings.Trim(out, "-")
	}
	return out
}

// ValidateSlugFormat checks a definition slug.
func ValidateSlugFormat(slug string) error {
	s := strings.ToLower(strings.TrimSpace(slug))
	if s == "" || len(s) > 64 || !slugRe.MatchString(s) {
		return ErrInvalidSlug
	}
	return nil
}

// CreateDefinition creates a badge definition with auto-slug when needed.
func CreateDefinition(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, in badgerepo.CreateDefinitionInput) (*badgerepo.Definition, error) {
	if !cfg.FFCompetencyBadges {
		return nil, ErrFeatureDisabled
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	in.Name = name
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = uniqueSlug(ctx, pool, in.CourseID, Slugify(name), nil)
	} else {
		if err := ValidateSlugFormat(slug); err != nil {
			return nil, err
		}
		taken, err := badgerepo.SlugExistsInCourse(ctx, pool, in.CourseID, slug, nil)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrSlugTaken
		}
	}
	in.Slug = slug
	if in.OutcomeID != nil {
		ok, err := badgerepo.OutcomeBelongsToCourse(ctx, pool, in.CourseID, *in.OutcomeID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%w: outcome not in course", ErrInvalidInput)
		}
	}
	return badgerepo.CreateDefinition(ctx, pool, in)
}

func uniqueSlug(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, base string, exclude *uuid.UUID) string {
	slug := base
	for i := 0; i < 50; i++ {
		taken, err := badgerepo.SlugExistsInCourse(ctx, pool, courseID, slug, exclude)
		if err != nil || !taken {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i+2)
	}
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8])
}

// AwardParams controls badge awarding.
type AwardParams struct {
	DefinitionID  uuid.UUID
	RecipientIDs  []uuid.UUID
	AwardedBy     *uuid.UUID
	AwardSource   badgerepo.AwardSource
	EvidenceJSON  json.RawMessage
	LearnerNames  map[uuid.UUID]string // optional display names
	DefaultPublic bool
}

// AwardResult is the outcome of awarding to one recipient.
type AwardResult struct {
	RecipientID uuid.UUID              `json:"recipientId"`
	Awarded     *badgerepo.AwardedBadge `json:"award,omitempty"`
	Skipped     bool                   `json:"skipped"`
	Reason      string                 `json:"reason,omitempty"`
}

// Award awards a badge to one or more recipients (idempotent per pair).
func Award(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, p AwardParams) ([]AwardResult, error) {
	if !cfg.FFCompetencyBadges {
		return nil, ErrFeatureDisabled
	}
	def, err := badgerepo.GetDefinitionByID(ctx, pool, p.DefinitionID)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, ErrNotFound
	}
	source := p.AwardSource
	if source == "" {
		source = badgerepo.AwardSourceManual
	}
	now := time.Now().UTC()
	institution := issuerName(cfg)
	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(cfg.PublicWebOrigin, "/")
	achievementID := fmt.Sprintf("%s/achievements/badge/%s", base, def.ID.String())

	results := make([]AwardResult, 0, len(p.RecipientIDs))
	for _, rid := range p.RecipientIDs {
		if rid == uuid.Nil {
			results = append(results, AwardResult{RecipientID: rid, Skipped: true, Reason: "invalid recipient"})
			continue
		}
		existing, err := badgerepo.GetAwardByDefinitionAndRecipient(ctx, pool, def.ID, rid)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			results = append(results, AwardResult{RecipientID: rid, Awarded: existing, Skipped: true, Reason: "already awarded"})
			continue
		}
		learnerName := ""
		if p.LearnerNames != nil {
			learnerName = p.LearnerNames[rid]
		}
		if learnerName == "" {
			learnerName, _ = badgerepo.UserDisplayName(ctx, pool, rid)
		}
		subject := map[string]any{
			"id":   fmt.Sprintf("urn:uuid:user:%s", rid.String()),
			"type": []string{"AchievementSubject"},
			"name": learnerName,
			"achievement": map[string]any{
				"id":          achievementID,
				"type":        []string{"Achievement"},
				"name":        def.Name,
				"description": def.Description,
				"criteria": map[string]any{
					"narrative": def.CriteriaNarrative,
				},
			},
		}
		if len(def.Tags) > 0 {
			if ach, ok := subject["achievement"].(map[string]any); ok {
				ach["tag"] = def.Tags
			}
		}
		vc, err := vcsigning.SignAchievementCredential(subject, institution, key, now)
		if err != nil {
			return nil, err
		}
		vcBytes, err := json.Marshal(vc)
		if err != nil {
			return nil, err
		}
		subjectBytes, err := json.Marshal(subject)
		if err != nil {
			return nil, err
		}
		shareSlug, err := badgerepo.NewShareSlug()
		if err != nil {
			return nil, err
		}
		// Ensure recipient has a badge profile (default handle).
		if _, err := badgerepo.EnsureProfile(ctx, pool, rid, ""); err != nil {
			return nil, err
		}
		isPublic := p.DefaultPublic
		if cfg.BadgesDefaultPublic {
			isPublic = true
		}
		// Minors default private regardless of tenant default.
		if minor, _ := badgerepo.UserIsMinor(ctx, pool, rid); minor {
			isPublic = false
		}
		created, wasNew, err := badgerepo.CreateAward(ctx, pool, badgerepo.CreateAwardInput{
			DefinitionID:   def.ID,
			RecipientID:    rid,
			AwardedBy:      p.AwardedBy,
			AwardSource:    source,
			EvidenceJSON:   p.EvidenceJSON,
			CredentialJSON: subjectBytes,
			Proof:          vcBytes,
			ShareSlug:      shareSlug,
			IsPublic:       isPublic,
			IssuedAt:       now,
		})
		if err != nil {
			return nil, err
		}
		if !wasNew {
			results = append(results, AwardResult{RecipientID: rid, Awarded: created, Skipped: true, Reason: "already awarded"})
			continue
		}
		results = append(results, AwardResult{RecipientID: rid, Awarded: created, Skipped: false})
	}
	return results, nil
}

// MaybeAutoAward awards auto_award definitions for an outcome when mastery is reached.
func MaybeAutoAward(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, studentID, outcomeID uuid.UUID) error {
	if !cfg.FFCompetencyBadges || pool == nil {
		return nil
	}
	ok, err := badgerepo.MasteryReached(ctx, pool, courseID, studentID, outcomeID)
	if err != nil || !ok {
		return err
	}
	defs, err := badgerepo.ListAutoAwardDefinitionsForOutcome(ctx, pool, courseID, outcomeID)
	if err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{
		"outcomeId": outcomeID.String(),
		"courseId":  courseID.String(),
		"trigger":   "sbg_mastery",
	})
	for _, def := range defs {
		_, err := Award(ctx, pool, cfg, AwardParams{
			DefinitionID: def.ID,
			RecipientIDs: []uuid.UUID{studentID},
			AwardSource:  badgerepo.AwardSourceAuto,
			EvidenceJSON: evidence,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// VerifyResult is the public verify endpoint payload.
type VerifyResult struct {
	Verified   bool            `json:"verified"`
	Revoked    bool            `json:"revoked"`
	IssuerDID  string          `json:"issuerDid"`
	Credential json.RawMessage `json:"credential"`
	CheckedAt  string          `json:"checkedAt"`
	Title      string          `json:"title,omitempty"`
	Status     string          `json:"status"`
}

// VerifyShareSlug verifies a badge by share slug (FR-18).
func VerifyShareSlug(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, shareSlug string) (*VerifyResult, error) {
	if !cfg.FFCompetencyBadges {
		return nil, ErrFeatureDisabled
	}
	award, err := badgerepo.GetAwardByShareSlug(ctx, pool, strings.TrimSpace(shareSlug))
	if err != nil {
		return nil, err
	}
	if award == nil {
		return nil, ErrNotFound
	}
	def, err := badgerepo.GetDefinitionByID(ctx, pool, award.DefinitionID)
	if err != nil {
		return nil, err
	}
	title := ""
	if def != nil {
		title = def.Name
	}
	key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
	issuerDID := ""
	if err == nil {
		issuerDID = key.DID
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	if award.Revoked {
		return &VerifyResult{
			Verified:   false,
			Revoked:    true,
			IssuerDID:  issuerDID,
			Credential: award.Proof,
			CheckedAt:  checkedAt,
			Title:      title,
			Status:     "revoked",
		}, nil
	}
	verified := false
	if err == nil {
		var vc map[string]any
		if json.Unmarshal(award.Proof, &vc) == nil {
			ok, verr := vcsigning.VerifyCredential(vc, key.PublicKey)
			if verr == nil {
				verified = ok
			}
		}
	}
	status := "unverified"
	if verified {
		status = "verified"
	}
	return &VerifyResult{
		Verified:   verified,
		Revoked:    false,
		IssuerDID:  issuerDID,
		Credential: award.Proof,
		CheckedAt:  checkedAt,
		Title:      title,
		Status:     status,
	}, nil
}

// AchievementJSON builds the public OB 3.0 Achievement object for a definition.
func AchievementJSON(cfg config.Config, def *badgerepo.Definition) map[string]any {
	base := strings.TrimRight(cfg.PublicWebOrigin, "/")
	out := map[string]any{
		"@context":    "https://purl.imsglobal.org/spec/ob/v3p0/context.json",
		"id":          fmt.Sprintf("%s/achievements/badge/%s", base, def.ID.String()),
		"type":        []string{"Achievement"},
		"name":        def.Name,
		"description": def.Description,
		"criteria": map[string]any{
			"narrative": def.CriteriaNarrative,
		},
	}
	if len(def.Tags) > 0 {
		out["tag"] = def.Tags
	}
	if len(def.AlignmentJSON) > 0 {
		var alignment any
		if json.Unmarshal(def.AlignmentJSON, &alignment) == nil {
			out["alignment"] = alignment
		}
	}
	return out
}

// UpdateHandle sets a learner handle with validation and rate limiting (FR-11/13).
func UpdateHandle(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID, newHandle string) (*badgerepo.BadgeProfile, error) {
	if !cfg.FFCompetencyBadges {
		return nil, ErrFeatureDisabled
	}
	h := strings.ToLower(strings.TrimSpace(newHandle))
	if err := ValidateHandleFormat(h); err != nil {
		return nil, err
	}
	if reserved, err := badgerepo.IsReservedHandle(ctx, pool, h); err != nil {
		return nil, err
	} else if reserved {
		return nil, ErrHandleReserved
	}
	taken, err := badgerepo.IsHandleTaken(ctx, pool, h, &userID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrHandleTaken
	}
	profile, err := badgerepo.EnsureProfile(ctx, pool, userID, "")
	if err != nil {
		return nil, err
	}
	if profile.Handle != nil && strings.EqualFold(*profile.Handle, h) {
		return profile, nil
	}
	// Rate limit: ≤5 changes / 30 days.
	now := time.Now().UTC()
	count := profile.HandleChangeCount30d
	if profile.HandleChangedAt != nil && now.Sub(*profile.HandleChangedAt) > handleChangeWindow {
		count = 0
	}
	if count >= handleChangeLimit {
		return nil, ErrHandleRateLimited
	}
	if profile.Handle != nil && strings.TrimSpace(*profile.Handle) != "" {
		if err := badgerepo.RecordHandleHistory(ctx, pool, userID, *profile.Handle); err != nil {
			return nil, err
		}
	}
	count++
	return badgerepo.UpdateProfile(ctx, pool, userID, badgerepo.UpdateProfileInput{
		Handle:               &h,
		HandleChangeCount30d: &count,
		HandleChangedAt:      &now,
	})
}

// SetPagePublic toggles whole-page public visibility with minor/consent gate (FR-15/19).
func SetPagePublic(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID, pagePublic bool) (*badgerepo.BadgeProfile, error) {
	if !cfg.FFCompetencyBadges {
		return nil, ErrFeatureDisabled
	}
	if pagePublic {
		minor, err := badgerepo.UserIsMinor(ctx, pool, userID)
		if err != nil {
			return nil, err
		}
		if minor {
			ok, err := badgerepo.HasActiveGuardianConsent(ctx, pool, userID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, ErrMinorNeedsConsent
			}
		}
	}
	if _, err := badgerepo.EnsureProfile(ctx, pool, userID, ""); err != nil {
		return nil, err
	}
	return badgerepo.UpdateProfile(ctx, pool, userID, badgerepo.UpdateProfileInput{
		PagePublic: &pagePublic,
	})
}

// PublicPageMeta is safe public profile metadata.
type PublicPageMeta struct {
	Handle          string `json:"handle"`
	DisplayName     string `json:"displayName"`
	PagePublic      bool   `json:"pagePublic"`
	SearchIndexable bool   `json:"searchIndexable"`
	RedirectTo      string `json:"redirectTo,omitempty"`
}

// ResolvePublicPage resolves a handle for public backpack access.
func ResolvePublicPage(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, handle string) (*PublicPageMeta, uuid.UUID, error) {
	if !cfg.FFCompetencyBadges {
		return nil, uuid.Nil, ErrFeatureDisabled
	}
	uid, current, redirected, err := badgerepo.ResolveHandle(ctx, pool, handle)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if uid == uuid.Nil {
		return nil, uuid.Nil, ErrNotFound
	}
	profile, err := badgerepo.GetProfile(ctx, pool, uid)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if profile == nil {
		return nil, uuid.Nil, ErrNotFound
	}
	display := ""
	if profile.HideRealName {
		if profile.Handle != nil {
			display = *profile.Handle
		}
	} else if profile.DisplayNameOverride != nil && strings.TrimSpace(*profile.DisplayNameOverride) != "" {
		display = strings.TrimSpace(*profile.DisplayNameOverride)
	} else {
		display, _ = badgerepo.UserDisplayName(ctx, pool, uid)
	}
	meta := &PublicPageMeta{
		Handle:          current,
		DisplayName:     display,
		PagePublic:      profile.PagePublic,
		SearchIndexable: profile.SearchIndexable,
	}
	if redirected && current != "" && !strings.EqualFold(current, handle) {
		meta.RedirectTo = current
	}
	return meta, uid, nil
}

// LinkedInParamsForAward builds LinkedIn add-to-profile params for a badge.
func LinkedInParamsForAward(cfg config.Config, def *badgerepo.Definition, award *badgerepo.AwardedBadge) credsvc.LinkedInParams {
	base := strings.TrimRight(cfg.PublicWebOrigin, "/")
	verifyURL := fmt.Sprintf("%s/api/v1/badges/verify/%s", base, award.ShareSlug)
	return credsvc.BuildLinkedInParams(def.Name, issuerName(cfg), verifyURL, award.ShareSlug, award.IssuedAt)
}

// BadgeExportToken reuses credential HMAC token pattern for awarded badge id.
func BadgeExportToken(cfg config.Config, awardID uuid.UUID, now time.Time) (string, time.Time, error) {
	return credsvc.BadgeExportToken(cfg, awardID, now)
}

// VerifyBadgeExportToken validates an export token.
func VerifyBadgeExportToken(cfg config.Config, token string, now time.Time) (uuid.UUID, error) {
	return credsvc.VerifyBadgeExportToken(cfg, token, now)
}

func issuerName(cfg config.Config) string {
	if strings.TrimSpace(cfg.CCRInstitutionName) != "" {
		return strings.TrimSpace(cfg.CCRInstitutionName)
	}
	return "Lextures"
}
