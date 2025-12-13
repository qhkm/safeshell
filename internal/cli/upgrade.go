package cli

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

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// httpClient with timeout to prevent hanging on slow/unresponsive servers
var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade SafeShell to the latest version",
	Long: `Downloads and installs the latest version of SafeShell.

Your checkpoints and settings are preserved.

Examples:
  safeshell upgrade          # Upgrade to latest version
  safeshell upgrade --check  # Just check for updates`,
	RunE: runUpgrade,
}

var upgradeCheckOnly bool

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for updates, don't install")
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	currentVersion := version

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	// Get latest version from GitHub
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Printf("Latest version:  %s\n", latestVersion)
	fmt.Println()

	// Compare versions (normalize by removing 'v' prefix)
	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(latestVersion, "v")

	if current == latest {
		color.Green("✓ Already up to date!")
		return nil
	}

	if upgradeCheckOnly {
		color.Yellow("Update available: %s → %s", currentVersion, latestVersion)
		fmt.Println("\nRun 'safeshell upgrade' to install.")
		return nil
	}

	// Proceed with upgrade
	fmt.Printf("Upgrading %s → %s\n\n", currentVersion, latestVersion)

	// Detect platform
	platform := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Platform: %s\n", platform)

	// Download URL
	downloadURL := fmt.Sprintf(
		"https://github.com/qhkm/safeshell/releases/download/%s/safeshell_%s.tar.gz",
		latestVersion, platform,
	)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	fmt.Printf("Downloading from GitHub...\n")

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "safeshell-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d (release may not exist for %s)", resp.StatusCode, platform)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract binary from tar.gz
	fmt.Println("Extracting...")

	tmpFile.Seek(0, 0)
	newBinary, err := extractBinaryFromTarGz(tmpFile)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	defer os.Remove(newBinary)

	// Replace current executable
	fmt.Println("Installing...")

	// Rename old binary as backup
	backupPath := execPath + ".old"
	os.Remove(backupPath) // Remove any existing backup
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := copyFile(newBinary, execPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	fmt.Println()
	color.Green("✓ Upgraded to %s", latestVersion)
	fmt.Println("\nYour checkpoints and settings are preserved.")

	return nil
}

func getLatestVersion() (string, error) {
	resp, err := httpClient.Get("https://api.github.com/repos/qhkm/safeshell/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func extractBinaryFromTarGz(r io.Reader) (string, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Security: reject any path traversal attempts
		if strings.Contains(header.Name, "..") {
			return "", fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		// Look for the safeshell binary - must be exactly "safeshell" or in a single subdirectory
		baseName := filepath.Base(header.Name)
		if header.Typeflag == tar.TypeReg && baseName == "safeshell" {
			tmpFile, err := os.CreateTemp("", "safeshell-bin-*")
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(tmpFile, tr); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				return "", err
			}

			tmpFile.Close()
			return tmpFile.Name(), nil
		}
	}

	return "", fmt.Errorf("safeshell binary not found in archive")
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}
