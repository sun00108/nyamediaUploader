package auth

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

type Session struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	Username    string     `json:"username,omitempty"`
	TelegramID  int64      `json:"telegram_id,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (s *Session) IsValidAt(now time.Time) bool {
	if s == nil || s.AccessToken == "" {
		return false
	}

	if s.ExpiresAt == nil {
		return true
	}

	return now.Before(*s.ExpiresAt)
}

func (s *Session) DisplayUser() string {
	if s == nil {
		return "unknown"
	}
	if s.Username != "" {
		return s.Username
	}
	if s.TelegramID != 0 {
		return "telegram:" + strconv.FormatInt(s.TelegramID, 10)
	}
	return "unknown"
}

func (s *Session) WriteToFile(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o600)
}
