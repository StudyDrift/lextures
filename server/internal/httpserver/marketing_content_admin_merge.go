package httpserver

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

type marketingArticleBody struct {
	Kind               string     `json:"kind"`
	Slug               string     `json:"slug"`
	Locale             string     `json:"locale"`
	CategoryID         *uuid.UUID `json:"categoryId"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	BodyMD             string     `json:"bodyMd"`
	AuthorSlug         string     `json:"authorSlug"`
	ReviewerSlug       *string    `json:"reviewerSlug"`
	PrimaryQuestion    string     `json:"primaryQuestion"`
	Cluster            string     `json:"cluster"`
	Pillar             string     `json:"pillar"`
	BriefRef           string     `json:"briefRef"`
	VerifiedAgainst    string     `json:"verifiedAgainst"`
	ReviewDueOn        *time.Time `json:"reviewDueOn"`
	Keywords           []string   `json:"keywords"`
	RelatedTo          []string   `json:"relatedTo"`
	Roles              []string   `json:"roles"`
	Segments           []string   `json:"segments"`
	Citations          []string   `json:"citations"`
	HeroMediaID        *uuid.UUID `json:"heroMediaId"`
	SocialTitle        string     `json:"socialTitle"`
	SocialDescription  string     `json:"socialDescription"`
	Noindex            bool       `json:"noindex"`
	CanonicalOverride  *string    `json:"canonicalOverride"`
	ExpectedRevisionNo int        `json:"expectedRevisionNo"`
	ChangeNote         string     `json:"changeNote"`
}

func (b marketingArticleBody) input(actor uuid.UUID) mcrepo.NewArticle {
	return mcrepo.NewArticle{Kind: b.Kind, Slug: b.Slug, Locale: b.Locale, CategoryID: b.CategoryID, Title: b.Title, Description: b.Description, BodyMD: b.BodyMD, AuthorSlug: b.AuthorSlug, ReviewerSlug: b.ReviewerSlug, ReviewDueOn: b.ReviewDueOn, PrimaryQuestion: b.PrimaryQuestion, Cluster: b.Cluster, Pillar: b.Pillar, BriefRef: b.BriefRef, VerifiedAgainst: b.VerifiedAgainst, Keywords: b.Keywords, RelatedTo: b.RelatedTo, Roles: b.Roles, Segments: b.Segments, Citations: b.Citations, HeroMediaID: b.HeroMediaID, SocialTitle: b.SocialTitle, SocialDescription: b.SocialDescription, Noindex: b.Noindex, CanonicalOverride: b.CanonicalOverride, ActorID: actor, ChangeNote: b.ChangeNote}
}

func validateArticleBody(b marketingArticleBody) error {
	if b.Kind != "blog" && b.Kind != "doc" {
		return errors.New("kind must be blog or doc")
	}
	if strings.TrimSpace(b.Slug) == "" || strings.ContainsAny(b.Slug, " /_") {
		return errors.New("slug must be non-empty kebab-case")
	}
	if b.Kind == "doc" && b.CategoryID == nil {
		return errors.New("categoryId is required for doc articles")
	}
	if b.Title == "" || b.AuthorSlug == "" {
		return errors.New("title and authorSlug are required")
	}
	return nil
}

func mergeMarketingArticle(a *mcrepo.Article, b marketingArticleBody, raw map[string]json.RawMessage, actor uuid.UUID) mcrepo.NewArticle {
	in := mcArticleInput(a, actor)
	if _, ok := raw["kind"]; ok {
		in.Kind = b.Kind
	}
	if _, ok := raw["slug"]; ok {
		in.Slug = b.Slug
	}
	if _, ok := raw["categoryId"]; ok {
		in.CategoryID = b.CategoryID
	}
	if _, ok := raw["title"]; ok {
		in.Title = b.Title
	}
	if _, ok := raw["description"]; ok {
		in.Description = b.Description
	}
	if _, ok := raw["bodyMd"]; ok {
		in.BodyMD = b.BodyMD
	}
	if _, ok := raw["authorSlug"]; ok {
		in.AuthorSlug = b.AuthorSlug
	}
	if _, ok := raw["reviewerSlug"]; ok {
		in.ReviewerSlug = b.ReviewerSlug
	}
	if _, ok := raw["reviewDueOn"]; ok {
		in.ReviewDueOn = b.ReviewDueOn
	}
	if _, ok := raw["locale"]; ok {
		in.Locale = b.Locale
	}
	if _, ok := raw["primaryQuestion"]; ok {
		in.PrimaryQuestion = b.PrimaryQuestion
	}
	if _, ok := raw["cluster"]; ok {
		in.Cluster = b.Cluster
	}
	if _, ok := raw["pillar"]; ok {
		in.Pillar = b.Pillar
	}
	if _, ok := raw["briefRef"]; ok {
		in.BriefRef = b.BriefRef
	}
	if _, ok := raw["verifiedAgainst"]; ok {
		in.VerifiedAgainst = b.VerifiedAgainst
	}
	if _, ok := raw["heroMediaId"]; ok {
		in.HeroMediaID = b.HeroMediaID
	}
	if _, ok := raw["socialTitle"]; ok {
		in.SocialTitle = b.SocialTitle
	}
	if _, ok := raw["socialDescription"]; ok {
		in.SocialDescription = b.SocialDescription
	}
	if _, ok := raw["keywords"]; ok {
		in.Keywords = b.Keywords
	}
	if _, ok := raw["relatedTo"]; ok {
		in.RelatedTo = b.RelatedTo
	}
	if _, ok := raw["roles"]; ok {
		in.Roles = b.Roles
	}
	if _, ok := raw["segments"]; ok {
		in.Segments = b.Segments
	}
	if _, ok := raw["citations"]; ok {
		in.Citations = b.Citations
	}
	if _, ok := raw["noindex"]; ok {
		in.Noindex = b.Noindex
	}
	if _, ok := raw["canonicalOverride"]; ok {
		in.CanonicalOverride = b.CanonicalOverride
	}
	in.ChangeNote = b.ChangeNote
	return in
}

func mcArticleInput(a *mcrepo.Article, actor uuid.UUID) mcrepo.NewArticle {
	return mcrepo.NewArticle{Kind: a.Kind, Slug: a.Slug, Locale: a.Locale, TranslationGroupID: a.TranslationGroupID, CategoryID: a.CategoryID, Title: a.Title, Description: a.Description, BodyMD: a.BodyMD, Status: a.Status, AuthorSlug: a.AuthorSlug, ReviewerSlug: a.ReviewerSlug, PublishedAt: a.PublishedAt, FirstPublishedAt: a.FirstPublishedAt, ScheduledFor: a.ScheduledFor, ContentUpdatedAt: a.ContentUpdatedAt, ReviewedAt: a.ReviewedAt, ReviewDueOn: a.ReviewDueOn, PrimaryQuestion: a.PrimaryQuestion, Cluster: a.Cluster, Pillar: a.Pillar, BriefRef: a.BriefRef, VerifiedAgainst: a.VerifiedAgainst, Keywords: a.Keywords, RelatedTo: a.RelatedTo, Roles: a.Roles, Segments: a.Segments, Citations: a.Citations, HeroMediaID: a.HeroMediaID, SocialTitle: a.SocialTitle, SocialDescription: a.SocialDescription, QualityScore: a.QualityScore, QualityReport: a.QualityReport, Noindex: a.Noindex, CanonicalOverride: a.CanonicalOverride, Extra: a.Extra, ActorID: actor}
}
