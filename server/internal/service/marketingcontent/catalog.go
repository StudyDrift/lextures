package marketingcontent

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

func (s *Service) ReplaceStaticKnownPaths(ctx context.Context, paths []string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error { return repo.ReplaceStaticKnownPaths(ctx, tx, paths) })
}

func (s *Service) SaveCategory(ctx context.Context, c repo.Category, actor uuid.UUID) (out *repo.Category, err error) {
	err = s.transaction(ctx, func(tx pgx.Tx) error { out, err = repo.UpsertCategory(ctx, tx, c, actor); return err })
	return
}
func (s *Service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.transaction(ctx, func(tx pgx.Tx) error { return repo.DeleteCategory(ctx, tx, id) })
}
func (s *Service) SaveAuthor(ctx context.Context, a repo.Author, actor uuid.UUID) (out *repo.Author, err error) {
	err = s.transaction(ctx, func(tx pgx.Tx) error { out, err = repo.UpsertAuthor(ctx, tx, a, actor); return err })
	return
}
func (s *Service) SaveTag(ctx context.Context, t repo.Tag, actor uuid.UUID) (out *repo.Tag, err error) {
	err = s.transaction(ctx, func(tx pgx.Tx) error { out, err = repo.UpsertTag(ctx, tx, t, actor); return err })
	return
}
func (s *Service) DeleteTag(ctx context.Context, id uuid.UUID) error {
	return s.transaction(ctx, func(tx pgx.Tx) error { return repo.DeleteTag(ctx, tx, id) })
}
func (s *Service) SaveRedirect(ctx context.Context, v repo.Redirect, actor uuid.UUID) (out *repo.Redirect, err error) {
	err = s.transaction(ctx, func(tx pgx.Tx) error { out, err = repo.InsertRedirect(ctx, tx, v, actor); return err })
	return
}
func (s *Service) DeleteRedirect(ctx context.Context, id uuid.UUID) error {
	return s.transaction(ctx, func(tx pgx.Tx) error { return repo.DeleteRedirect(ctx, tx, id) })
}
