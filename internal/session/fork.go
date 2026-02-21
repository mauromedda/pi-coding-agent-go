// ABOUTME: Session forking and PR linking for branching conversations
// ABOUTME: Copies session JSONL to new ID; tracks PR-to-session mappings atomically

package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// ForkResult holds the result of forking a session.
type ForkResult struct {
	OriginalID string
	NewID      string
	Branch     string // git branch associated with fork, if any
}

// Fork duplicates a session's messages file under a new random ID.
// The original session is left unchanged.
func Fork(sessionDir, sessionID string) (*ForkResult, error) {
	srcPath := filepath.Join(sessionDir, sessionID+".jsonl")

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("reading session %s: %w", sessionID, err)
	}

	newID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generating session ID: %w", err)
	}

	dstPath := filepath.Join(sessionDir, newID+".jsonl")
	if err := os.WriteFile(dstPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("writing forked session: %w", err)
	}

	return &ForkResult{
		OriginalID: sessionID,
		NewID:      newID,
	}, nil
}

// LinkPR associates a PR number with a session ID.
// Creates or updates pr_links.json in sessionDir atomically.
func LinkPR(sessionDir string, prNumber int, sessionID string) error {
	links, err := readPRLinks(sessionDir)
	if err != nil {
		return err
	}

	links[strconv.Itoa(prNumber)] = sessionID

	return writePRLinks(sessionDir, links)
}

// GetPRSession returns the session ID linked to the given PR number.
// Returns an error if the PR is not linked to any session.
func GetPRSession(sessionDir string, prNumber int) (string, error) {
	links, err := readPRLinks(sessionDir)
	if err != nil {
		return "", err
	}

	sid, ok := links[strconv.Itoa(prNumber)]
	if !ok {
		return "", fmt.Errorf("no session linked to PR #%d", prNumber)
	}
	return sid, nil
}

// ListPRLinks returns all PR-to-session mappings.
// The returned map uses PR numbers as keys.
func ListPRLinks(sessionDir string) (map[int]string, error) {
	raw, err := readPRLinks(sessionDir)
	if err != nil {
		return nil, err
	}

	result := make(map[int]string, len(raw))
	for k, v := range raw {
		n, err := strconv.Atoi(k)
		if err != nil {
			continue // skip malformed keys
		}
		result[n] = v
	}
	return result, nil
}

// generateSessionID creates a 16-byte cryptographically random hex string.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

const prLinksFile = "pr_links.json"

// readPRLinks reads the PR links file, returning an empty map if it doesn't exist.
func readPRLinks(sessionDir string) (map[string]string, error) {
	path := filepath.Join(sessionDir, prLinksFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("reading PR links: %w", err)
	}

	var links map[string]string
	if err := json.Unmarshal(data, &links); err != nil {
		return nil, fmt.Errorf("parsing PR links: %w", err)
	}
	return links, nil
}

// writePRLinks writes PR links atomically using write-rename.
func writePRLinks(sessionDir string, links map[string]string) error {
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling PR links: %w", err)
	}

	path := filepath.Join(sessionDir, prLinksFile)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing temp PR links: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming PR links: %w", err)
	}
	return nil
}
