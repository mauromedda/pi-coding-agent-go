// ABOUTME: Dependency graph tool: builds Go import graph via go/parser ImportsOnly
// ABOUTME: Groups imports by package directory, supports substring filtering

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewDependencyGraphTool creates a read-only tool that shows Go import dependencies.
func NewDependencyGraphTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "dependency_graph",
		Label:       "Dependency Graph",
		Description: "Show the Go import dependency graph for packages under a directory.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":           {"type": "string", "description": "Root directory to scan (default: current dir)"},
				"package_filter": {"type": "string", "description": "Only show packages whose imports contain this substring"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeDependencyGraph,
	}
}

// pkgImports holds the imports for a single package directory.
type pkgImports struct {
	dir     string
	imports []string
}

func executeDependencyGraph(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	root := stringParam(params, "path", ".")
	filter := stringParam(params, "package_filter", "")

	fset := token.NewFileSet()
	pkgs := make(map[string]map[string]bool) // dir -> set of import paths

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		dir, _ := filepath.Rel(root, filepath.Dir(path))
		if dir == "" || dir == "." {
			dir = "."
		}

		if pkgs[dir] == nil {
			pkgs[dir] = make(map[string]bool)
		}
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			pkgs[dir][importPath] = true
		}
		return nil
	})
	if walkErr != nil {
		return errResult(fmt.Errorf("walking %s: %w", root, walkErr)), nil
	}

	if len(pkgs) == 0 {
		return agent.ToolResult{Content: "no Go packages found"}, nil
	}

	// Collect and sort.
	var results []pkgImports
	for dir, imps := range pkgs {
		sorted := make([]string, 0, len(imps))
		for imp := range imps {
			sorted = append(sorted, imp)
		}
		sort.Strings(sorted)
		results = append(results, pkgImports{dir: dir, imports: sorted})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].dir < results[j].dir })

	var b strings.Builder
	for _, pkg := range results {
		if filter != "" {
			matched := false
			for _, imp := range pkg.imports {
				if strings.Contains(imp, filter) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		fmt.Fprintf(&b, "package ./%s\n", pkg.dir)
		for _, imp := range pkg.imports {
			if filter != "" && !strings.Contains(imp, filter) {
				continue
			}
			fmt.Fprintf(&b, "  -> %s\n", imp)
		}
		b.WriteString("\n")
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return agent.ToolResult{Content: "no packages match filter"}, nil
	}
	return agent.ToolResult{Content: out}, nil
}
