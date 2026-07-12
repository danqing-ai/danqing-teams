package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"danqing-teams/core/domain"
)

type Edit struct{}

func (h *Edit) Name() string                { return "edit" }
func (h *Edit) RiskLevel() domain.RiskLevel { return domain.RiskMedium }
func (h *Edit) Describe(args map[string]any) string {
	path, _ := args["path"].(string)
	oldStr, _ := args["oldString"].(string)
	newStr, _ := args["newString"].(string)
	oldShort := oldStr
	newShort := newStr
	if len(oldStr) > 40 {
		oldShort = oldStr[:40] + "..."
	}
	if len(newStr) > 40 {
		newShort = newStr[:40] + "..."
	}
	return path + " (" + oldShort + " → " + newShort + ")"
}
func (h *Edit) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "edit",
		Description: "Performs exact string replacements in an existing file.\n\n" +
			"**Important**: All paths are relative to the project root directory. Use relative paths like 'src/main.go' instead of absolute paths.\n\n" +
			"- You MUST use read_file first — the edit will fail if you haven't read the file.\n" +
			"- oldString must match exactly one occurrence in the file. If it matches multiple times, set replaceAll=true or provide more surrounding context to make it unique.\n" +
			"- newString must be different from oldString.\n" +
			"- Preserve exact indentation (tabs/spaces) as it appears in the file.\n" +
			"- Use replaceAll for replacing and renaming strings across the file.\n" +
			"- For multi-hunk or multi-file edits, prefer apply_patch instead.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string", "description": "Relative file path from project root (e.g., 'src/main.go')"},
				"oldString":  map[string]any{"type": "string", "description": "The text to replace"},
				"newString":  map[string]any{"type": "string", "description": "The text to replace it with (must be different from oldString)"},
				"replaceAll": map[string]any{"type": "boolean", "description": "Replace all occurrences of oldString (default: false)"},
			},
			"required": []string{"path", "oldString", "newString"},
		},
	}
}

func (h *Edit) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	path, _ := input["path"].(string)
	oldStr, _ := input["oldString"].(string)
	newStr, _ := input["newString"].(string)
	replaceAll, _ := input["replaceAll"].(bool)

	if path == "" {
		return domain.ToolResult{}, fmt.Errorf("path is required")
	}
	if oldStr == "" {
		return domain.ToolResult{}, fmt.Errorf("oldString is required")
	}
	if oldStr == newStr {
		return domain.ToolResult{}, fmt.Errorf("oldString and newString must be different")
	}

	path, err := resolvePath(workDirFromInput(input), path)
	if err != nil {
		return domain.ToolResult{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("cannot read file %q: %w", path, err)
	}
	content := string(data)

	count := strings.Count(content, oldStr)
	if count == 0 {
		return domain.ToolResult{}, fmt.Errorf("oldString not found in file %q", path)
	}
	if !replaceAll && count > 1 {
		return domain.ToolResult{}, fmt.Errorf("found %d occurrences of oldString in %q; set replaceAll=true to replace all, or use more context to make oldString unique", count, path)
	}

	if replaceAll {
		content = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		content = strings.Replace(content, oldStr, newStr, 1)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return domain.ToolResult{}, fmt.Errorf("cannot write file %q: %w", path, err)
	}

	return domain.ToolResult{Content: fmt.Sprintf("Edited file %q, replaced %d occurrence(s)", path, map[bool]int{true: count, false: 1}[replaceAll])}, nil
}
