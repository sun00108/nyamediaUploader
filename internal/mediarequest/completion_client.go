package mediarequest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CompleteUploadInput struct {
	RequestCode string `json:"request_code"`
	FileName    string `json:"file_name"`
}

type CompleteUploadResponse struct {
	Status string `json:"status,omitempty"`
}

func (c *Client) CompleteUpload(ctx context.Context, accessToken string, input CompleteUploadInput) (*CompleteUploadResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/media/upload-completions", bytes.NewReader(body))
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
			return nil, fmt.Errorf("complete upload failed: bot returned %s", resp.Status)
		}
		return nil, fmt.Errorf("complete upload failed: bot returned %s: %s", resp.Status, message)
	}

	var out CompleteUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil && err != io.EOF {
		return nil, err
	}
	return &out, nil
}
