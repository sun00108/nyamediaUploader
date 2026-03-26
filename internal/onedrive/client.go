package onedrive

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

type CreateUploadSessionInput struct {
	Path             string `json:"path,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	FileSize         int64  `json:"file_size"`
	ConflictBehavior string `json:"conflict_behavior,omitempty"`
	RequestCode      string `json:"request_code,omitempty"`
}

type CreateUploadSessionResponse struct {
	UploadURL          string `json:"upload_url"`
	ExpirationDateTime string `json:"expiration_date_time"`
	Path               string `json:"path,omitempty"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateUploadSession(ctx context.Context, accessToken string, input CreateUploadSessionInput) (*CreateUploadSessionResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	endpoint := c.baseURL + "/api/onedrive/upload-sessions"
	if input.RequestCode != "" {
		endpoint = c.baseURL + "/api/media/upload-sessions"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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
			return nil, fmt.Errorf("create upload session failed: bot returned %s", resp.Status)
		}
		return nil, fmt.Errorf("create upload session failed: bot returned %s: %s", resp.Status, message)
	}

	var out CreateUploadSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.UploadURL == "" {
		return nil, fmt.Errorf("create upload session failed: bot returned empty upload_url")
	}

	return &out, nil
}
