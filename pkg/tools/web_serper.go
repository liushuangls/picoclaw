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

type SerperSearchProvider struct {
	apiKey  string
	baseURL string
	proxy   string
}

func (p *SerperSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(p.baseURL), "/")
	if baseURL == "" {
		baseURL = "https://google.serper.dev"
	}
	searchURL := baseURL + "/search"

	payload := map[string]any{
		"q":           query,
		"type":        "search",
		"autocorrect": true,
		"page":        1,
		"num":         count,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", p.apiKey)
	req.Header.Set("User-Agent", userAgent)

	client, err := createHTTPClient(p.proxy, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("serper api error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		AnswerBox struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"answerBox"`
		KnowledgeGraph struct {
			Title           string `json:"title"`
			Description     string `json:"description"`
			DescriptionLink string `json:"descriptionLink"`
		} `json:"knowledgeGraph"`
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via Serper)", query))

	serial := 0
	appendResult := func(title, link, snippet string) {
		if serial >= count {
			return
		}
		if strings.TrimSpace(title) == "" || strings.TrimSpace(link) == "" {
			return
		}
		serial++
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", serial, title, link))
		if strings.TrimSpace(snippet) != "" {
			lines = append(lines, fmt.Sprintf("   %s", snippet))
		}
	}

	appendResult(searchResp.AnswerBox.Title, searchResp.AnswerBox.Link, searchResp.AnswerBox.Snippet)
	appendResult(
		searchResp.KnowledgeGraph.Title,
		searchResp.KnowledgeGraph.DescriptionLink,
		searchResp.KnowledgeGraph.Description,
	)
	for _, item := range searchResp.Organic {
		if serial >= count {
			break
		}
		appendResult(item.Title, item.Link, item.Snippet)
	}

	if serial == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	return strings.Join(lines, "\n"), nil
}
