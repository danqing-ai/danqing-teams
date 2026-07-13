package runtime

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"danqing-teams/core/domain"
)

func buildSystemPrompt(agentPersona string, skillList []domain.Skill, checkpoint string) string {
	var b strings.Builder
	b.WriteString(agentPersona)

	meta := buildSkillMetadata(skillList)
	if meta != "" {
		b.WriteString("\n\n")
		b.WriteString(meta)
	}
	if checkpoint != "" {
		b.WriteString("\n\n")
		b.WriteString("<compaction-checkpoint>\n")
		b.WriteString(checkpoint)
		b.WriteString("\n</compaction-checkpoint>")
	}
	b.WriteString("\n\n")
	b.WriteString(buildRuntimeEnvironment())

	return b.String()
}

// buildRuntimeEnvironment returns a static block describing the runtime OS environment.
// This is injected into the system prompt (never changes during a session).
func buildRuntimeEnvironment() string {
	osName := runtime.GOOS
	osLabel := osName
	switch osName {
	case "darwin":
		osLabel = "macOS"
	case "linux":
		osLabel = "Linux"
	case "windows":
		osLabel = "Windows"
	}
	sep := string(filepath.Separator)
	shell := "sh"
	if osName == "windows" {
		shell = "cmd"
	}
	return "<runtime-environment>\n" +
		"OS: " + osName + " (" + osLabel + ")\n" +
		"Path separator: " + sep + "\n" +
		"Shell: " + shell + "\n" +
		"</runtime-environment>"
}

func buildSkillMetadata(skills []domain.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available_skills>\n")
	b.WriteString("  <!-- Use read_skill tool with path to load instructions: read_skill(path=\"skill-name\") -->\n")
	b.WriteString("  <!-- Resource files: read_skill(path=\"skill-name/references/file.md\") -->\n")
	for _, sk := range skills {
		fmt.Fprintf(&b, "  <skill>\n")
		fmt.Fprintf(&b, "    <path>%s</path>\n", escapeXML(sk.Name))
		fmt.Fprintf(&b, "    <description>%s</description>\n", escapeXML(sk.Description))
		if sk.SystemHint != "" {
			fmt.Fprintf(&b, "    <hint>%s</hint>\n", escapeXML(sk.SystemHint))
		}
		fmt.Fprintf(&b, "  </skill>\n")
	}
	b.WriteString("</available_skills>")
	return b.String()
}

func skillToolSchemas(skills []domain.Skill, toolBindings []domain.ToolBinding) []domain.ToolSchema {
	bindings := make(map[string]domain.ToolBinding, len(toolBindings))
	for _, tb := range toolBindings {
		bindings[tb.ToolID] = tb
	}
	toolSet := map[string]struct{}{}
	for _, sk := range skills {
		for _, tid := range sk.ToolIDs {
			toolSet[tid] = struct{}{}
		}
	}
	var schemas []domain.ToolSchema
	for tid := range toolSet {
		if tb, ok := bindings[tid]; ok {
			schemas = append(schemas, domain.ToolSchema{
				Name:        tb.ToolID,
				Description: tb.ToolID,
				Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
				RiskLevel:   tb.RiskLevel,
			})
		}
	}
	return schemas
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
