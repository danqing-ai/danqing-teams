package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepInclude(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main\nfunc Hello() {}"), 0644)
	os.WriteFile(filepath.Join(dir, "app.ts"), []byte("function Hello() {}\n"), 0644)

	h := &Grep{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":          "hello",
		"path":             dir,
		"include":          "*.go",
		"case_insensitive": true,
		"__work_dir":       dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "app.go") {
		t.Error("expected app.go match")
	}
	if strings.Contains(result.Content, "app.ts") {
		t.Error("app.ts should be excluded by include filter")
	}
}

func TestGrepContextLines(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3 - match\nline4\nline5\n"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644)

	h := &Grep{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":       "match",
		"path":          dir,
		"context_lines": float64(1),
		"__work_dir":    dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "line2") || !strings.Contains(result.Content, "line4") {
		t.Error("expected context lines around match")
	}
}

func TestGrepCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("HELLO world\n"), 0644)

	h := &Grep{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":          "hello",
		"path":             dir,
		"case_insensitive": true,
		"__work_dir":       dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "HELLO") {
		t.Error("expected case-insensitive match")
	}
}

func TestGrepExcludesDefaultDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("function hello() {}"), 0644)
	os.WriteFile(filepath.Join(dir, "src.go"), []byte("hello world\n"), 0644)

	h := &Grep{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":    "hello",
		"path":       dir,
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "src.go") {
		t.Error("expected src.go match")
	}
	if strings.Contains(result.Content, "node_modules") {
		t.Error("node_modules should be excluded by default")
	}
}

func TestGrepBinarySkipped(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	os.WriteFile(filepath.Join(dir, "binary.bin"), data, 0644)

	h := &Grep{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":    ".*",
		"path":       dir,
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result.Content, "binary.bin") {
		t.Error("binary file should be skipped")
	}
}

func TestGlobWithPath(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.MkdirAll(filepath.Join(dir, "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "app.go"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "lib", "util.go"), []byte("x"), 0644)

	h := &Glob{}
	result, err := h.Execute(nil, map[string]any{
		"pattern":    "*.go",
		"path":       filepath.Join(dir, "src"),
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result.Content, "util.go") {
		t.Error("util.go should not appear when scoped to src/")
	}
}

func TestGlobMatchPath(t *testing.T) {
	tests := []struct {
		pattern []string
		path    []string
		want    bool
	}{
		{[]string{"**", "*.go"}, []string{"main.go"}, true},
		{[]string{"**", "*.go"}, []string{"a", "b", "main.go"}, true},
		{[]string{"**", "*.go"}, []string{"main.ts"}, false},
		{[]string{"src", "**", "*.ts"}, []string{"src", "a", "b", "app.ts"}, true},
		{[]string{"src", "**", "*.ts"}, []string{"src", "app.ts"}, true},
		{[]string{"src", "**", "*.ts"}, []string{"lib", "app.ts"}, false},
		{[]string{"**"}, []string{"any", "path", "here"}, true},
		{[]string{"*.go"}, []string{"main.go"}, true},
		{[]string{"*.go"}, []string{"a", "main.go"}, false},
	}

	for _, tt := range tests {
		got := globMatchPath(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("globMatchPath(%v, %v) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestDoubleGlob(t *testing.T) {
	dir := t.TempDir()
	dirs := []string{
		filepath.Join(dir, "src"),
		filepath.Join(dir, "src", "sub"),
		filepath.Join(dir, "lib"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
	files := []string{
		filepath.Join(dir, "main.go"),
		filepath.Join(dir, "src", "app.go"),
		filepath.Join(dir, "src", "app.ts"),
		filepath.Join(dir, "src", "sub", "deep.go"),
		filepath.Join(dir, "lib", "util.go"),
	}
	for _, f := range files {
		os.WriteFile(f, []byte("x"), 0644)
	}

	tests := []struct {
		pattern string
		wantMin int
		desc    string
	}{
		{filepath.Join(dir, "**", "*.go"), 4, "** should match all .go files recursively"},
		{filepath.Join(dir, "src", "**", "*.go"), 2, "src/**/*.go should match files in src and sub"},
		{filepath.Join(dir, "**", "*.ts"), 1, "**/*.ts should match 1 .ts file"},
	}

	for _, tt := range tests {
		matches, err := doubleGlob(tt.pattern)
		if err != nil {
			t.Fatalf("doubleGlob(%q) error: %v", tt.pattern, err)
		}
		if len(matches) < tt.wantMin {
			t.Errorf("%s: got %d matches (%v), want at least %d", tt.desc, len(matches), matches, tt.wantMin)
		}
	}
}
