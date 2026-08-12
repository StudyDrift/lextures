package marketingcontent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

type ImportResult struct {
	Article *repo.Article
	Action  string
}

// ImportArticle creates or replaces one file-backed article atomically. SourceHash
// is kept in extra.import and makes an unchanged rerun a true no-op (no revision).
func (s *Service) ImportArticle(ctx context.Context, in repo.NewArticle, sourceHash string, force bool) (result ImportResult, err error) {
	report := s.applyQuality(ctx, &in)
	var quality map[string]any
	if e := json.Unmarshal(in.QualityReport, &quality); e == nil {
		quality["grandfathered"] = true
		in.QualityReport, _ = json.Marshal(quality)
	}
	in.QualityScore = &report.Score
	err = s.transaction(ctx, func(tx pgx.Tx) error {
		current, e := repo.GetArticleByKey(ctx, tx, in.Kind, in.Locale, in.Slug)
		if errors.Is(e, pgx.ErrNoRows) {
			in.Extra = setImportedRevision(in.Extra, 1)
			result.Article, e = repo.InsertArticle(ctx, tx, in)
			result.Action = "created"
		} else if e != nil {
			return e
		} else {
			var extra struct {
				Import struct {
					SourceHash       string `json:"sourceHash"`
					ImportedRevision int    `json:"importedRevision"`
				} `json:"import"`
			}
			_ = json.Unmarshal(current.Extra, &extra)
			if extra.Import.SourceHash == sourceHash {
				result = ImportResult{Article: current, Action: "unchanged"}
				return nil
			}
			if current.RevisionNo > 1 && current.RevisionNo != extra.Import.ImportedRevision && !force {
				return fmt.Errorf("%s has revision_no=%d; use --force to overwrite human edits", current.Path, current.RevisionNo)
			}
			in.TranslationGroupID = current.TranslationGroupID
			in.Extra = setImportedRevision(in.Extra, current.RevisionNo+1)
			result.Article, e = repo.UpdateArticle(ctx, tx, repo.ArticleUpdate{ID: current.ID, ExpectedRevisionNo: current.RevisionNo, Article: in})
			result.Action = "updated"
		}
		if e == nil && result.Action != "unchanged" {
			e = repo.SyncArticleMedia(ctx, tx, result.Article.ID, result.Article.BodyMD, result.Article.HeroMediaID)
		}
		return e
	})
	return
}

func setImportedRevision(raw json.RawMessage, revision int) json.RawMessage {
	var extra map[string]any
	if json.Unmarshal(raw, &extra) != nil {
		extra = map[string]any{}
	}
	imp, _ := extra["import"].(map[string]any)
	if imp == nil {
		imp = map[string]any{}
	}
	imp["importedRevision"] = revision
	extra["import"] = imp
	out, _ := json.Marshal(extra)
	return out
}
