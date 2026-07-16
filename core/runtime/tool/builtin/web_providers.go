package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	duckDuckGoEndpoint          = "https://html.duckduckgo.com/html/"
	tavilyEndpoint              = "https://api.tavily.com/search"
	bochaEndpoint               = "https://api.bochaai.com/v1/web-search"
	metasoEndpoint              = "https://metaso.cn/api/v1"
	baiduEndpoint               = "https://qianfan.baidubce.com/v2/ai_search/web_search"
	volcengineResponsesEndpoint = "https://ark.cn-beijing.volces.com/api/v3/responses"
	sofyaEndpoint               = "https://sofya.co/v1/search"
	braveSearchEndpoint         = "https://api.search.brave.com/res/v1/web/search"
)

func searchDuckDuckGo(ctx context.Context, query string, max int, opts webClientOpts, baseURL string) ([]webSearchEntry, error) {
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

	resp, err := fetchWithOpts(ctx, u.String(), opts)
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
	html := string(body)
	results := parseDuckDuckGoResults(html, max)
	if len(results) == 0 && isDuckDuckGoChallenge(html) {
		return nil, fmt.Errorf("duckduckgo returned a bot challenge")
	}
	return results, nil
}

func parseDuckDuckGoResults(html string, max int) []webSearchEntry {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}
	var results []webSearchEntry
	doc.Find(".result").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= max {
			return
		}
		a := s.Find("a.result__a").First()
		href, _ := a.Attr("href")
		title := normalizeText(a.Text())
		if href == "" || title == "" {
			return
		}
		snippet := normalizeText(s.Find(".result__snippet").First().Text())
		results = append(results, webSearchEntry{
			Title:   title,
			URL:     normalizeDuckDuckGoURL(href),
			Snippet: snippet,
		})
	})
	if len(results) == 0 {
		doc.Find("a.result__a").Each(func(_ int, a *goquery.Selection) {
			if len(results) >= max {
				return
			}
			href, _ := a.Attr("href")
			title := normalizeText(a.Text())
			if href == "" || title == "" {
				return
			}
			results = append(results, webSearchEntry{
				Title: title,
				URL:   normalizeDuckDuckGoURL(href),
			})
		})
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

func searchBing(ctx context.Context, query string, max int, opts webClientOpts) ([]webSearchEntry, error) {
	u := "https://www.bing.com/search?q=" + url.QueryEscape(query) + "&setmkt=en-US&setlang=en"
	resp, err := fetchWithOpts(ctx, u, opts)
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
	return parseBingResults(string(body), max), nil
}

func parseBingResults(html string, max int) []webSearchEntry {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}
	var results []webSearchEntry
	doc.Find("li.b_algo").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= max {
			return
		}
		a := s.Find("h2 a").First()
		href, _ := a.Attr("href")
		title := normalizeText(a.Text())
		if href == "" || title == "" {
			return
		}
		snippet := normalizeText(s.Find(".b_caption p").First().Text())
		if snippet == "" {
			snippet = normalizeText(s.Find("p").First().Text())
		}
		results = append(results, webSearchEntry{
			Title:   title,
			URL:     normalizeBingURL(href),
			Snippet: snippet,
		})
	})
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
		if raw, err := base64.StdEncoding.DecodeString(padded); err == nil {
			if urlStr := string(raw); strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
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

func runTavilySearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string, locale searchLocale) ([]webSearchEntry, string, error) {
	if apiKey == "" {
		return nil, "", fmt.Errorf("Tavily search requires an API key")
	}
	payload := map[string]any{
		"api_key":        apiKey,
		"query":          query,
		"search_depth":   "advanced",
		"include_answer": true,
		"max_results":    max,
	}
	if locale.Region != "" {
		payload["country"] = locale.Region
	}
	resp, err := postJSON(ctx, tavilyEndpoint, payload, nil, opts)
	if err != nil {
		return nil, "", err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Tavily search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
	}
	var parsed struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("failed to parse Tavily response: %w", err)
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
	answer := strings.TrimSpace(parsed.Answer)
	return results, answer, nil
}

func runBraveSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string, locale searchLocale) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("BRAVE_SEARCH_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("BRAVE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Brave search requires an API key")
	}
	u, err := url.Parse(braveSearchEndpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", max))
	if locale.Region != "" {
		q.Set("country", locale.Region)
	}
	if locale.Language != "" {
		q.Set("search_lang", locale.Language)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("User-Agent", effectiveUserAgent(opts))

	resp, err := doRequest(ctx, req, opts)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Brave search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			msg = "Brave search API key rejected"
		case http.StatusTooManyRequests:
			msg = "Brave search rate-limited"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	var parsed struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse Brave response: %w", err)
	}
	var results []webSearchEntry
	for _, item := range parsed.Web.Results {
		if item.Title == "" || item.URL == "" {
			continue
		}
		results = append(results, webSearchEntry{Title: item.Title, URL: item.URL, Snippet: item.Description})
		if len(results) >= max {
			break
		}
	}
	return results, nil
}

func runBochaSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Bocha search requires an API key")
	}
	payload := map[string]any{
		"query":     query,
		"freshness": "noLimit",
		"summary":   true,
		"count":     max,
	}
	resp, err := postJSON(ctx, bochaEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, opts)
	if err != nil {
		return nil, err
	}
	body, err := readResponseBody(resp, 0)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Bocha search failed: HTTP %d — %s", resp.StatusCode, truncateErrorBody(string(body)))
		if resp.StatusCode == http.StatusTooManyRequests {
			msg = "Bocha search rate-limited"
		}
		return nil, fmt.Errorf("%s", msg)
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

func runMetasoSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("METASO_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Metaso search requires an API key")
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
	resp, err := postJSON(ctx, metasoEndpoint+"/search", payload, map[string]string{"Authorization": "Bearer " + apiKey}, opts)
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

func runSearxngSearch(ctx context.Context, query string, max int, opts webClientOpts, baseURL string, locale searchLocale) ([]webSearchEntry, error) {
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
	if locale.Language != "" {
		q.Set("language", locale.Language)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", effectiveUserAgent(opts))
	req.Header.Set("Accept", "application/json")
	resp, err := doRequest(ctx, req, opts)
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

func runBaiduSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("BAIDU_SEARCH_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Baidu search requires an API key")
	}
	payload := map[string]any{
		"messages":      []map[string]any{{"role": "user", "content": query}},
		"search_source": "baidu_search_v2",
		"resource_type_filter": []map[string]any{
			{"type": "web", "top_k": max},
		},
	}
	resp, err := postJSON(ctx, baiduEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, opts)
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

func runVolcengineSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string) ([]webSearchEntry, error) {
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

	effectiveOpts := opts
	if effectiveOpts.Timeout < 90*time.Second {
		effectiveOpts.Timeout = 90 * time.Second
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
		resp, err := postJSON(ctx, volcengineResponsesEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, effectiveOpts)
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

func runSofyaSearch(ctx context.Context, query string, max int, opts webClientOpts, apiKey string) ([]webSearchEntry, error) {
	if apiKey == "" {
		apiKey = os.Getenv("SOFYA_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Sofya search requires an API key")
	}
	payload := map[string]any{
		"query":       query,
		"max_results": max,
	}
	resp, err := postJSON(ctx, sofyaEndpoint, payload, map[string]string{"Authorization": "Bearer " + apiKey}, opts)
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
