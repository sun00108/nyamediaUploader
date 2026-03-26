package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var nowFunc = time.Now

func Now() time.Time {
	return nowFunc()
}

type Service struct {
	config Config
	store  *Store
	api    ExchangeAPI
}

type CompleteLoginInput struct {
	State             string
	AuthorizationCode string
}

func NewService(cfg Config) *Service {
	return &Service{
		config: cfg,
		store:  NewStore(cfg.ConfigDir),
		api:    NewHTTPExchangeAPI(cfg.BotAPIBaseURL, cfg.ClientID),
	}
}

func (s *Service) SessionPath() string {
	return s.store.SessionPath()
}

func (s *Service) LoadSession(ctx context.Context) (*Session, error) {
	return s.store.LoadSession(ctx)
}

func (s *Service) Logout(ctx context.Context) error {
	return s.store.DeleteSession(ctx)
}

func (s *Service) BeginLogin(_ context.Context) (loginURL string, state string, err error) {
	state, err = generateStateToken()
	if err != nil {
		return "", "", err
	}

	u, err := url.Parse(strings.TrimRight(s.config.BotPublicBaseURL, "/"))
	if err != nil {
		return "", "", fmt.Errorf("invalid bot public base url: %w", err)
	}

	u.Path += "/login"
	query := u.Query()
	query.Set("client_id", s.config.ClientID)
	query.Set("state", state)
	query.Set("source", "cli")
	u.RawQuery = query.Encode()

	return u.String(), state, nil
}

func (s *Service) CompleteLogin(ctx context.Context, input CompleteLoginInput) (*Session, error) {
	if strings.TrimSpace(input.State) == "" {
		return nil, errors.New("missing state")
	}
	if strings.TrimSpace(input.AuthorizationCode) == "" {
		return nil, errors.New("missing authorization code")
	}

	session, err := s.api.ExchangeAuthorizationCode(ctx, ExchangeAuthorizationCodeInput{
		State:             input.State,
		AuthorizationCode: input.AuthorizationCode,
	})
	if err != nil {
		return nil, err
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = Now()
	}
	if session.TokenType == "" {
		session.TokenType = "Bearer"
	}

	if err := s.store.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func generateStateToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
