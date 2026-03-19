package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
"github.com/kettleofketchup/AuthentikOperator/src/AuthentikOperator/version"
)

// GitHubRelease represents the structure of a GitHub release API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// GitLabRelease represents the structure of a GitLab release API response
type GitLabRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  struct {
		Links []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"links"`
	} `json:"assets"`
}

// ReleaseInfo contains release information from either platform
type ReleaseInfo struct {
	TagName     string
	DownloadURL string
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update AuthentikOperator to the latest version",
	Long: `Download and install the latest version of AuthentikOperator from releases.

This command automatically detects the release source from git remote URL
and downloads the appropriate binary for your platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate() error {
	fmt.Println("Checking for latest version...")

	// Detect release source from git remote
	platform, repoPath, err := detectReleaseSource()
	if err != nil {
		return fmt.Errorf("failed to detect release source: %w", err)
	}

	fmt.Printf("Release source: %s (%s)\n", platform, repoPath)

	// Get latest release info
	release, err := getLatestRelease(platform, repoPath)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(version.Version, "v")

	fmt.Printf("Current version: v%s\n", currentVersion)
	fmt.Printf("Latest version:  v%s\n", latestVersion)

	// Compare versions
	if compareVersions(currentVersion, latestVersion) >= 0 {
		fmt.Println("You are already running the latest version!")
		return nil
	}

	fmt.Println("A newer version is available. Starting update...")

	// Download and install
	if err := downloadAndInstall(release.DownloadURL); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated from v%s to v%s!\n", currentVersion, latestVersion)
	fmt.Println("You may need to restart your terminal for changes to take effect.")

	return nil
}

// detectReleaseSource detects the git hosting platform from remote URL
func detectReleaseSource() (platform string, repoPath string, err error) {
	// Try to get git remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to configured platform

		return "github", "kettleofketchup/AuthentikOperator", nil

	}

	remoteURL := strings.TrimSpace(string(output))
	return parseRemoteURL(remoteURL)
}

// parseRemoteURL extracts platform and repo path from git remote URL
func parseRemoteURL(remoteURL string) (platform string, repoPath string, err error) {
	// Handle SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); matches != nil {
		host := matches[1]
		path := matches[2]
		return detectPlatformFromHost(host), path, nil
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	httpsPattern := regexp.MustCompile(`^https?://([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(remoteURL); matches != nil {
		host := matches[1]
		path := matches[2]
		return detectPlatformFromHost(host), path, nil
	}

	return "", "", fmt.Errorf("unable to parse remote URL: %s", remoteURL)
}

func detectPlatformFromHost(host string) string {
	if strings.Contains(host, "github.com") {
		return "github"
	}
	// Assume GitLab for anything else (gitlab.com, self-hosted gitlab, etc.)
	return "gitlab"
}

// getLatestRelease fetches the latest release from GitHub or GitLab
func getLatestRelease(platform, repoPath string) (*ReleaseInfo, error) {
	switch platform {
	case "github":
		return getGitHubRelease(repoPath)
	case "gitlab":
		return getGitLabRelease(repoPath)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func getGitHubRelease(repoPath string) (*ReleaseInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoPath)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	// Find asset for current OS/arch
	assetName := getBinaryName()
	for _, asset := range release.Assets {
		if asset.Name == assetName || asset.Name == "AuthentikOperator" {
			return &ReleaseInfo{
				TagName:     release.TagName,
				DownloadURL: asset.BrowserDownloadURL,
			}, nil
		}
	}

	// Fallback to generic binary name
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", repoPath, assetName)
	return &ReleaseInfo{
		TagName:     release.TagName,
		DownloadURL: downloadURL,
	}, nil
}

func getGitLabRelease(repoPath string) (*ReleaseInfo, error) {
	// URL encode the project path
	encodedPath := strings.ReplaceAll(repoPath, "/", "%2F")

	// Try GitLab.com first, then fall back to self-hosted
	hosts := []string{
		"https://gitlab.com",

	}

	var lastErr error
	for _, host := range hosts {
		apiURL := fmt.Sprintf("%s/api/v4/projects/%s/releases/permalink/latest", host, encodedPath)

		resp, err := http.Get(apiURL)
		if err != nil {
			lastErr = err
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("GitLab API returned status: %s", resp.Status)
			continue
		}

		var release GitLabRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			lastErr = fmt.Errorf("failed to parse release: %w", err)
			continue
		}

		// Find asset for current OS/arch
		assetName := getBinaryName()
		for _, link := range release.Assets.Links {
			if link.Name == assetName || link.Name == "AuthentikOperator" {
				return &ReleaseInfo{
					TagName:     release.TagName,
					DownloadURL: link.URL,
				}, nil
			}
		}

		// Fallback to generic binary
		return &ReleaseInfo{
			TagName:     release.TagName,
			DownloadURL: fmt.Sprintf("%s/%s/-/releases/%s/downloads/%s", host, repoPath, release.TagName, assetName),
		}, nil
	}

	return nil, fmt.Errorf("failed to get GitLab release: %w", lastErr)
}

func getBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	name := fmt.Sprintf("%s_%s_%s", "AuthentikOperator", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}

	return name
}

func downloadAndInstall(downloadURL string) error {
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create temp file
	tmpFile := filepath.Join(os.TempDir(), "AuthentikOperator-update")

	fmt.Printf("Downloading from %s...\n", downloadURL)

	// Download
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	if closeErr := out.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Replace current binary
	fmt.Println("Installing update...")
	if err := atomicReplace(currentExe, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

// atomicReplace replaces target with source atomically
func atomicReplace(target, source string) error {
	switch runtime.GOOS {
	case "windows":
		return windowsReplace(target, source)
	default:
		return os.Rename(source, target)
	}
}

// windowsReplace handles binary replacement on Windows
func windowsReplace(target, source string) error {
	batchScript := target + ".update.bat"
	batchContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move "%s" "%s"
del "%%~f0"
`, source, target)

	if err := os.WriteFile(batchScript, []byte(batchContent), 0755); err != nil {
		return fmt.Errorf("failed to create update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "/B", batchScript)
	if err := cmd.Start(); err != nil {
		return err
	}

	fmt.Println("Update script started. Binary will be replaced after exit.")
	return nil
}

// compareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Handle dev versions
	if v1 == "dev" && v2 != "dev" {
		return -1
	}
	if v1 != "dev" && v2 == "dev" {
		return 1
	}
	if v1 == "dev" && v2 == "dev" {
		return 0
	}

	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	for i := 0; i < 3; i++ {
		if parts1[i] < parts2[i] {
			return -1
		}
		if parts1[i] > parts2[i] {
			return 1
		}
	}

	return 0
}

// parseVersion parses a version string into [major, minor, patch]
func parseVersion(ver string) [3]int {
	parts := [3]int{0, 0, 0}
	versionParts := strings.Split(ver, ".")

	for i, part := range versionParts {
		if i >= 3 {
			break
		}
		val := 0
		for _, char := range part {
			if char >= '0' && char <= '9' {
				val = val*10 + int(char-'0')
			} else {
				break
			}
		}
		parts[i] = val
	}

	return parts
}
