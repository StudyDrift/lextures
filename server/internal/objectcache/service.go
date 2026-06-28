package objectcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lextures/lextures/server/internal/redisclient"
)

const staleExtensionFactor = 6 // stale window = TTL * factor (86400/3600 for HTTP; here 5min fresh → 30min stale)

// entry wraps cached JSON with stale-while-revalidate metadata (plan 17.5 FR-8).
type entry struct {
	Data       json.RawMessage `json:"data"`
	FreshUntil int64           `json:"freshUntil"`
	StaleUntil int64           `json:"staleUntil"`
}

// Service provides Redis-backed object caching with graceful degradation (plan 17.5).
type Service struct {
	redis   *redisclient.Client
	enabled func() bool
}

// New builds a cache service. When redis is nil or enabled returns false, all
// operations degrade to cache miss without error (AC-6).
func New(redis *redisclient.Client, enabled func() bool) *Service {
	if enabled == nil {
		enabled = func() bool { return false }
	}
	return &Service{redis: redis, enabled: enabled}
}

func (s *Service) active() bool {
	return s != nil && s.enabled() && s.redis != nil
}

// GetJSON loads a fresh cached value into dest. Returns true on hit.
func (s *Service) GetJSON(ctx context.Context, key string, resource ResourceType, dest any) (bool, error) {
	if !s.active() {
		RecordMiss(resource)
		return false, nil
	}
	raw, err := s.redis.Get(ctx, key)
	if err != nil || raw == "" {
		RecordMiss(resource)
		return false, err
	}
	var e entry
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		RecordMiss(resource)
		return false, nil
	}
	now := time.Now().Unix()
	if now > e.StaleUntil {
		RecordMiss(resource)
		return false, nil
	}
	if now > e.FreshUntil {
		RecordStaleHit(resource)
	} else {
		RecordHit(resource)
	}
	if err := json.Unmarshal(e.Data, dest); err != nil {
		RecordMiss(resource)
		return false, nil
	}
	return true, nil
}

// SetJSON stores value with the given TTL as the fresh window; stale window extends further.
func (s *Service) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !s.active() || ttl <= 0 {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	now := time.Now()
	e := entry{
		Data:       data,
		FreshUntil: now.Add(ttl).Unix(),
		StaleUntil: now.Add(ttl * staleExtensionFactor).Unix(),
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	// Redis key TTL covers the full stale window so stale-while-revalidate can serve.
	return s.redis.Set(ctx, key, string(b), ttl*staleExtensionFactor)
}

// RefreshAsync re-fetches and stores value in the background after serving stale data.
func (s *Service) RefreshAsync(key string, ttl time.Duration, refresh func(context.Context) (any, error)) {
	if !s.active() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		val, err := refresh(ctx)
		if err != nil {
			return
		}
		_ = s.SetJSON(ctx, key, val, ttl)
	}()
}

// Delete removes a cache key (invalidation).
func (s *Service) Delete(ctx context.Context, keys ...string) error {
	if !s.active() || len(keys) == 0 {
		return nil
	}
	return s.redis.Del(ctx, keys...)
}

// InvalidateCourseStructure removes staff and student structure cache entries.
func (s *Service) InvalidateCourseStructure(ctx context.Context, courseID string) error {
	return s.Delete(ctx,
		CourseStructureKey(courseID, true),
		CourseStructureKey(courseID, false),
	)
}

// InvalidateCourseEnrollments removes the roster cache for a course.
func (s *Service) InvalidateCourseEnrollments(ctx context.Context, courseID string) error {
	return s.Delete(ctx, CourseEnrollmentsKey(courseID))
}

// InvalidateCatalog removes all catalog page caches (publish/unpublish).
func (s *Service) InvalidateCatalog(ctx context.Context) error {
	if !s.active() {
		return nil
	}
	return s.redis.DelByPrefix(ctx, prefix+"catalog:page:")
}

// InvalidateUserCalendar removes calendar feed caches for a user.
func (s *Service) InvalidateUserCalendar(ctx context.Context, userID string) error {
	if !s.active() {
		return nil
	}
	return s.redis.DelByPrefix(ctx, prefix+"user:"+userID+":calendar")
}

// InvalidateCourseCalendar removes course-scoped calendar caches for all users.
func (s *Service) InvalidateCourseCalendar(ctx context.Context, courseID string) error {
	if !s.active() {
		return nil
	}
	pattern := prefix + "user:*:calendar:course:" + courseID
	return s.redis.DelByPattern(ctx, pattern)
}
