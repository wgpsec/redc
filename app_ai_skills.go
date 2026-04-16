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

	// Mark installed skills
	installed := make(map[string]bool)
	dir := a.skillsDir()
	if dir != "" {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() {
				skillMD := filepath.Join(dir, e.Name(), "SKILL.md")
				if _, err := os.Stat(skillMD); err == nil {
					installed[e.Name()] = true
				}
			}
		}
	}

	for i := range index.Skills {
		if installed[index.Skills[i].ID] {
			index.Skills[i].Installed = true
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
