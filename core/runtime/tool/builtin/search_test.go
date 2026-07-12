package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

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
	// Create a temp directory structure
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
		pattern  string
		wantMin  int
		desc     string
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
