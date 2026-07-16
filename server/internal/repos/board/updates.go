package board

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reearth/ygo/crdt"
)

// PostsMapName is the root Y.Map key holding board post arrangement state.
const PostsMapName = "posts"

// CompactUpdateThreshold triggers compaction when a board has at least this many
// append-only updates since the latest snapshot.
const CompactUpdateThreshold = 100

// ReplayState is the ordered binary payload a connecting client must apply.
type ReplayState struct {
	Snapshot []byte
	Updates  [][]byte
}

// StoreUpdate persists a raw Y.js binary update for a board.
// Pass uuid.Nil for authorID to store a system/reconciler update with NULL author.
func StoreUpdate(ctx context.Context, pool *pgxpool.Pool, boardID, authorID uuid.UUID, update []byte) error {
	var author any
	if authorID != uuid.Nil {
		author = authorID
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO board.board_updates (board_id, author_id, update) VALUES ($1, $2, $3)`,
		boardID, author, update,
	)
	return err
}

// GetReplayState returns the latest snapshot (if any) plus subsequent updates.
func GetReplayState(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) (ReplayState, error) {
	var out ReplayState
	var snapTaken *time.Time
	err := pool.QueryRow(ctx, `
		SELECT state, taken_at FROM board.board_snapshots
		WHERE board_id = $1
		ORDER BY taken_at DESC
		LIMIT 1
	`, boardID).Scan(&out.Snapshot, &snapTaken)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return ReplayState{}, err
		}
		// No snapshot yet — load all updates.
		updates, uerr := getUpdatesSince(ctx, pool, boardID, nil)
		if uerr != nil {
			return ReplayState{}, uerr
		}
		out.Updates = updates
		return out, nil
	}
	updates, err := getUpdatesSince(ctx, pool, boardID, snapTaken)
	if err != nil {
		return ReplayState{}, err
	}
	out.Updates = updates
	return out, nil
}

func getUpdatesSince(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID, since *time.Time) ([][]byte, error) {
	var (
		q    string
		args []any
	)
	if since == nil {
		q = `SELECT update FROM board.board_updates WHERE board_id=$1 ORDER BY created_at ASC, id ASC`
		args = []any{boardID}
	} else {
		q = `SELECT update FROM board.board_updates WHERE board_id=$1 AND created_at > $2 ORDER BY created_at ASC, id ASC`
		args = []any{boardID, *since}
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]byte
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CountUpdatesSinceSnapshot returns how many updates exist after the latest snapshot.
func CountUpdatesSinceSnapshot(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM board.board_updates u
		WHERE u.board_id = $1
		  AND (
		    NOT EXISTS (SELECT 1 FROM board.board_snapshots s WHERE s.board_id = $1)
		    OR u.created_at > (
		      SELECT MAX(taken_at) FROM board.board_snapshots WHERE board_id = $1
		    )
		  )
	`, boardID).Scan(&n)
	return n, err
}

// ListBoardIDsNeedingCompaction returns boards with enough append-only updates to compact.
func ListBoardIDsNeedingCompaction(ctx context.Context, pool *pgxpool.Pool, threshold, limit int) ([]uuid.UUID, error) {
	if threshold < 1 {
		threshold = CompactUpdateThreshold
	}
	if limit < 1 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
		SELECT b.id
		FROM board.boards b
		WHERE b.archived = FALSE
		  AND (
		    SELECT COUNT(*) FROM board.board_updates u
		    WHERE u.board_id = b.id
		      AND (
		        NOT EXISTS (SELECT 1 FROM board.board_snapshots s WHERE s.board_id = b.id)
		        OR u.created_at > (
		          SELECT MAX(taken_at) FROM board.board_snapshots WHERE board_id = b.id
		        )
		      )
		  ) >= $1
		ORDER BY b.updated_at DESC
		LIMIT $2
	`, threshold, limit)
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

// BuildDocFromReplay applies snapshot + updates into a fresh Y.js document.
func BuildDocFromReplay(replay ReplayState) (*crdt.Doc, error) {
	doc := crdt.New()
	if len(replay.Snapshot) > 0 {
		if err := doc.ApplyUpdate(replay.Snapshot); err != nil {
			return nil, err
		}
	}
	for _, upd := range replay.Updates {
		if len(upd) == 0 {
			continue
		}
		if err := doc.ApplyUpdate(upd); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// CompactBoard merges stored updates into a new snapshot and prunes folded rows.
func CompactBoard(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) (bool, error) {
	replay, err := GetReplayState(ctx, pool, boardID)
	if err != nil {
		return false, err
	}
	if len(replay.Updates) == 0 && len(replay.Snapshot) > 0 {
		return false, nil
	}
	if len(replay.Updates) == 0 && len(replay.Snapshot) == 0 {
		return false, nil
	}
	doc, err := BuildDocFromReplay(replay)
	if err != nil {
		return false, err
	}
	state := doc.EncodeStateAsUpdate()
	takenAt := time.Now().UTC()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO board.board_snapshots (board_id, state, taken_at) VALUES ($1, $2, $3)
	`, boardID, state, takenAt); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM board.board_updates WHERE board_id = $1 AND created_at <= $2
	`, boardID, takenAt); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM board.board_snapshots
		WHERE board_id = $1 AND taken_at < $2
	`, boardID, takenAt); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
