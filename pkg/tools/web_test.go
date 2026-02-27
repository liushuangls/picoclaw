package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestWebTool_WebFetch_Success verifies successful URL fetching
func TestWebTool_WebFetch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Page</h1><p>Content here</p></body></html>"))
	}))
	defer server.Close()

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain the fetched content
	if !strings.Contains(result.ForUser, "Test Page") {
		t.Errorf("Expected ForUser to contain 'Test Page', got: %s", result.ForUser)
	}

	// ForLLM should contain summary
	if !strings.Contains(result.ForLLM, "bytes") && !strings.Contains(result.ForLLM, "extractor") {
		t.Errorf("Expected ForLLM to contain summary, got: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_JSON verifies JSON content handling
func TestWebTool_WebFetch_JSON(t *testing.T) {
	testData := map[string]string{"key": "value", "number": "123"}
	expectedJSON, _ := json.MarshalIndent(testData, "", "  ")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedJSON)
	}))
	defer server.Close()

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain formatted JSON
	if !strings.Contains(result.ForUser, "key") && !strings.Contains(result.ForUser, "value") {
		t.Errorf("Expected ForUser to contain JSON data, got: %s", result.ForUser)
	}
}

// TestWebTool_WebFetch_InvalidURL verifies error handling for invalid URL
func TestWebTool_WebFetch_InvalidURL(t *testing.T) {
	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": "not-a-valid-url",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for invalid URL")
	}

	// Should contain error message (either "invalid URL" or scheme error)
	if !strings.Contains(result.ForLLM, "URL") && !strings.Contains(result.ForUser, "URL") {
		t.Errorf("Expected error message for invalid URL, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_UnsupportedScheme verifies error handling for non-http URLs
func TestWebTool_WebFetch_UnsupportedScheme(t *testing.T) {
	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": "ftp://example.com/file.txt",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for unsupported URL scheme")
	}

	// Should mention only http/https allowed
	if !strings.Contains(result.ForLLM, "http/https") && !strings.Contains(result.ForUser, "http/https") {
		t.Errorf("Expected scheme error message, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_MissingURL verifies error handling for missing URL
func TestWebTool_WebFetch_MissingURL(t *testing.T) {
	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when URL is missing")
	}

	// Should mention URL is required
	if !strings.Contains(result.ForLLM, "url is required") && !strings.Contains(result.ForUser, "url is required") {
		t.Errorf("Expected 'url is required' message, got ForLLM: %s", result.ForLLM)
	}
}

// TestWebTool_WebFetch_Truncation verifies content truncation
func TestWebTool_WebFetch_Truncation(t *testing.T) {
	longContent := strings.Repeat("x", 20000)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longContent))
	}))
	defer server.Close()

	tool := NewWebFetchTool(1000) // Limit to 1000 chars
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain truncated content (not the full 20000 chars)
	resultMap := make(map[string]any)
	json.Unmarshal([]byte(result.ForUser), &resultMap)
	if text, ok := resultMap["text"].(string); ok {
		if len(text) > 1100 { // Allow some margin
			t.Errorf("Expected content to be truncated to ~1000 chars, got: %d", len(text))
		}
	}

	// Should be marked as truncated
	if truncated, ok := resultMap["truncated"].(bool); !ok || !truncated {
		t.Errorf("Expected 'truncated' to be true in result")
	}
}

// TestWebTool_WebSearch_NoApiKey verifies that no tool is created when API key is missing
func TestWebTool_WebSearch_NoApiKey(t *testing.T) {
	tool := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKey: ""})
	if tool != nil {
		t.Errorf("Expected nil tool when Brave API key is empty")
	}

	// Also nil when nothing is enabled
	tool = NewWebSearchTool(WebSearchToolOptions{})
	if tool != nil {
		t.Errorf("Expected nil tool when no provider is enabled")
	}
}

// TestWebTool_WebSearch_MissingQuery verifies error handling for missing query
func TestWebTool_WebSearch_MissingQuery(t *testing.T) {
	tool := NewWebSearchTool(WebSearchToolOptions{BraveEnabled: true, BraveAPIKey: "test-key", BraveMaxResults: 5})
	ctx := context.Background()
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when query is missing")
	}
}

// TestWebTool_WebFetch_HTMLExtraction verifies HTML text extraction
func TestWebTool_WebFetch_HTMLExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<html><body><script>alert('test');</script><style>body{color:red;}</style><h1>Title</h1><p>Content</p></body></html>`,
			),
		)
	}))
	defer server.Close()

	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": server.URL,
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain extracted text (without script/style tags)
	if !strings.Contains(result.ForUser, "Title") && !strings.Contains(result.ForUser, "Content") {
		t.Errorf("Expected ForUser to contain extracted text, got: %s", result.ForUser)
	}

	// Should NOT contain script or style tags
	if strings.Contains(result.ForUser, "<script>") || strings.Contains(result.ForUser, "<style>") {
		t.Errorf("Expected script/style tags to be removed, got: %s", result.ForUser)
	}
}

// TestWebFetchTool_extractText verifies text extraction preserves newlines
func TestWebFetchTool_extractText(t *testing.T) {
	tool := &WebFetchTool{}

	tests := []struct {
		name     string
		input    string
		wantFunc func(t *testing.T, got string)
	}{
		{
			name:  "preserves newlines between block elements",
			input: "<html><body><h1>Title</h1>\n<p>Paragraph 1</p>\n<p>Paragraph 2</p></body></html>",
			wantFunc: func(t *testing.T, got string) {
				lines := strings.Split(got, "\n")
				if len(lines) < 2 {
					t.Errorf("Expected multiple lines, got %d: %q", len(lines), got)
				}
				if !strings.Contains(got, "Title") || !strings.Contains(got, "Paragraph 1") ||
					!strings.Contains(got, "Paragraph 2") {
					t.Errorf("Missing expected text: %q", got)
				}
			},
		},
		{
			name:  "removes script and style tags",
			input: "<script>alert('x');</script><style>body{}</style><p>Keep this</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "alert") || strings.Contains(got, "body{}") {
					t.Errorf("Expected script/style content removed, got: %q", got)
				}
				if !strings.Contains(got, "Keep this") {
					t.Errorf("Expected 'Keep this' to remain, got: %q", got)
				}
			},
		},
		{
			name:  "collapses excessive blank lines",
			input: "<p>A</p>\n\n\n\n\n<p>B</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "\n\n\n") {
					t.Errorf("Expected excessive blank lines collapsed, got: %q", got)
				}
			},
		},
		{
			name:  "collapses horizontal whitespace",
			input: "<p>hello     world</p>",
			wantFunc: func(t *testing.T, got string) {
				if strings.Contains(got, "     ") {
					t.Errorf("Expected spaces collapsed, got: %q", got)
				}
				if !strings.Contains(got, "hello world") {
					t.Errorf("Expected 'hello world', got: %q", got)
				}
			},
		},
		{
			name:  "empty input",
			input: "",
			wantFunc: func(t *testing.T, got string) {
				if got != "" {
					t.Errorf("Expected empty string, got: %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tool.extractText(tt.input)
			tt.wantFunc(t, got)
		})
	}
}

// TestWebTool_WebFetch_MissingDomain verifies error handling for URL without domain
func TestWebTool_WebFetch_MissingDomain(t *testing.T) {
	tool := NewWebFetchTool(50000)
	ctx := context.Background()
	args := map[string]any{
		"url": "https://",
	}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error for URL without domain")
	}

	// Should mention missing domain
	if !strings.Contains(result.ForLLM, "domain") && !strings.Contains(result.ForUser, "domain") {
		t.Errorf("Expected domain error message, got ForLLM: %s", result.ForLLM)
	}
}

func TestWebTool_WebFetch_FallbackToBigModelOnRequestFailure(t *testing.T) {
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/reader" {
			t.Errorf("Expected path /reader, got %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload failed: %v", err)
		}
		if payload["url"] != "http://127.0.0.1:1" {
			t.Errorf("Expected payload url, got %v", payload["url"])
		}

		resp := map[string]any{
			"id":         "rdr_123",
			"created":    123,
			"request_id": "req_123",
			"model":      "reader",
			"reader_result": map[string]any{
				"title":       "Fallback Page",
				"url":         "https://example.com/fallback",
				"description": "Fallback description",
				"content":     "Fallback content from BigModel",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tool := &WebFetchTool{
		maxChars:          50000,
		bigModelAPIKey:    "test-zhipu-key",
		bigModelReaderURL: server.URL + "/reader",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"url": "http://127.0.0.1:1",
	})

	if result.IsError {
		t.Fatalf("Expected fallback success, got error: %s", result.ForLLM)
	}
	if seenAuthorization != "Bearer test-zhipu-key" {
		t.Fatalf("Authorization header = %q, want %q", seenAuthorization, "Bearer test-zhipu-key")
	}
	if !strings.Contains(result.ForUser, "bigmodel-reader") {
		t.Fatalf("Expected bigmodel extractor, got: %s", result.ForUser)
	}

	var resultMap map[string]any
	if err := json.Unmarshal([]byte(result.ForUser), &resultMap); err != nil {
		t.Fatalf("failed to parse result json: %v", err)
	}

	text, _ := resultMap["text"].(string)
	if text == "" {
		t.Fatalf("Expected non-empty fallback text, got: %s", result.ForUser)
	}

	titleIdx := strings.Index(text, "Fallback Page")
	descriptionIdx := strings.Index(text, "Fallback description")
	contentIdx := strings.Index(text, "Fallback content from BigModel")
	if titleIdx == -1 || descriptionIdx == -1 || contentIdx == -1 {
		t.Fatalf("Expected title/description/content in text, got: %s", text)
	}
	if !(titleIdx < descriptionIdx && descriptionIdx < contentIdx) {
		t.Fatalf("Expected order title->description->content, got: %s", text)
	}
}

func TestWebTool_WebFetch_RequestFailureWithoutBigModelKey(t *testing.T) {
	tool := NewWebFetchTool(50000)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"url": "http://127.0.0.1:1",
	})
	if !result.IsError {
		t.Fatalf("Expected error when BigModel key not configured, got success: %s", result.ForUser)
	}
}

func TestCreateHTTPClient_ProxyConfigured(t *testing.T) {
	client, err := createHTTPClient("http://127.0.0.1:7890", 12*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}
	if client.Timeout != 12*time.Second {
		t.Fatalf("client.Timeout = %v, want %v", client.Timeout, 12*time.Second)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want non-nil")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "http://127.0.0.1:7890")
	}
}

func TestCreateHTTPClient_InvalidProxy(t *testing.T) {
	_, err := createHTTPClient("://bad-proxy", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for invalid proxy URL, got nil")
	}
}

func TestCreateHTTPClient_Socks5ProxyConfigured(t *testing.T) {
	client, err := createHTTPClient("socks5://127.0.0.1:1080", 8*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy URL = %v, want %q", proxyURL, "socks5://127.0.0.1:1080")
	}
}

func TestCreateHTTPClient_UnsupportedProxyScheme(t *testing.T) {
	_, err := createHTTPClient("ftp://127.0.0.1:21", 10*time.Second)
	if err == nil {
		t.Fatal("createHTTPClient() expected error for unsupported scheme, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "unsupported proxy scheme")
	}
}

func TestCreateHTTPClient_ProxyFromEnvironmentWhenConfigEmpty(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("http_proxy", "http://127.0.0.1:8888")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("https_proxy", "http://127.0.0.1:8888")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	client, err := createHTTPClient("", 10*time.Second)
	if err != nil {
		t.Fatalf("createHTTPClient() error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("transport.Proxy is nil, want proxy function from environment")
	}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	if _, err := tr.Proxy(req); err != nil {
		t.Fatalf("transport.Proxy(req) error: %v", err)
	}
}

func TestNewWebFetchToolWithProxy(t *testing.T) {
	tool := NewWebFetchToolWithProxy(1024, "http://127.0.0.1:7890")
	if tool.maxChars != 1024 {
		t.Fatalf("maxChars = %d, want %d", tool.maxChars, 1024)
	}
	if tool.proxy != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %q, want %q", tool.proxy, "http://127.0.0.1:7890")
	}

	tool = NewWebFetchToolWithProxy(0, "http://127.0.0.1:7890")
	if tool.maxChars != 50000 {
		t.Fatalf("default maxChars = %d, want %d", tool.maxChars, 50000)
	}
}

func TestNewWebSearchTool_PropagatesProxy(t *testing.T) {
	t.Run("serper", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			SerperEnabled:    true,
			SerperAPIKey:     "k",
			SerperBaseURL:    "https://google.serper.dev",
			SerperMaxResults: 3,
			Proxy:            "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*SerperSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *SerperSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("perplexity", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			PerplexityEnabled:    true,
			PerplexityAPIKey:     "k",
			PerplexityMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*PerplexitySearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *PerplexitySearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("brave", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			BraveEnabled:    true,
			BraveAPIKey:     "k",
			BraveMaxResults: 3,
			Proxy:           "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*BraveSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *BraveSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})

	t.Run("duckduckgo", func(t *testing.T) {
		tool := NewWebSearchTool(WebSearchToolOptions{
			DuckDuckGoEnabled:    true,
			DuckDuckGoMaxResults: 3,
			Proxy:                "http://127.0.0.1:7890",
		})
		p, ok := tool.provider.(*DuckDuckGoSearchProvider)
		if !ok {
			t.Fatalf("provider type = %T, want *DuckDuckGoSearchProvider", tool.provider)
		}
		if p.proxy != "http://127.0.0.1:7890" {
			t.Fatalf("provider proxy = %q, want %q", p.proxy, "http://127.0.0.1:7890")
		}
	})
}

func TestNewWebSearchTool_PrefersSerper(t *testing.T) {
	tool := NewWebSearchTool(WebSearchToolOptions{
		SerperEnabled:        true,
		SerperAPIKey:         "serper-key",
		PerplexityEnabled:    true,
		PerplexityAPIKey:     "perplexity-key",
		BraveEnabled:         true,
		BraveAPIKey:          "brave-key",
		DuckDuckGoEnabled:    true,
		DuckDuckGoMaxResults: 3,
	})
	p, ok := tool.provider.(*SerperSearchProvider)
	if !ok {
		t.Fatalf("provider type = %T, want *SerperSearchProvider", tool.provider)
	}
	if p.apiKey != "serper-key" {
		t.Fatalf("provider apiKey = %q, want %q", p.apiKey, "serper-key")
	}
}

// TestWebTool_TavilySearch_Success verifies successful Tavily search
func TestWebTool_TavilySearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify payload
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["api_key"] != "test-key" {
			t.Errorf("Expected api_key test-key, got %v", payload["api_key"])
		}
		if payload["query"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", payload["query"])
		}

		// Return mock response
		response := map[string]any{
			"results": []map[string]any{
				{
					"title":   "Test Result 1",
					"url":     "https://example.com/1",
					"content": "Content for result 1",
				},
				{
					"title":   "Test Result 2",
					"url":     "https://example.com/2",
					"content": "Content for result 2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool(WebSearchToolOptions{
		TavilyEnabled:    true,
		TavilyAPIKey:     "test-key",
		TavilyBaseURL:    server.URL,
		TavilyMaxResults: 5,
	})

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	// Success should not be an error
	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}

	// ForUser should contain result titles and URLs
	if !strings.Contains(result.ForUser, "Test Result 1") ||
		!strings.Contains(result.ForUser, "https://example.com/1") {
		t.Errorf("Expected results in output, got: %s", result.ForUser)
	}

	// Should mention via Tavily
	if !strings.Contains(result.ForUser, "via Tavily") {
		t.Errorf("Expected 'via Tavily' in output, got: %s", result.ForUser)
	}
}

func TestWebTool_SerperSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("Expected path /search, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-KEY") != "serper-key" {
			t.Errorf("Expected X-API-KEY serper-key, got %s", r.Header.Get("X-API-KEY"))
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		if payload["q"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", payload["q"])
		}
		if payload["num"] != float64(4) {
			t.Errorf("Expected num 4, got %v", payload["num"])
		}

		response := map[string]any{
			"organic": []map[string]any{
				{
					"title":   "Serper Result 1",
					"link":    "https://example.com/1",
					"snippet": "Serper snippet 1",
				},
				{
					"title":   "Serper Result 2",
					"link":    "https://example.com/2",
					"snippet": "Serper snippet 2",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool(WebSearchToolOptions{
		SerperEnabled:    true,
		SerperAPIKey:     "serper-key",
		SerperBaseURL:    server.URL,
		SerperMaxResults: 4,
	})

	ctx := context.Background()
	args := map[string]any{
		"query": "test query",
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success, got IsError=true: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "via Serper") {
		t.Errorf("Expected 'via Serper' in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "Serper Result 1") ||
		!strings.Contains(result.ForUser, "https://example.com/1") {
		t.Errorf("Expected Serper result in output, got: %s", result.ForUser)
	}
}

func TestWebTool_SerperSearch_RealRequest(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("SERPER_API_KEY"))
	if apiKey == "" {
		t.Skip("SERPER_API_KEY is empty")
	}

	tool := NewWebSearchTool(WebSearchToolOptions{
		SerperEnabled:    true,
		SerperAPIKey:     apiKey,
		SerperMaxResults: 5,
	})
	if tool == nil {
		t.Fatal("NewWebSearchTool returned nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"query": "golang context package tutorial",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "via Serper") {
		t.Fatalf("expected output contains via Serper, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "http") {
		t.Fatalf("expected output contains link, got: %s", result.ForUser)
	}
	t.Logf("Serper search result:\n%s", result.ForUser)
}

func TestDirectWebFetcher_RealRequest(t *testing.T) {
	if strings.TrimSpace(os.Getenv("PICOCLAW_INTEGRATION_TESTS")) != "1" {
		t.Skip("skipping integration test (set PICOCLAW_INTEGRATION_TESTS=1 to enable)")
	}

	fetcher := &directWebFetcher{
		redirectLimit: 5,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := fetcher.Fetch(ctx, "https://docs.bigmodel.cn/cn/guide/start/model-overview")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status < 200 || result.Status >= 400 {
		t.Fatalf("unexpected status code: %d", result.Status)
	}
	if strings.TrimSpace(result.Text) == "" {
		t.Fatal("expected non-empty text")
	}
	t.Logf("Direct web fetcher result preview: %+v", result)
}

func TestBigModelReaderFetcher_RealRequest(t *testing.T) {
	if strings.TrimSpace(os.Getenv("PICOCLAW_INTEGRATION_TESTS")) != "1" {
		t.Skip("skipping integration test (set PICOCLAW_INTEGRATION_TESTS=1 to enable)")
	}

	apiKey := strings.TrimSpace(os.Getenv("BIGMODEL_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("PICOCLAW_TOOLS_WEB_FETCH_BIGMODEL_API_KEY"))
	}
	if apiKey == "" {
		t.Skip("BIGMODEL_API_KEY / PICOCLAW_TOOLS_WEB_FETCH_BIGMODEL_API_KEY is empty")
	}

	fetcher := &bigModelReaderFetcher{
		apiKey: apiKey,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := fetcher.Fetch(ctx, "https://docs.bigmodel.cn/cn/guide/start/model-overview")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Extractor != "bigmodel-reader" {
		t.Fatalf("unexpected extractor: %s", result.Extractor)
	}
	if strings.TrimSpace(result.Text) == "" {
		t.Fatal("expected non-empty text")
	}
	t.Logf("BigModel reader result preview: %+v", result)
}
