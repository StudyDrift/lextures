package httpserver

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const gradingAgentRunStatusCacheTTL = 2 * time.Second

type gradingAgentRunStatusCacheEntry struct {
	status    string
	expiresAt time.Time
}

var gradingAgentRunStatusCache sync.Map // map[uuid.UUID]gradingAgentRunStatusCacheEntry

func gradingAgentRunStatusFromCache(runID uuid.UUID) (string, bool) {
	raw, ok := gradingAgentRunStatusCache.Load(runID)
	if !ok {
		return "", false
	}
	entry, ok := raw.(gradingAgentRunStatusCacheEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		gradingAgentRunStatusCache.Delete(runID)
		return "", false
	}
	return entry.status, true
}

func gradingAgentRunStatusCacheSet(runID uuid.UUID, status string) {
	gradingAgentRunStatusCache.Store(runID, gradingAgentRunStatusCacheEntry{
		status:    status,
		expiresAt: time.Now().Add(gradingAgentRunStatusCacheTTL),
	})
}

func gradingAgentRunStatusCacheInvalidate(runID uuid.UUID) {
	gradingAgentRunStatusCache.Delete(runID)
}
