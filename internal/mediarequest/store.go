package mediarequest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

var ErrRequestStoreNotFound = errors.New("request store not found")

type Item struct {
	RequestID   int64     `json:"request_id"`
	RequestCode string    `json:"request_code"`
	MediaTitle  string    `json:"media_title"`
	Season      *int      `json:"season,omitempty"`
	Episode     *int      `json:"episode,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	configDir string
}

func NewStore(configDir string) *Store {
	return &Store{configDir: configDir}
}

func (s *Store) Path() string {
	return filepath.Join(s.configDir, "upload_requests.json")
}

func (s *Store) Load(_ context.Context) ([]Item, error) {
	data, err := os.ReadFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrRequestStoreNotFound
	}
	if err != nil {
		return nil, err
	}

	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) Save(_ context.Context, items []Item) error {
	if err := os.MkdirAll(s.configDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path(), append(data, '\n'), 0o600)
}

func (s *Store) Add(ctx context.Context, item Item) error {
	items, err := s.Load(ctx)
	if err != nil && !errors.Is(err, ErrRequestStoreNotFound) {
		return err
	}

	items = slices.DeleteFunc(items, func(existing Item) bool {
		return existing.RequestCode == item.RequestCode
	})
	items = append(items, item)
	return s.Save(ctx, items)
}

func (s *Store) RemoveByCode(ctx context.Context, requestCode string) error {
	items, err := s.Load(ctx)
	if err != nil {
		if errors.Is(err, ErrRequestStoreNotFound) {
			return nil
		}
		return err
	}

	items = slices.DeleteFunc(items, func(item Item) bool {
		return item.RequestCode == requestCode
	})
	return s.Save(ctx, items)
}
