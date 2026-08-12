package marketingcontent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MediaRendition struct {
	Name   string `json:"name"`
	Ext    string `json:"ext"`
	MIME   string `json:"mime"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Key    string `json:"-"`
	URL    string `json:"url"`
	Bytes  int64  `json:"bytes"`
}

type MediaAsset struct {
	ID         uuid.UUID        `json:"id"`
	Checksum   string           `json:"checksum"`
	MIMEType   string           `json:"mimeType"`
	ByteSize   int64            `json:"byteSize"`
	Width      *int             `json:"width"`
	Height     *int             `json:"height"`
	AltText    string           `json:"altText"`
	Decorative bool             `json:"decorative"`
	Title      string           `json:"title"`
	Credit     string           `json:"credit"`
	StorageKey string           `json:"-"`
	Renditions []MediaRendition `json:"renditions"`
	UploadedBy *uuid.UUID       `json:"uploadedBy"`
	CreatedAt  time.Time        `json:"createdAt"`
	UsageCount int              `json:"usageCount"`
	UsedBy     []MediaUsage     `json:"usedBy,omitempty"`
}
type MediaUsage struct {
	ArticleID uuid.UUID `json:"articleId"`
	Path      string    `json:"path"`
	Usage     string    `json:"usage"`
}
type MediaFilter struct {
	Q, MIMEType, Cursor string
	UnusedOnly          bool
	Limit               int
}

type mediaStore interface {
	querier
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const mediaSelect = `m.id,m.checksum,m.mime_type,m.byte_size,m.width,m.height,m.alt_text,m.decorative,m.title,m.credit,m.storage_key,m.renditions,m.uploaded_by,m.created_at,
 (SELECT count(*) FROM marketing.content_article_media am JOIN marketing.content_articles a ON a.id=am.article_id WHERE am.media_id=m.id AND a.deleted_at IS NULL)`

func scanMedia(row pgx.Row) (*MediaAsset, error) {
	var m MediaAsset
	var raw []byte
	if err := row.Scan(&m.ID, &m.Checksum, &m.MIMEType, &m.ByteSize, &m.Width, &m.Height, &m.AltText, &m.Decorative, &m.Title, &m.Credit, &m.StorageKey, &raw, &m.UploadedBy, &m.CreatedAt, &m.UsageCount); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &m.Renditions); err != nil {
		return nil, err
	}
	return &m, nil
}

func GetMediaByChecksum(ctx context.Context, q querier, checksum string) (*MediaAsset, error) {
	return scanMedia(q.QueryRow(ctx, `SELECT `+mediaSelect+` FROM marketing.content_media m WHERE checksum=$1 AND deleted_at IS NULL`, checksum))
}
func GetMedia(ctx context.Context, q querier, id uuid.UUID) (*MediaAsset, error) {
	m, err := scanMedia(q.QueryRow(ctx, `SELECT `+mediaSelect+` FROM marketing.content_media m WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT a.id,a.path,am.usage FROM marketing.content_article_media am JOIN marketing.content_articles a ON a.id=am.article_id WHERE am.media_id=$1 AND a.deleted_at IS NULL ORDER BY a.path`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u MediaUsage
		if err = rows.Scan(&u.ArticleID, &u.Path, &u.Usage); err != nil {
			return nil, err
		}
		m.UsedBy = append(m.UsedBy, u)
	}
	return m, rows.Err()
}
func GetPublicMedia(ctx context.Context, q querier, id uuid.UUID) (*MediaAsset, error) {
	return scanMedia(q.QueryRow(ctx, `SELECT `+mediaSelect+` FROM marketing.content_media m WHERE m.id=$1 AND m.deleted_at IS NULL AND EXISTS (SELECT 1 FROM marketing.content_article_media am JOIN marketing.content_articles a ON a.id=am.article_id WHERE am.media_id=m.id AND a.deleted_at IS NULL)`, id))
}
func InsertMedia(ctx context.Context, tx pgx.Tx, m MediaAsset) (*MediaAsset, error) {
	raw, err := json.Marshal(m.Renditions)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO marketing.content_media(id,checksum,mime_type,byte_size,width,height,alt_text,decorative,title,credit,storage_key,renditions,uploaded_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, m.ID, m.Checksum, m.MIMEType, m.ByteSize, m.Width, m.Height, m.AltText, m.Decorative, m.Title, m.Credit, m.StorageKey, raw, m.UploadedBy); err != nil {
		return nil, err
	}
	return GetMediaByChecksum(ctx, tx, m.Checksum)
}
func ListMedia(ctx context.Context, q querier, f MediaFilter) ([]MediaAsset, string, error) {
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 30
	}
	var ct time.Time
	var cid uuid.UUID
	if f.Cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(f.Cursor)
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		p := strings.SplitN(string(b), "/", 2)
		if len(p) != 2 {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		ct, e = time.Parse(time.RFC3339Nano, p[0])
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		cid, e = uuid.Parse(p[1])
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
	}
	rows, e := q.Query(ctx, `SELECT `+mediaSelect+` FROM marketing.content_media m WHERE m.deleted_at IS NULL AND ($1='' OR m.mime_type=$1) AND ($2='' OR m.alt_text ILIKE '%'||$2||'%' OR m.title ILIKE '%'||$2||'%') AND (NOT $3 OR NOT EXISTS(SELECT 1 FROM marketing.content_article_media x JOIN marketing.content_articles a ON a.id=x.article_id WHERE x.media_id=m.id AND a.deleted_at IS NULL)) AND ($4::timestamptz IS NULL OR (m.created_at,m.id)<($4,$5)) ORDER BY m.created_at DESC,m.id DESC LIMIT $6`, f.MIMEType, f.Q, f.UnusedOnly, nullTime(ct), nullUUID(cid), f.Limit+1)
	if e != nil {
		return nil, "", e
	}
	defer rows.Close()
	out := make([]MediaAsset, 0, f.Limit+1)
	for rows.Next() {
		m, e := scanMedia(rows)
		if e != nil {
			return nil, "", e
		}
		out = append(out, *m)
	}
	if e = rows.Err(); e != nil {
		return nil, "", e
	}
	next := ""
	if len(out) > f.Limit {
		last := out[f.Limit-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.Format(time.RFC3339Nano) + "/" + last.ID.String()))
		out = out[:f.Limit]
	}
	return out, next, nil
}
func UpdateMedia(ctx context.Context, q mediaStore, id uuid.UUID, alt string, decorative bool, title, credit string) (*MediaAsset, error) {
	return scanMedia(q.QueryRow(ctx, `UPDATE marketing.content_media m SET alt_text=$2,decorative=$3,title=$4,credit=$5 WHERE id=$1 AND deleted_at IS NULL RETURNING `+mediaSelect, id, alt, decorative, title, credit))
}
func SoftDeleteMedia(ctx context.Context, q mediaStore, id uuid.UUID) error {
	tag, e := q.Exec(ctx, `UPDATE marketing.content_media m SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM marketing.content_article_media am JOIN marketing.content_articles a ON a.id=am.article_id WHERE am.media_id=m.id AND a.deleted_at IS NULL)`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func SyncArticleMedia(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, body string, hero *uuid.UUID) error {
	if _, e := tx.Exec(ctx, `DELETE FROM marketing.content_article_media WHERE article_id=$1`, articleID); e != nil {
		return e
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range ExtractMediaIDs(body) {
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, e := tx.Exec(ctx, `INSERT INTO marketing.content_article_media(article_id,media_id,usage) VALUES($1,$2,'body')`, articleID, id); e != nil {
			return e
		}
	}
	if hero != nil {
		_, e := tx.Exec(ctx, `INSERT INTO marketing.content_article_media(article_id,media_id,usage) VALUES($1,$2,'hero')`, articleID, *hero)
		return e
	}
	return nil
}
func ExtractMediaIDs(body string) []uuid.UUID {
	const prefix = "/api/v1/public/content/media/"
	var out []uuid.UUID
	for {
		i := strings.Index(body, prefix)
		if i < 0 {
			break
		}
		body = body[i+len(prefix):]
		if len(body) >= 36 {
			if id, e := uuid.Parse(body[:36]); e == nil {
				out = append(out, id)
			}
		}
	}
	return out
}

func ListArticleMedia(ctx context.Context, q querier, articleID uuid.UUID) ([]MediaAsset, error) {
	rows, err := q.Query(ctx, `SELECT `+mediaSelect+` FROM marketing.content_media m JOIN marketing.content_article_media am ON am.media_id=m.id WHERE am.article_id=$1 AND m.deleted_at IS NULL ORDER BY m.created_at,m.id`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaAsset
	for rows.Next() {
		m, e := scanMedia(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
