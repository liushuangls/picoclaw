package tools

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

const bigModelReaderAPIURL = "https://open.bigmodel.cn/api/paas/v4/reader"

type bigModelReaderErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type bigModelReaderResponse struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	RequestID string `json:"request_id"`
	Model     string `json:"model"`
	Result    struct {
		Content     string `json:"content"`
		Description string `json:"description"`
		Title       string `json:"title"`
		URL         string `json:"url"`
	} `json:"reader_result"`
}

type bigModelReaderFetcher struct {
	apiKey    string
	readerURL string
	proxy     string
}

func (f *bigModelReaderFetcher) Fetch(ctx context.Context, urlStr string) (*webFetcherResult, error) {
	if strings.TrimSpace(f.apiKey) == "" {
		return nil, fmt.Errorf("bigmodel api key not configured")
	}

	apiURL := strings.TrimSpace(f.readerURL)
	if apiURL == "" {
		apiURL = bigModelReaderAPIURL
	}

	payload := map[string]string{
		"url": urlStr,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+f.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client, err := createHTTPClient(f.proxy, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("create HTTP client failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		var errResp bigModelReaderErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil &&
			(strings.TrimSpace(errResp.Error.Code) != "" || strings.TrimSpace(errResp.Error.Message) != "") {
			return nil, fmt.Errorf("api error %s: %s", strings.TrimSpace(errResp.Error.Code), strings.TrimSpace(errResp.Error.Message))
		}
		return nil, fmt.Errorf("api status %d: %s", resp.StatusCode, string(body))
	}

	var readerResp bigModelReaderResponse
	if err := json.Unmarshal(body, &readerResp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	parts := make([]string, 0, 3)
	if title := strings.TrimSpace(readerResp.Result.Title); title != "" {
		parts = append(parts, title)
	}
	if description := strings.TrimSpace(readerResp.Result.Description); description != "" {
		parts = append(parts, description)
	}
	if content := strings.TrimSpace(readerResp.Result.Content); content != "" {
		parts = append(parts, content)
	}

	text := strings.Join(parts, "\n")
	if text == "" {
		return nil, fmt.Errorf("empty reader_result content")
	}

	resultURL := strings.TrimSpace(readerResp.Result.URL)
	if resultURL == "" {
		resultURL = urlStr
	}

	return &webFetcherResult{
		URL:       resultURL,
		Status:    http.StatusOK,
		Extractor: "bigmodel-reader",
		Text:      text,
	}, nil
}
