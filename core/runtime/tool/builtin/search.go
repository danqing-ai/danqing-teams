package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"danqing-teams/core/domain"
)

type Grep struct{}

func (h *Grep) Name() string                { return "grep" }
func (h *Grep) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *Grep) Describe(args map[string]any) string {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	return "\"" + pattern + "\" in " + path
}
func (h *Grep) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "grep",
		Description: "Fast content search tool that searches file contents using regular expressions.\n\n" +
			"**Important**: Searches within the project root directory by default. Use relative paths when specifying a search location.\n\n" +
			"- Supports full regex syntax (e.g. \"log.*Error\", \"function\\s+\\w+\").\n" +
			"- Returns file paths, line numbers, and matching lines.\n" +
			"- Use this tool when you need to find files containing specific patterns.\n" +
			"- You can call multiple grep searches in parallel to batch lookups.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "Regex pattern to search"},
				"path":    map[string]any{"type": "string", "description": "Relative directory or file path to search (default: project root)"},
				"max":     map[string]any{"type": "integer", "description": "Maximum number of results (default: 20)"},
			},
			"required": []string{"pattern"},
		},
	}
}
func (h *Grep) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	pattern, _ := input["pattern"].(string)
	if pattern == "" {
		return domain.ToolResult{}, fmt.Errorf("pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("invalid regex: %w", err)
	}
	root, _ := input["path"].(string)
	if root == "" {
		root = "."
	}
	root, err = resolvePath(workDirFromInput(input), root)
	if err != nil {
		return domain.ToolResult{}, err
	}
	max := 20
	if m, ok := input["max"].(float64); ok {
		max = int(m)
	}

	var results []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
				if len(results) >= max {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	return domain.ToolResult{Content: strings.Join(results, "\n")}, nil
}

type Glob struct{}

func (h *Glob) Name() string                { return "glob" }
func (h *Glob) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *Glob) Describe(args map[string]any) string {
	pattern, _ := args["pattern"].(string)
	return pattern
}
func (h *Glob) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "glob",
		Description: "Fast file pattern matching tool to find files by name.\n\n" +
			"**Important**: Searches within the project root directory. Use relative paths in patterns.\n\n" +
			"- Supports glob patterns like \"**/*.go\" or \"src/**/*.ts\".\n" +
			"- The ** pattern matches any number of directory levels (recursive).\n" +
			"- Returns matching file paths, including dotfiles.\n" +
			"- Use this tool when you need to find files by name patterns.\n" +
			"- Batch multiple glob searches in parallel when looking for different file types.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "Glob pattern, e.g. \"**/*.go\" or \"src/**/*.ts\""},
				"max":     map[string]any{"type": "integer", "description": "Maximum number of results (default: 50)"},
			},
			"required": []string{"pattern"},
		},
	}
}
func (h *Glob) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	pattern, _ := input["pattern"].(string)
	if pattern == "" {
		return domain.ToolResult{}, fmt.Errorf("pattern is required")
	}
	max := 50
	if m, ok := input["max"].(float64); ok {
		max = int(m)
	}

	pattern, err := resolvePath(workDirFromInput(input), pattern)
	if err != nil {
		return domain.ToolResult{}, err
	}

	var matches []string
	if strings.Contains(pattern, "**") {
		matches, err = doubleGlob(pattern)
	} else {
		matches, err = filepath.Glob(pattern)
	}
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("invalid glob: %w", err)
	}
	if len(matches) > max {
		matches = matches[:max]
	}
	return domain.ToolResult{Content: strings.Join(matches, "\n")}, nil
}

// doubleGlob handles glob patterns containing ** (recursive directory matching).
// It uses filepath.WalkDir to enumerate all files and globMatchPath to check each
// path against the pattern. This replaces filepath.Glob which does not support **.
func doubleGlob(pattern string) ([]string, error) {
	// Validate: reject malformed patterns early.
	if _, err := filepath.Match("", ""); err != nil {
		return nil, err
	}

	base := pattern
	for strings.Contains(base, "**") {
		parent := filepath.Dir(base)
		if parent == base {
			break
		}
		base = parent
	}
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		base = filepath.Dir(base)
	}

	patSegs := strings.Split(filepath.ToSlash(pattern), "/")
	var matches []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		pathSegs := strings.Split(filepath.ToSlash(path), "/")
		if globMatchPath(patSegs, pathSegs) {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

// globMatchPath matches a file path (split into segments) against a pattern
// (split into segments). The ** segment matches zero or more directory levels.
// Other segments are matched using filepath.Match (supports *, ?, [charset]).
func globMatchPath(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		rest := pattern[1:]
		// ** at end matches everything below
		if len(rest) == 0 {
			return true
		}
		// Try matching the rest at every possible depth
		for i := 0; i <= len(path); i++ {
			if globMatchPath(rest, path[i:]) {
				return true
			}
		}
		return false
	}
	if len(path) == 0 {
		return false
	}
	ok, _ := filepath.Match(pattern[0], path[0])
	if !ok {
		return false
	}
	return globMatchPath(pattern[1:], path[1:])
}
