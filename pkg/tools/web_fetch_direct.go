package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type directWebFetcher struct {
	proxy         string
	failOnNon2xx  bool
	redirectLimit int
}

func (f *directWebFetcher) Fetch(ctx context.Context, urlStr string) (*webFetcherResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client, err := createHTTPClient(f.proxy, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	redirectLimit := f.redirectLimit
	if redirectLimit <= 0 {
		redirectLimit = 5
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= redirectLimit {
			return fmt.Errorf("stopped after %d redirects", redirectLimit)
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if f.failOnNon2xx && (resp.StatusCode < http.StatusOK || resp.StatusCode >= 300) {
		return nil, fmt.Errorf("upstream status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	var text string
	extractor := "raw"

	if strings.Contains(contentType, "application/json") {
		var jsonData any
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
			extractor = "json"
		} else {
			text = string(body)
		}
	} else if strings.Contains(contentType, "text/html") || len(body) > 0 &&
		(strings.HasPrefix(string(body), "<!DOCTYPE") || strings.HasPrefix(strings.ToLower(string(body)), "<html")) {
		text = extractWebText(string(body))
		extractor = "text"
	} else {
		text = string(body)
	}

	return &webFetcherResult{
		URL:       urlStr,
		Status:    resp.StatusCode,
		Extractor: extractor,
		Text:      text,
	}, nil
}
