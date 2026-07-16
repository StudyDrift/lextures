package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/reearth/ygo/crdt"
)

func TestBoardWS_RealtimeFlagOff_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, h, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"title": "Realtime board"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create board: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	boardID, _ := created["id"].(string)

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/ws", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when realtime flag off, got %d %s", rr2.Code, rr2.Body.String())
	}
}

func TestBoardWS_RealtimeFlagOnDoesNotFeature404_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, tok, cc, _ := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{FFVisualBoards: true, FFBoardsRealtime: true},
	})

	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"title": "Live"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+cc+"/boards", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	boardID, _ := created["id"].(string)

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+cc+"/boards/"+boardID+"/ws", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code == http.StatusNotFound {
		t.Fatalf("realtime on should not feature-gate 404, got %d %s", rr2.Code, rr2.Body.String())
	}
}

func TestBoardWS_StoreCompactReconcile_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, _, _, _, courseID := setupBoardTest(t, ctx, "teacher", true, true)
	defer pool.Close()

	var boardID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO board.boards (course_id, title, slug)
		VALUES ($1, 'CRDT', 'crdt-board') RETURNING id
	`, courseID).Scan(&boardID); err != nil {
		t.Fatalf("insert board: %v", err)
	}

	var postID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO board.posts (board_id, content_type, title, sort_index)
		VALUES ($1, 'text', 'A', 0) RETURNING id
	`, boardID).Scan(&postID); err != nil {
		t.Fatalf("post: %v", err)
	}

	doc := crdt.New()
	m := doc.GetMap(board.PostsMapName)
	doc.Transact(func(txn *crdt.Transaction) {
		m.Set(txn, postID.String(), map[string]any{
			"id":        postID.String(),
			"sortIndex": 42.0,
			"position":  map[string]any{"x": 5.0, "y": 6.0, "w": 100.0, "h": 80.0},
		})
	})
	if err := board.StoreUpdate(ctx, pool, boardID, uuid.Nil, doc.EncodeStateAsUpdate()); err != nil {
		t.Fatalf("store: %v", err)
	}

	replay, err := board.GetReplayState(ctx, pool, boardID)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replay.Updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(replay.Updates))
	}

	ok, err := board.CompactBoard(ctx, pool, boardID)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if !ok {
		t.Fatal("expected compaction")
	}
	replay2, err := board.GetReplayState(ctx, pool, boardID)
	if err != nil {
		t.Fatalf("replay2: %v", err)
	}
	if len(replay2.Snapshot) == 0 {
		t.Fatal("expected snapshot after compact")
	}

	n, err := board.ReconcileBoard(ctx, pool, boardID)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected arrangement update, got %d", n)
	}
	var sortIndex float64
	if err := pool.QueryRow(ctx, `SELECT sort_index FROM board.posts WHERE id=$1`, postID).Scan(&sortIndex); err != nil {
		t.Fatalf("read post: %v", err)
	}
	if sortIndex != 42.0 {
		t.Fatalf("sort_index=%v want 42", sortIndex)
	}
}
