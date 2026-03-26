package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	graphFragmentSize = 10 * 1024 * 1024
)

type Result struct {
	StatusCode int
	Body       map[string]any
}

func UploadFile(ctx context.Context, localPath, uploadURL string, progress io.Writer) (*Result, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	totalSize := info.Size()
	if totalSize <= 0 {
		return nil, fmt.Errorf("local file is empty")
	}

	client := &http.Client{Timeout: 0}
	buffer := make([]byte, graphFragmentSize)
	var offset int64

	for offset < totalSize {
		chunkSize := int64(len(buffer))
		remaining := totalSize - offset
		if remaining < chunkSize {
			chunkSize = remaining
		}

		n, err := file.ReadAt(buffer[:chunkSize], offset)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if int64(n) != chunkSize {
			return nil, fmt.Errorf("short read at offset %d", offset)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(buffer[:chunkSize]))
		if err != nil {
			return nil, err
		}
		req.ContentLength = chunkSize
		req.Header.Set("Content-Length", fmt.Sprintf("%d", chunkSize))
		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, offset+chunkSize-1, totalSize))

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upload failed at bytes %d-%d: onedrive returned %s", offset, offset+chunkSize-1, resp.Status)
		}

		offset += chunkSize
		if progress != nil {
			fmt.Fprintf(progress, "\rUploaded %d/%d bytes", offset, totalSize)
		}

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			if progress != nil {
				fmt.Fprintln(progress)
			}
			return &Result{
				StatusCode: resp.StatusCode,
				Body:       body,
			}, nil
		}
	}

	if progress != nil {
		fmt.Fprintln(progress)
	}

	return &Result{StatusCode: http.StatusAccepted}, nil
}
