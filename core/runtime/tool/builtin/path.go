package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
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
