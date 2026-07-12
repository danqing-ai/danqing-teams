package builtin

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"danqing-teams/core/domain"
)

type ListDir struct{}

func (h *ListDir) Name() string                { return "list_directory" }
func (h *ListDir) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *ListDir) Describe(args map[string]any) string {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	return path
}
func (h *ListDir) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "list_directory",
		Description: "Lists directory contents with file sizes and modification times. Fully replaces the need for `ls` or `ls -la` shell commands.\n\n" +
			"**Important**: All paths are relative to the project root directory. Use '.' or omit path to list the project root.\n\n" +
			"- Directories are prefixed with '[dir ]' and suffixed with '/' for easy identification.\n" +
			"- File sizes are shown in bytes (raw number).\n" +
			"- Modification times are shown for each entry.\n" +
			"- Dotfiles (hidden files) are included.\n" +
			"- Results are sorted alphabetically (directories first, then files).\n" +
			"- Use this tool FIRST when you're unsure about the project structure or file locations. " +
			"Also use it to verify files after writes — shows size and time, same as `ls -la`.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Relative directory path from project root (default: '.' for project root)"},
			},
		},
	}
}
func (h *ListDir) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	path, _ := input["path"].(string)
	if path == "" {
		path = "."
	}
	path, err := resolvePath(workDirFromInput(input), path)
	if err != nil {
		return domain.ToolResult{}, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return domain.ToolResult{}, err
	}

	type entry struct {
		name     string
		isDir    bool
		size     int64
		modTime  string
	}
	var items []entry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			items = append(items, entry{name: e.Name(), isDir: e.IsDir(), size: -1})
			continue
		}
		items = append(items, entry{
			name:    e.Name(),
			isDir:   e.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime().Format("2006-01-02 15:04"),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Contents of %s (%d items):\n", path, len(items)))
	for _, item := range items {
		if item.isDir {
			result.WriteString(fmt.Sprintf("  [dir ] %s/\n", item.name))
		} else {
			result.WriteString(fmt.Sprintf("  [file] %-40s %10d bytes  %s\n", item.name, item.size, item.modTime))
		}
	}
	content := result.String()
	return domain.ToolResult{Content: content}, nil
}

