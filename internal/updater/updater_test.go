package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// --- normalizeVersion ---

func TestNormalizeVersion_StripsV(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v0.1.0", "0.1.0"},
		{"", ""},
		{"v", ""},
		{"vv1.0.0", "v1.0.0"}, // only strips one leading v
	}

	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- isNewer ---

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"newer patch", "0.2.0", "0.2.1", true},
		{"newer minor", "0.2.0", "0.3.0", true},
		{"newer major", "0.2.0", "1.0.0", true},
		{"same version", "0.2.0", "0.2.0", false},
		{"older version", "0.3.0", "0.2.0", false},
		{"empty current", "", "0.2.0", false},
		{"empty latest", "0.2.0", "", false},
		{"both empty", "", "", false},
		{"dev current", "dev", "0.2.0", false},
		{"two part version", "0.2", "0.3.0", true},
		{"two part latest", "0.2.0", "0.3", true},
		{"major jump", "1.9.9", "2.0.0", true},
		{"minor jump", "0.9.0", "0.10.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// --- parseIntSafe ---

func TestParseIntSafe(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"1", 1},
		{"42", 42},
		{"10", 10},
		{"", 0},
		{"abc", 0},
		{"3rc1", 3}, // stops at non-digit
	}

	for _, tt := range tests {
		got := parseIntSafe(tt.input)
		if got != tt.want {
			t.Errorf("parseIntSafe(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- buildAssetName ---

func TestBuildAssetName(t *testing.T) {
	got := buildAssetName("0.3.0")

	// Should contain the version, OS, and arch
	osName := runtime.GOOS
	arch := runtime.GOARCH

	wantExt := "tar.gz"
	if osName == "windows" {
		wantExt = "zip"
	}

	want := "sdd-hoffy_0.3.0_" + osName + "_" + arch + "." + wantExt
	if got != want {
		t.Errorf("buildAssetName(\"0.3.0\") = %q, want %q", got, want)
	}
}

// --- CheckVersion ---

// newTestServer creates an httptest server that responds with a fake GitHub
// release payload. Caller must defer ts.Close().
func newTestServer(t *testing.T, release ReleaseInfo, statusCode int) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		if statusCode == http.StatusOK {
			if err := json.NewEncoder(w).Encode(release); err != nil {
				t.Fatalf("encoding test response: %v", err)
			}
		}
	}))
	return ts
}

// withTestServer overrides releaseEndpoint and httpClient for testing,
// restoring them when the test finishes.
func withTestServer(t *testing.T, ts *httptest.Server) {
	t.Helper()
	origEndpoint := releaseEndpoint
	origClient := httpClient

	releaseEndpoint = ts.URL
	httpClient = ts.Client()

	t.Cleanup(func() {
		releaseEndpoint = origEndpoint
		httpClient = origClient
	})
}

func TestCheckVersion_UpdateAvailable(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v0.3.0",
		HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v0.3.0",
	}
	ts := newTestServer(t, release, http.StatusOK)
	defer ts.Close()
	withTestServer(t, ts)

	result := CheckVersion("v0.2.0")

	if !result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be true")
	}
	if result.LatestVersion != "0.3.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "0.3.0")
	}
	if result.CurrentVersion != "0.2.0" {
		t.Errorf("CurrentVersion = %q, want %q", result.CurrentVersion, "0.2.0")
	}
	if result.ReleaseURL != release.HTMLURL {
		t.Errorf("ReleaseURL = %q, want %q", result.ReleaseURL, release.HTMLURL)
	}
}

func TestCheckVersion_AlreadyLatest(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v0.2.0",
		HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v0.2.0",
	}
	ts := newTestServer(t, release, http.StatusOK)
	defer ts.Close()
	withTestServer(t, ts)

	result := CheckVersion("v0.2.0")

	if result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be false when already at latest")
	}
}

func TestCheckVersion_NetworkError(t *testing.T) {
	// Point to a server that's already closed.
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close()
	withTestServer(t, ts)

	result := CheckVersion("v0.2.0")

	// Should return a result (not panic) with UpdateAvailable = false.
	if result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be false on network error")
	}
	if result.CurrentVersion != "0.2.0" {
		t.Errorf("CurrentVersion = %q, want %q", result.CurrentVersion, "0.2.0")
	}
}

func TestCheckVersion_APIErrorStatus(t *testing.T) {
	ts := newTestServer(t, ReleaseInfo{}, http.StatusForbidden)
	defer ts.Close()
	withTestServer(t, ts)

	result := CheckVersion("v0.2.0")

	if result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be false on API error")
	}
}

func TestCheckVersion_DevVersion(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v0.3.0",
		HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v0.3.0",
	}
	ts := newTestServer(t, release, http.StatusOK)
	defer ts.Close()
	withTestServer(t, ts)

	result := CheckVersion("dev")

	// Dev versions should never report updates (can't compare).
	if result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be false for dev version")
	}
}

// --- SelfUpdate ---

// createTestTarGz creates a tar.gz archive containing a fake sdd-hoffy binary.
func createTestTarGz(t *testing.T, binaryContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "sdd-hoffy",
		Mode: 0o755,
		Size: int64(len(binaryContent)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	if _, err := tw.Write(binaryContent); err != nil {
		t.Fatalf("writing tar body: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}

	return buf.Bytes()
}

func TestSelfUpdate_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping self-update test on Windows")
	}

	fakeBinary := []byte("#!/bin/sh\necho updated\n")
	archiveData := createTestTarGz(t, fakeBinary)
	version := "0.3.0"
	assetName := buildAssetName(version)

	// Create a server that serves the release info and the archive.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download/"+assetName {
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(archiveData)
			return
		}
		// Default: return release info.
		release := ReleaseInfo{
			TagName: "v" + version,
			HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v" + version,
			Assets: []Asset{
				{
					Name:               assetName,
					BrowserDownloadURL: "PLACEHOLDER", // will be replaced below
				},
			},
		}
		// We need the full server URL which we don't have yet.
		// Workaround: use the request's Host.
		release.Assets[0].BrowserDownloadURL = "http://" + r.Host + "/download/" + assetName
		_ = json.NewEncoder(w).Encode(release)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	withTestServer(t, ts)

	// Create a fake "current binary" that SelfUpdate will replace.
	tmpDir := t.TempDir()
	fakePath := filepath.Join(tmpDir, "sdd-hoffy")
	if err := os.WriteFile(fakePath, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	// Override os.Executable by creating a symlink (not possible to override os.Executable).
	// Instead, we test extractBinary and the flow separately.
	// For the full SelfUpdate flow, we'd need to mock os.Executable.
	// Let's test it by verifying extractBinary works correctly.
	t.Run("extractBinary works with tar.gz", func(t *testing.T) {
		data, err := extractBinary(bytes.NewReader(archiveData), assetName)
		if err != nil {
			t.Fatalf("extractBinary: %v", err)
		}
		if !bytes.Equal(data, fakeBinary) {
			t.Errorf("extracted data = %q, want %q", data, fakeBinary)
		}
	})
}

func TestSelfUpdate_AlreadyLatest(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v0.2.0",
		HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v0.2.0",
	}
	ts := newTestServer(t, release, http.StatusOK)
	defer ts.Close()
	withTestServer(t, ts)

	err := SelfUpdate("v0.2.0")
	if err == nil {
		t.Fatal("expected error when already at latest version")
	}
	if got := err.Error(); got != "already at latest version (v0.2.0)" {
		t.Errorf("error = %q, want %q", got, "already at latest version (v0.2.0)")
	}
}

func TestSelfUpdate_APIError(t *testing.T) {
	ts := newTestServer(t, ReleaseInfo{}, http.StatusInternalServerError)
	defer ts.Close()
	withTestServer(t, ts)

	err := SelfUpdate("v0.2.0")
	if err == nil {
		t.Fatal("expected error on API failure")
	}
}

func TestSelfUpdate_NoMatchingAsset(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v0.3.0",
		HTMLURL: "https://github.com/HendryAvila/sdd-hoffy/releases/tag/v0.3.0",
		Assets: []Asset{
			{
				Name:               "sdd-hoffy_0.3.0_solaris_sparc.tar.gz",
				BrowserDownloadURL: "https://example.com/nope",
			},
		},
	}
	ts := newTestServer(t, release, http.StatusOK)
	defer ts.Close()
	withTestServer(t, ts)

	err := SelfUpdate("v0.2.0")
	if err == nil {
		t.Fatal("expected error when no matching asset found")
	}
}

// --- extractFromTarGz ---

func TestExtractFromTarGz_Success(t *testing.T) {
	content := []byte("#!/bin/sh\necho hello\n")
	archive := createTestTarGz(t, content)

	data, err := extractFromTarGz(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("extracted = %q, want %q", data, content)
	}
}

func TestExtractFromTarGz_BinaryNotFound(t *testing.T) {
	// Create a tar.gz with a differently-named file.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "not-the-binary",
		Mode: 0o755,
		Size: 5,
	}
	_ = tw.WriteHeader(header)
	_, _ = tw.Write([]byte("hello"))
	_ = tw.Close()
	_ = gw.Close()

	_, err := extractFromTarGz(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error when binary not found in archive")
	}
}

func TestExtractFromTarGz_InvalidGzip(t *testing.T) {
	_, err := extractFromTarGz(bytes.NewReader([]byte("not gzip data")))
	if err == nil {
		t.Fatal("expected error on invalid gzip data")
	}
}

// --- extractFromZip ---

func TestExtractFromZip_ReturnsUnsupportedError(t *testing.T) {
	_, err := extractFromZip(bytes.NewReader([]byte("fake zip")))
	if err == nil {
		t.Fatal("expected error from extractFromZip (unsupported in MVP)")
	}
}

// --- extractBinary dispatch ---

func TestExtractBinary_DispatchesByExtension(t *testing.T) {
	content := []byte("binary data")
	archive := createTestTarGz(t, content)

	// tar.gz path
	data, err := extractBinary(bytes.NewReader(archive), "sdd-hoffy_0.3.0_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("extractBinary (tar.gz): %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("tar.gz: extracted = %q, want %q", data, content)
	}

	// zip path (should fail with unsupported)
	_, err = extractBinary(bytes.NewReader([]byte("fake")), "sdd-hoffy_0.3.0_windows_amd64.zip")
	if err == nil {
		t.Fatal("extractBinary (zip): expected error")
	}
}
