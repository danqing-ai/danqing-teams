//go:build ignore

// Layer import boundary checker for Spring-style backend layout.
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
	{
		prefix: "internal/api/rest/controller",
		forbidden: []string{
			"/internal/application/service",
			"/internal/persistence/",
			"/internal/provider/",
			"/internal/domain/repository",
			"/internal/service",
			"/internal/contract",
		},
	},
	{
		prefix: "internal/api/rest/dto",
		forbidden: []string{
			"/internal/persistence/",
			"/internal/provider/",
			"/internal/application/service",
			"/internal/service",
		},
	},
	{
		prefix: "internal/application/port",
		forbidden: []string{
			"/internal/persistence/",
			"/internal/provider/",
			"/internal/api/rest/controller",
			"/internal/application/service",
			"/internal/service",
		},
	},
	{
		prefix: "internal/application/service",
		forbidden: []string{
			"/internal/api/rest/controller",
			"/internal/persistence/sqlstore",
			"/internal/persistence/memory",
			"/internal/provider/",
			"/internal/service",
		},
	},
	{
		prefix: "internal/domain/",
		forbidden: []string{
			"/internal/application/",
			"/internal/api/",
			"/internal/persistence/",
			"/internal/provider/",
			"/internal/service",
			"/internal/contract",
		},
	},
	{
		prefix: "internal/core/",
		forbidden: []string{
			"/internal/application/",
			"/internal/api/",
			"/internal/persistence/",
			"/internal/provider/",
			"/internal/service",
			"/internal/contract",
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
	fset := token.NewFileSet()
	_ = filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
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
						violations = append(violations, fmt.Sprintf("%s imports %s (forbidden for %s)", rel, importPath, r.prefix))
					}
				}
			}
		}
		return nil
	})

	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "layer violations:")
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, " ", v)
		}
		os.Exit(1)
	}
	fmt.Println("layer check OK")
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
