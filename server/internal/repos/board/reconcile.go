package board

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reearth/ygo/crdt"
)

// ArrangementCRDT is the shared per-post arrangement fields stored in the Y.Map.
type ArrangementCRDT struct {
	ID         string           `json:"id"`
	SectionID  *string          `json:"sectionId,omitempty"`
	SortIndex  *float64         `json:"sortIndex,omitempty"`
	Position   *PostPosition    `json:"position,omitempty"`
	EventDate  *string          `json:"eventDate,omitempty"`
	Lat        *float64         `json:"lat,omitempty"`
	Lng        *float64         `json:"lng,omitempty"`
	Deleted    bool             `json:"deleted,omitempty"`
}

// ReconcileBoard folds CRDT arrangement into board.posts and seeds missing CRDT
// entries from the DB. CRDT is authoritative for arrangement fields.
func ReconcileBoard(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) (int, error) {
	replay, err := GetReplayState(ctx, pool, boardID)
	if err != nil {
		return 0, err
	}
	doc, err := BuildDocFromReplay(replay)
	if err != nil {
		return 0, err
	}

	postsMap := doc.GetMap(PostsMapName)
	crdtByID := map[string]ArrangementCRDT{}
	postsMap.ForEach(func(key string, value any) {
		arr, ok := decodeArrangement(key, value)
		if !ok {
			return
		}
		crdtByID[key] = arr
	})

	dbPosts, err := listPostsByBoardID(ctx, pool, boardID)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, p := range dbPosts {
		arr, ok := crdtByID[p.ID]
		if !ok {
			continue
		}
		if arr.Deleted {
			continue
		}
		n, err := applyArrangementToPost(ctx, pool, boardID, p.ID, arr)
		if err != nil {
			return updated, err
		}
		updated += n
	}

	// Forward DB-side posts missing from the CRDT so late joins see them after reconcile.
	missing := false
	for _, p := range dbPosts {
		if _, ok := crdtByID[p.ID]; !ok {
			missing = true
			break
		}
	}
	if missing {
		var seedUpdate []byte
		unsub := doc.OnUpdate(func(update []byte, _ any) {
			seedUpdate = append([]byte(nil), update...)
		})
		doc.Transact(func(txn *crdt.Transaction) {
			for _, p := range dbPosts {
				if _, ok := crdtByID[p.ID]; ok {
					continue
				}
				postsMap.Set(txn, p.ID, arrangementFromPost(p))
			}
		})
		unsub()
		if len(seedUpdate) > 0 {
			if err := StoreUpdate(ctx, pool, boardID, uuid.Nil, seedUpdate); err != nil {
				return updated, err
			}
		}
	}
	return updated, nil
}

func decodeArrangement(key string, value any) (ArrangementCRDT, bool) {
	arr := ArrangementCRDT{ID: key}
	switch v := value.(type) {
	case map[string]any:
		b, err := json.Marshal(v)
		if err != nil {
			return ArrangementCRDT{}, false
		}
		if err := json.Unmarshal(b, &arr); err != nil {
			return ArrangementCRDT{}, false
		}
	case *crdt.YMap:
		b, err := v.ToJSON()
		if err != nil {
			return ArrangementCRDT{}, false
		}
		if err := json.Unmarshal(b, &arr); err != nil {
			return ArrangementCRDT{}, false
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ArrangementCRDT{}, false
		}
		if err := json.Unmarshal(b, &arr); err != nil {
			return ArrangementCRDT{}, false
		}
	}
	if arr.ID == "" {
		arr.ID = key
	}
	return arr, true
}

func arrangementFromPost(p Post) map[string]any {
	m := map[string]any{
		"id":        p.ID,
		"sortIndex": p.SortIndex,
	}
	if p.SectionID != nil {
		m["sectionId"] = *p.SectionID
	}
	if len(p.Position) > 0 {
		var pos PostPosition
		if json.Unmarshal(p.Position, &pos) == nil {
			m["position"] = map[string]any{"x": pos.X, "y": pos.Y, "w": pos.W, "h": pos.H}
		}
	}
	if p.EventDate != nil {
		m["eventDate"] = p.EventDate.UTC().Format(time.RFC3339)
	}
	if p.Lat != nil {
		m["lat"] = *p.Lat
	}
	if p.Lng != nil {
		m["lng"] = *p.Lng
	}
	return m
}

func applyArrangementToPost(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID, postID string, arr ArrangementCRDT) (int, error) {
	pid, err := uuid.Parse(postID)
	if err != nil {
		return 0, nil
	}
	var section any
	if arr.SectionID != nil && *arr.SectionID != "" {
		sid, err := uuid.Parse(*arr.SectionID)
		if err != nil {
			return 0, fmt.Errorf("sectionId: %w", err)
		}
		section = sid
	}
	var posJSON []byte
	if arr.Position != nil {
		posJSON, _ = json.Marshal(arr.Position)
	}
	var eventDate *time.Time
	if arr.EventDate != nil && *arr.EventDate != "" {
		t, err := time.Parse(time.RFC3339, *arr.EventDate)
		if err != nil {
			t2, err2 := time.Parse(time.RFC3339Nano, *arr.EventDate)
			if err2 != nil {
				return 0, fmt.Errorf("eventDate: %w", err)
			}
			t = t2
		}
		tt := t.UTC()
		eventDate = &tt
	}
	sortIndex := 0.0
	if arr.SortIndex != nil {
		sortIndex = *arr.SortIndex
	}
	tag, err := pool.Exec(ctx, `
		UPDATE board.posts SET
			section_id = $3,
			sort_index = $4,
			position = COALESCE($5::jsonb, position),
			event_date = $6,
			lat = $7,
			lng = $8,
			updated_at = NOW()
		WHERE id = $1 AND board_id = $2
		  AND (
			section_id IS DISTINCT FROM $3::uuid
			OR sort_index IS DISTINCT FROM $4::double precision
			OR ($5::jsonb IS NOT NULL AND position IS DISTINCT FROM $5::jsonb)
			OR event_date IS DISTINCT FROM $6::timestamptz
			OR lat IS DISTINCT FROM $7::double precision
			OR lng IS DISTINCT FROM $8::double precision
		  )
	`, pid, boardID, section, sortIndex, posJSON, eventDate, arr.Lat, arr.Lng)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func listPostsByBoardID(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) ([]Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+selectPostCols()+`
		FROM board.posts p
		WHERE p.board_id = $1
		ORDER BY p.sort_index ASC, p.created_at DESC
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListBoardIDsForReconcile returns recently active boards to fold CRDT → DB.
func ListBoardIDsForReconcile(ctx context.Context, pool *pgxpool.Pool, since time.Time, limit int) ([]uuid.UUID, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT u.board_id
		FROM board.board_updates u
		INNER JOIN board.boards b ON b.id = u.board_id
		WHERE u.created_at >= $1 AND b.archived = FALSE
		ORDER BY u.board_id
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
