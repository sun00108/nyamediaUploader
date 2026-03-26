package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ExchangeAPI interface {
	ExchangeAuthorizationCode(ctx context.Context, input ExchangeAuthorizationCodeInput) (*Session, error)
}

type ExchangeAuthorizationCodeInput struct {
	State             string `json:"state"`
	AuthorizationCode string `json:"authorization_code"`
}

type HTTPExchangeAPI struct {
	baseURL    string
	clientID   string
	httpClient *http.Client
}

func NewHTTPExchangeAPI(baseURL, clientID string) *HTTPExchangeAPI {
	return &HTTPExchangeAPI{
		baseURL:  strings.TrimRight(baseURL, "/"),
		clientID: clientID,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (a *HTTPExchangeAPI) ExchangeAuthorizationCode(ctx context.Context, input ExchangeAuthorizationCodeInput) (*Session, error) {
	body, err := json.Marshal(struct {
		ClientID string `json:"client_id"`
		ExchangeAuthorizationCodeInput
	}{
		ClientID:                       a.clientID,
		ExchangeAuthorizationCodeInput: input,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/cli/login/exchange", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login exchange failed: bot returned %s", resp.Status)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}
