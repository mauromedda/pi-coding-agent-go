// ABOUTME: Search definitions tool: finds function, type, and method definitions
// ABOUTME: Go files use AST parsing; other languages use regex-based pattern matching

package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const maxDefResults = 200

// NewSearchDefinitionsTool creates a read-only tool that finds definitions.
func NewSearchDefinitionsTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "search_definitions",
		Label:       "Search Definitions",
		Description: "Find function, type, struct, and interface definitions by name or pattern. Uses Go AST for .go files, regex fallback for other languages.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["pattern"],
			"properties": {
				"pattern":  {"type": "string", "description": "Name or regex pattern to match definition names"},
				"path":     {"type": "string", "description": "Directory to search (default: current dir)"},
				"language": {"type": "string", "description": "Language hint: go, python, javascript, typescript, rust, ruby, java"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeSearchDefinitions,
	}
}

// defResult holds a single definition match.
type defResult struct {
	file string
	line int
	kind string
	name string
}

func executeSearchDefinitions(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	pattern, err := requireStringParam(params, "pattern")
	if err != nil {
		return errResult(err), nil
	}

	root := stringParam(params, "path", ".")
	lang := stringParam(params, "language", "")

	// Auto-detect Go if go.mod exists and no language specified.
	if lang == "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			lang = "go"
		}
	}

	nameRe, err := regexp.Compile(pattern)
	if err != nil {
		return errResult(fmt.Errorf("invalid pattern: %w", err)), nil
	}

	var results []defResult

	switch lang {
	case "go":
		results = searchGoAST(root, nameRe)
	default:
		results = searchRegex(root, lang, nameRe)
	}

	if len(results) == 0 {
		return agent.ToolResult{Content: "no definitions found"}, nil
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].file != results[j].file {
			return results[i].file < results[j].file
		}
		return results[i].line < results[j].line
	})

	var b strings.Builder
	for i, r := range results {
		if i >= maxDefResults {
			fmt.Fprintf(&b, "\n... truncated at %d results", maxDefResults)
			break
		}
		fmt.Fprintf(&b, "%s:%d: %s %s\n", r.file, r.line, r.kind, r.name)
	}
	return agent.ToolResult{Content: strings.TrimSpace(b.String())}, nil
}

// searchGoAST walks Go files and uses AST to find definitions.
func searchGoAST(root string, nameRe *regexp.Regexp) []defResult {
	fset := token.NewFileSet()
	var results []defResult

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if len(results) >= maxDefResults {
			return fs.SkipAll
		}

		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "" {
			rel = path
		}

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !nameRe.MatchString(d.Name.Name) {
					continue
				}
				pos := fset.Position(d.Pos())
				if d.Recv != nil && len(d.Recv.List) > 0 {
					recv := receiverTypeName(d.Recv.List[0].Type)
					results = append(results, defResult{
						file: rel, line: pos.Line,
						kind: "method", name: fmt.Sprintf("(%s) %s", recv, d.Name.Name),
					})
				} else {
					results = append(results, defResult{
						file: rel, line: pos.Line,
						kind: "func", name: d.Name.Name,
					})
				}

			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !nameRe.MatchString(ts.Name.Name) {
						continue
					}
					pos := fset.Position(ts.Pos())
					kind := "type"
					switch ts.Type.(type) {
					case *ast.StructType:
						kind = "struct"
					case *ast.InterfaceType:
						kind = "interface"
					}
					results = append(results, defResult{
						file: rel, line: pos.Line, kind: kind, name: ts.Name.Name,
					})
				}
			}
		}
		return nil
	})
	return results
}

// receiverTypeName extracts the type name from a method receiver.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	default:
		return "?"
	}
}

// langPatterns maps languages to regex patterns that match definitions.
var langPatterns = map[string]*regexp.Regexp{
	"python":     regexp.MustCompile(`^\s*(def|class)\s+(\w+)`),
	"javascript": regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|class|const|let|var)\s+(\w+)`),
	"typescript": regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?(?:function|class|interface|type|enum|const|let|var)\s+(\w+)`),
	"rust":       regexp.MustCompile(`^\s*(?:pub\s+)?(?:fn|struct|enum|trait|type|impl|mod)\s+(\w+)`),
	"ruby":       regexp.MustCompile(`^\s*(?:def|class|module)\s+(\w+)`),
	"java":       regexp.MustCompile(`^\s*(?:public|private|protected|static|final|abstract)?\s*(?:class|interface|enum|record)\s+(\w+)`),
}

// langExtensions maps languages to file extensions.
var langExtensions = map[string][]string{
	"python":     {".py"},
	"javascript": {".js", ".jsx", ".mjs"},
	"typescript": {".ts", ".tsx"},
	"rust":       {".rs"},
	"ruby":       {".rb"},
	"java":       {".java"},
}

// searchRegex walks files and uses regex to find definitions.
func searchRegex(root, lang string, nameRe *regexp.Regexp) []defResult {
	// Collect which patterns and extensions to use.
	type langMatch struct {
		pattern *regexp.Regexp
		exts    map[string]bool
	}
	var matchers []langMatch

	if lang != "" {
		p, ok := langPatterns[lang]
		if !ok {
			return nil
		}
		exts := make(map[string]bool)
		for _, e := range langExtensions[lang] {
			exts[e] = true
		}
		matchers = append(matchers, langMatch{pattern: p, exts: exts})
	} else {
		// All languages.
		for l, p := range langPatterns {
			exts := make(map[string]bool)
			for _, e := range langExtensions[l] {
				exts[e] = true
			}
			matchers = append(matchers, langMatch{pattern: p, exts: exts})
		}
	}

	var results []defResult

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if len(results) >= maxDefResults {
			return fs.SkipAll
		}

		ext := filepath.Ext(path)
		for _, m := range matchers {
			if !m.exts[ext] {
				continue
			}

			found := scanFileForDefs(path, root, m.pattern, nameRe, &results)
			if found || len(results) >= maxDefResults {
				break
			}
		}
		if len(results) >= maxDefResults {
			return fs.SkipAll
		}
		return nil
	})
	return results
}

// scanFileForDefs scans a single file for definition matches and appends to results.
// Returns true if the file matched the extension (regardless of whether definitions were found).
func scanFileForDefs(path, root string, defPattern, nameRe *regexp.Regexp, results *[]defResult) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	rel, _ := filepath.Rel(root, path)
	if rel == "" {
		rel = path
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := defPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		defName := matches[len(matches)-1]
		if !nameRe.MatchString(defName) {
			continue
		}
		*results = append(*results, defResult{
			file: rel, line: lineNum,
			kind: "def", name: strings.TrimSpace(line),
		})
		if len(*results) >= maxDefResults {
			return true
		}
	}
	return true
}
