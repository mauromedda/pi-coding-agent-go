// ABOUTME: Tests for search_definitions tool: AST-based + regex definition finder
// ABOUTME: Covers Go AST path (func, struct, interface) and regex fallback for other languages

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupDefsTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

type Server struct {
	port int
}

type Handler interface {
	ServeHTTP()
}

func NewServer(port int) *Server {
	return &Server{port: port}
}

func (s *Server) Start() error {
	return nil
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "helper.py"), []byte(`class MyClass:
    def helper(self):
        pass

def standalone():
    return True
`), 0o644)

	os.WriteFile(filepath.Join(dir, "lib.rs"), []byte(`struct Point {
    x: f64,
    y: f64,
}

fn distance(a: &Point, b: &Point) -> f64 {
    0.0
}

trait Drawable {
    fn draw(&self);
}
`), 0o644)

	return dir
}

func TestSearchDefinitions_GoAST_Func(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "NewServer",
		"path":     dir,
		"language": "go",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "func NewServer") {
		t.Errorf("expected 'func NewServer' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_GoAST_Struct(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "Server",
		"path":     dir,
		"language": "go",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "struct Server") {
		t.Errorf("expected 'struct Server' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_GoAST_Interface(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "Handler",
		"path":     dir,
		"language": "go",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "interface Handler") {
		t.Errorf("expected 'interface Handler' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_GoAST_Method(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "Start",
		"path":     dir,
		"language": "go",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "method (Server) Start") {
		t.Errorf("expected 'method (Server) Start' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_RegexFallback_Python(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "helper",
		"path":     dir,
		"language": "python",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "helper.py") {
		t.Errorf("expected helper.py in output:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "def helper") {
		t.Errorf("expected 'def helper' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_RegexFallback_Rust(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern":  "Point",
		"path":     dir,
		"language": "rust",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "struct Point") {
		t.Errorf("expected 'struct Point' in output:\n%s", result.Content)
	}
}

func TestSearchDefinitions_NoMatches(t *testing.T) {
	t.Parallel()

	dir := setupDefsTestDir(t)
	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"pattern": "NoSuchThing",
		"path":    dir,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "no definitions found") {
		t.Errorf("expected 'no definitions found', got:\n%s", result.Content)
	}
}

func TestSearchDefinitions_MissingPattern(t *testing.T) {
	t.Parallel()

	tool := NewSearchDefinitionsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing pattern param")
	}
}

func TestSearchDefinitions_IsReadOnly(t *testing.T) {
	t.Parallel()

	tool := NewSearchDefinitionsTool()
	if !tool.ReadOnly {
		t.Error("search_definitions must be read-only")
	}
}
