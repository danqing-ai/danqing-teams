package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditExactReplace(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)
	ft.NoteRead(f)

	h := &Edit{}
	result, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "\"hello\"",
		"newString":     "\"world\"",
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "\"world\"") {
		t.Errorf("expected replacement, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "--- a/") {
		t.Error("expected unified diff in output")
	}
}

func TestEditReplaceAll(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "foo bar foo bar foo\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)
	ft.NoteRead(f)

	h := &Edit{}
	result, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "foo",
		"newString":     "baz",
		"replaceAll":    true,
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "3 occurrence") {
		t.Errorf("expected 3 replacements, got: %s", result.Content)
	}
}

func TestEditFuzzyIndent(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	content := "func main() {\n\t\tprintln(\"hello\")\n}\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)
	ft.NoteRead(f)

	h := &Edit{}
	result, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "\tprintln(\"hello\")",
		"newString":     "\tprintln(\"world\")",
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err != nil {
		t.Fatalf("fuzzy indent matching failed: %v", err)
	}

	if !strings.Contains(result.Content, "println(\"world\")") {
		t.Errorf("expected fuzzy replacement, got: %s", result.Content)
	}
}

func TestEditFuzzyWhitespace(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "hello\nworld\nfoo  bar\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)
	ft.NoteRead(f)

	h := &Edit{}
	result, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "foo bar",
		"newString":     "baz qux",
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err != nil {
		t.Fatalf("fuzzy whitespace matching failed: %v", err)
	}
	if !strings.Contains(result.Content, "baz qux") {
		t.Errorf("expected fuzzy replacement, got: %s", result.Content)
	}
}

func TestEditRequiresReadFirst(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "hello world\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)

	h := &Edit{}
	_, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "hello",
		"newString":     "goodbye",
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err == nil {
		t.Fatal("expected error for editing without reading first")
	}
	if !strings.Contains(err.Error(), "has not been read yet") {
		t.Errorf("expected read-first error, got: %v", err)
	}
}

func TestEditMultipleExactMatches(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "foo bar foo baz foo\n"
	os.WriteFile(f, []byte(content), 0644)

	ft := setupTracker(dir)
	ft.NoteRead(f)

	h := &Edit{}
	_, err := h.Execute(nil, map[string]any{
		"path":          f,
		"oldString":     "foo",
		"newString":     "bar",
		"replaceAll":    false,
		"__work_dir":    dir,
		"__file_tracker": ft,
	})
	if err == nil {
		t.Fatal("expected error for multiple matches without replaceAll")
	}
	if !strings.Contains(err.Error(), "set replaceAll=true") {
		t.Errorf("expected replaceAll hint, got: %v", err)
	}
}
