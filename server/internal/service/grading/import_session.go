package grading

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// ImportSession is a validated gradebook import waiting for confirm or cancel.
type ImportSession struct {
	Token      uuid.UUID
	CourseID   uuid.UUID
	ActorID    uuid.UUID
	Grades     map[string]map[string]string
	RequireAck bool
	Expires    time.Time
}

// ImportSessionStore keeps pending imports in memory for ImportSessionTTL.
type ImportSessionStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[uuid.UUID]ImportSession
	now   func() time.Time
}

// DefaultImportSessions is the process-wide pending-import table (plan 3.11).
var DefaultImportSessions = NewImportSessionStore(ImportSessionTTL)

// NewImportSessionStore returns an empty store.
func NewImportSessionStore(ttl time.Duration) *ImportSessionStore {
	if ttl <= 0 {
		ttl = ImportSessionTTL
	}
	return &ImportSessionStore{
		ttl:   ttl,
		items: map[uuid.UUID]ImportSession{},
		now:   time.Now,
	}
}

// Save stores a session, replacing any previous token.
func (s *ImportSessionStore) Save(sess ImportSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	if sess.Expires.IsZero() {
		sess.Expires = s.now().Add(s.ttl)
	}
	s.items[sess.Token] = sess
}

// Delete drops a session when the token, course, and actor match.
func (s *ImportSessionStore) Delete(token, courseID, actorID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	sess, ok := s.items[token]
	if !ok || sess.CourseID != courseID || sess.ActorID != actorID {
		return false
	}
	delete(s.items, token)
	return true
}

// Take returns and removes a matching unexpired session.
func (s *ImportSessionStore) Take(token, courseID, actorID uuid.UUID) (ImportSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	sess, ok := s.items[token]
	if !ok || sess.CourseID != courseID || sess.ActorID != actorID {
		return ImportSession{}, false
	}
	delete(s.items, token)
	return sess, true
}

func (s *ImportSessionStore) sweepLocked() {
	now := s.now()
	for id, sess := range s.items {
		if !sess.Expires.After(now) {
			delete(s.items, id)
		}
	}
}
