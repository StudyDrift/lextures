package boardexport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/repos/board"
)

func TestSerializeCards_ExcludesHiddenPendingRemoved(t *testing.T) {
	author := "user-1"
	secID := "sec-1"
	posts := []board.Post{
		{ID: "1", Title: "ok", Status: board.PostStatusApproved, AuthorID: &author, SectionID: &secID, SortIndex: 1, Body: json.RawMessage(`{"text":"hello"}`), CreatedAt: time.Unix(1, 0)},
		{ID: "2", Title: "hidden", Status: board.PostStatusApproved, Hidden: true, SortIndex: 2, CreatedAt: time.Unix(2, 0)},
		{ID: "3", Title: "pending", Status: board.PostStatusPending, SortIndex: 3, CreatedAt: time.Unix(3, 0)},
		{ID: "4", Title: "removed", Status: board.PostStatusApproved, Removed: true, SortIndex: 4, CreatedAt: time.Unix(4, 0)},
	}
	sections := map[string]board.Section{
		secID: {ID: secID, Title: "Ideas", SortIndex: 0},
	}
	rows := SerializeCards(posts, SerializeOpts{
		RevealAuthors: true,
		SectionByID:   sections,
		Layout:        "columns",
	})
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Title != "ok" || rows[0].SectionTitle != "Ideas" || rows[0].BodyText != "hello" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestSerializeCards_RedactsAuthors(t *testing.T) {
	author := "user-secret"
	posts := []board.Post{
		{ID: "1", Title: "anon", Status: board.PostStatusApproved, AuthorID: &author, CreatedAt: time.Unix(1, 0)},
	}
	rows := SerializeCards(posts, SerializeOpts{
		Attribution:   board.AttributionAnonymous,
		RevealAuthors: false,
	})
	if len(rows) != 1 || rows[0].Author != "" {
		t.Fatalf("author should be redacted, got %+v", rows[0])
	}
}

func TestSerializeCards_IncludeModeration(t *testing.T) {
	posts := []board.Post{
		{ID: "1", Title: "hidden", Status: board.PostStatusApproved, Hidden: true, CreatedAt: time.Unix(1, 0)},
		{ID: "2", Title: "pending", Status: board.PostStatusPending, CreatedAt: time.Unix(2, 0)},
	}
	rows := SerializeCards(posts, SerializeOpts{IncludeModeration: true})
	if len(rows) != 2 {
		t.Fatalf("want 2 rows with moderation, got %d", len(rows))
	}
}

func TestRenderCSV_ColumnsAndRedaction(t *testing.T) {
	cards := []CardRow{
		{SectionTitle: "A", Author: "", ContentType: "text", Title: "T", BodyText: "B", ReactionCount: 2, CommentCount: 1, CreatedAt: time.Unix(100, 0).UTC()},
	}
	raw, err := RenderCSV(cards, false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "section,author,content_type") {
		t.Fatalf("missing headers: %s", s)
	}
	if strings.Contains(s, "user-secret") {
		t.Fatal("leaked author")
	}
	if !strings.Contains(s, ",text,T,B,") {
		t.Fatalf("missing row data: %s", s)
	}
}

func TestRenderPDF_AndPNG(t *testing.T) {
	cards := []CardRow{
		{SectionTitle: "Sec", Title: "Hello", BodyText: "World", ContentType: "text", AttachmentFilename: "a.png", AttachmentAltText: "alt", CreatedAt: time.Now()},
	}
	pdf, err := RenderPDF("Board", cards)
	if err != nil || len(pdf) < 100 {
		t.Fatalf("pdf: err=%v len=%d", err, len(pdf))
	}
	if string(pdf[0:4]) != "%PDF" {
		t.Fatalf("not a pdf: %q", pdf[:4])
	}
	pngBytes, err := RenderPNG("Board", cards)
	if err != nil || len(pngBytes) < 50 {
		t.Fatalf("png: err=%v len=%d", err, len(pngBytes))
	}
	if string(pngBytes[0:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("not a png")
	}
}

func TestRenderQRPNG(t *testing.T) {
	b, err := RenderQRPNG("https://example.com/board-links/bsh_test", 128)
	if err != nil || len(b) < 50 {
		t.Fatalf("qr: err=%v len=%d", err, len(b))
	}
}
