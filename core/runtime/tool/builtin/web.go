package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"danqing-teams/core/domain"
)

const webUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

const (
	duckDuckGoEndpoint       = "https://html.duckduckgo.com/html/"
	bingHost                 = "www.bing.com"
	tavilyEndpoint           = "https://api.tavily.com/search"
	bochaEndpoint            = "https://api.bochaai.com/v1/web-search"
	metasoEndpoint           = "https://metaso.cn/api/v1"
	baiduEndpoint            = "https://qianfan.baidubce.com/v2/ai_search/web_search"
	volcengineResponsesEndpoint = "https://ark.cn-beijing.volces.com/api/v3/responses"
	sofyaEndpoint            = "https://sofya.co/v1/search"
	metasoDefaultAPIKey      = "mk-E384C1DD5E8501BB7EFE27C949AFDE5B"
	defaultMaxResults        = 5
	maxResults               = 10
	defaultTimeoutMs         = 15000
	maxTimeoutMs             = 60000
	errorBodyPreviewBytes    = 512
	maxFetchBytes            = 10 * 1024 * 1024
)

type webSearchEntry struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type webSearchResult struct {
	Query   string           `json:"query"`
	Source  string           `json:"source"`
	Count   int              `json:"count"`
	Message string           `json:"message"`
	Results []webSearchEntry `json:"results"`
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https allowed")
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}

func stripHTMLTags(s string) string {
	var out strings.Builder
	inTag := false
	for i := 0; i < len(s); {
		if i+7 < len(s) && strings.ToLower(s[i:i+7]) == "<script" {
			if end := strings.Index(strings.ToLower(s[i:]), "</script>"); end >= 0 {
				i += end + 9
				continue
			}
		}
		c := s[i]
		if c == '<' {
			inTag = true
			i++
			continue
		}
		if c == '>' {
			inTag = false
			i++
			continue
		}
		if !inTag {
			out.WriteByte(c)
		}
		i++
	}
	return out.String()
}

func normalizeText(s string) string {
	s = htmlUnescape(s)
	s = stripHTMLTags(s)
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func htmlUnescape(s string) string {
	replacements := map[string]string{
		"\u0026amp;":  "\u0026",
		"\u0026lt;":   "\u003c",
		"\u0026gt;":   "\u003e",
		"\u0026quot;": `"`,
		"\u0026#39;": "'",
		"\u0026nbsp;": " ",
	}
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}

func decodeHTMLEntities(s string) string {
	re := regexp.MustCompile(`&(?:#(\d+)|#x([0-9A-Fa-f]+)|([a-zA-Z]+));`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		groups := re.FindStringSubmatch(match)
		if groups[1] != "" {
			var n int
			_, _ = fmt.Sscanf(groups[1], "%d", &n)
			if n > 0 {
				return string(rune(n))
			}
		}
		if groups[2] != "" {
			var n int
			_, _ = fmt.Sscanf(groups[2], "%x", &n)
			if n > 0 {
				return string(rune(n))
			}
		}
		switch groups[3] {
		case "amp":
			return "&"
		case "lt":
			return "<"
		case "gt":
			return ">"
		case "quot":
			return `"`
		case "apos":
			return "'"
		case "nbsp":
			return " "
		case "copy":
			return "\u00A9"
		case "reg":
			return "\u00AE"
		case "mdash":
			return "\u2014"
		case "ndash":
			return "\u2013"
		case "lsquo":
			return "\u2018"
		case "rsquo":
			return "\u2019"
		case "ldquo":
			return "\u201C"
		case "rdquo":
			return "\u201D"
		case "hellip":
			return "\u2026"
		}
		return match
	})
}

func httpClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func fetchWithUserAgent(ctx context.Context, urlStr string, timeout time.Duration) (*http.Response, error) {
	if err := validateURL(urlStr); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	return httpClient(timeout).Do(req)
}

func postJSON(ctx context.Context, urlStr string, payload any, headers map[string]string, timeout time.Duration) (*http.Response, error) {
	if err := validateURL(urlStr); err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient(timeout).Do(req)
}

func readResponseBody(resp *http.Response, limit int64) ([]byte, error) {
	defer resp.Body.Close()
	if limit > 0 {
		return io.ReadAll(io.LimitReader(resp.Body, limit))
	}
	return io.ReadAll(resp.Body)
}

func truncateErrorBody(body string) string {
	stripped := sanitizeErrorBody(body)
	if len(stripped) <= errorBodyPreviewBytes {
		return stripped
	}
	return stripped[:errorBodyPreviewBytes] + "..."
}

func sanitizeErrorBody(body string) string {
	visible := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, stripHTMLTags(body))
	re := regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	return re.ReplaceAllString(visible, "Bearer [REDACTED]")
}

// WebSearch searches the web using configurable providers.
type WebSearch struct {
	ConfigFunc func(context.Context) (domain.SearchConfig, error)
}

func (h *WebSearch) Name() string                { return "web_search" }
func (h *WebSearch) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *WebSearch) Describe(args map[string]any) string {
	query := extractSearchQuery(args)
	if len(query) > 80 {
		query = query[:80] + "..."
	}
	return query
}
func (h *WebSearch) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "web_search",
		Description: "Search the web and return ranked results with URLs and snippets.\n\n" +
			"- query: search query string (required).\n" +
			"- max_results: result count (default 5, max 10).\n" +
			"- timeout_ms: timeout in milliseconds (default 15000, max 60000).\n" +
			"- Use for accessing current events, latest docs, or best practices beyond your knowledge cutoff.\n" +
			"- Results include title, URL, and snippet for each match.\n" +
			"- Provider is configurable (DuckDuckGo default; also supports Bing, Tavily, SearXNG, Baidu, Volcengine, Sofya, Metaso, Bocha).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":       map[string]any{"type": "string", "description": "Search query"},
				"q":           map[string]any{"type": "string", "description": "Search query alias"},
				"max_results": map[string]any{"type": "integer", "description": "Maximum number of results (default 5, max 10)"},
				"timeout_ms":  map[string]any{"type": "integer", "description": "Timeout in milliseconds (default 15000, max 60000)"},
			},
			"required": []string{"query"},
		},
	}
}

func (h *WebSearch) Execute(ctx context.Context, input map[string]any) (domain.ToolResult, error) {
	query := extractSearchQuery(input)
	if query == "" {
		return domain.ToolResult{}, fmt.Errorf("query is required")
	}

	cfg := h.currentConfig(ctx)

	max := cfg.MaxResults
	if max <= 0 {
		max = defaultMaxResults
	}
	if m, ok := input["max_results"].(float64); ok {
		max = int(m)
	}
	if max < 1 {
		max = 1
	}
	if max > maxResults {
		max = maxResults
	}
	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}
	if t, ok := input["timeout_ms"].(float64); ok {
		timeoutMs = int(t)
	}
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > maxTimeoutMs {
		timeoutMs = maxTimeoutMs
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond

	if cfg.BaseURL != "" && cfg.Provider != domain.SearchProviderDuckDuckGo && cfg.Provider != domain.SearchProviderSearxng {
		return domain.ToolResult{}, fmt.Errorf("base_url is only supported with provider \"duckduckgo\" or \"searxng\"")
	}

	var results []webSearchEntry
	var source string
	var err error

	switch cfg.Provider {
	case domain.SearchProviderTavily:
		results, err = runTavilySearch(ctx, query, max, timeout, cfg.APIKey)
		source = "tavily"
	case domain.SearchProviderBocha:
		results, err = runBochaSearch(ctx, query, max, timeout, cfg.APIKey)
		source = "bocha"
	case domain.SearchProviderMetaso:
		results, err = runMetasoSearch(ctx, query, max, timeout, cfg.APIKey)
		source = "metaso"
	case domain.SearchProviderSearxng:
		results, err = runSearxngSearch(ctx, query, max, timeout, cfg.BaseURL)
		source = "searxng"
	case domain.SearchProviderBaidu:
		results, err = runBaiduSearch(ctx, query, max, timeout, cfg.APIKey)
		source = "baidu"
	case domain.SearchProviderVolcengine:
		results, err = runVolcengineSearch(ctx, query, max, timeout, cfg.APIKey)
		source = "volcengine"
	case domain.SearchProviderSofya:
		results, err = runSofyaSearch(ctx, query, max, timeout, cfg.APIKey)
		source = "sofya"
	case domain.SearchProviderBing:
		results, err = runBingSearch(ctx, query, max, timeout)
		source = "bing"
		if err == nil && len(results) == 0 {
			fallback, err2 := searchDuckDuckGo(ctx, query, max, timeout, "")
			if err2 == nil && len(fallback) > 0 {
				results = fallback
				source = "duckduckgo"
			}
		}
	default:
		results, source, err = runDuckDuckGoWithBingFallback(ctx, query, max, timeout, cfg.BaseURL)
	}

	if err != nil {
		return domain.ToolResult{}, err
	}

	msg := fmt.Sprintf("Found %d result(s)", len(results))
	if len(results) == 0 {
		msg = "No results found"
	}
	res := webSearchResult{
		Query:   query,
		Source:  source,
		Count:   len(results),
		Message: msg,
		Results: results,
	}
	b, _ := json.Marshal(res)
	return domain.ToolResult{Content: string(b)}, nil
}

func (h *WebSearch) currentConfig(ctx context.Context) domain.SearchConfig {
	if h.ConfigFunc == nil {
		return domain.SearchConfig{Provider: domain.SearchProviderDuckDuckGo}
	}
	cfg, err := h.ConfigFunc(ctx)
	if err != nil || cfg.Provider == "" {
		return domain.SearchConfig{Provider: domain.SearchProviderDuckDuckGo}
	}
	return cfg
}

func extractSearchQuery(input map[string]any) string {
	for _, key := range []string{"query", "q"} {
		if v, ok := input[key].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	if arr, ok := input["search_query"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				for _, key := range []string{"q", "query"} {
					if v, ok := m[key].(string); ok {
						v = strings.TrimSpace(v)
						if v != "" {
							return v
						}
					}
				}
			}
		}
	}
	return ""
}

func runDuckDuckGoWithBingFallback(ctx context.Context, query string, max int, timeout time.Duration, baseURL string) ([]webSearchEntry, string, error) {
	results, err := searchDuckDuckGo(ctx, query, max, timeout, baseURL)
	if err != nil || len(results) == 0 {
		fallback, err2 := searchBing(ctx, query, max, timeout)
		if err2 == nil && len(fallback) > 0 {
			return fallback, "bing", nil
		}
	}
	if err != nil {
		return nil, "", err
	}
	if baseURL == "" {
		return results, "duckduckgo", nil
	}
	u, _ := url.Parse(strings.TrimSpace(baseURL))
	if u != nil && u.Host != "" {
		return results, u.Host, nil
	}
	return results, "duckduckgo", nil
}

func searchDuckDuckGo(ctx context.Context, query string, max int, timeout time.Duration, baseURL string) ([]webSearchEntry, error) {
	endpoint := duckDuckGoEndpoint
	if strings.TrimSpace(baseURL) != "" {
		endpoint = strings.TrimSpace(baseURL)
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid duckduckgo base_url: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	resp, err := fetchWithUserAgent(ctx, u.String(), timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo search failed: HTTP %d", resp.StatusCode)
	}
	results := parseDuckDuckGoResults(string(body), max)
	if len(results) == 0 && isDuckDuckGoChallenge(string(body)) {
		return nil, fmt.Errorf("duckduckgo returned a bot challenge")
	}
	return results, nil
}

func parseDuckDuckGoResults(html string, max int) []webSearchEntry {
	var results []webSearchEntry
	linkRe := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	snipRe := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	links := linkRe.FindAllStringSubmatch(html, -1)
	snips := snipRe.FindAllStringSubmatch(html, -1)
	for i, m := range links {
		if len(results) >= max {
			break
		}
		url := normalizeDuckDuckGoURL(normalizeText(m[1]))
		title := normalizeText(m[2])
		snippet := ""
		if i < len(snips) {
			snippet = normalizeText(snips[i][1])
		}
		results = append(results, webSearchEntry{Title: title, URL: url, Snippet: snippet})
	}
	return results
}

func isDuckDuckGoChallenge(html string) bool {
	return strings.Contains(html, "anomaly-modal") || strings.Contains(html, "Unfortunately, bots use DuckDuckGo too")
}

func normalizeDuckDuckGoURL(href string) string {
	if uddg := extractQueryParam(href, "uddg"); uddg != "" {
		return uddg
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return "https://duckduckgo.com" + href
	}
	return href
}

func searchBing(ctx context.Context, query string, max int, timeout time.Duration) ([]webSearchEntry, error) {
	u := "https://www.bing.com/search?q=" + url.QueryEscape(query) + "&setmkt=en-US&setlang=en"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	resp, err := httpClient(timeout).Do(req)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing search failed: HTTP %d", resp.StatusCode)
	}
	results := parseBingResults(string(body), max)
	return results, nil
}

func runBingSearch(ctx context.Context, query string, max int, timeout time.Duration) ([]webSearchEntry, error) {
	return searchBing(ctx, query, max, timeout)
}

func parseBingResults(html string, max int) []webSearchEntry {
	var results []webSearchEntry
	resultRe := regexp.MustCompile(`(?is)<li[^>]*class="[^"]*\bb_algo\b[^"]*"[^>]*>(.*?)</li>`)
	titleRe := regexp.MustCompile(`(?is)<h2[^>]*>.*?<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	snipRe := regexp.MustCompile(`(?is)<div[^>]*class="[^"]*\bb_caption\b[^"]*"[^>]*>.*?<p[^>]*>(.*?)</p>`)
	for _, m := range resultRe.FindAllStringSubmatch(html, -1) {
		if len(results) >= max {
			break
		}
		block := m[1]
		tm := titleRe.FindStringSubmatch(block)
		if tm == nil {
			continue
		}
		sm := snipRe.FindStringSubmatch(block)
		snippet := ""
		if sm != nil {
			snippet = normalizeText(sm[1])
		}
		results = append(results, webSearchEntry{
			Title:   normalizeText(tm[2]),
			URL:     normalizeBingURL(normalizeText(tm[1])),
			Snippet: snippet,
		})
	}
	return results
}

func normalizeBingURL(href string) string {
	href = decodeHTMLEntities(href)
	if encoded := extractQueryParam(href, "u"); encoded != "" {
		decoded := percentDecode(encoded)
		if strings.HasPrefix(decoded, "a1") {
			decoded = decoded[2:]
		}
		padded := strings.ReplaceAll(strings.ReplaceAll(decoded, "-", "+"), "_", "/")
		for len(padded)%4 != 0 {
			padded += "="
		}
		if bytes, err := base64.StdEncoding.DecodeString(padded); err == nil {
			if urlStr := string(bytes); strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
				return urlStr
			}
		}
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return "https://www.bing.com" + href
	}
	return href
}

func extractQueryParam(raw, key string) string {
	parts := strings.SplitN(raw, "?", 2)
	if len(parts) != 2 {
		return ""
	}
	for _, part := range strings.Split(parts[1], "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			return percentDecode(kv[1])
		}
	}
	return ""
}

func percentDecode(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) {
			var b byte
			if _, err := fmt.Sscanf(s[i+1:i+3], "%02x", &b); err == nil {
				out.WriteByte(b)
				i += 3
				continue
			}
		}
		if s[i] == '+' {
			out.WriteByte(' ')
		} else {
			out.WriteByte(s[i])
		}
		i++
	}
	return out.String()
}

func runTavilySearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Tavily search requires an API key")
	}
	payload := map[string]any{
		"api_key": apiKey,
		"query":   query,
		"search_depth": "basic",
		"max_results":  max,
	}
	resp, err := postJSON(ctx, tavilyEndpoint, payload, nil, timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tavily search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
	}
	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Tavily response: %w", err)
	}
	var results []webSearchEntry
	for _, item := range parsed.Results {
		if item.Title == "" || item.URL == "" {
			continue
		}
		snippet := item.Content
		if snippet == "" {
			snippet = item.Snippet
		}
		results = append(results, webSearchEntry{Title: item.Title, URL: item.URL, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results, nil
}

func runBochaSearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Bocha search requires an API key")
	}
	payload := map[string]any{
		"query":   query,
		"freshness": "noLimit",
		"count":   max,
	}
	resp, err := postJSON(ctx, bochaEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bocha search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Bocha response: %w", err)
	}
	if errMsg := bochaErrorMessage(parsed); errMsg != "" {
		return nil, fmt.Errorf("%s", errMsg)
	}
	return parseBochaResults(parsed, max), nil
}

func bochaErrorMessage(parsed map[string]any) string {
	codeVal, ok := parsed["code"]
	if !ok {
		return ""
	}
	code, ok := codeVal.(float64)
	if !ok {
		return ""
	}
	if code == 0 || code == 200 {
		return ""
	}
	msg := "unknown error"
	if m, ok := parsed["msg"].(string); ok && m != "" {
		msg = m
	} else if m, ok := parsed["message"].(string); ok && m != "" {
		msg = m
	}
	return fmt.Sprintf("Bocha search API error (code %d: %s)", int(code), msg)
}

func parseBochaResults(parsed map[string]any, max int) []webSearchEntry {
	var results []webSearchEntry
	var arr []any
	if data, ok := parsed["data"].(map[string]any); ok {
		if wp, ok := data["webPages"].(map[string]any); ok {
			arr, _ = wp["value"].([]any)
		}
		if arr == nil {
			arr, _ = data["pages"].([]any)
		}
	}
	if arr == nil {
		arr, _ = parsed["pages"].([]any)
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, []string{"name", "title"})
		urlStr := firstString(m, []string{"url", "link"})
		if title == "" || urlStr == "" {
			continue
		}
		snippet := firstString(m, []string{"summary", "snippet", "description"})
		results = append(results, webSearchEntry{Title: title, URL: urlStr, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func runMetasoSearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("METASO_API_KEY")
	}
	if apiKey == "" {
		apiKey = metasoDefaultAPIKey
	}
	size := max
	if size < 1 {
		size = 1
	}
	if size > 100 {
		size = 100
	}
	payload := map[string]any{
		"q":     query,
		"scope": "webpage",
		"size":  size,
	}
	resp, err := postJSON(ctx, metasoEndpoint+"/search", payload, map[string]string{"Authorization": "Bearer " + apiKey}, timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Metaso search failed: HTTP %d", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			msg = "Metaso API key rejected"
		case http.StatusTooManyRequests:
			msg = "Metaso rate-limited"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Metaso response: %w", err)
	}
	if codeVal, ok := parsed["code"].(float64); ok && codeVal != 0 {
		msg := "unknown error"
		if m, ok := parsed["message"].(string); ok && m != "" {
			msg = m
		}
		switch int(codeVal) {
		case 3003:
			msg = "Metaso: daily search limit reached"
		case 2005:
			msg = "Metaso API key rejected"
		}
		return nil, fmt.Errorf("Metaso API error (code %d: %s)", int(codeVal), msg)
	}
	return parseMetasoResults(parsed, size), nil
}

func parseMetasoResults(parsed map[string]any, max int) []webSearchEntry {
	var results []webSearchEntry
	arr, _ := parsed["webpages"].([]any)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, []string{"title"})
		urlStr := firstString(m, []string{"link", "url"})
		if title == "" || urlStr == "" {
			continue
		}
		snippet := firstString(m, []string{"snippet", "summary"})
		results = append(results, webSearchEntry{Title: title, URL: urlStr, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func runSearxngSearch(ctx context.Context, query string, max int, timeout time.Duration, baseURL string) ([]webSearchEntry, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("SearXNG search requires a configured base_url")
	}
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid searxng base_url: %w", err)
	}
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" {
		u.Path = "/search"
	} else if path != "/search" && !strings.HasSuffix(path, "/search") {
		u.Path = path + "/search"
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient(timeout).Do(req)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SearXNG search failed: HTTP %d", resp.StatusCode)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse SearXNG response: %w", err)
	}
	return parseSearxngResults(parsed, max), nil
}

func parseSearxngResults(parsed map[string]any, max int) []webSearchEntry {
	var results []webSearchEntry
	arr, _ := parsed["results"].([]any)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, []string{"title"})
		urlStr := firstString(m, []string{"url"})
		if title == "" || urlStr == "" {
			continue
		}
		snippet := firstString(m, []string{"content", "snippet"})
		results = append(results, webSearchEntry{Title: title, URL: urlStr, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func runBaiduSearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("BAIDU_SEARCH_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Baidu search requires an API key")
	}
	payload := map[string]any{
		"messages": []map[string]any{{"role": "user", "content": query}},
		"search_source": "baidu_search_v2",
		"resource_type_filter": []map[string]any{
			{"type": "web", "top_k": max},
		},
	}
	resp, err := postJSON(ctx, baiduEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Baidu search failed: HTTP %d", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			msg = "Baidu search API key rejected"
		case http.StatusTooManyRequests:
			msg = "Baidu search rate-limited"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Baidu response: %w", err)
	}
	if errMsg := baiduErrorMessage(parsed); errMsg != "" {
		return nil, fmt.Errorf("%s", errMsg)
	}
	return parseBaiduResults(parsed, max), nil
}

func baiduErrorMessage(parsed map[string]any) string {
	var code float64
	if c, ok := parsed["error_code"].(float64); ok {
		code = c
	} else if c, ok := parsed["code"].(float64); ok {
		code = c
	}
	if code == 0 {
		return ""
	}
	msg := "unknown error"
	if m, ok := parsed["error_msg"].(string); ok && m != "" {
		msg = m
	} else if m, ok := parsed["message"].(string); ok && m != "" {
		msg = m
	}
	return fmt.Sprintf("Baidu search API error (code %d: %s)", int(code), msg)
}

func parseBaiduResults(parsed map[string]any, max int) []webSearchEntry {
	var results []webSearchEntry
	arr, _ := parsed["references"].([]any)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, []string{"title", "name"})
		urlStr := firstString(m, []string{"url", "link"})
		if title == "" || urlStr == "" {
			continue
		}
		snippet := firstString(m, []string{"content", "snippet", "summary"})
		results = append(results, webSearchEntry{Title: title, URL: urlStr, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func runVolcengineSearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("VOLCENGINE_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("VOLCENGINE_ARK_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ARK_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Volcengine search requires an API key")
	}

	effectiveTimeout := timeout
	if effectiveTimeout < 90*time.Second {
		effectiveTimeout = 90 * time.Second
	}
	payload := map[string]any{
		"model":  "doubao-seed-2-0-lite-260428",
		"stream": false,
		"tools":  []map[string]any{{"type": "web_search"}},
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": fmt.Sprintf(
							"Search the web for: %s\n\n"+
								"CRITICAL: Respond ONLY with a valid JSON object. No markdown, no explanation.\n"+
								"Schema: {\"results\":[{\"title\":\"...\",\"url\":\"https://...\",\"snippet\":\"...\"}]}\n"+
								"- results: 1-%d most relevant pages\n"+
								"- title: page title (required)\n"+
								"- url: full URL starting with https:// (required)\n"+
								"- snippet: 1-2 sentence factual summary (required)\n"+
								"- If zero results: {\"results\":[]}\n"+
								"- Your entire response must be valid, parseable JSON.",
							query, max,
						),
					},
				},
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<(attempt-1)) * time.Second):
			}
		}
		resp, err := postJSON(ctx, volcengineResponsesEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, effectiveTimeout)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := readResponseBody(resp, 0)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Volcengine search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
		}
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse Volcengine response: %w", err)
		}
		if errMsg := volcengineErrorMessage(parsed); errMsg != "" {
			return nil, fmt.Errorf("%s", errMsg)
		}
		text := volcengineExtractText(parsed)
		if text == "" {
			return nil, fmt.Errorf("Volcengine response contains no output text")
		}
		return parseVolcengineResults(text, max), nil
	}
	return nil, fmt.Errorf("Volcengine search request failed: %w", lastErr)
}

func volcengineErrorMessage(parsed map[string]any) string {
	errVal, ok := parsed["error"].(map[string]any)
	if !ok {
		return ""
	}
	code := "unknown"
	if c, ok := errVal["code"].(string); ok {
		code = c
	}
	msg := "no details"
	if m, ok := errVal["message"].(string); ok {
		msg = m
	}
	return fmt.Sprintf("Volcengine API error (code %s: %s)", code, msg)
}

func volcengineExtractText(parsed map[string]any) string {
	output, ok := parsed["output"].([]any)
	if !ok {
		return ""
	}
	for i := len(output) - 1; i >= 0; i-- {
		item, ok := output[i].(map[string]any)
		if !ok {
			continue
		}
		if t, ok := item["type"].(string); ok && t == "message" {
			content, ok := item["content"].([]any)
			if !ok {
				continue
			}
			for _, c := range content {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				if text, ok := cm["text"].(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func parseVolcengineResults(text string, max int) []webSearchEntry {
	jsonText := extractJSONBlock(text)
	if jsonText == "" {
		jsonText = text
	}
	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil
	}
	var results []webSearchEntry
	for _, item := range parsed.Results {
		if item.Title == "" || item.URL == "" {
			continue
		}
		results = append(results, webSearchEntry{Title: item.Title, URL: item.URL, Snippet: item.Snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func extractJSONBlock(text string) string {
	if start := strings.Index(text, "```json"); start >= 0 {
		inner := text[start+7:]
		if end := strings.Index(inner, "```"); end >= 0 {
			return strings.TrimSpace(inner[:end])
		}
	}
	if start := strings.Index(text, "{"); start >= 0 {
		if end := strings.LastIndex(text, "}"); end >= start {
			return text[start : end+1]
		}
	}
	return ""
}

func runSofyaSearch(ctx context.Context, query string, max int, timeout time.Duration, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("SOFYA_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Sofya search requires an API key")
	}
	payload := map[string]any{
		"query":      query,
		"max_results": max,
	}
	resp, err := postJSON(ctx, sofyaEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, timeout)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Sofya search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Sofya response: %w", err)
	}
	return parseSofyaResults(parsed, max), nil
}

func parseSofyaResults(parsed map[string]any, max int) []webSearchEntry {
	var results []webSearchEntry
	arr, _ := parsed["results"].([]any)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := firstString(m, []string{"title"})
		urlStr := firstString(m, []string{"url"})
		if title == "" || urlStr == "" {
			continue
		}
		snippet := firstString(m, []string{"content", "description"})
		results = append(results, webSearchEntry{Title: title, URL: urlStr, Snippet: snippet})
		if len(results) >= max {
			break
		}
	}
	return results
}

func firstString(m map[string]any, keys []string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	return ""
}

// WebFetch fetches a URL and returns readable text or JSON.
type WebFetch struct{}

func (h *WebFetch) Name() string                { return "web_fetch" }
func (h *WebFetch) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *WebFetch) Describe(args map[string]any) string {
	urlStr, _ := args["url"].(string)
	if len(urlStr) > 80 {
		urlStr = urlStr[:80] + "..."
	}
	return urlStr
}
func (h *WebFetch) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "web_fetch",
		Description: "Fetch the content of a web page or API endpoint and return readable text.\n\n" +
			"- url: must be a valid HTTP/HTTPS URL (required).\n" +
			"- max_chars: limits returned content length (default 8000, min 100).\n" +
			"- timeout_ms: timeout in milliseconds (default 15000, max 60000).\n" +
			"- HTML pages are converted to readable text; JSON endpoints are returned as-is.\n" +
			"- Use after web_search to get full content from interesting results.\n" +
			"- HTTP URLs are automatically upgraded to HTTPS.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":        map[string]any{"type": "string", "description": "URL to fetch"},
				"max_chars":  map[string]any{"type": "integer", "description": "Maximum characters to return (default 8000)"},
				"timeout_ms": map[string]any{"type": "integer", "description": "Timeout in milliseconds (default 15000)"},
			},
			"required": []string{"url"},
		},
	}
}
func (h *WebFetch) Execute(ctx context.Context, input map[string]any) (domain.ToolResult, error) {
	urlStr, _ := input["url"].(string)
	if urlStr == "" {
		return domain.ToolResult{}, fmt.Errorf("url is required")
	}
	maxChars := 8000
	if m, ok := input["max_chars"].(float64); ok {
		maxChars = int(m)
	}
	if maxChars < 100 {
		maxChars = 100
	}
	timeoutMs := 15000
	if t, ok := input["timeout_ms"].(float64); ok {
		timeoutMs = int(t)
	}
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > 60000 {
		timeoutMs = 60000
	}

	resp, err := fetchWithUserAgent(ctx, urlStr, time.Duration(timeoutMs)*time.Millisecond)
	if err != nil {
		return domain.ToolResult{}, err
	}
	body, err := readResponseBody(resp, maxFetchBytes)
	if err != nil {
		return domain.ToolResult{}, err
	}
	contentType := resp.Header.Get("Content-Type")
	if isRejectedContentType(contentType) {
		return domain.ToolResult{}, fmt.Errorf("content type %s is not supported", contentType)
	}
	content := string(body)

	extractor := "raw"
	if strings.Contains(contentType, "application/json") {
		extractor = "json"
	} else if strings.Contains(contentType, "text/html") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(content)), "<!doctype") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(content)), "<html") {
		content = extractReadableText(content)
		extractor = "html"
	}

	truncated := false
	if len(content) > maxChars {
		content = content[:maxChars]
		truncated = true
	}

	result := map[string]any{
		"url":       urlStr,
		"finalUrl":  resp.Request.URL.String(),
		"status":    resp.StatusCode,
		"extractor": extractor,
		"truncated": truncated,
		"length":    len(content),
		"content":   content,
	}
	b, _ := json.Marshal(result)
	return domain.ToolResult{Content: string(b)}, nil
}

func isRejectedContentType(contentType string) bool {
	lower := strings.ToLower(contentType)
	prefixes := []string{
		"image/", "audio/", "video/", "application/octet-stream",
		"application/zip", "application/gzip", "application/x-tar",
		"application/pdf", "application/msword", "application/vnd.ms",
		"application/vnd.openxml",
	}
	for _, p := range prefixes {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func extractReadableText(html string) string {
	replacements := []string{
		"\u003c/p\u003e", "\n\n",
		"\u003c/div\u003e", "\n\n",
		"\u003c/section\u003e", "\n\n",
		"\u003c/article\u003e", "\n\n",
		"\u003cbr\u003e", "\n",
		"\u003cbr/\u003e", "\n",
		"\u003cbr /\u003e", "\n",
	}
	for i := 0; i < len(replacements); i += 2 {
		html = strings.ReplaceAll(html, replacements[i], replacements[i+1])
	}
	text := normalizeText(html)
	lines := strings.Split(text, "\n")
	var nonEmpty []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}
