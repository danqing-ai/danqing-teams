package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileLineNumbers(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	os.WriteFile(f, []byte(content), 0644)

	h := &ReadFile{}
	result, err := h.Execute(nil, map[string]any{
		"path":       f,
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "1: line one") {
		t.Error("expected line number prefix '1: line one'")
	}
	if !strings.Contains(result.Content, "2: line two") {
		t.Error("expected line number prefix '2: line two'")
	}
}

func TestReadFileOffset1Indexed(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	os.WriteFile(f, []byte(content), 0644)

	h := &ReadFile{}
	result, err := h.Execute(nil, map[string]any{
		"path":       f,
		"__work_dir": dir,
		"offset":     float64(3),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result.Content, "1: line one") {
		t.Error("offset=3 should not include line 1")
	}
	if !strings.Contains(result.Content, "3: line three") {
		t.Error("expected line 3 as first line with offset=3")
	}
}

func TestReadFileTruncated(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line")
	}
	os.WriteFile(f, []byte(strings.Join(lines, "\n")+"\n"), 0644)

	h := &ReadFile{}
	result, err := h.Execute(nil, map[string]any{
		"path":       f,
		"__work_dir": dir,
		"limit":      float64(5),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "[Truncated") {
		t.Error("expected truncation hint")
	}
}

func TestReadFileBinary(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "binary.bin")
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	os.WriteFile(f, data, 0644)

	h := &ReadFile{}
	result, err := h.Execute(nil, map[string]any{
		"path":       f,
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "[Binary file not displayed") {
		t.Error("expected binary file annotation")
	}
}

func TestReadFileDirectory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)

	h := &ReadFile{}
	result, err := h.Execute(nil, map[string]any{
		"path":       dir,
		"__work_dir": dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "[dir ]") {
		t.Error("expected directory entry")
	}
	if !strings.Contains(result.Content, "[file]") {
		t.Error("expected file entry")
	}
}

func TestReadFileNotFoundSuggestion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main"), 0644)

	h := &ReadFile{}
	_, err := h.Execute(nil, map[string]any{
		"path":       "mainn.go",
		"__work_dir": dir,
	})
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "Did you mean") {
		t.Errorf("expected 'Did you mean' suggestion, got: %v", err)
	}
}
