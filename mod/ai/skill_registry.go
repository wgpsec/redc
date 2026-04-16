package ai

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultSkillRegistryURL = "https://redc.wgpsec.org/skills/skill-registry.json"

// SkillRegistryIndex is the remote skill registry.
type SkillRegistryIndex struct {
	Version int             `json:"version"`
	Updated string          `json:"updated"`
	Skills  []RegistrySkill `json:"skills"`
}

// RegistrySkill describes an available skill in the remote registry.
type RegistrySkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	URL         string   `json:"url"`
	SHA256      string   `json:"sha256,omitempty"`
	Installed   bool     `json:"installed,omitempty"`
}

// FetchSkillRegistry fetches the skill registry from the remote URL.
func FetchSkillRegistry(registryURL string) (*SkillRegistryIndex, error) {
	if registryURL == "" {
		registryURL = DefaultSkillRegistryURL
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(registryURL)
	if err != nil {
		return nil, fmt.Errorf("fetch skill registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skill registry returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read skill registry: %w", err)
	}

	var index SkillRegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("parse skill registry: %w", err)
	}

	return &index, nil
}

// InstallSkill downloads a skill zip from URL and extracts to skillsDir/<id>/.
func InstallSkill(skillsDir, id, downloadURL string) error {
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	destDir := filepath.Join(skillsDir, id)
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("skill %q already installed", id)
	}

	// Download zip
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "redc-skill-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("save skill zip: %w", err)
	}
	tmpFile.Close()

	// Extract zip
	if err := extractZip(tmpPath, destDir); err != nil {
		os.RemoveAll(destDir)
		return fmt.Errorf("extract skill: %w", err)
	}

	// Verify SKILL.md exists
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
		os.RemoveAll(destDir)
		return fmt.Errorf("invalid skill: missing SKILL.md")
	}

	return nil
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") {
			continue // skip path traversal
		}
		target := filepath.Join(destDir, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
