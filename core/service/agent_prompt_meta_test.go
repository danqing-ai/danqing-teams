package service

import "testing"

func TestEncodeDecodeAgentSystemPrompt(t *testing.T) {
	body := "You are a facilitator."
	meta := map[string]string{
		"market.source":  "official-github",
		"market.version": "0.1.0",
	}
	stored := EncodeAgentSystemPrompt(body, meta)
	gotBody, gotMeta := DecodeAgentSystemPrompt(stored)
	if gotBody != body {
		t.Fatalf("body: got %q want %q", gotBody, body)
	}
	if gotMeta["market.source"] != "official-github" {
		t.Fatalf("meta: %+v", gotMeta)
	}
	plain, empty := DecodeAgentSystemPrompt(body)
	if plain != body || empty != nil {
		t.Fatalf("plain decode failed: %q %+v", plain, empty)
	}
}
