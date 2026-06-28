package objectcache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/redisclient"
)

func TestService_GetSetJSON(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	defer func() { _ = rc.Close() }()

	s := New(rc, func() bool { return true })
	ctx := context.Background()
	key := CourseStructureKey("c1", true)

	var out struct {
		Items []string `json:"items"`
	}
	if hit, err := s.GetJSON(ctx, key, ResourceCourseStructure, &out); err != nil || hit {
		t.Fatalf("expected miss, hit=%v err=%v", hit, err)
	}

	val := struct {
		Items []string `json:"items"`
	}{Items: []string{"a", "b"}}
	if err := s.SetJSON(ctx, key, val, time.Minute); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}
	if hit, err := s.GetJSON(ctx, key, ResourceCourseStructure, &out); err != nil || !hit {
		t.Fatalf("expected hit, hit=%v err=%v", hit, err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items=%v", out.Items)
	}
}

func TestService_DisabledDegrades(t *testing.T) {
	s := New(nil, func() bool { return false })
	ctx := context.Background()
	key := CourseEnrollmentsKey("c1")
	if err := s.SetJSON(ctx, key, "x", time.Minute); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}
	var out string
	if hit, err := s.GetJSON(ctx, key, ResourceCourseEnrollments, &out); err != nil || hit {
		t.Fatalf("expected miss without error, hit=%v err=%v", hit, err)
	}
}

func TestService_InvalidateCourseStructure(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	defer func() { _ = rc.Close() }()
	s := New(rc, func() bool { return true })
	ctx := context.Background()

	_ = s.SetJSON(ctx, CourseStructureKey("c1", true), "staff", time.Minute)
	_ = s.SetJSON(ctx, CourseStructureKey("c1", false), "student", time.Minute)
	if err := s.InvalidateCourseStructure(ctx, "c1"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	var dummy string
	if hit, _ := s.GetJSON(ctx, CourseStructureKey("c1", true), ResourceCourseStructure, &dummy); hit {
		t.Fatal("expected staff key invalidated")
	}
	if hit, _ := s.GetJSON(ctx, CourseStructureKey("c1", false), ResourceCourseStructure, &dummy); hit {
		t.Fatal("expected student key invalidated")
	}
}

func TestService_StaleWhileRevalidate(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	defer func() { _ = rc.Close() }()
	s := New(rc, func() bool { return true })
	ctx := context.Background()
	key := CatalogPageKey(repoCourse.PublicCatalogFilter{Limit: 10})

	_ = s.SetJSON(ctx, key, map[string]string{"v": "1"}, 100*time.Millisecond)
	mr.FastForward(150 * time.Millisecond)

	var out map[string]string
	hit, err := s.GetJSON(ctx, key, ResourceCatalogPage, &out)
	if err != nil || !hit {
		t.Fatalf("expected stale hit, hit=%v err=%v", hit, err)
	}
	if out["v"] != "1" {
		t.Fatalf("stale value=%v", out)
	}
}
