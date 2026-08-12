// Package marketingcontent is the sole SQL boundary for database-backed marketing articles.
package marketingcontent

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRevisionConflict = errors.New("marketingcontent: revision conflict")
	ErrDuplicateSlug    = errors.New("marketingcontent: duplicate slug or path")
)

type Article struct {
	ID                 uuid.UUID       `json:"id"`
	Kind               string          `json:"kind"`
	Slug               string          `json:"slug"`
	Locale             string          `json:"locale"`
	Path               string          `json:"path"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	BodyMD             string          `json:"bodyMd"`
	Status             string          `json:"status"`
	TranslationGroupID uuid.UUID       `json:"translationGroupId"`
	CategoryID         *uuid.UUID      `json:"categoryId"`
	AuthorSlug         string          `json:"authorSlug"`
	ReviewerSlug       *string         `json:"reviewerSlug"`
	PublishedAt        *time.Time      `json:"publishedAt"`
	FirstPublishedAt   *time.Time      `json:"firstPublishedAt"`
	ScheduledFor       *time.Time      `json:"scheduledFor"`
	ContentUpdatedAt   *time.Time      `json:"contentUpdatedAt"`
	ReviewedAt         *time.Time      `json:"reviewedAt"`
	ReviewDueOn        *time.Time      `json:"reviewDueOn"`
	PrimaryQuestion    string          `json:"primaryQuestion"`
	Cluster            string          `json:"cluster"`
	Pillar             string          `json:"pillar"`
	BriefRef           string          `json:"briefRef"`
	VerifiedAgainst    string          `json:"verifiedAgainst"`
	Keywords           []string        `json:"keywords"`
	RelatedTo          []string        `json:"relatedTo"`
	Roles              []string        `json:"roles"`
	Segments           []string        `json:"segments"`
	Citations          []string        `json:"citations"`
	HeroMediaID        *uuid.UUID      `json:"heroMediaId"`
	QualityScore       *float64        `json:"qualityScore"`
	QualityReport      json.RawMessage `json:"qualityReport"`
	Noindex            bool            `json:"noindex"`
	CanonicalOverride  *string         `json:"canonicalOverride"`
	Extra              json.RawMessage `json:"extra"`
	RevisionNo         int             `json:"revisionNo"`
	CreatedBy          *uuid.UUID      `json:"createdBy"`
	UpdatedBy          *uuid.UUID      `json:"updatedBy"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	DeletedAt          *time.Time      `json:"deletedAt,omitempty"`
}

type ArticleSummary struct {
	ID          uuid.UUID  `json:"id"`
	Kind        string     `json:"kind"`
	Slug        string     `json:"slug"`
	Locale      string     `json:"locale"`
	Path        string     `json:"path"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	AuthorSlug  string     `json:"authorSlug"`
	CategoryID  *uuid.UUID `json:"categoryId"`
	PublishedAt *time.Time `json:"publishedAt"`
	RevisionNo  int        `json:"revisionNo"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type ArticleFilter struct {
	Kind, Status, Locale, CategorySlug, Q, Cursor string
	Limit                                         int
}

// PublicArticleFilter is deliberately status-free: public callers cannot widen
// the repository's published-only visibility predicate.
type PublicArticleFilter struct {
	Kind, Locale, CategorySlug, AuthorSlug, Tag, Q, Cursor string
	Limit                                                  int
}

type NewArticle struct {
	Kind, Slug, Locale, Title, Description, BodyMD, Status        string
	TranslationGroupID                                            uuid.UUID
	CategoryID                                                    *uuid.UUID
	AuthorSlug                                                    string
	ReviewerSlug                                                  *string
	PublishedAt, FirstPublishedAt, ScheduledFor, ContentUpdatedAt *time.Time
	ReviewedAt, ReviewDueOn                                       *time.Time
	PrimaryQuestion, Cluster, Pillar, BriefRef, VerifiedAgainst   string
	Keywords, RelatedTo, Roles, Segments, Citations               []string
	HeroMediaID                                                   *uuid.UUID
	QualityScore                                                  *float64
	QualityReport                                                 json.RawMessage
	Noindex                                                       bool
	CanonicalOverride                                             *string
	Extra                                                         json.RawMessage
	ActorID                                                       uuid.UUID
	ChangeNote                                                    string
}

type ArticleUpdate struct {
	ID                 uuid.UUID
	ExpectedRevisionNo int
	Article            NewArticle
}

type Revision struct {
	ID          uuid.UUID       `json:"id"`
	ArticleID   uuid.UUID       `json:"articleId"`
	RevisionNo  int             `json:"revisionNo"`
	BodyMD      string          `json:"bodyMd"`
	Metadata    json.RawMessage `json:"metadata"`
	ChangeNote  string          `json:"changeNote"`
	StatusAfter string          `json:"statusAfter"`
	ActorID     *uuid.UUID      `json:"actorId"`
	CreatedAt   time.Time       `json:"createdAt"`
}

type Category struct {
	ID           uuid.UUID  `json:"id"`
	Slug         string     `json:"slug"`
	Locale       string     `json:"locale"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	PlatformPath string     `json:"platformPath"`
	SortOrder    int        `json:"sortOrder"`
	CreatedBy    *uuid.UUID `json:"createdBy"`
	UpdatedBy    *uuid.UUID `json:"updatedBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Author struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	JobTitle     string          `json:"jobTitle"`
	Bio          string          `json:"bio"`
	Status       string          `json:"status"`
	KnowsAbout   []string        `json:"knowsAbout"`
	ImageMediaID *uuid.UUID      `json:"imageMediaId"`
	UserID       *uuid.UUID      `json:"userId"`
	Links        json.RawMessage `json:"links"`
	CreatedBy    *uuid.UUID      `json:"createdBy"`
	UpdatedBy    *uuid.UUID      `json:"updatedBy"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type Redirect struct {
	ID         uuid.UUID  `json:"id"`
	FromPath   string     `json:"from"`
	ToPath     string     `json:"to"`
	Source     string     `json:"source"`
	StatusCode int        `json:"statusCode"`
	ArticleID  *uuid.UUID `json:"articleId"`
	CreatedBy  *uuid.UUID `json:"createdBy"`
	UpdatedBy  *uuid.UUID `json:"updatedBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Tag struct {
	ID        uuid.UUID  `json:"id"`
	Slug      string     `json:"slug"`
	Label     string     `json:"label"`
	CreatedBy *uuid.UUID `json:"createdBy"`
	UpdatedBy *uuid.UUID `json:"updatedBy"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}
