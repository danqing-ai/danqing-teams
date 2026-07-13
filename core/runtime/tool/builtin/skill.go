package builtin

import (
	"context"
	"fmt"
	"strings"

	"danqing-teams/core/domain"
	"danqing-teams/core/service"
)

// ReadSkill reads a skill's instructions or its bundled resource files.
// All data is served from the DB (single runtime data source).
//
// Path convention:
//   - "git-workflow"              → returns the skill's body (instructions)
//   - "debugging/references/patterns.md" → returns the resource file content
//   - Resource paths are relative to the skill directory
//   - Valid resource subdirectories: scripts/, references/, assets/
type ReadSkill struct {
	Skills *service.SkillManager
}

func (h *ReadSkill) Name() string                { return "read_skill" }
func (h *ReadSkill) RiskLevel() domain.RiskLevel { return domain.RiskLow }

func (h *ReadSkill) Describe(args map[string]any) string {
	path, _ := args["path"].(string)
	return path
}

func (h *ReadSkill) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "read_skill",
		Description: "Read a skill's instructions or bundled resource files.\n\n" +
			"**Path format:**\n" +
			"- Skill instructions: path=\"git-workflow\" (skill name only)\n" +
			"- Resource file: path=\"debugging/references/patterns.md\" (skill name + relative path)\n" +
			"- Resource subdirectories: scripts/, references/, assets/\n" +
			"- See <available_skills> in system prompt for available skill paths and descriptions.\n\n" +
			"**Examples:**\n" +
			"- path=\"git-workflow\"                       → skill instructions\n" +
			"- path=\"debugging/references/patterns.md\"   → reference file\n" +
			"- path=\"git-workflow/scripts/commit.sh\"     → script file",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Skill name (e.g. \"git-workflow\") or skill name + resource path (e.g. \"debugging/references/patterns.md\")"},
			},
			"required": []string{"path"},
		},
	}
}

func (h *ReadSkill) Execute(ctx context.Context, input map[string]any) (domain.ToolResult, error) {
	path, _ := input["path"].(string)
	if path == "" {
		return domain.ToolResult{}, fmt.Errorf("path is required")
	}

	// Security: reject path traversal
	if strings.Contains(path, "..") {
		return domain.ToolResult{}, fmt.Errorf("invalid path: must not contain \"..\"")
	}

	// Split path: first segment is skill ID, rest is resource path
	parts := strings.SplitN(path, "/", 2)
	skillID := parts[0]
	resPath := ""
	if len(parts) > 1 {
		resPath = parts[1]
	}

	sk, err := h.Skills.Get(ctx, skillID)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("skill %q not found", skillID)
	}

	// No resource path → return skill body (instructions)
	if resPath == "" {
		if sk.Body == "" {
			return domain.ToolResult{Content: fmt.Sprintf("Skill %q has no instructions.", skillID)}, nil
		}
		return domain.ToolResult{Content: sk.Body}, nil
	}

	// Validate resource subdirectory
	if !isValidResourcePath(resPath) {
		return domain.ToolResult{}, fmt.Errorf("invalid resource path %q: must be under scripts/, references/, or assets/", resPath)
	}

	// Look up resource file from DB
	files, err := h.Skills.Files(ctx, skillID)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("failed to list files for skill %q: %w", skillID, err)
	}
	for _, f := range files {
		if f.Path == resPath {
			return domain.ToolResult{Content: string(f.Content)}, nil
		}
	}

	// List available files as hint
	var available []string
	for _, f := range files {
		available = append(available, skillID+"/"+f.Path)
	}
	if len(available) > 0 {
		return domain.ToolResult{}, fmt.Errorf("resource %q not found in skill %q. Available: %s",
			path, skillID, strings.Join(available, ", "))
	}
	return domain.ToolResult{}, fmt.Errorf("resource %q not found in skill %q (no resource files available)", path, skillID)
}

func isValidResourcePath(p string) bool {
	for _, prefix := range []string{"scripts/", "references/", "assets/"} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
