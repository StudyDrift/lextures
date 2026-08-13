package marketingcontent

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func UpdateArticle(ctx context.Context, tx pgx.Tx, u ArticleUpdate) (*Article, error) {
	in := u.Article
	in = normalizeArticleInput(in)
	var oldPath, oldStatus, oldLocale string
	if err := tx.QueryRow(ctx, `SELECT path,status,locale FROM marketing.content_articles WHERE id=$1 AND deleted_at IS NULL`, u.ID).Scan(&oldPath, &oldStatus, &oldLocale); err != nil {
		return nil, err
	}
	if oldLocale == "" {
		oldLocale = "en"
	}
	in.Locale = oldLocale
	path, err := articlePath(ctx, tx, in.Kind, in.Locale, in.Slug, in.CategoryID)
	if err != nil {
		return nil, err
	}
	a, err := scanArticle(tx.QueryRow(ctx, `UPDATE marketing.content_articles SET
	 kind=$3,slug=$4,category_id=$5,path=$6,title=$7,description=$8,
	 body_md=$9,status=$10,author_slug=$11,reviewer_slug=$12,published_at=$13,
	 first_published_at=COALESCE(first_published_at,$14),scheduled_for=$15,content_updated_at=$16,
	 reviewed_at=$17,review_due_on=$18,primary_question=$19,cluster=$20,pillar=$21,brief_ref=$22,
	 verified_against=$23,keywords=$24,related_to=$25,roles=$26,segments=$27,citations=$28,
	 hero_media_id=$29,quality_score=$30,quality_report=$31,noindex=$32,canonical_override=$33,
	 extra=$34,source_article_id=COALESCE($35,source_article_id),source_synced_revision=COALESCE($36,source_synced_revision),source_synced_at=COALESCE($37,source_synced_at),revision_no=revision_no+1,updated_by=$38
	 WHERE id=$1 AND revision_no=$2 AND deleted_at IS NULL RETURNING `+articleColumns,
		u.ID, u.ExpectedRevisionNo, in.Kind, in.Slug, in.CategoryID, path, in.Title, in.Description, in.BodyMD, in.Status, in.AuthorSlug, in.ReviewerSlug, in.PublishedAt, in.FirstPublishedAt, in.ScheduledFor, in.ContentUpdatedAt, in.ReviewedAt, in.ReviewDueOn, in.PrimaryQuestion, in.Cluster, in.Pillar, in.BriefRef, in.VerifiedAgainst, in.Keywords, in.RelatedTo, in.Roles, in.Segments, in.Citations, in.HeroMediaID, in.QualityScore, in.QualityReport, in.Noindex, in.CanonicalOverride, in.Extra, in.SourceArticleID, in.SourceSyncedRevision, in.SourceSyncedAt, nullUUID(in.ActorID)))
	if err == pgx.ErrNoRows {
		return nil, ErrRevisionConflict
	}
	if err != nil {
		return nil, mapConstraint(err)
	}
	if oldPath != a.Path && (oldStatus == "published" || a.Status == "published") {
		_, err = tx.Exec(ctx, `INSERT INTO marketing.content_redirects (from_path,to_path,status_code,source,article_id,created_by,updated_by)
		 VALUES ($1,$2,301,'slug_change',$3,$4,$4)
		 ON CONFLICT (from_path) DO UPDATE SET to_path=EXCLUDED.to_path,status_code=301,source='slug_change',article_id=EXCLUDED.article_id,updated_by=EXCLUDED.updated_by`, oldPath, a.Path, a.ID, in.ActorID)
		if err != nil {
			return nil, err
		}
	}
	if err := insertRevision(ctx, tx, a, in.ChangeNote, in.ActorID); err != nil {
		return nil, err
	}
	return a, nil
}

func SoftDeleteArticle(ctx context.Context, tx pgx.Tx, id, actor uuid.UUID) error {
	tag, err := tx.Exec(ctx, `UPDATE marketing.content_articles SET deleted_at=now(),updated_by=$2,revision_no=revision_no+1 WHERE id=$1 AND deleted_at IS NULL`, id, actor)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	a, err := scanArticle(tx.QueryRow(ctx, `SELECT `+articleColumns+` FROM marketing.content_articles WHERE id=$1`, id))
	if err != nil {
		return err
	}
	return insertRevision(ctx, tx, a, "Soft deleted", actor)
}

func insertRevision(ctx context.Context, tx pgx.Tx, a *Article, note string, actor uuid.UUID) error {
	metadata, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO marketing.content_revisions (article_id,revision_no,body_md,metadata,change_note,status_after,actor_id) VALUES ($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.RevisionNo, a.BodyMD, metadata, note, a.Status, nullUUID(actor))
	return err
}

func ListRevisions(ctx context.Context, q querier, articleID uuid.UUID, limit int) ([]Revision, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	rows, err := q.Query(ctx, `SELECT id,article_id,revision_no,body_md,metadata,change_note,status_after,actor_id,created_at FROM marketing.content_revisions WHERE article_id=$1 ORDER BY revision_no DESC LIMIT $2`, articleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Revision, 0, limit)
	for rows.Next() {
		var r Revision
		if err := rows.Scan(&r.ID, &r.ArticleID, &r.RevisionNo, &r.BodyMD, &r.Metadata, &r.ChangeNote, &r.StatusAfter, &r.ActorID, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetRevision(ctx context.Context, q querier, articleID uuid.UUID, no int) (*Revision, error) {
	var r Revision
	err := q.QueryRow(ctx, `SELECT id,article_id,revision_no,body_md,metadata,change_note,status_after,actor_id,created_at FROM marketing.content_revisions WHERE article_id=$1 AND revision_no=$2`, articleID, no).Scan(&r.ID, &r.ArticleID, &r.RevisionNo, &r.BodyMD, &r.Metadata, &r.ChangeNote, &r.StatusAfter, &r.ActorID, &r.CreatedAt)
	return &r, err
}
