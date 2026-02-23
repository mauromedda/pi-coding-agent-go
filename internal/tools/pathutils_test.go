// ABOUTME: Tests for path normalization: Unicode spaces, NFD, curly quotes, tilde
// ABOUTME: Uses temp directories to verify ResolveReadPath variant matching

package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestNormalizeSpaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no-break space", "hello\u00A0world", "hello world"},
		{"narrow no-break space", "hello\u202Fworld", "hello world"},
		{"ideographic space", "hello\u3000world", "hello world"},
		{"en space", "hello\u2002world", "hello world"},
		{"em space", "hello\u2003world", "hello world"},
		{"thin space", "hello\u2009world", "hello world"},
		{"hair space", "hello\u200Aworld", "hello world"},
		{"medium mathematical space", "hello\u205Fworld", "hello world"},
		{"mixed unicode spaces", "a\u00A0b\u202Fc\u3000d", "a b c d"},
		{"ascii unchanged", "hello world", "hello world"},
		{"empty string", "", ""},
		{"only spaces", "\u00A0\u202F\u3000", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeSpaces(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSpaces(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"tilde expansion", "~/Documents", filepath.Join(home, "Documents")},
		{"at prefix strip", "@/some/file.txt", "/some/file.txt"},
		{"at prefix with tilde", "@~/foo", filepath.Join(home, "foo")},
		{"no-op absolute", "/usr/local/bin", "/usr/local/bin"},
		{"relative unchanged", "foo/bar", "foo/bar"},
		{"unicode space normalized", "foo\u00A0bar", "foo bar"},
		{"combined at + tilde + unicode", "@~/hello\u202Fworld", filepath.Join(home, "hello world")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExpandPath(tt.path)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q; want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveToCwd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		cwd  string
		want string
	}{
		{"relative path", "foo/bar.txt", "/home/user/project", "/home/user/project/foo/bar.txt"},
		{"absolute path", "/etc/config", "/home/user", "/etc/config"},
		{"already clean", "/usr/local/../bin", "/tmp", "/usr/bin"},
		{"dot relative", "./test.go", "/work", "/work/test.go"},
		{"tilde in path", "~/docs", "/work", ""}, // computed below
	}

	// Compute expected value for tilde test.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}
	tests[4].want = filepath.Clean(filepath.Join(home, "docs"))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveToCwd(tt.path, tt.cwd)
			if got != tt.want {
				t.Errorf("ResolveToCwd(%q, %q) = %q; want %q", tt.path, tt.cwd, got, tt.want)
			}
		})
	}
}

func TestResolveReadPath(t *testing.T) {
	t.Parallel()

	// Helper to create a file in a temp dir and return (dir, filename).
	createFile := func(t *testing.T, dir, name string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("ok"), 0o644); err != nil {
			t.Fatalf("create %q: %v", p, err)
		}
		return p
	}

	t.Run("direct match", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		want := createFile(t, dir, "plain.txt")

		got := ResolveReadPath("plain.txt", dir)
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("narrow no-break space fallback", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// File on disk has regular space.
		want := createFile(t, dir, "10 AM.txt")

		// Query with narrow no-break space (U+202F) between 10 and AM.
		got := ResolveReadPath("10\u202FAM.txt", dir)
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("curly quote fallback", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// File on disk has straight apostrophe.
		want := createFile(t, dir, "it's.txt")

		// Query with right single quotation mark (U+2019).
		got := ResolveReadPath("it\u2019s.txt", dir)
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("NFD normalization fallback", func(t *testing.T) {
		// macOS APFS is Unicode-normalizing: both NFC and NFD stat calls
		// succeed for the same on-disk file. We verify that the function
		// finds the file regardless of normalization form.
		if runtime.GOOS != "darwin" {
			t.Skip("NFD filesystem test only reliable on macOS")
		}
		t.Parallel()
		dir := t.TempDir()

		// Create file with NFD name (decomposed e-acute: e + combining acute).
		nfdName := norm.NFD.String("café.txt")
		createFile(t, dir, nfdName)

		// Query with NFC name (precomposed e-acute). The function must
		// find the file; on APFS the direct variant already matches.
		nfcName := norm.NFC.String("café.txt")
		got := ResolveReadPath(nfcName, dir)
		if _, err := os.Stat(got); err != nil {
			t.Errorf("ResolveReadPath returned non-existent path %q: %v", got, err)
		}
	})

	t.Run("NFD plus curly quote fallback", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("NFD filesystem test only reliable on macOS")
		}
		t.Parallel()
		dir := t.TempDir()

		// File on disk: NFD + straight apostrophe.
		nfdName := norm.NFD.String("café") + "'s.txt"
		createFile(t, dir, nfdName)

		// Query: NFC + curly apostrophe. The function must find the file
		// via one of the fallback variants.
		query := norm.NFC.String("café") + "\u2019s.txt"
		got := ResolveReadPath(query, dir)
		if _, err := os.Stat(got); err != nil {
			t.Errorf("ResolveReadPath returned non-existent path %q: %v", got, err)
		}
	})

	t.Run("no match returns direct resolution", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		got := ResolveReadPath("nonexistent.txt", dir)
		want := filepath.Join(dir, "nonexistent.txt")
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})
}
