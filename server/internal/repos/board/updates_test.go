package board

import (
	"testing"

	"github.com/reearth/ygo/crdt"
)

func TestBuildDocFromReplayAndCompactRoundTrip(t *testing.T) {
	doc := crdt.New()
	m := doc.GetMap(PostsMapName)
	doc.Transact(func(txn *crdt.Transaction) {
		m.Set(txn, "post-1", map[string]any{
			"id":        "post-1",
			"sortIndex": 1.5,
			"position":  map[string]any{"x": 10.0, "y": 20.0, "w": 100.0, "h": 80.0},
		})
	})
	upd1 := doc.EncodeStateAsUpdate()

	doc2 := crdt.New()
	m2 := doc2.GetMap(PostsMapName)
	doc2.Transact(func(txn *crdt.Transaction) {
		m2.Set(txn, "post-2", map[string]any{
			"id":        "post-2",
			"sortIndex": 2.0,
		})
	})
	upd2 := doc2.EncodeStateAsUpdate()

	merged, err := BuildDocFromReplay(ReplayState{Updates: [][]byte{upd1, upd2}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	posts := merged.GetMap(PostsMapName)
	if !posts.Has("post-1") || !posts.Has("post-2") {
		t.Fatalf("expected both posts, keys=%v", posts.Keys())
	}

	snap := merged.EncodeStateAsUpdate()
	again, err := BuildDocFromReplay(ReplayState{Snapshot: snap})
	if err != nil {
		t.Fatalf("from snap: %v", err)
	}
	if !again.GetMap(PostsMapName).Has("post-1") {
		t.Fatal("snapshot missing post-1")
	}
}

func TestDecodeArrangement(t *testing.T) {
	arr, ok := decodeArrangement("abc", map[string]any{
		"sortIndex": 3.0,
		"sectionId": "sec-1",
		"lat":       40.0,
		"lng":       -74.0,
	})
	if !ok {
		t.Fatal("decode failed")
	}
	if arr.ID != "abc" || arr.SortIndex == nil || *arr.SortIndex != 3.0 {
		t.Fatalf("got %+v", arr)
	}
	if arr.SectionID == nil || *arr.SectionID != "sec-1" {
		t.Fatalf("section: %+v", arr.SectionID)
	}
}

func TestArrangementFromPost(t *testing.T) {
	sec := "s1"
	lat, lng := 1.0, 2.0
	p := Post{
		ID:        "p1",
		SortIndex: 9,
		SectionID: &sec,
		Position:  []byte(`{"x":1,"y":2,"w":3,"h":4}`),
		Lat:       &lat,
		Lng:       &lng,
	}
	m := arrangementFromPost(p)
	if m["id"] != "p1" || m["sortIndex"] != 9.0 {
		t.Fatalf("got %#v", m)
	}
}
