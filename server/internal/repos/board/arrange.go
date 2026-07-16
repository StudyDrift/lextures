package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrLayoutLocked is returned when a non-manager tries to arrange on a locked board.
var ErrLayoutLocked = errors.New("board: layout is locked")

// ErrArrangeForbidden is returned when the viewer cannot arrange the post.
var ErrArrangeForbidden = errors.New("board: arrange not permitted")

// PostPosition is the canvas layout rect for a card.
type PostPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// ArrangePostInput is a partial arrangement update (FR-4…FR-7).
type ArrangePostInput struct {
	SectionID *string
	ClearSection bool // when true, set section_id NULL
	SortIndex *float64
	Position  *PostPosition
	EventDate *time.Time
	ClearEventDate bool
	Lat       *float64
	Lng       *float64
	ClearGeo  bool
}

// CanArrangePost reports whether viewer may change arrangement fields.
// Managers (item:create) always can; authors can unless the board is layout-locked.
func CanArrangePost(layoutLocked, isManager bool, authorID *string, viewer uuid.UUID) error {
	if isManager {
		return nil
	}
	if layoutLocked {
		return ErrLayoutLocked
	}
	if authorID != nil && *authorID == viewer.String() {
		return nil
	}
	return ErrArrangeForbidden
}

// ValidateArrangeCoords checks lat/lng bounds when provided.
func ValidateArrangeCoords(lat, lng *float64) error {
	if lat != nil {
		if math.IsNaN(*lat) || *lat < -90 || *lat > 90 {
			return fmt.Errorf("board: lat must be between -90 and 90")
		}
	}
	if lng != nil {
		if math.IsNaN(*lng) || *lng < -180 || *lng > 180 {
			return fmt.Errorf("board: lng must be between -180 and 180")
		}
	}
	return nil
}

// ArrangePost updates layout fields on a post (section, sort, position, event_date, geo).
func ArrangePost(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	in ArrangePostInput,
) (*Post, error) {
	existing, err := GetPost(ctx, pool, courseCode, boardID, postID)
	if err != nil || existing == nil {
		return existing, err
	}

	sectionID := existing.SectionID
	if in.ClearSection {
		sectionID = nil
	} else if in.SectionID != nil {
		sid := *in.SectionID
		if sid == "" {
			sectionID = nil
		} else {
			sec, err := GetSection(ctx, pool, courseCode, boardID, sid)
			if err != nil {
				return nil, err
			}
			if sec == nil {
				return nil, fmt.Errorf("board: section not found")
			}
			sectionID = &sid
		}
	}

	sortIndex := existing.SortIndex
	if in.SortIndex != nil {
		sortIndex = *in.SortIndex
	}

	position := existing.Position
	if in.Position != nil {
		pos := *in.Position
		if pos.W <= 0 {
			pos.W = 240
		}
		if pos.H <= 0 {
			pos.H = 160
		}
		b, err := json.Marshal(pos)
		if err != nil {
			return nil, err
		}
		position = json.RawMessage(b)
	}

	eventDate := existing.EventDate
	if in.ClearEventDate {
		eventDate = nil
	} else if in.EventDate != nil {
		t := in.EventDate.UTC()
		eventDate = &t
	}

	lat := existing.Lat
	lng := existing.Lng
	if in.ClearGeo {
		lat = nil
		lng = nil
	} else {
		if in.Lat != nil {
			lat = in.Lat
		}
		if in.Lng != nil {
			lng = in.Lng
		}
		if err := ValidateArrangeCoords(lat, lng); err != nil {
			return nil, err
		}
	}

	var sectionUUID *uuid.UUID
	if sectionID != nil {
		parsed, err := uuid.Parse(*sectionID)
		if err != nil {
			return nil, fmt.Errorf("board: invalid section_id")
		}
		sectionUUID = &parsed
	}

	pid, _ := uuid.Parse(postID)
	bid, _ := uuid.Parse(boardID)
	row := pool.QueryRow(ctx, `
		UPDATE board.posts p
		SET
			section_id = $4,
			sort_index = $5,
			position = COALESCE($6, p.position),
			event_date = $7,
			lat = $8,
			lng = $9,
			updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		RETURNING `+selectPostCols()+`
	`, courseCode, bid, pid, sectionUUID, sortIndex, nullableJSON(position), eventDate, lat, lng)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// NextSortIndexBetween returns a fractional index between before and after.
// Nil before → prepend relative to after; nil after → append relative to before.
func NextSortIndexBetween(before, after *float64) float64 {
	switch {
	case before == nil && after == nil:
		return 0
	case before == nil:
		return PrependSortIndex(after)
	case after == nil:
		return AppendSortIndex(before)
	default:
		return MidpointSortIndex(*before, *after)
	}
}
