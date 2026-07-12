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

type ApplyPatch struct{}

func (h *ApplyPatch) Name() string                { return "apply_patch" }
func (h *ApplyPatch) RiskLevel() domain.RiskLevel { return domain.RiskMedium }
func (h *ApplyPatch) Describe(args map[string]any) string {
	return "apply_patch"
}
func (h *ApplyPatch) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "apply_patch",
		Description: "Applies a unified diff patch to files. Preferred for multi-file or multi-hunk edits.\n\n" +
			"**Important**: File paths in the patch must be relative to the project root directory (e.g., 'src/main.go', not '/absolute/path/src/main.go').\n\n" +
			"- patch must be a valid unified diff string with ---/+++ file headers and @@ hunk headers.\n" +
			"- Can apply multiple hunks across multiple files in a single call.\n" +
			"- Always read the target files first to understand the current state.\n" +
			"- Use this instead of multiple edit calls when changing several locations at once.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"patch": map[string]any{"type": "string", "description": "A unified diff patch string to apply"},
			},
			"required": []string{"patch"},
		},
	}
}

var (
	hunkHeaderRe    = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
	fileHeaderRe    = regexp.MustCompile(`^--- (?:a/|b/)?(\S+)`)
	fileHeaderRe2   = regexp.MustCompile(`^\+\+\+ (?:a/|b/)?(\S+)`)
)

type hunk struct {
	oldStart, oldCount int
	newStart, newCount int
	lines              []hunkLine
}

type hunkLine struct {
	op   byte
	text string
}

type filePatch struct {
	path  string
	hunks []hunk
}

func (h *ApplyPatch) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	patch, _ := input["patch"].(string)
	if patch == "" {
		return domain.ToolResult{}, fmt.Errorf("patch is required")
	}

	workDir := workDirFromInput(input)
	patches, err := parsePatch(patch)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("invalid patch: %w", err)
	}

	var results []string
	for _, fp := range patches {
		fp.path, err = resolvePath(workDir, fp.path)
		if err != nil {
			return domain.ToolResult{}, fmt.Errorf("cannot resolve path %q: %w", fp.path, err)
		}
		data, err := os.ReadFile(fp.path)
		if err != nil {
			return domain.ToolResult{}, fmt.Errorf("cannot read file %q: %w", fp.path, err)
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

		newLines, err := applyHunks(lines, fp.hunks)
		if err != nil {
			return domain.ToolResult{}, fmt.Errorf("cannot apply patch to %q: %w", fp.path, err)
		}

		if err := os.WriteFile(fp.path, []byte(strings.Join(newLines, "\n")+"\n"), 0644); err != nil {
			return domain.ToolResult{}, fmt.Errorf("cannot write file %q: %w", fp.path, err)
		}
		results = append(results, fmt.Sprintf("Patched %q (%d hunks)", fp.path, len(fp.hunks)))
	}

	if len(results) == 0 {
		return domain.ToolResult{Content: "No files to patch"}, nil
	}

	return domain.ToolResult{Content: strings.Join(results, "\n")}, nil
}

func parsePatch(patch string) ([]filePatch, error) {
	lines := strings.Split(strings.ReplaceAll(patch, "\r\n", "\n"), "\n")
	var patches []filePatch
	var cur *filePatch
	var curHunk *hunk

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "--- "):
			if cur != nil {
				return nil, fmt.Errorf("unexpected --- header at line %d", i+1)
			}
			m := fileHeaderRe.FindStringSubmatch(line)
			if m != nil {
				cur = &filePatch{path: filepath.Clean(m[1])}
			}
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				return nil, fmt.Errorf("unexpected +++ header at line %d", i+1)
			}
			m := fileHeaderRe2.FindStringSubmatch(line)
			if m != nil && cur.path == "" {
				cur.path = filepath.Clean(m[1])
			}
		case strings.HasPrefix(line, "@@ "):
			if cur == nil {
				return nil, fmt.Errorf("hunk without file header at line %d", i+1)
			}
			m := hunkHeaderRe.FindStringSubmatch(line)
			if m == nil {
				return nil, fmt.Errorf("invalid hunk header at line %d", i+1)
			}
			h := hunk{lines: make([]hunkLine, 0)}
			h.oldStart = parseInt(m[1])
			if m[2] != "" {
				h.oldCount = parseInt(m[2])
			} else {
				h.oldCount = 1
			}
			h.newStart = parseInt(m[3])
			if m[4] != "" {
				h.newCount = parseInt(m[4])
			} else {
				h.newCount = 1
			}
			curHunk = &h
			cur.hunks = append(cur.hunks, h)
		case strings.HasPrefix(line, " "):
			if curHunk != nil {
				curHunk.lines = append(curHunk.lines, hunkLine{op: ' ', text: line[1:]})
			}
		case strings.HasPrefix(line, "-"):
			if curHunk != nil {
				curHunk.lines = append(curHunk.lines, hunkLine{op: '-', text: line[1:]})
			}
		case strings.HasPrefix(line, "+"):
			if curHunk != nil {
				curHunk.lines = append(curHunk.lines, hunkLine{op: '+', text: line[1:]})
			}
		case line == "" || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") || strings.HasPrefix(line, "deleted file"):
			if cur != nil && len(cur.hunks) > 0 {
				patches = append(patches, *cur)
			}
			cur = nil
			curHunk = nil
		}
	}
	if cur != nil && len(cur.hunks) > 0 {
		patches = append(patches, *cur)
	}
	return patches, nil
}

func applyHunks(lines []string, hunks []hunk) ([]string, error) {
	result := make([]string, 0, len(lines))
	offset := 0

	for _, h := range hunks {
		oldIdx := h.oldStart - 1 - offset

		for len(result) < oldIdx {
			result = append(result, lines[len(result)])
		}

		newIdx := oldIdx + h.oldCount
		offset += h.oldCount - h.newCount

		for _, hl := range h.lines {
			switch hl.op {
			case ' ':
				if oldIdx >= len(lines) || lines[oldIdx] != hl.text {
					return nil, fmt.Errorf("context mismatch at line %d: expected %q, got %q", oldIdx+1, hl.text, safeGet(lines, oldIdx))
				}
				result = append(result, hl.text)
				oldIdx++
			case '-':
				if oldIdx >= len(lines) || lines[oldIdx] != hl.text {
					return nil, fmt.Errorf("removal mismatch at line %d: expected %q, got %q", oldIdx+1, hl.text, safeGet(lines, oldIdx))
				}
				oldIdx++
			case '+':
				result = append(result, hl.text)
			}
		}

		for i := oldIdx; i < newIdx; i++ {
			if i < len(lines) {
				result = append(result, lines[i])
			}
		}
	}

	for len(result) < len(lines) {
		result = append(result, lines[len(result)])
	}

	for i := range result {
		if result[i] == "" {
			result[i] = ""
		}
	}

	return result, nil
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func safeGet(lines []string, idx int) string {
	if idx < len(lines) {
		return lines[idx]
	}
	return "<EOF>"
}
