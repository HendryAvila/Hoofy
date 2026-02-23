// Package updater checks for new versions on GitHub and can self-update
// the binary in place. It uses the GitHub Releases API (no auth required
// for public repos) and replaces the running binary atomically.
//
// Design decisions:
//   - Zero external dependencies (only net/http + encoding/json)
//   - Atomic replace: download to temp file, then rename over current binary
//   - Non-blocking: CheckVersion runs in a goroutine during "serve"
//   - No auto-restart: user must restart the MCP server after update
package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// githubRepo is the repository path for API calls.
	githubRepo = "HendryAvila/Hoofy"

	// releaseURL is the GitHub API endpoint for the latest release.
	releaseURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"

	// checkTimeout is how long we wait for the GitHub API.
	checkTimeout = 10 * time.Second
)

// For testing: allow overriding the release URL and HTTP client.
var (
	releaseEndpoint = releaseURL
	httpClient      = &http.Client{Timeout: checkTimeout}
)

// ReleaseInfo holds the relevant fields from a GitHub release.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a downloadable file in a GitHub release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// UpdateResult is returned by CheckVersion to communicate the outcome.
type UpdateResult struct {
	// CurrentVersion is the running version (e.g. "0.2.0").
	CurrentVersion string
	// LatestVersion is the newest release (e.g. "0.3.0").
	LatestVersion string
	// UpdateAvailable is true when latest > current.
	UpdateAvailable bool
	// ReleaseURL is the GitHub page for the release.
	ReleaseURL string
}

// CheckVersion queries GitHub for the latest release and compares it
// against the current version. It never returns an error to the caller —
// network failures are silently ignored (this is a best-effort check).
func CheckVersion(currentVersion string) *UpdateResult {
	result := &UpdateResult{
		CurrentVersion: normalizeVersion(currentVersion),
	}

	req, err := http.NewRequest("GET", releaseEndpoint, nil)
	if err != nil {
		return result
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hoofy/"+currentVersion)

	resp, err := httpClient.Do(req)
	if err != nil {
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return result
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return result
	}

	result.LatestVersion = normalizeVersion(release.TagName)
	result.ReleaseURL = release.HTMLURL
	result.UpdateAvailable = isNewer(result.CurrentVersion, result.LatestVersion)

	return result
}

// SelfUpdate downloads the appropriate binary for the current OS/arch
// and replaces the running executable atomically.
func SelfUpdate(currentVersion string) error {
	// 1. Fetch latest release info
	req, err := http.NewRequest("GET", releaseEndpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "hoofy/"+currentVersion)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("parsing release info: %w", err)
	}

	latestVersion := normalizeVersion(release.TagName)
	if !isNewer(normalizeVersion(currentVersion), latestVersion) {
		return fmt.Errorf("already at latest version (%s)", currentVersion)
	}

	// 2. Find the right asset for this OS/arch
	assetName := buildAssetName(latestVersion)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (looking for %s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	// 3. Download the archive
	archiveResp, err := http.Get(downloadURL) //nolint:gosec // URL comes from GitHub API
	if err != nil {
		return fmt.Errorf("downloading release: %w", err)
	}
	defer func() { _ = archiveResp.Body.Close() }()

	if archiveResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", archiveResp.StatusCode)
	}

	// 4. Extract the binary from the archive
	binaryData, err := extractBinary(archiveResp.Body, assetName)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// 5. Atomic replace: write to temp file, then rename
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	tmpPath := execPath + ".new"
	if err := os.WriteFile(tmpPath, binaryData, 0o755); err != nil {
		return fmt.Errorf("writing new binary: %w", err)
	}

	// On Windows, we can't replace a running binary directly.
	// Rename old → .old, new → current.
	if runtime.GOOS == "windows" {
		oldPath := execPath + ".old"
		_ = os.Remove(oldPath) // clean up any previous .old
		if err := os.Rename(execPath, oldPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("backing up current binary: %w", err)
		}
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// extractBinary reads a .tar.gz (or .zip for Windows) archive and returns
// the raw bytes of the hoofy binary inside it.
func extractBinary(reader io.Reader, assetName string) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(reader)
	}
	return extractFromTarGz(reader)
}

// extractFromTarGz pulls the hoofy binary out of a .tar.gz archive.
func extractFromTarGz(reader io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("opening gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		name := filepath.Base(header.Name)
		if name == "hoofy" || name == "hoofy.exe" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("reading binary from tar: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("hoofy binary not found in archive")
}

// extractFromZip reads the entire zip into memory (Windows .zip files are small)
// and extracts the binary. We read to a temp file since zip needs seeking.
func extractFromZip(reader io.Reader) ([]byte, error) {
	// For zip we need to read the whole thing since zip requires random access.
	// The binary is ~10MB, so this is fine.
	tmpFile, err := os.CreateTemp("", "hoofy-*.zip")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		return nil, fmt.Errorf("writing zip to temp: %w", err)
	}

	// Re-open for reading with archive/zip
	// (We'd need archive/zip which requires ReadSeeker)
	// For simplicity in MVP, Windows users can download manually.
	return nil, fmt.Errorf("automatic zip extraction not yet supported on Windows — please download manually from GitHub releases")
}

// buildAssetName constructs the expected archive filename for the
// current OS and architecture, matching GoReleaser's name_template.
func buildAssetName(version string) string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("hoofy_%s_%s_%s.%s", version, osName, arch, ext)
}

// normalizeVersion strips the leading "v" from version strings.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// isNewer returns true if latest is a higher version than current.
// Uses simple string comparison of semver parts.
func isNewer(current, latest string) bool {
	if current == "" || latest == "" || current == "dev" {
		return false
	}

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	// Pad to 3 parts
	for len(currentParts) < 3 {
		currentParts = append(currentParts, "0")
	}
	for len(latestParts) < 3 {
		latestParts = append(latestParts, "0")
	}

	for i := 0; i < 3; i++ {
		c := parseIntSafe(currentParts[i])
		l := parseIntSafe(latestParts[i])
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	return false
}

// parseIntSafe converts a string to int, returning 0 on error.
func parseIntSafe(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		} else {
			break
		}
	}
	return n
}
