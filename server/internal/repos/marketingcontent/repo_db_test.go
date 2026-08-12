package marketingcontent

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestArticleRevisionConflictAndSlugRedirectDB(t *testing.T) {
	if testing.Short() {
		t.Skip("database integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actor uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	suffix := uuid.NewString()
	_, err = UpsertAuthor(ctx, tx, Author{Slug: "author-" + suffix, Name: "DB Test", Status: "active"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Now().UTC()
	created, err := InsertArticle(ctx, tx, NewArticle{
		Kind: "blog", Slug: "before-" + suffix, Locale: "en", Title: "Before",
		BodyMD: "revision one", Status: "published", AuthorSlug: "author-" + suffix,
		PublishedAt: &publishedAt, ActorID: actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.RevisionNo != 1 {
		t.Fatalf("initial revision = %d, want 1", created.RevisionNo)
	}

	update := NewArticle{
		Kind: "blog", Slug: "after-" + suffix, Locale: "en", Title: "After",
		BodyMD: "revision two", Status: "published", AuthorSlug: "author-" + suffix,
		PublishedAt: &publishedAt, TranslationGroupID: created.TranslationGroupID, ActorID: actor,
	}
	updated, err := UpdateArticle(ctx, tx, ArticleUpdate{ID: created.ID, ExpectedRevisionNo: 1, Article: update})
	if err != nil {
		t.Fatal(err)
	}
	if updated.RevisionNo != 2 {
		t.Fatalf("updated revision = %d, want 2", updated.RevisionNo)
	}
	if _, err := UpdateArticle(ctx, tx, ArticleUpdate{ID: created.ID, ExpectedRevisionNo: 1, Article: update}); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale update error = %v, want ErrRevisionConflict", err)
	}
	var revisions, redirects int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM marketing.content_revisions WHERE article_id=$1`, created.ID).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM marketing.content_redirects WHERE article_id=$1 AND from_path=$2 AND to_path=$3`, created.ID, created.Path, updated.Path).Scan(&redirects); err != nil {
		t.Fatal(err)
	}
	if revisions != 2 || redirects != 1 {
		t.Fatalf("revisions=%d redirects=%d, want 2 and 1", revisions, redirects)
	}
}

func TestPublicContentProjectionVisibilityAndSearchDB(t *testing.T) {
	if testing.Short() {
		t.Skip("database integration test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actor uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM "user".users LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	suffix := uuid.NewString()
	authorSlug := "public-author-" + suffix
	if _, err := UpsertAuthor(ctx, tx, Author{Slug: authorSlug, Name: "Public Author", Status: "active"}, actor); err != nil {
		t.Fatal(err)
	}
	category, err := UpsertCategory(ctx, tx, Category{Slug: "category-" + suffix, Locale: "en", Title: "Category"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	published, err := InsertArticle(ctx, tx, NewArticle{Kind: "doc", Slug: "published-" + suffix, Locale: "en", Title: "Rubric guide", Description: "Public description", BodyMD: "A rubric helps assessment.", Status: "published", AuthorSlug: authorSlug, CategoryID: &category.ID, PublishedAt: &now, ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	draft, err := InsertArticle(ctx, tx, NewArticle{Kind: "doc", Slug: "draft-" + suffix, Locale: "en", Title: "Secret rubric roadmap", BodyMD: "unpublished rubric", Status: "draft", AuthorSlug: authorSlug, CategoryID: &category.ID, ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}

	items, _, err := ListPublishedArticles(ctx, tx, PublicArticleFilter{CategorySlug: category.Slug, Q: "rubric", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != published.ID || items[0].BodyMD != published.BodyMD || items[0].CategorySlug == nil || *items[0].CategorySlug != category.Slug {
		t.Fatalf("unexpected public projection: %#v", items)
	}
	if _, err := GetPublishedArticleByPath(ctx, tx, draft.Path); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("draft public lookup error = %v, want pgx.ErrNoRows", err)
	}
	results, err := SearchPublished(ctx, tx, "rubric", "doc", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Path == draft.Path {
			t.Fatal("draft leaked into public search")
		}
	}
	found := false
	for _, result := range results {
		if result.Path == published.Path {
			found = strings.Contains(result.Snippet, "<mark>")
		}
	}
	if !found {
		t.Fatalf("published result missing highlighted snippet: %#v", results)
	}
}
