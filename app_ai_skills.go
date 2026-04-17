package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	redc "red-cloud/mod"
	"red-cloud/mod/ai"
)

func (a *App) skillsDir() string {
	if redc.RedcPath != "" {
		return filepath.Join(redc.RedcPath, "skills")
	}
	return ""
}

// GetSkillsDir returns the skills installation directory path.
func (a *App) GetSkillsDir() string {
	return a.skillsDir()
}

// ListSkills returns all locally installed skills.
func (a *App) ListSkills(keyword string) []ai.SkillIndex {
	engine := ai.NewSkillsEngine(a.skillsDir())
	return engine.List(keyword)
}

// GetSkill returns the full content of a skill by ID.
func (a *App) GetSkill(id string) (*ai.Skill, error) {
	engine := ai.NewSkillsEngine(a.skillsDir())
	return engine.Read(id)
}

// SaveCustomSkill saves a custom skill to ~/redc/skills/<id>/SKILL.md.
func (a *App) SaveCustomSkill(id, content string) error {
	dir := a.skillsDir()
	if dir == "" {
		return fmt.Errorf("skills directory not available")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}

	skillDir := filepath.Join(dir, id)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}
	return nil
}

// DeleteCustomSkill removes a skill directory.
func (a *App) DeleteCustomSkill(id string) error {
	dir := a.skillsDir()
	if dir == "" {
		return fmt.Errorf("skills directory not available")
	}

	skillDir := filepath.Join(dir, id)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill %q not found", id)
	}
	return os.RemoveAll(skillDir)
}

// FetchSkillsRegistry fetches the remote skills registry.
func (a *App) FetchSkillsRegistry() ([]ai.RegistrySkill, error) {
	index, err := ai.FetchSkillRegistry("")
	if err != nil {
		return nil, err
	}

	dir := a.skillsDir()
	for i := range index.Skills {
		skill := &index.Skills[i]
		if dir == "" {
			continue
		}
		skillMD := filepath.Join(dir, skill.ID, "SKILL.md")
		if _, err := os.Stat(skillMD); err == nil {
			skill.Installed = true
			// Compare SHA256 to detect updates
			if skill.SHA256 != "" {
				localHash := ai.ReadLocalSkillHash(dir, skill.ID)
				if localHash != "" && localHash != skill.SHA256 {
					skill.HasUpdate = true
				} else if localHash == "" {
					// Legacy install without hash — treat as updatable
					skill.HasUpdate = true
				}
			}
		}
	}

	return index.Skills, nil
}

// InstallSkill downloads and installs a skill from the registry.
func (a *App) InstallSkill(id, downloadURL string) error {
	dir := a.skillsDir()
	if dir == "" {
		return fmt.Errorf("skills directory not available")
	}
	return ai.InstallSkill(dir, id, downloadURL)
}

// UpdateSkill re-downloads and reinstalls a skill from the registry.
func (a *App) UpdateSkill(id, downloadURL, sha256Hash string) error {
	dir := a.skillsDir()
	if dir == "" {
		return fmt.Errorf("skills directory not available")
	}
	return ai.UpdateSkill(dir, id, downloadURL, sha256Hash)
}

// InstallAllSkills installs all uninstalled skills from the registry.
func (a *App) InstallAllSkills() (int, error) {
	dir := a.skillsDir()
	if dir == "" {
		return 0, fmt.Errorf("skills directory not available")
	}

	index, err := ai.FetchSkillRegistry("")
	if err != nil {
		return 0, err
	}

	installed := 0
	var errs []string
	for _, skill := range index.Skills {
		destDir := filepath.Join(dir, skill.ID)
		if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); err == nil {
			continue
		}
		if err := ai.InstallSkill(dir, skill.ID, skill.URL); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", skill.ID, err))
		} else {
			installed++
		}
	}

	if len(errs) > 0 {
		return installed, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return installed, nil
}

// UpdateAllSkills updates all installed skills that have a new version in the registry.
func (a *App) UpdateAllSkills() (int, error) {
	dir := a.skillsDir()
	if dir == "" {
		return 0, fmt.Errorf("skills directory not available")
	}

	index, err := ai.FetchSkillRegistry("")
	if err != nil {
		return 0, err
	}

	updated := 0
	var errs []string
	for _, skill := range index.Skills {
		destDir := filepath.Join(dir, skill.ID)
		if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
			continue
		}
		// Skip if hash matches (already up to date)
		if skill.SHA256 != "" {
			localHash := ai.ReadLocalSkillHash(dir, skill.ID)
			if localHash == skill.SHA256 {
				continue
			}
		}
		if err := ai.UpdateSkill(dir, skill.ID, skill.URL, skill.SHA256); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", skill.ID, err))
		} else {
			updated++
		}
	}

	if len(errs) > 0 {
		return updated, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return updated, nil
}
