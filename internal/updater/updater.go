package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const repo = "viktorfroberg/koll"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckAsync checks for updates in the background and sends the result
// on the returned channel. Never blocks startup — times out after 3 seconds.
func CheckAsync(currentVersion string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		if currentVersion == "dev" {
			return
		}

		client := &http.Client{Timeout: 3 * time.Second}
		url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
		resp, err := client.Get(url)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return
		}
		var release githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return
		}

		latestClean := strings.TrimPrefix(release.TagName, "v")
		currentClean := strings.TrimPrefix(currentVersion, "v")
		if latestClean != currentClean {
			ch <- release.TagName
		}
	}()
	return ch
}

// Update checks for the latest release and replaces the current binary if newer.
func Update(currentVersion string) error {
	latest, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Strip "v" prefix for comparison
	latestClean := strings.TrimPrefix(latest, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if currentClean == latestClean {
		fmt.Printf("koll %s is already the latest version\n", currentVersion)
		return nil
	}

	if currentVersion == "dev" {
		fmt.Println("Running a dev build. To update, reinstall using your original install method.")
		return nil
	}

	fmt.Printf("Updating koll %s -> %s\n", currentVersion, latest)

	// Determine binary URL
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/koll_%s_%s.tar.gz", repo, latest, goos, goarch)

	// Download to temp file
	tmpDir, err := os.MkdirTemp("", "koll-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := tmpDir + "/koll.tar.gz"
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", tmpDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Find current binary location
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate current binary: %w", err)
	}

	// Replace binary
	newBin := tmpDir + "/koll"
	if err := replaceBinary(newBin, self); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("Updated to koll %s\n", latest)
	return nil
}

func getLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func replaceBinary(newPath, oldPath string) error {
	// Get permissions from old binary
	info, err := os.Stat(oldPath)
	if err != nil {
		return err
	}

	// Read new binary
	newBin, err := os.ReadFile(newPath)
	if err != nil {
		return err
	}

	// Write to old path (atomic-ish: write temp next to target, then rename)
	tmpPath := oldPath + ".new"
	if err := os.WriteFile(tmpPath, newBin, info.Mode()); err != nil {
		// May need sudo — tell the user
		return fmt.Errorf("permission denied writing to %s (try: sudo koll --update)", oldPath)
	}

	if err := os.Rename(tmpPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
