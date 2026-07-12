package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"danqing-teams/core/domain"
)

type Write struct{}

func (h *Write) Name() string                { return "write" }
func (h *Write) RiskLevel() domain.RiskLevel { return domain.RiskMedium }
func (h *Write) Describe(args map[string]any) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	writeType, _ := args["write_type"].(string)
	if writeType == "directory" {
		return "创建目录 " + path
	}
	return path + " (" + fmt.Sprintf("%d", len(content)) + " chars)"
}
func (h *Write) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "write",
		Description: "Writes a file or creates a directory on the local filesystem.\n\n" +
			"**Important**: All paths are relative to the project root directory. Use relative paths like 'src/main.go' instead of absolute paths.\n\n" +
			"- Auto-creates all parent directories — do NOT use exec_shell mkdir beforehand.\n" +
			"- This tool will overwrite the existing file if there is one at the provided path.\n" +
			"- If the file already exists, you MUST use read_file first to read its contents.\n" +
			"- ALWAYS prefer editing existing files with edit or apply_patch. NEVER write new files unless explicitly required.\n" +
			"- Do NOT use exec_shell with cat/echo/heredoc for writing files.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string", "description": "Relative file path from project root (e.g., 'src/main.go')"},
				"content":    map[string]any{"type": "string", "description": "The content to write to the file (required when type is 'file')"},
				"write_type": map[string]any{"type": "string", "enum": []string{"file", "directory"}, "default": "file", "description": "'file' to write a file (default), 'directory' to create a directory. content is optional for 'directory'."},
			},
			"required": []string{"path"},
		},
	}
}

func (h *Write) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	path, _ := input["path"].(string)
	content, _ := input["content"].(string)
	writeType, _ := input["write_type"].(string)

	if path == "" {
		return domain.ToolResult{}, fmt.Errorf("path is required")
	}
	if writeType == "" {
		writeType = "file"
	}

	path, err := resolvePath(workDirFromInput(input), path)
	if err != nil {
		return domain.ToolResult{}, err
	}

	if writeType == "directory" {
		if err := os.MkdirAll(path, 0755); err != nil {
			return domain.ToolResult{}, fmt.Errorf("cannot create directory %q: %w", path, err)
		}
		return domain.ToolResult{Content: fmt.Sprintf("Created directory %q", path)}, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return domain.ToolResult{}, fmt.Errorf("cannot create directory %q: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return domain.ToolResult{}, fmt.Errorf("cannot write file %q: %w", path, err)
	}

	return domain.ToolResult{Content: fmt.Sprintf("Wrote file %q (%d bytes)", path, len(content))}, nil
}
