package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"danqing-teams/core/runtime/tool"
)

func resolvePath(workDir, path string) (string, error) {
	// Normalize path separators: convert any mix of / and \ to the OS-native format.
	// This handles models outputting Windows-style paths on Unix or vice versa.
	path = filepath.FromSlash(filepath.ToSlash(path))

	resolved := path
	if !filepath.IsAbs(path) {
		resolved = filepath.Join(workDir, path)
	}
	resolved = filepath.Clean(resolved)

	rel, err := filepath.Rel(workDir, resolved)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %q: use relative paths from project root, or use list_directory to discover valid paths", path)
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q is outside project directory. Use relative paths (e.g., 'src/main.go') or start with list_directory to explore the project structure", path)
	}
	return resolved, nil
}

func workDirFromInput(input map[string]any) string {
	s, _ := input["__work_dir"].(string)
	return s
}

func fileTrackerFromInput(input map[string]any) *tool.FileTracker {
	t, _ := input["__file_tracker"].(*tool.FileTracker)
	return t
}

func noteReadFile(input map[string]any, path string) {
	t := fileTrackerFromInput(input)
	if t != nil {
		_ = t.NoteRead(path)
	}
}

func requireFreshRead(input map[string]any, path string) error {
	t := fileTrackerFromInput(input)
	if t == nil {
		return nil
	}
	return t.RequireRead(path)
}

func checkBinary(data []byte, path string) error {
	if isBinary(data) {
		return fmt.Errorf("file %q appears to be binary (annotated as binary)", path)
	}
	return nil
}

func readFilePath(workDir, pathName string) (string, os.FileInfo, error) {
	resolvedPath, err := resolvePath(workDir, pathName)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			dir := filepath.Dir(resolvedPath)
			if suggestions := fuzzyFileSuggestions(dir, filepath.Base(resolvedPath)); len(suggestions) > 0 {
				return "", nil, fmt.Errorf("file not found: %q. Did you mean: %s?", pathName, strings.Join(suggestions, ", "))
			}
		}
		return "", nil, err
	}
	return resolvedPath, info, nil
}
