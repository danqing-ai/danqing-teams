package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"danqing-teams/core/port"
)

type HTTPProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewHTTPProvider(baseURL, apiKey string) *HTTPProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &HTTPProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *HTTPProvider) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}

	messages := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := map[string]any{
			"role": m.Role,
		}
		// Assistant messages with tool_calls must omit content (or set null)
		// per OpenAI API spec. Empty string causes some providers (e.g. DeepSeek)
		// to not recognize the message as a tool_calls carrier, making subsequent
		// tool messages appear unpaired → 400 error.
		if m.Content != "" || len(m.ToolCalls) == 0 {
			msg["content"] = m.Content
		}
		if len(m.ToolCalls) > 0 {
			var tcs []map[string]any
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": marshalArgs(tc.Arguments),
					},
				})
			}
			msg["tool_calls"] = tcs
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
			msg["name"] = m.Name
		}
		messages = append(messages, msg)
	}

	body := map[string]any{
		"model":        model,
		"messages":     messages,
		"temperature":  0.2,
		"max_tokens":   16384,
	}
	if len(req.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			})
		}
		body["tools"] = tools
		if req.ToolChoice != "" {
			body["tool_choice"] = req.ToolChoice
		} else {
			body["tool_choice"] = "auto"
		}
	}

	b, err := json.Marshal(body)
	if err != nil {
		return port.LLMChatResponse{}, err
	}

	hReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	hReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		hReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(hReq)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return port.LLMChatResponse{}, fmt.Errorf("llm http %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return port.LLMChatResponse{}, err
	}
	if len(result.Choices) == 0 {
		return port.LLMChatResponse{}, fmt.Errorf("no choices in llm response")
	}

	usage := &port.LLMUsage{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		usage = nil
	}

	choice := result.Choices[0].Message
	if len(choice.ToolCalls) > 0 {
		var tcs []port.ChatToolCall
		for _, tc := range choice.ToolCalls {
			args, err := parseArgs(tc.Function.Arguments)
			if err != nil {
				return port.LLMChatResponse{}, fmt.Errorf("tool '%s' arguments: %w (raw: %s)", tc.Function.Name, err, string(tc.Function.Arguments))
			}
			tcs = append(tcs, port.ChatToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
		return port.LLMChatResponse{ToolCalls: tcs, Usage: usage}, nil
	}

	return port.LLMChatResponse{Content: choice.Content, Usage: usage, Done: true}, nil

}

func marshalArgs(args map[string]any) string {
	b, _ := json.Marshal(args)
	return string(b)
}

// parseArgs parses tool call arguments from an OpenAI-compatible API response.
// arguments may be a JSON string or a JSON object; both are handled.
func parseArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("arguments is null")
	}
	// OpenAI-compatible APIs return arguments as a JSON string, not an object.
	var str string
	if err := json.Unmarshal(raw, &str); err == nil && str != "" {
		raw = json.RawMessage(str)
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args == nil {
		return nil, fmt.Errorf("arguments parsed to nil")
	}
	return args, nil
}
