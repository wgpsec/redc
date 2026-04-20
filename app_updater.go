package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	redc "red-cloud/mod"
)

// UpdateState represents the current state of the auto-update process.
type UpdateState struct {
	Status       string  `json:"status"`       // "none"|"checking"|"available"|"downloading"|"ready"|"error"
	CurrentVer   string  `json:"currentVer"`
	LatestVer    string  `json:"latestVer"`
	ReleaseNotes string  `json:"releaseNotes"`
	DownloadURL  string  `json:"downloadURL"`
	AssetURL     string  `json:"assetURL"`
	AssetSize    int64   `json:"assetSize"`
	Progress     float64 `json:"progress"`     // 0-100
	Downloaded   int64   `json:"downloaded"`
	Error        string  `json:"error"`
}

// githubRelease represents a GitHub release API response.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset represents a single asset in a GitHub release.
type githubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// platformAssetName returns the expected GitHub Release asset filename for the
// current OS/architecture combination.
func platformAssetName() string {
	switch runtime.GOOS {
	case "darwin":
		return "redc-gui-macos-universal.zip"
	case "windows":
		return "redc-gui-windows-amd64.zip"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "redc-gui-linux-arm64.tar.gz"
		}
		return "redc-gui-linux-amd64.tar.gz"
	default:
		return ""
	}
}

// updatesDir returns ~/redc/updates/ and ensures it exists.
func updatesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "redc", "updates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// stagingDir returns ~/redc/updates/staging/ and ensures it exists.
func stagingDir() (string, error) {
	dir, err := updatesDir()
	if err != nil {
		return "", err
	}
	staging := filepath.Join(dir, "staging")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", err
	}
	return staging, nil
}

// setUpdateError is a helper that sets the update state to error.
func (a *App) setUpdateError(msg string) {
	a.mu.Lock()
	a.updateState.Status = "error"
	a.updateState.Error = msg
	a.mu.Unlock()
}

// CheckForUpdatesOnStartup checks for a new version in the background after startup.
// It is intended to be called as a goroutine from startup().
func (a *App) CheckForUpdatesOnStartup() {
	time.Sleep(3 * time.Second)

	a.mu.Lock()
	a.updateState = UpdateState{
		Status:     "checking",
		CurrentVer: redc.Version,
	}
	a.mu.Unlock()

	client := redc.NewProxyHTTPClient(30 * time.Second)
	resp, err := client.Get("https://api.github.com/repos/wgpsec/redc/releases/latest")
	if err != nil {
		a.mu.Lock()
		a.updateState.Status = "none"
		a.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		a.mu.Lock()
		a.updateState.Status = "none"
		a.mu.Unlock()
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		a.mu.Lock()
		a.updateState.Status = "none"
		a.mu.Unlock()
		return
	}

	tagName := release.TagName
	if tagName == "" {
		a.mu.Lock()
		a.updateState.Status = "none"
		a.mu.Unlock()
		return
	}

	currentVer := strings.TrimPrefix(redc.Version, "v")
	latestVer := strings.TrimPrefix(tagName, "v")

	if compareVersions(currentVer, latestVer) >= 0 {
		// Already up-to-date or newer
		a.mu.Lock()
		a.updateState.Status = "none"
		a.updateState.LatestVer = tagName
		a.mu.Unlock()
		return
	}

	// Find the matching platform asset
	wantAsset := platformAssetName()
	var matchedAsset *githubAsset
	if wantAsset != "" {
		for i := range release.Assets {
			if release.Assets[i].Name == wantAsset {
				matchedAsset = &release.Assets[i]
				break
			}
		}
	}

	a.mu.Lock()
	a.updateState.Status = "available"
	a.updateState.CurrentVer = redc.Version
	a.updateState.LatestVer = tagName
	a.updateState.ReleaseNotes = release.Body
	a.updateState.DownloadURL = release.HTMLURL
	if matchedAsset != nil {
		a.updateState.AssetURL = matchedAsset.BrowserDownloadURL
		a.updateState.AssetSize = matchedAsset.Size
	}
	stateCopy := a.updateState
	a.mu.Unlock()

	a.emitEvent("updateAvailable", stateCopy)
}

// GetUpdateStatus returns the cached update state (mutex-protected).
func (a *App) GetUpdateStatus() UpdateState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updateState
}

// DownloadUpdate downloads the release asset and extracts it to the staging directory.
func (a *App) DownloadUpdate() error {
	a.mu.Lock()
	if a.updateState.Status != "available" {
		a.mu.Unlock()
		return fmt.Errorf("no update available to download (status: %s)", a.updateState.Status)
	}
	assetURL := a.updateState.AssetURL
	assetSize := a.updateState.AssetSize
	if assetURL == "" {
		a.mu.Unlock()
		return fmt.Errorf("no asset URL for this platform")
	}
	a.updateState.Status = "downloading"
	a.updateState.Progress = 0
	a.updateState.Downloaded = 0
	a.mu.Unlock()

	dir, err := updatesDir()
	if err != nil {
		a.setUpdateError(fmt.Sprintf("failed to create updates dir: %v", err))
		return err
	}

	// Clean staging before extraction
	staging, err := stagingDir()
	if err != nil {
		a.setUpdateError(fmt.Sprintf("failed to create staging dir: %v", err))
		return err
	}
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		a.setUpdateError(fmt.Sprintf("failed to recreate staging dir: %v", err))
		return err
	}

	assetName := platformAssetName()
	destPath := filepath.Join(dir, assetName)

	client := redc.NewProxyHTTPClient(10 * time.Minute)
	resp, err := client.Get(assetURL)
	if err != nil {
		a.setUpdateError(fmt.Sprintf("download failed: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("download returned status %d", resp.StatusCode)
		a.setUpdateError(msg)
		return fmt.Errorf("%s", msg)
	}

	totalSize := assetSize
	if totalSize == 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		a.setUpdateError(fmt.Sprintf("failed to create file: %v", err))
		return err
	}

	buf := make([]byte, 32*1024)
	var downloaded int64
	lastEmit := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := outFile.Write(buf[:n]); writeErr != nil {
				outFile.Close()
				a.setUpdateError(fmt.Sprintf("write error: %v", writeErr))
				return writeErr
			}
			downloaded += int64(n)

			if time.Since(lastEmit) >= 200*time.Millisecond {
				var progress float64
				if totalSize > 0 {
					progress = float64(downloaded) / float64(totalSize) * 100
				}
				a.mu.Lock()
				a.updateState.Progress = progress
				a.updateState.Downloaded = downloaded
				stateCopy := a.updateState
				a.mu.Unlock()
				a.emitEvent("updateProgress", stateCopy)
				lastEmit = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			outFile.Close()
			a.setUpdateError(fmt.Sprintf("read error: %v", readErr))
			return readErr
		}
	}
	outFile.Close()

	// Extract
	if strings.HasSuffix(assetName, ".zip") {
		if err := extractZip(destPath, staging); err != nil {
			a.setUpdateError(fmt.Sprintf("extract zip failed: %v", err))
			return err
		}
	} else if strings.HasSuffix(assetName, ".tar.gz") {
		if err := extractTarGz(destPath, staging); err != nil {
			a.setUpdateError(fmt.Sprintf("extract tar.gz failed: %v", err))
			return err
		}
	}

	a.mu.Lock()
	a.updateState.Status = "ready"
	a.updateState.Progress = 100
	a.updateState.Downloaded = downloaded
	stateCopy := a.updateState
	a.mu.Unlock()
	a.emitEvent("updateReady", stateCopy)

	return nil
}

// extractZip extracts a zip archive to dest with zip-slip protection.
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		// Zip-slip protection
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal zip entry path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// extractTarGz extracts a .tar.gz archive to dest with zip-slip protection.
func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		// Zip-slip protection
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal tar entry path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// ApplyUpdateAndRestart replaces the running binary with the downloaded update
// and restarts the application. Behaviour is platform-specific.
func (a *App) ApplyUpdateAndRestart() error {
	a.mu.Lock()
	if a.updateState.Status != "ready" {
		a.mu.Unlock()
		return fmt.Errorf("no update ready to apply (status: %s)", a.updateState.Status)
	}
	a.mu.Unlock()

	staging, err := stagingDir()
	if err != nil {
		return fmt.Errorf("staging dir error: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return a.applyUpdateMacOS(staging)
	case "windows":
		return a.applyUpdateWindows(staging)
	case "linux":
		return a.applyUpdateLinux(staging)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findAppBundle walks up from the executable path to find the .app bundle on macOS.
func findAppBundle(execPath string) (string, error) {
	dir := execPath
	for i := 0; i < 10; i++ {
		dir = filepath.Dir(dir)
		if strings.HasSuffix(dir, ".app") {
			return dir, nil
		}
		if dir == "/" || dir == "." {
			break
		}
	}
	return "", fmt.Errorf("could not find .app bundle from %s", execPath)
}

func (a *App) applyUpdateMacOS(staging string) error {
	// Find .app in staging
	var newApp string
	entries, err := os.ReadDir(staging)
	if err != nil {
		return fmt.Errorf("read staging dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			newApp = filepath.Join(staging, e.Name())
			break
		}
	}
	if newApp == "" {
		return fmt.Errorf("no .app bundle found in staging directory")
	}

	// Find current .app bundle
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %v", err)
	}
	currentApp, err := findAppBundle(execPath)
	if err != nil {
		return fmt.Errorf("cannot find current app bundle: %v", err)
	}

	// Replace old with new
	if err := os.RemoveAll(currentApp); err != nil {
		return fmt.Errorf("remove old app: %v", err)
	}
	if err := os.Rename(newApp, currentApp); err != nil {
		return fmt.Errorf("move new app: %v", err)
	}

	// Relaunch
	_ = exec.Command("open", currentApp).Start()
	os.Exit(0)
	return nil // unreachable
}

func (a *App) applyUpdateWindows(staging string) error {
	// Find .exe in staging
	var newExe string
	entries, err := os.ReadDir(staging)
	if err != nil {
		return fmt.Errorf("read staging dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ".exe") {
			newExe = filepath.Join(staging, e.Name())
			break
		}
	}
	if newExe == "" {
		return fmt.Errorf("no .exe found in staging directory")
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %v", err)
	}

	// Write a temporary bat script to do the replacement
	batContent := fmt.Sprintf(`@echo off
ping -n 3 127.0.0.1 >nul
copy /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, newExe, execPath, execPath)

	batPath := filepath.Join(os.TempDir(), "redc_update.bat")
	if err := os.WriteFile(batPath, []byte(batContent), 0o755); err != nil {
		return fmt.Errorf("write bat script: %v", err)
	}

	cmd := exec.Command("cmd", "/C", batPath)
	cmd.Dir = os.TempDir()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch bat script: %v", err)
	}

	os.Exit(0)
	return nil // unreachable
}

func (a *App) applyUpdateLinux(staging string) error {
	// Find binary in staging (first regular file that is executable or has no extension)
	var newBin string
	entries, err := os.ReadDir(staging)
	if err != nil {
		return fmt.Errorf("read staging dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Pick the first regular file
		if info.Mode().IsRegular() {
			newBin = filepath.Join(staging, e.Name())
			break
		}
	}
	if newBin == "" {
		return fmt.Errorf("no binary found in staging directory")
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %v", err)
	}

	// Remove old binary, rename new one into place
	if err := os.Remove(execPath); err != nil {
		return fmt.Errorf("remove old binary: %v", err)
	}
	if err := os.Rename(newBin, execPath); err != nil {
		return fmt.Errorf("move new binary: %v", err)
	}
	if err := os.Chmod(execPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %v", err)
	}

	_ = exec.Command(execPath).Start()
	os.Exit(0)
	return nil // unreachable
}
