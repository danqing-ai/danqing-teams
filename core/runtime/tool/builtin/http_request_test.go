package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRequest_GETJSON(t *testing.T) {
	testSkipSSRF = true
	defer func() { testSkipSSRF = false }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	h := &HTTPRequest{}
	res, err := h.Execute(context.Background(), map[string]any{
		"url": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(res.Content), &parsed); err != nil {
		t.Fatal(err)
	}
	if int(parsed["status"].(float64)) != 200 {
		t.Fatalf("status = %v", parsed["status"])
	}
	if !strings.Contains(parsed["body"].(string), `"ok"`) {
		t.Fatalf("body = %v", parsed["body"])
	}
}

func TestHTTPRequest_POSTWithHeaders(t *testing.T) {
	testSkipSSRF = true
	defer func() { testSkipSSRF = false }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("X-Test") != "1" {
			t.Fatalf("missing header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %s", r.Header.Get("Content-Type"))
		}
		buf := make([]byte, 64)
		n, _ := r.Body.Read(buf)
		if !strings.Contains(string(buf[:n]), "hello") {
			t.Fatalf("body = %q", buf[:n])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer srv.Close()

	h := &HTTPRequest{}
	res, err := h.Execute(context.Background(), map[string]any{
		"method": "POST",
		"url":    srv.URL,
		"headers": map[string]any{
			"X-Test": "1",
		},
		"body": `{"hello":"world"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Content, "received") {
		t.Fatalf("content = %s", res.Content)
	}
}

func TestHTTPRequest_BlocksLocalhost(t *testing.T) {
	h := &HTTPRequest{}
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

func TestHTTPRequest_RejectsBinaryBodyArg(t *testing.T) {
	h := &HTTPRequest{}
	_, err := h.Execute(context.Background(), map[string]any{
		"method": "POST",
		"url":    "https://example.com/",
		"body":   "a\x00b",
	})
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPRequest_TruncatesBinaryResponse(t *testing.T) {
	testSkipSSRF = true
	defer func() { testSkipSSRF = false }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x00, 0x01, 0x02, 0xff})
	}))
	defer srv.Close()

	h := &HTTPRequest{}
	res, err := h.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(res.Content), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["body"] != "" {
		t.Fatalf("expected empty body, got %v", parsed["body"])
	}
	if parsed["truncated"] != true {
		t.Fatalf("expected truncated=true")
	}
	msg, _ := parsed["message"].(string)
	if !strings.Contains(msg, "non-text") {
		t.Fatalf("message = %q", msg)
	}
}

func TestHTTPRequest_Describe(t *testing.T) {
	h := &HTTPRequest{}
	got := h.Describe(map[string]any{"method": "post", "url": "https://api.example.com/v1/items"})
	if got != "POST https://api.example.com/v1/items" {
		t.Fatalf("describe = %q", got)
	}
}

func TestIsNonTextHTTPBody(t *testing.T) {
	if isNonTextHTTPBody("application/json", []byte(`{}`)) {
		t.Fatal("json should be text")
	}
	if !isNonTextHTTPBody("image/png", []byte{0x89, 0x50}) {
		t.Fatal("png should be non-text")
	}
	if !isNonTextHTTPBody("", []byte{0x00, 0x01}) {
		t.Fatal("null bytes should be non-text")
	}
}
