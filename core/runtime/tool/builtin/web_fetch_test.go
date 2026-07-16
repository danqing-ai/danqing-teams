package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractPageContent_Readability(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Hello Article</title></head>
<body>
<article>
<h1>Hello Article</h1>
<p>This is a long enough paragraph about cats and dogs to satisfy readability extraction heuristics for unit testing purposes.</p>
<p>Another paragraph with more meaningful content so the article extractor has enough text to work with reliably.</p>
</article>
</body></html>`
	content, extractor, title, spa := extractPageContent(html, "https://example.com/post")
	if extractor != "readability" && extractor != "html" {
		t.Fatalf("extractor = %q", extractor)
	}
	if !strings.Contains(content, "cats and dogs") {
		t.Fatalf("content missing body text: %q", content)
	}
	if spa {
		t.Fatal("did not expect SPA hint for article page")
	}
	_ = title
}

func TestIsSPAShell(t *testing.T) {
	html := `<html><body><div id="root"></div><script></script><script></script><script></script></body></html>`
	if !isSPAShell(html, "ok") {
		// extracted is short (<200)
		if !isSPAShell(html, "x") {
			t.Fatal("expected SPA shell detection")
		}
	}
	if !isSPAShell(html, "short") {
		t.Fatal("expected SPA shell for short extract")
	}
	long := strings.Repeat("word ", 80)
	if isSPAShell(html, long) {
		t.Fatal("did not expect SPA hint when extract is long")
	}
}

func TestWebFetch_ExecuteJSON(t *testing.T) {
	testSkipSSRF = true
	defer func() { testSkipSSRF = false }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	h := &WebFetch{}
	res, err := h.Execute(context.Background(), map[string]any{
		"url":       srv.URL,
		"max_chars": float64(8000),
	})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(res.Content), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["extractor"] != "json" {
		t.Fatalf("extractor = %v", parsed["extractor"])
	}
	if !strings.Contains(parsed["content"].(string), "hello") {
		t.Fatalf("content = %v", parsed["content"])
	}
}

func TestWebFetch_BlocksLocalhost(t *testing.T) {
	h := &WebFetch{}
	_, err := h.Execute(context.Background(), map[string]any{
		"url": "http://127.0.0.1:9/",
	})
	if err == nil {
		t.Fatal("expected SSRF error")
	}
	if !strings.Contains(err.Error(), "blocked") && !strings.Contains(err.Error(), "private") && !strings.Contains(err.Error(), "local") {
		t.Fatalf("unexpected error: %v", err)
	}
}
