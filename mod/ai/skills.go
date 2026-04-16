package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Skill represents a knowledge base document for IaC best practices.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Content     string   `json:"content,omitempty"`
}

// SkillIndex is a lightweight entry used for search without loading full content.
type SkillIndex struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// SkillsEngine manages loading, indexing, and searching skills from a directory.
type SkillsEngine struct {
	mu     sync.RWMutex
	dir    string
	index  []SkillIndex
	loaded bool
}

// NewSkillsEngine creates a new engine. dir is the path to the skills directory.
func NewSkillsEngine(dir string) *SkillsEngine {
	return &SkillsEngine{dir: dir}
}

// ensureLoaded lazily builds the index on first access.
func (e *SkillsEngine) ensureLoaded() {
	e.mu.RLock()
	if e.loaded {
		e.mu.RUnlock()
		return
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.loaded {
		return
	}

	e.index = make([]SkillIndex, 0)

	// Scan directory for skills
	if e.dir != "" {
		entries, err := os.ReadDir(e.dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				skillMD := filepath.Join(e.dir, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillMD); err != nil {
					continue
				}
				data, err := os.ReadFile(skillMD)
				if err != nil {
					continue
				}
				si := parseSkillFrontmatter(entry.Name(), string(data))
				e.index = append(e.index, si)
			}
		}
	}

	e.loaded = true
}

// Reload forces a rescan of the skills directory.
func (e *SkillsEngine) Reload() {
	e.mu.Lock()
	e.loaded = false
	e.mu.Unlock()
	e.ensureLoaded()
}

// List returns all available skill index entries. Optionally filter by keyword.
func (e *SkillsEngine) List(keyword string) []SkillIndex {
	e.ensureLoaded()
	e.mu.RLock()
	defer e.mu.RUnlock()

	if keyword == "" {
		result := make([]SkillIndex, len(e.index))
		copy(result, e.index)
		return result
	}

	kw := strings.ToLower(keyword)
	var matched []SkillIndex
	for _, si := range e.index {
		if strings.Contains(strings.ToLower(si.Name), kw) ||
			strings.Contains(strings.ToLower(si.Description), kw) ||
			strings.Contains(strings.ToLower(si.ID), kw) {
			matched = append(matched, si)
			continue
		}
		for _, tag := range si.Tags {
			if strings.Contains(strings.ToLower(tag), kw) {
				matched = append(matched, si)
				break
			}
		}
	}
	return matched
}

// Read returns the full content of a skill by ID.
func (e *SkillsEngine) Read(id string) (*Skill, error) {
	e.ensureLoaded()

	// Read from directory
	if e.dir != "" {
		skillMD := filepath.Join(e.dir, id, "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err == nil {
			si := parseSkillFrontmatter(id, string(data))
			return &Skill{
				ID:          si.ID,
				Name:        si.Name,
				Description: si.Description,
				Tags:        si.Tags,
				Content:     string(data),
			}, nil
		}
	}

	return nil, fmt.Errorf("skill %q not found", id)
}

// Suggest returns recommended skill IDs based on context (target, tool usage, errors).
func (e *SkillsEngine) Suggest(context string, maxResults int) []SkillIndex {
	e.ensureLoaded()
	e.mu.RLock()
	defer e.mu.RUnlock()

	if maxResults <= 0 {
		maxResults = 5
	}

	ctxLower := strings.ToLower(context)
	ctxTokens := extractTokens(ctxLower)

	type scored struct {
		si    SkillIndex
		score int
	}
	var results []scored

	for _, si := range e.index {
		score := 0
		for _, tag := range si.Tags {
			tagLower := strings.ToLower(tag)
			if _, ok := ctxTokens[tagLower]; ok {
				score += 3
			} else if strings.Contains(ctxLower, tagLower) && len(tagLower) >= 3 {
				score += 1
			}
		}
		descLower := strings.ToLower(si.Description)
		for tok := range ctxTokens {
			if len(tok) >= 3 && strings.Contains(descLower, tok) {
				score += 1
			}
		}
		if score > 0 {
			results = append(results, scored{si: si, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	out := make([]SkillIndex, len(results))
	for i, r := range results {
		out[i] = r.si
	}
	return out
}

// FormatSuggestions formats skill suggestions as a prompt injection block.
func FormatSuggestions(suggestions []SkillIndex) string {
	if len(suggestions) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n## Recommended Skills (auto-matched to context)\n")
	sb.WriteString("Load on demand when encountering related scenarios:\n")
	for _, si := range suggestions {
		desc := si.Description
		if len(desc) > 60 {
			desc = desc[:60] + "..."
		}
		sb.WriteString(fmt.Sprintf("- `read_skill(id=\"%s\")` — %s\n", si.ID, desc))
	}
	sb.WriteString("\nUse `list_skills(keyword=\"...\")` to search for more.\n")
	return sb.String()
}

// --- Internal helpers ---

func parseSkillFrontmatter(id, content string) SkillIndex {
	si := SkillIndex{ID: id, Name: id}

	if !strings.HasPrefix(content, "---") {
		return si
	}
	end := strings.Index(content[3:], "---")
	if end == -1 {
		return si
	}
	front := content[3 : 3+end]

	for _, line := range strings.Split(front, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			si.Name = strings.Trim(strings.TrimPrefix(line, "name:"), " \"'")
		} else if strings.HasPrefix(line, "description:") {
			si.Description = strings.Trim(strings.TrimPrefix(line, "description:"), " \"'")
		} else if strings.HasPrefix(line, "tags:") {
			tagStr := strings.TrimPrefix(line, "tags:")
			tagStr = strings.Trim(tagStr, " \"'")
			for _, t := range regexp.MustCompile(`[,，]`).Split(tagStr, -1) {
				t = strings.TrimSpace(t)
				if t != "" {
					si.Tags = append(si.Tags, t)
				}
			}
		}
	}
	return si
}

func extractTokens(s string) map[string]struct{} {
	re := regexp.MustCompile(`[a-z\p{Han}]{2,}`)
	matches := re.FindAllString(s, -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		tokens[m] = struct{}{}
	}
	return tokens
}
