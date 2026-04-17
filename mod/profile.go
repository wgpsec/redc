package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const profilesDirName = "profiles"
const activeProfileFile = "active_profile"

// ProfileInfo represents a GUI profile with an external ID (derived from filename).
type ProfileInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ConfigPath  string    `json:"configPath"`
	TemplateDir string    `json:"templateDir"`
	AIConfig    *AIConfig `json:"aiConfig,omitempty"`
}

// AIConfig represents AI provider configuration
type AIConfig struct {
	Provider      string `json:"provider"`
	APIKey        string `json:"apiKey,omitempty"`
	BaseURL       string `json:"baseUrl"`
	Model         string `json:"model"`
	MaxToolRounds int    `json:"maxToolRounds,omitempty"`
	ContextWindow int    `json:"contextWindow,omitempty"` // model context window size in tokens (default 120000)
	EnableAskUser *bool  `json:"enableAskUser,omitempty"`
	// FallbackProviders are backup AI providers used when the primary fails
	FallbackProviders []FallbackProvider `json:"fallbackProviders,omitempty"`
}

// FallbackProvider represents a backup AI provider for failover
type FallbackProvider struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
}

type profilePayload struct {
	Name        string    `json:"name"`
	ConfigPath  string    `json:"configPath"`
	TemplateDir string    `json:"templateDir"`
	AIConfig    *AIConfig `json:"aiConfig,omitempty"`
}

func ensureRedcPath() error {
	if RedcPath != "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("无法获取用户目录: %v", err)
	}
	RedcPath = filepath.Join(home, "redc")
	return nil
}

func profilesDir() (string, error) {
	if err := ensureRedcPath(); err != nil {
		return "", err
	}
	return filepath.Join(RedcPath, profilesDirName), nil
}

func defaultTemplateDir() (string, error) {
	if err := ensureRedcPath(); err != nil {
		return "", err
	}
	return filepath.Join(RedcPath, "templates"), nil
}

func ensureProfilesDir() (string, error) {
	dir, err := profilesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建 profiles 目录失败: %v", err)
	}
	return dir, nil
}

func profileFilePath(id string) (string, error) {
	dir, err := profilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

func makeProfileID(name string) string {
	base := strings.TrimSpace(strings.ToLower(name))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "profile"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func readProfileFile(id string) (ProfileInfo, error) {
	path, err := profileFilePath(id)
	if err != nil {
		return ProfileInfo{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProfileInfo{}, err
	}
	var payload profilePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return ProfileInfo{}, err
	}
	return ProfileInfo{
		ID:          id,
		Name:        payload.Name,
		ConfigPath:  payload.ConfigPath,
		TemplateDir: payload.TemplateDir,
		AIConfig:    payload.AIConfig,
	}, nil
}

func writeProfileFile(id string, payload profilePayload) (ProfileInfo, error) {
	path, err := profileFilePath(id)
	if err != nil {
		return ProfileInfo{}, err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ProfileInfo{}, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return ProfileInfo{}, err
	}
	return ProfileInfo{
		ID:          id,
		Name:        payload.Name,
		ConfigPath:  payload.ConfigPath,
		TemplateDir: payload.TemplateDir,
		AIConfig:    payload.AIConfig,
	}, nil
}

func EnsureDefaultProfile() (ProfileInfo, error) {
	dir, err := ensureProfilesDir()
	if err != nil {
		return ProfileInfo{}, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ProfileInfo{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			return readProfileFile(id)
		}
	}
	configPath, err := GetConfigPath("")
	if err != nil {
		return ProfileInfo{}, err
	}
	templateDir, err := defaultTemplateDir()
	if err != nil {
		return ProfileInfo{}, err
	}
	payload := profilePayload{
		Name:        "default",
		ConfigPath:  configPath,
		TemplateDir: templateDir,
	}
	profile, err := writeProfileFile("default", payload)
	if err != nil {
		return ProfileInfo{}, err
	}
	if err := setActiveProfileID("default"); err != nil {
		return ProfileInfo{}, err
	}
	return profile, nil
}

func ListProfiles() ([]ProfileInfo, error) {
	if _, err := EnsureDefaultProfile(); err != nil {
		return nil, err
	}
	dir, err := profilesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	profiles := make([]ProfileInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		profile, err := readProfileFile(id)
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})
	return profiles, nil
}

func GetActiveProfile() (ProfileInfo, error) {
	if _, err := EnsureDefaultProfile(); err != nil {
		return ProfileInfo{}, err
	}
	id, err := getActiveProfileID()
	if err != nil || id == "" {
		id = "default"
		_ = setActiveProfileID(id)
	}
	return readProfileFile(id)
}

func CreateProfile(name string, configPath string, templateDir string) (ProfileInfo, error) {
	if strings.TrimSpace(name) == "" {
		return ProfileInfo{}, fmt.Errorf("profile name cannot be empty")
	}
	if _, err := EnsureDefaultProfile(); err != nil {
		return ProfileInfo{}, err
	}
	if configPath == "" {
		path, err := GetConfigPath("")
		if err != nil {
			return ProfileInfo{}, err
		}
		configPath = path
	}
	if templateDir == "" {
		path, err := defaultTemplateDir()
		if err != nil {
			return ProfileInfo{}, err
		}
		templateDir = path
	}
	id := makeProfileID(name)
	return writeProfileFile(id, profilePayload{
		Name:        name,
		ConfigPath:  configPath,
		TemplateDir: templateDir,
	})
}

func UpdateProfile(id string, name string, configPath string, templateDir string) (ProfileInfo, error) {
	if strings.TrimSpace(id) == "" {
		return ProfileInfo{}, fmt.Errorf("profile id cannot be empty")
	}
	if strings.TrimSpace(name) == "" {
		return ProfileInfo{}, fmt.Errorf("profile name cannot be empty")
	}
	if configPath == "" {
		path, err := GetConfigPath("")
		if err != nil {
			return ProfileInfo{}, err
		}
		configPath = path
	}
	if templateDir == "" {
		path, err := defaultTemplateDir()
		if err != nil {
			return ProfileInfo{}, err
		}
		templateDir = path
	}
	return writeProfileFile(id, profilePayload{
		Name:        name,
		ConfigPath:  configPath,
		TemplateDir: templateDir,
	})
}

// UpdateProfileAIConfig updates only the AI config for a profile
func UpdateProfileAIConfig(id string, aiConfig *AIConfig) error {
	profile, err := readProfileFile(id)
	if err != nil {
		return err
	}
	return writeProfileFileWithAIConfig(id, profile, aiConfig)
}

func writeProfileFileWithAIConfig(id string, profile ProfileInfo, aiConfig *AIConfig) error {
	path, err := profileFilePath(id)
	if err != nil {
		return err
	}
	payload := profilePayload{
		Name:        profile.Name,
		ConfigPath:  profile.ConfigPath,
		TemplateDir: profile.TemplateDir,
		AIConfig:    aiConfig,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func DeleteProfile(id string) error {
	profiles, err := ListProfiles()
	if err != nil {
		return err
	}
	if len(profiles) <= 1 {
		return fmt.Errorf("至少需要保留一个 profile")
	}
	if id == "default" {
		return fmt.Errorf("default profile 不能删除")
	}
	path, err := profileFilePath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	activeID, _ := getActiveProfileID()
	if activeID == id {
		_ = setActiveProfileID("default")
	}
	return nil
}

func SetActiveProfile(id string) (ProfileInfo, error) {
	profile, err := readProfileFile(id)
	if err != nil {
		return ProfileInfo{}, err
	}
	if err := setActiveProfileID(id); err != nil {
		return ProfileInfo{}, err
	}
	ActiveConfigPath = profile.ConfigPath
	if profile.TemplateDir != "" {
		TemplateDir = profile.TemplateDir
	} else if path, err := defaultTemplateDir(); err == nil {
		TemplateDir = path
	}
	_ = ApplyConfig(profile.ConfigPath)
	return profile, nil
}

func activeProfilePath() (string, error) {
	dir, err := profilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, activeProfileFile), nil
}

func getActiveProfileID() (string, error) {
	path, err := activeProfilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func setActiveProfileID(id string) error {
	path, err := activeProfilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(id), 0600)
}
