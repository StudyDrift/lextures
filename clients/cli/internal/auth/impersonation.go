package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const impersonationFileName = ".lextures-impersonation"

// ImpersonationSession holds the real and impersonation tokens separately from
// the default profile token store.
type ImpersonationSession struct {
	RealAccessToken    string `json:"realAccessToken"`
	ImpersonationToken string `json:"impersonationToken"`
	TargetUserID       string `json:"targetUserId,omitempty"`
	ExpiresAt          string `json:"expiresAt,omitempty"`
}

// ImpersonationStore persists active impersonation sessions per profile.
type ImpersonationStore interface {
	Load(profile string) (*ImpersonationSession, error)
	Save(profile string, session *ImpersonationSession) error
	Delete(profile string) error
	Backend() string
}

type impersonationFileStore struct {
	path string
	mu   sync.Mutex
}

func newImpersonationFileStore() *impersonationFileStore {
	home, _ := os.UserHomeDir()
	return &impersonationFileStore{path: filepath.Join(home, impersonationFileName)}
}

// NewImpersonationStoreAt returns a file-backed impersonation store at path (tests).
func NewImpersonationStoreAt(path string) ImpersonationStore {
	return &impersonationFileStore{path: path}
}

// NewImpersonationStore returns the impersonation session store.
func NewImpersonationStore() ImpersonationStore {
	return newImpersonationFileStore()
}

type impersonationMap map[string]*ImpersonationSession

func (f *impersonationFileStore) read() (impersonationMap, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(impersonationMap), nil
	}
	if err != nil {
		return nil, err
	}
	var m impersonationMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("corrupt impersonation file %s: %w", f.path, err)
	}
	return m, nil
}

func (f *impersonationFileStore) write(m impersonationMap) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0o600)
}

func (f *impersonationFileStore) Load(profile string) (*ImpersonationSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return nil, err
	}
	return m[profile], nil
}

func (f *impersonationFileStore) Save(profile string, session *ImpersonationSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	m[profile] = session
	return f.write(m)
}

func (f *impersonationFileStore) Delete(profile string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	delete(m, profile)
	return f.write(m)
}

func (f *impersonationFileStore) Backend() string { return "file" }