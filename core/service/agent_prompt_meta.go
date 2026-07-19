package service

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// agentStoredMeta is the slim YAML frontmatter embedded in SystemPrompt
// to persist market provenance without adding DB columns.
type agentStoredMeta struct {
	Metadata map[string]string `yaml:"metadata"`
}

// DecodeAgentSystemPrompt strips provenance frontmatter from a stored system
// prompt and returns the body plus metadata (may be nil).
func DecodeAgentSystemPrompt(stored string) (body string, meta map[string]string) {
	stored = strings.TrimSpace(stored)
	if !strings.HasPrefix(stored, "---") {
		return stored, nil
	}
	parts := strings.SplitN(stored, "---", 3)
	if len(parts) < 3 {
		return stored, nil
	}
	var fm agentStoredMeta
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(parts[1])), &fm); err != nil {
		return stored, nil
	}
	if fm.Metadata == nil || fm.Metadata["market.source"] == "" {
		// Not our provenance wrapper — leave prompt untouched.
		return stored, nil
	}
	return strings.TrimSpace(parts[2]), fm.Metadata
}

// EncodeAgentSystemPrompt embeds market provenance as YAML frontmatter.
func EncodeAgentSystemPrompt(body string, meta map[string]string) string {
	body = strings.TrimSpace(body)
	if meta == nil || meta["market.source"] == "" {
		return body
	}
	raw, err := yaml.Marshal(agentStoredMeta{Metadata: meta})
	if err != nil {
		return body
	}
	return "---\n" + strings.TrimSpace(string(raw)) + "\n---\n\n" + body
}

func marketSourceFromMeta(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	return meta["market.source"]
}
