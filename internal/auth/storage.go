package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	configDir string
}

func NewStore(configDir string) *Store {
	return &Store{configDir: configDir}
}

func (s *Store) SessionPath() string {
	return filepath.Join(s.configDir, "session.json")
}

func (s *Store) LoadSession(_ context.Context) (*Session, error) {
	data, err := os.ReadFile(s.SessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Store) SaveSession(_ context.Context, session *Session) error {
	if err := os.MkdirAll(s.configDir, 0o755); err != nil {
		return err
	}

	return session.WriteToFile(s.SessionPath())
}

func (s *Store) DeleteSession(_ context.Context) error {
	err := os.Remove(s.SessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete session file %s: %w", s.SessionPath(), err)
	}

	return nil
}
