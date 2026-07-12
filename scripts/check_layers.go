//go:build ignore

// Layer import boundary checker.
// Usage: go run scripts/check_layers.go
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type rule struct {
	prefix    string
	forbidden []string
}

var rules = []rule{
	// app — business orchestration; only port + domain
	{
		prefix: "core/service",
		forbidden: []string{
			"/core/runtime",
			"/core/adapter",
			"/core/store/",
			"/core/bootstrap",
		},
	},
	// runtime — mission engine; no concrete adapters or persistence impls
	{
		prefix: "core/runtime",
		forbidden: []string{
			"/core/adapter",
			"/core/store/",
		},
	},
	// REST transport — delegates to app, not runtime internals
	{
		prefix: "server/api",
		forbidden: []string{
			"/core/runtime",
			"/core/adapter",
			"/core/store/sqlite",
			"/core/store/turnlog",
			"/core/bootstrap",
		},
	},
	// shared core must not depend on HTTP
	{
		prefix: "core/",
		forbidden: []string{
			"/server/api/",
		},
	},
	// port — pure contracts
	{
		prefix: "core/port",
		forbidden: []string{
			"/core/runtime",
			"/core/adapter",
			"/core/store/",
			"/core/service",
			"/core/bootstrap",
		},
	},
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
		os.Exit(1)
	}
	modulePath := readModulePath(root)
	if modulePath == "" {
		fmt.Fprintln(os.Stderr, "could not read module path from go.mod")
		os.Exit(1)
	}

	var violations []string
	for _, dir := range []string{"core", "server"} {
		walkDir(filepath.Join(root, dir), modulePath, &violations)
	}

	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "layer violations:")
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, " ", v)
		}
		os.Exit(1)
	}
	fmt.Println("layer check OK")
}

func walkDir(dir, modulePath string, violations *[]string) {
	fset := token.NewFileSet()
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, filepath.Dir(path))
		rel = filepath.ToSlash(filepath.Join(filepath.Base(dir), rel))
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(importPath, modulePath) {
				continue
			}
			suffix := strings.TrimPrefix(importPath, modulePath)
			for _, r := range rules {
				if !strings.HasPrefix(rel, r.prefix) {
					continue
				}
				for _, forbidden := range r.forbidden {
					if strings.Contains(suffix, forbidden) {
						*violations = append(*violations, fmt.Sprintf("%s imports %s (forbidden for %s)", rel, importPath, r.prefix))
					}
				}
			}
		}
		return nil
	})
}

func readModulePath(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
