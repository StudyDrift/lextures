// Package canvassubmissionsyncjobs tracks in-flight Canvas submission sync jobs in memory.
package canvassubmissionsyncjobs

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status is the lifecycle of a sync job.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Job is an ephemeral sync job (not persisted to Postgres).
type Job struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Status       Status
	ErrorMessage string
	Result       map[string]any
	CreatedAt    time.Time
}

// Registry stores sync jobs for WebSocket reconnect and authorization checks.
type Registry struct {
	mu   sync.RWMutex
	jobs map[uuid.UUID]*Job
}

// NewRegistry returns an empty job registry.
func NewRegistry() *Registry {
	return &Registry{jobs: make(map[uuid.UUID]*Job)}
}

// Create registers a queued job and returns its ID.
func (r *Registry) Create(userID uuid.UUID) uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	id := uuid.New()
	r.mu.Lock()
	r.jobs[id] = &Job{
		ID:        id,
		UserID:    userID,
		Status:    StatusQueued,
		CreatedAt: time.Now(),
	}
	r.mu.Unlock()
	return id
}

// Get returns a copy of the job, or nil when missing.
func (r *Registry) Get(jobID uuid.UUID) *Job {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[jobID]
	if !ok || j == nil {
		return nil
	}
	cp := *j
	if j.Result != nil {
		cp.Result = make(map[string]any, len(j.Result))
		for k, v := range j.Result {
			cp.Result[k] = v
		}
	}
	return &cp
}

// MarkProcessing sets status to processing.
func (r *Registry) MarkProcessing(jobID uuid.UUID) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if j := r.jobs[jobID]; j != nil {
		j.Status = StatusProcessing
	}
	r.mu.Unlock()
}

// MarkCompleted stores the grade payload and marks the job completed.
func (r *Registry) MarkCompleted(jobID uuid.UUID, result map[string]any) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if j := r.jobs[jobID]; j != nil {
		j.Status = StatusCompleted
		j.ErrorMessage = ""
		if result != nil {
			j.Result = make(map[string]any, len(result))
			for k, v := range result {
				j.Result[k] = v
			}
		}
	}
	r.mu.Unlock()
}

// MarkFailed stores the error and marks the job failed.
func (r *Registry) MarkFailed(jobID uuid.UUID, errMsg string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	if j := r.jobs[jobID]; j != nil {
		j.Status = StatusFailed
		j.ErrorMessage = errMsg
	}
	r.mu.Unlock()
}