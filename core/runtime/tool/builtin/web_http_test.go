package builtin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpgradeToHTTPS(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://example.com/a", "https://example.com/a"},
		{"http://example.com/a", "https://example.com/a"},
		{"http://example.com:80/a", "https://example.com/a"},
		{"http://example.com:8080/a", "http://example.com:8080/a"},
	}
	for _, tt := range tests {
		if got := upgradeToHTTPS(tt.in); got != tt.want {
			t.Errorf("upgradeToHTTPS(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAssertPublicURL_BlocksPrivate(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/",
		"http://localhost/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://169.254.169.254/",
		"http://[::1]/",
	}
	for _, raw := range blocked {
		if err := assertPublicURL(raw); err == nil {
			t.Errorf("assertPublicURL(%q) expected error", raw)
		}
	}
}

func TestAssertPublicURL_AllowsPublicIPLiteral(t *testing.T) {
	// 8.8.8.8 is a public IP literal — no DNS needed.
	if err := assertPublicURL("https://8.8.8.8/"); err != nil {
		t.Fatalf("assertPublicURL public IP: %v", err)
	}
}

func TestDoRequest_RetriesOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	opts := webClientOpts{Timeout: 5 * time.Second, SkipSSRF: true}
	resp, err := doRequest(context.Background(), req, opts)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestPostJSON_SendsJSON(t *testing.T) {
	var gotCT string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	opts := webClientOpts{Timeout: 5 * time.Second, SkipSSRF: true}
	resp, err := postJSON(context.Background(), srv.URL, map[string]any{"q": "test"}, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.Contains(gotCT, "application/json") {
		t.Fatalf("Content-Type = %q", gotCT)
	}
	if !strings.Contains(gotBody, `"q":"test"`) {
		t.Fatalf("body = %q", gotBody)
	}
}
