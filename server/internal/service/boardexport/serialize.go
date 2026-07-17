package boardexport

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/repos/board"
)

// CardRow is one exportable card in reading order (shared by PDF/CSV/image).
type CardRow struct {
	SectionTitle       string
	Author             string // empty when redacted
	ContentType        string
	Title              string
	BodyText           string
	Link               string
	AttachmentFilename string
	AttachmentAltText  string
	ReactionCount      int
	ReactionAvg        *float64
	CommentCount       int
	CreatedAt          time.Time
	// Moderation-only fields (populated when IncludeModeration).
	Status  string
	Hidden  bool
	Removed bool
}

// SerializeOpts controls redaction and inclusion rules (VC.9 FR-5/FR-7).
type SerializeOpts struct {
	Attribution       string
	RevealAuthors     bool
	IncludeModeration bool
	Engagements       map[string]board.PostEngagement
	SectionByID       map[string]board.Section
	Layout            string
}

// SerializeCards builds reading-order rows, excluding hidden/pending/removed
// unless IncludeModeration is set (manager export).
func SerializeCards(posts []board.Post, opts SerializeOpts) []CardRow {
	sectionOrder := make(map[string]float64)
	for id, s := range opts.SectionByID {
		sectionOrder[id] = s.SortIndex
	}

	type ranked struct {
		post  board.Post
		secIx float64
		hasSec bool
	}
	rankedPosts := make([]ranked, 0, len(posts))
	for _, p := range posts {
		if !opts.IncludeModeration {
			if p.Removed || p.Hidden || p.Status == board.PostStatusPending || p.Status == board.PostStatusRejected {
				continue
			}
		}
		r := ranked{post: p}
		if p.SectionID != nil {
			if ix, ok := sectionOrder[*p.SectionID]; ok {
				r.secIx = ix
				r.hasSec = true
			}
		}
		rankedPosts = append(rankedPosts, r)
	}

	sort.SliceStable(rankedPosts, func(i, j int) bool {
		a, b := rankedPosts[i], rankedPosts[j]
		// Sections first by section sort index; unsectioned after.
		if a.hasSec != b.hasSec {
			return a.hasSec
		}
		if a.hasSec && b.hasSec && a.secIx != b.secIx {
			return a.secIx < b.secIx
		}
		// Layout-aware tie-breakers within a section.
		switch strings.ToLower(opts.Layout) {
		case "timeline":
			if a.post.EventDate != nil && b.post.EventDate != nil && !a.post.EventDate.Equal(*b.post.EventDate) {
				return a.post.EventDate.Before(*b.post.EventDate)
			}
			if a.post.EventDate != nil && b.post.EventDate == nil {
				return true
			}
			if a.post.EventDate == nil && b.post.EventDate != nil {
				return false
			}
		case "map":
			if a.post.Lat != nil && b.post.Lat != nil && *a.post.Lat != *b.post.Lat {
				return *a.post.Lat > *b.post.Lat // north → south
			}
		}
		if a.post.SortIndex != b.post.SortIndex {
			return a.post.SortIndex < b.post.SortIndex
		}
		return a.post.CreatedAt.Before(b.post.CreatedAt)
	})

	out := make([]CardRow, 0, len(rankedPosts))
	for _, r := range rankedPosts {
		p := r.post
		row := CardRow{
			ContentType: p.ContentType,
			Title:       p.Title,
			BodyText:    extractBodyText(p.Body),
			CreatedAt:   p.CreatedAt,
			Status:      p.Status,
			Hidden:      p.Hidden,
			Removed:     p.Removed,
		}
		if p.SectionID != nil {
			if s, ok := opts.SectionByID[*p.SectionID]; ok {
				row.SectionTitle = s.Title
			}
		}
		if p.LinkURL != nil {
			row.Link = *p.LinkURL
		}
		if p.Attachment != nil {
			row.AttachmentFilename = p.Attachment.FileName
			row.AttachmentAltText = p.Attachment.AltText
		}
		if opts.RevealAuthors {
			if p.AuthorID != nil {
				row.Author = *p.AuthorID
			} else if p.GuestDisplayName != "" {
				row.Author = p.GuestDisplayName
			}
		}
		if eng, ok := opts.Engagements[p.ID]; ok {
			row.ReactionCount = eng.ReactionCount
			row.ReactionAvg = eng.AvgStars
			row.CommentCount = eng.CommentCount
		}
		out = append(out, row)
	}
	return out
}

func extractBodyText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return strings.TrimSpace(string(raw))
	}
	if t, ok := obj["text"].(string); ok {
		return strings.TrimSpace(t)
	}
	if t, ok := obj["caption"].(string); ok {
		return strings.TrimSpace(t)
	}
	return ""
}
