package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"danqing-teams/core/domain"
)

type ReadFile struct{}

func (h *ReadFile) Name() string                { return "read_file" }
func (h *ReadFile) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *ReadFile) Describe(args map[string]any) string {
	path, _ := args["path"].(string)
	offset, limit := parseOffsetLimit(args)
	if offset == 0 && limit == 0 {
		return path
	}
	if limit == 0 {
		return fmt.Sprintf("%s (offset=%d)", path, offset)
	}
	return fmt.Sprintf("%s (offset=%d, limit=%d)", path, offset, limit)
}
func (h *ReadFile) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "read_file",
		Description: "Read the contents of a local file. If the path does not exist, an error is returned.\n\n" +
			"**Important**: All paths are relative to the project root directory. Use relative paths like 'src/main.go' instead of absolute paths.\n\n" +
			"**Line-based reading**: Use offset and limit to read specific line ranges instead of the whole file.\n" +
			"- offset: 0-based line number to start reading from (default 0). Use with grep results to jump to a specific line.\n" +
			"- limit: maximum number of lines to read (default 0 = read to end). Use to limit output size for large files.\n\n" +
			"Usage:\n" +
			"- Use relative paths from project root (e.g., 'pkg/handler.go', 'frontend/src/App.vue').\n" +
			"- If unsure of the correct path, use list_directory first to explore the project structure.\n" +
			"- Use grep to find specific content in large files first, then use offset+limit to read around matches.\n" +
			"- Use glob to look up filenames by pattern.\n" +
			"- Call this tool in parallel when reading multiple files.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "Relative file path from project root (e.g., 'src/main.go')"},
				"offset": map[string]any{"type": "integer", "description": "0-based line number to start reading from (default: 0)"},
				"limit":  map[string]any{"type": "integer", "description": "Maximum number of lines to read (default: 0 = read to end)"},
			},
			"required": []string{"path"},
		},
	}
}
func (h *ReadFile) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	path, _ := input["path"].(string)
	if path == "" {
		return domain.ToolResult{}, fmt.Errorf("path is required")
	}
	path, err := resolvePath(workDirFromInput(input), path)
	if err != nil {
		return domain.ToolResult{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.ToolResult{}, err
	}

	offset, limit := parseOffsetLimit(input)
	if offset == 0 && limit == 0 {
		return domain.ToolResult{Content: string(data)}, nil
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if offset >= len(lines) {
		return domain.ToolResult{}, fmt.Errorf("offset %d exceeds file length %d lines", offset, len(lines))
	}

	end := len(lines)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return domain.ToolResult{Content: strings.Join(lines[offset:end], "\n")}, nil
}

func parseOffsetLimit(input map[string]any) (int, int) {
	offset := 0
	if o, ok := input["offset"].(float64); ok {
		offset = int(o)
	}
	limit := 0
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}
	return offset, limit
}
