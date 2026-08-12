package marketingcontent

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UpsertCategory(ctx context.Context, tx pgx.Tx, c Category, actor uuid.UUID) (*Category, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Locale == "" {
		c.Locale = "en"
	}
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_categories (id,slug,locale,title,description,sort_order,platform_path,created_by,updated_by)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) ON CONFLICT (locale,slug) DO UPDATE SET title=EXCLUDED.title,description=EXCLUDED.description,sort_order=EXCLUDED.sort_order,platform_path=EXCLUDED.platform_path,updated_by=EXCLUDED.updated_by
	 RETURNING id,slug,locale,title,description,sort_order,platform_path,created_by,updated_by,created_at,updated_at`, c.ID, c.Slug, c.Locale, c.Title, c.Description, c.SortOrder, c.PlatformPath, actor).Scan(&c.ID, &c.Slug, &c.Locale, &c.Title, &c.Description, &c.SortOrder, &c.PlatformPath, &c.CreatedBy, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func ListCategories(ctx context.Context, q querier, locale string) ([]Category, error) {
	rows, err := q.Query(ctx, `SELECT id,slug,locale,title,description,sort_order,platform_path,created_by,updated_by,created_at,updated_at FROM marketing.content_categories WHERE ($1='' OR locale=$1) ORDER BY sort_order,slug`, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Locale, &c.Title, &c.Description, &c.SortOrder, &c.PlatformPath, &c.CreatedBy, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func UpsertAuthor(ctx context.Context, tx pgx.Tx, a Author, actor uuid.UUID) (*Author, error) {
	if a.KnowsAbout == nil {
		a.KnowsAbout = []string{}
	}
	if len(a.Links) == 0 {
		a.Links = json.RawMessage(`{}`)
	}
	if a.Status == "" {
		a.Status = "active"
	}
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_authors (slug,name,job_title,bio,knows_about,image_media_id,links,user_id,status,created_by,updated_by)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10) ON CONFLICT (slug) DO UPDATE SET name=EXCLUDED.name,job_title=EXCLUDED.job_title,bio=EXCLUDED.bio,knows_about=EXCLUDED.knows_about,image_media_id=EXCLUDED.image_media_id,links=EXCLUDED.links,user_id=EXCLUDED.user_id,status=EXCLUDED.status,updated_by=EXCLUDED.updated_by
	 RETURNING slug,name,job_title,bio,knows_about,image_media_id,links,user_id,status,created_by,updated_by,created_at,updated_at`, a.Slug, a.Name, a.JobTitle, a.Bio, a.KnowsAbout, a.ImageMediaID, a.Links, a.UserID, a.Status, actor).Scan(&a.Slug, &a.Name, &a.JobTitle, &a.Bio, &a.KnowsAbout, &a.ImageMediaID, &a.Links, &a.UserID, &a.Status, &a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func ListAuthors(ctx context.Context, q querier) ([]Author, error) {
	rows, err := q.Query(ctx, `SELECT slug,name,job_title,bio,knows_about,image_media_id,links,user_id,status,created_by,updated_by,created_at,updated_at FROM marketing.content_authors ORDER BY name,slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Author
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.Slug, &a.Name, &a.JobTitle, &a.Bio, &a.KnowsAbout, &a.ImageMediaID, &a.Links, &a.UserID, &a.Status, &a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func InsertRedirect(ctx context.Context, tx pgx.Tx, r Redirect, actor uuid.UUID) (*Redirect, error) {
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_redirects (from_path,to_path,status_code,source,article_id,created_by,updated_by) VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id,from_path,to_path,status_code,source,article_id,created_by,updated_by,created_at,updated_at`, r.FromPath, r.ToPath, r.StatusCode, r.Source, r.ArticleID, actor).Scan(&r.ID, &r.FromPath, &r.ToPath, &r.StatusCode, &r.Source, &r.ArticleID, &r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt)
	return &r, err
}

func ListRedirects(ctx context.Context, q querier) ([]Redirect, error) {
	rows, err := q.Query(ctx, `SELECT id,from_path,to_path,status_code,source,article_id,created_by,updated_by,created_at,updated_at FROM marketing.content_redirects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Redirect
	for rows.Next() {
		var r Redirect
		if err := rows.Scan(&r.ID, &r.FromPath, &r.ToPath, &r.StatusCode, &r.Source, &r.ArticleID, &r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func DeleteRedirect(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM marketing.content_redirects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func DeleteCategory(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return deleteCatalogRow(ctx, tx, `DELETE FROM marketing.content_categories WHERE id=$1`, id)
}
func DeleteTag(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return deleteCatalogRow(ctx, tx, `DELETE FROM marketing.content_tags WHERE id=$1`, id)
}
func deleteCatalogRow(ctx context.Context, tx pgx.Tx, sql string, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, sql, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func UpsertTag(ctx context.Context, tx pgx.Tx, t Tag, actor uuid.UUID) (*Tag, error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_tags (id,slug,label,created_by,updated_by) VALUES ($1,$2,$3,$4,$4) ON CONFLICT (slug) DO UPDATE SET label=EXCLUDED.label,updated_by=EXCLUDED.updated_by RETURNING id,slug,label,created_by,updated_by,created_at,updated_at`, t.ID, t.Slug, t.Label, actor).Scan(&t.ID, &t.Slug, &t.Label, &t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}
func ListTags(ctx context.Context, q querier) ([]Tag, error) {
	rows, err := q.Query(ctx, `SELECT id,slug,label,created_by,updated_by,created_at,updated_at FROM marketing.content_tags ORDER BY label,slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Slug, &t.Label, &t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
