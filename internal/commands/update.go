package commands

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

	"github.com/spf13/cobra"
)

const (
	repoOwner   = "dea-exmachina"
	repoName    = "dea-cli"
	releasesAPI = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type githubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// newUpdateCommand returns the `dea update` cobra command.
func newUpdateCommand(currentVersion, currentCommit, currentDate string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update dea to the latest version",
		Long:  "Checks GitHub Releases for a newer version and replaces the running binary.",
		RunE:  runUpdate(currentVersion),
	}
}

func runUpdate(currentVersion string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", currentVersion)
		fmt.Println("Checking for updates...")

		// 1. GET latest release from GitHub API.
		release, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to fetch latest release: %w", err)
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentClean := strings.TrimPrefix(currentVersion, "v")

		// 2. Compare versions.
		if latestVersion == currentClean || currentClean == "dev" && latestVersion == "" {
			fmt.Printf("Already at latest version: %s\n", currentVersion)
			return nil
		}
		if latestVersion == currentClean {
			fmt.Printf("Already at latest version: %s\n", currentVersion)
			return nil
		}

		fmt.Printf("New version available: %s -> %s\n", currentVersion, release.TagName)

		// 3. Find asset for current GOOS/GOARCH.
		assetName := buildAssetName(release.TagName)
		checksumAssetName := "checksums.txt"

		assetURL := findAssetURL(release.Assets, assetName)
		if assetURL == "" {
			return fmt.Errorf("no asset found for %s/%s (looking for %s)", runtime.GOOS, runtime.GOARCH, assetName)
		}

		checksumURL := findAssetURL(release.Assets, checksumAssetName)
		if checksumURL == "" {
			return fmt.Errorf("no checksums.txt found in release")
		}

		// 4. Download the asset.
		fmt.Printf("Downloading %s...\n", assetName)
		assetData, err := downloadBytes(assetURL)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		// 5. Verify SHA256 against checksums.txt.
		fmt.Println("Verifying checksum...")
		checksumData, err := downloadBytes(checksumURL)
		if err != nil {
			return fmt.Errorf("failed to download checksums: %w", err)
		}
		if err := verifyChecksum(assetData, checksumData, assetName); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("Checksum OK.")

		// 6. Extract binary from archive.
		binaryData, err := extractBinary(assetData, assetName)
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}

		// 7. Write to temp file alongside current executable.
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("could not resolve symlinks: %w", err)
		}

		tmpPath := execPath + ".new"
		if err := os.WriteFile(tmpPath, binaryData, 0755); err != nil {
			return fmt.Errorf("failed to write new binary: %w", err)
		}

		// 8. Atomic replace (rename is atomic on the same filesystem).
		if err := os.Rename(tmpPath, execPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to replace binary: %w", err)
		}

		fmt.Printf("Updated to %s. Run `dea --version` to confirm.\n", release.TagName)
		return nil
	}
}

// fetchLatestRelease calls the GitHub API for the latest release.
func fetchLatestRelease() (*githubRelease, error) {
	resp, err := http.Get(releasesAPI) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// buildAssetName constructs the expected GoReleaser archive filename.
// Pattern: dea_<version>_<os>_<arch>.<ext>
func buildAssetName(tag string) string {
	ver := strings.TrimPrefix(tag, "v")
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("dea_%s_%s_%s.%s", ver, goos, goarch, ext)
}

// findAssetURL searches release assets for a matching name.
func findAssetURL(assets []releaseAsset, name string) string {
	for _, a := range assets {
		if a.Name == name {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// downloadBytes performs a GET and returns the full response body.
func downloadBytes(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum checks the SHA256 of data against the checksums file.
func verifyChecksum(data, checksumFile []byte, assetName string) error {
	h := sha256.Sum256(data)
	actualHash := hex.EncodeToString(h[:])

	// Parse checksums.txt (format: "<hash>  <filename>")
	for _, line := range strings.Split(string(checksumFile), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			if parts[0] != actualHash {
				return fmt.Errorf("expected %s, got %s", parts[0], actualHash)
			}
			return nil
		}
	}
	return fmt.Errorf("no checksum entry found for %s", assetName)
}

// extractBinary extracts the `dea` or `dea.exe` binary from a tar.gz or zip archive.
func extractBinary(archiveData []byte, assetName string) ([]byte, error) {
	binaryName := "dea"
	if runtime.GOOS == "windows" {
		binaryName = "dea.exe"
	}

	if strings.HasSuffix(assetName, ".tar.gz") {
		return extractFromTarGz(archiveData, binaryName)
	}
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archiveData, binaryName)
	}
	return nil, fmt.Errorf("unsupported archive format: %s", assetName)
}

// extractFromTarGz extracts a named file from a tar.gz archive.
func extractFromTarGz(data []byte, name string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		// Match on the base name to handle directories in the archive.
		if filepath.Base(hdr.Name) == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}

// extractFromZip extracts a named file from a zip archive.
func extractFromZip(data []byte, name string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) == name {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open zip entry: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}
