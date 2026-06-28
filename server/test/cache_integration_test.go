package test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/objectcache"
	"github.com/lextures/lextures/server/internal/redisclient"
)

func TestRedisCache_CourseStructureInvalidate(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	defer func() { _ = rc.Close() }()

	s := objectcache.New(rc, func() bool { return true })
	ctx := context.Background()
	courseID := uuid.New().String()
	key := objectcache.CourseStructureKey(courseID, true)

	_ = s.SetJSON(ctx, key, map[string]string{"v": "1"}, time.Minute)
	if err := s.InvalidateCourseStructure(ctx, courseID); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	var out map[string]string
	if hit, _ := s.GetJSON(ctx, key, objectcache.ResourceCourseStructure, &out); hit {
		t.Fatal("expected cache miss after invalidation")
	}
}

func TestRedisCache_UnavailableDegrades(t *testing.T) {
	s := objectcache.New(nil, func() bool { return true })
	ctx := context.Background()
	key := objectcache.CourseEnrollmentsKey("missing")
	var out string
	if hit, err := s.GetJSON(ctx, key, objectcache.ResourceCourseEnrollments, &out); err != nil || hit {
		t.Fatalf("expected graceful miss, hit=%v err=%v", hit, err)
	}
}
