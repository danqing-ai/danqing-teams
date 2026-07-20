package runtime

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"danqing-teams/core/domain"
)

func buildSystemPrompt(agentPersona string, skillList []domain.Skill, agentList []domain.Agent, checkpoint string, activeTodos string, sandboxStatus domain.SandboxStatus) string {
	var b strings.Builder
	b.WriteString(agentPersona)

	meta := buildSkillMetadata(skillList)
	if meta != "" {
		b.WriteString("\n\n")
		b.WriteString(meta)
	}
	agentMeta := buildAgentMetadata(agentList)
	if agentMeta != "" {
		b.WriteString("\n\n")
		b.WriteString(agentMeta)
	}
	if checkpoint != "" {
		b.WriteString("\n\n")
		b.WriteString("<compaction-checkpoint>\n")
		b.WriteString(checkpoint)
		b.WriteString("\n</compaction-checkpoint>")
	}
	if activeTodos != "" {
		b.WriteString("\n\n")
		b.WriteString(activeTodos)
	}
	b.WriteString("\n\n")
	b.WriteString(buildRuntimeEnvironment(sandboxStatus))

	return b.String()
}

// buildRuntimeEnvironment returns a block describing the runtime OS / shell environment.
// Injected into the system prompt; shell fields come from the same resolve path as exec_shell.
func buildRuntimeEnvironment(st domain.SandboxStatus) string {
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

	shell := st.Shell
	if shell == "" {
		shell = "sh"
		if osName == "windows" {
			shell = "cmd"
		}
	}

	var b strings.Builder
	b.WriteString("<runtime-environment>\n")
	b.WriteString("OS: " + osName + " (" + osLabel + ")\n")
	b.WriteString("Path separator: " + sep + "\n")
	b.WriteString("Shell: " + shell + "\n")
	if st.ShellPath != "" {
		b.WriteString("Shell path: " + st.ShellPath + "\n")
	}
	if st.Shell == "bash (WSL2)" {
		b.WriteString("via: wsl -e bash -lc\n")
	}
	if st.Backend != "" {
		b.WriteString("Sandbox backend: " + string(st.Backend) + "\n")
	}
	switch {
	case st.Shell == "cmd":
		b.WriteString("Note: Git Bash not detected; exec_shell uses cmd.exe syntax, or install Git for Windows / set runtime.sandbox.backend=wsl2 for bash.\n")
	case strings.HasPrefix(st.Shell, "bash"):
		b.WriteString("Note: exec_shell invokes the Shell above under the OS sandbox when enabled (workspace-write by default). Prefer POSIX shell syntax. Avoid cmd.exe builtins unless necessary.\n")
	default:
		b.WriteString("Note: exec_shell runs under the OS sandbox when enabled (workspace-write by default).\n")
	}
	b.WriteString("</runtime-environment>")
	return b.String()
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
		if cat := skillCategory(sk); cat != "" {
			fmt.Fprintf(&b, "    <category>%s</category>\n", escapeXML(cat))
		}
		fmt.Fprintf(&b, "    <description>%s</description>\n", escapeXML(sk.Description))
		if sk.SystemHint != "" {
			fmt.Fprintf(&b, "    <hint>%s</hint>\n", escapeXML(sk.SystemHint))
		}
		fmt.Fprintf(&b, "  </skill>\n")
	}
	b.WriteString("</available_skills>")
	return b.String()
}

func buildAgentMetadata(agents []domain.Agent) string {
	if len(agents) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available_agents>\n")
	b.WriteString("  <!-- Delegate work to these agents with delegate_agent(agent_id=..., goal=...) -->\n")
	for _, a := range agents {
		fmt.Fprintf(&b, "  <agent>\n")
		fmt.Fprintf(&b, "    <id>%s</id>\n", escapeXML(a.ID))
		fmt.Fprintf(&b, "    <name>%s</name>\n", escapeXML(a.Name))
		fmt.Fprintf(&b, "    <description>%s</description>\n", escapeXML(a.Description))
		fmt.Fprintf(&b, "  </agent>\n")
	}
	b.WriteString("</available_agents>")
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

// skillCategory returns metadata.category when set (general | coding | work).
func skillCategory(sk domain.Skill) string {
	if sk.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(sk.Metadata["category"])
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
