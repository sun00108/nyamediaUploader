package mediarequest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type CreateInput struct {
	RequestID int64 `json:"request_id"`
	Season    *int  `json:"season,omitempty"`
	Episode   *int  `json:"episode,omitempty"`
}

type CreateResponse struct {
	MediaTitle  string `json:"media_title"`
	Season      *int   `json:"season,omitempty"`
	Episode     *int   `json:"episode,omitempty"`
	RequestCode string `json:"request_code"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Create(ctx context.Context, accessToken string, input CreateInput) (*CreateResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/media/upload-requests", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return nil, fmt.Errorf("create upload request failed: bot returned %s", resp.Status)
		}
		return nil, fmt.Errorf("create upload request failed: bot returned %s: %s", resp.Status, message)
	}

	var out CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.RequestCode == "" {
		return nil, fmt.Errorf("create upload request failed: bot returned empty request_code")
	}
	return &out, nil
}
