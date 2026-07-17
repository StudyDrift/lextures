package boardexport

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/board"
)

// BuildOpts configures a board export render.
type BuildOpts struct {
	CourseCode        string
	BoardID           string
	ViewerID          uuid.UUID
	Format            string
	IncludeModeration bool
	Caps              board.Capabilities
}

// Result is the rendered export bytes plus content type / extension.
type Result struct {
	Bytes       []byte
	ContentType string
	Extension   string
}

// Build loads board content and renders the requested format through the shared serializer.
func Build(ctx context.Context, pool *pgxpool.Pool, opts BuildOpts) (*Result, error) {
	format, err := board.NormalizeExportFormat(opts.Format)
	if err != nil {
		return nil, err
	}
	b, err := board.Get(ctx, pool, opts.CourseCode, opts.BoardID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("boardexport: board not found")
	}
	posts, err := board.ListPosts(ctx, pool, opts.CourseCode, opts.BoardID)
	if err != nil {
		return nil, err
	}
	sections, err := board.ListSections(ctx, pool, opts.CourseCode, opts.BoardID)
	if err != nil {
		return nil, err
	}
	secMap := make(map[string]board.Section, len(sections))
	for _, s := range sections {
		secMap[s.ID] = s
	}
	eng, err := board.LoadPostEngagements(ctx, pool, opts.CourseCode, opts.BoardID, opts.ViewerID, b.ReactionMode, opts.Caps.CanManage)
	if err != nil {
		return nil, err
	}
	reveal := board.RevealAuthor(b.Attribution, opts.Caps)
	cards := SerializeCards(posts, SerializeOpts{
		Attribution:       b.Attribution,
		RevealAuthors:     reveal,
		IncludeModeration: opts.IncludeModeration && opts.Caps.CanManage,
		Engagements:       eng,
		SectionByID:       secMap,
		Layout:            b.Layout,
	})

	switch format {
	case board.ExportFormatCSV:
		raw, err := RenderCSV(cards, opts.IncludeModeration && opts.Caps.CanManage)
		if err != nil {
			return nil, err
		}
		return &Result{Bytes: raw, ContentType: "text/csv; charset=utf-8", Extension: "csv"}, nil
	case board.ExportFormatPDF:
		raw, err := RenderPDF(b.Title, cards)
		if err != nil {
			return nil, err
		}
		return &Result{Bytes: raw, ContentType: "application/pdf", Extension: "pdf"}, nil
	case board.ExportFormatImage:
		raw, err := RenderPNG(b.Title, cards)
		if err != nil {
			return nil, err
		}
		return &Result{Bytes: raw, ContentType: "image/png", Extension: "png"}, nil
	default:
		return nil, fmt.Errorf("boardexport: unsupported format")
	}
}
