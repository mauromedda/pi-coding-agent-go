// ABOUTME: Self-update command that downloads the latest release from GitHub
// ABOUTME: Verifies SHA256 checksum and performs atomic binary replacement

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubOwner = "mauromedda"
	githubRepo  = "pi-coding-agent-go"
	releasesURL = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
)

// githubRelease represents the relevant fields from the GitHub Releases API.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset represents a single release asset.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// TODO(security): add GPG signature verification for release assets.
// The checksum alone proves integrity but not authenticity; verifying a
// detached GPG signature (signed by a known release key) would close that gap.

// runSelfUpdate checks for a newer version and replaces the current binary.
func runSelfUpdate(currentVersion string) error {
	fmt.Println("checking for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetching latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if latestVersion == currentClean {
		fmt.Printf("already at latest version %s\n", currentVersion)
		return nil
	}

	fmt.Printf("updating %s -> %s\n", currentVersion, latestVersion)

	binaryName := buildAssetName()
	checksumName := binaryName + ".sha256"

	binaryURL, err := findAssetURL(release, binaryName)
	if err != nil {
		return fmt.Errorf("finding binary asset: %w", err)
	}

	checksumURL, err := findAssetURL(release, checksumName)
	if err != nil {
		return fmt.Errorf("finding checksum asset: %w", err)
	}

	expectedHash, err := fetchChecksum(checksumURL)
	if err != nil {
		return fmt.Errorf("fetching checksum: %w", err)
	}

	tmpPath, err := downloadBinary(binaryURL)
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer os.Remove(tmpPath) // Clean up on failure; on success the file is already renamed

	if err := verifyChecksum(tmpPath, expectedHash); err != nil {
		return fmt.Errorf("checksum verification: %w", err)
	}

	if err := replaceBinary(tmpPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("updated to %s\n", latestVersion)
	return nil
}

// fetchLatestRelease queries the GitHub Releases API for the latest release.
func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, releasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "pi-go-self-update")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, body)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}

	return &release, nil
}

// buildAssetName constructs the expected binary asset name for the current platform.
func buildAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("pi-go_%s_%s%s", goos, goarch, ext)
}

// findAssetURL searches the release assets for a matching name.
func findAssetURL(release *githubRelease, name string) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %q not found in release %s", name, release.TagName)
}

// fetchChecksum downloads and parses the SHA256 checksum file.
// Expected format: "<hex-hash>  <filename>" or just "<hex-hash>".
func fetchChecksum(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from GitHub API
	if err != nil {
		return "", fmt.Errorf("downloading checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading checksum body: %w", err)
	}

	line := strings.TrimSpace(string(data))
	// Handle "hash  filename" format
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}

	return parts[0], nil
}

// downloadBinary downloads the release binary to a temporary file.
func downloadBinary(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is constructed from GitHub API
	if err != nil {
		return "", fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("binary download returned status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "pi-go-update-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing binary: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// verifyChecksum computes the SHA256 hash of the file and compares it to the expected value.
func verifyChecksum(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}

	actualHex := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualHex, expectedHex) {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHex, actualHex)
	}

	return nil
}

// replaceBinary performs an atomic rename dance to replace the running binary.
// Steps: (1) find current executable, (2) rename current to .old, (3) rename new to current, (4) remove .old.
func replaceBinary(tmpPath string) error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}

	// Resolve symlinks to get the real path
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Make the new binary executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("setting executable permissions: %w", err)
	}

	oldPath := currentPath + ".old"

	// Rename current binary out of the way
	if err := os.Rename(currentPath, oldPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, currentPath); err != nil {
		// Attempt to restore the old binary
		_ = os.Rename(oldPath, currentPath)
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Best-effort cleanup of the old binary
	_ = os.Remove(oldPath)

	return nil
}
