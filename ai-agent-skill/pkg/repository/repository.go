package repository

import (
	"encoding/json"
	"fmt"
	"os"
)

// Skill represents a skill in the repository
type Skill struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Author      string   `json:"author"`
	Downloads   int      `json:"downloads"`
	URL         string   `json:"url"`
	Homepage    string   `json:"homepage"`
	License     string   `json:"license"`
	UpdatedAt   string   `json:"updated_at"`
}

// SkillsData represents the structure of skills.json file
type SkillsData struct {
	Skills []Skill `json:"skills"`
}

// Repository manages the skill repository
type Repository struct {
	filePath string
	skills   []Skill
}

// NewRepository creates a new repository instance
func NewRepository(filePath string) (*Repository, error) {
	repo := &Repository{
		filePath: filePath,
	}

	if err := repo.Load(); err != nil {
		return nil, fmt.Errorf("failed to load repository: %w", err)
	}

	return repo, nil
}

// Load loads skills from the JSON file
func (r *Repository) Load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to read skills file: %w", err)
	}

	var skillsData SkillsData
	if err := json.Unmarshal(data, &skillsData); err != nil {
		return fmt.Errorf("failed to parse skills file: %w", err)
	}

	r.skills = skillsData.Skills
	return nil
}

// GetAllSkills returns all skills in the repository
func (r *Repository) GetAllSkills() []Skill {
	return r.skills
}

// GetSkill retrieves a skill by name
func (r *Repository) GetSkill(name string) (*Skill, error) {
	for _, skill := range r.skills {
		if skill.Name == name {
			return &skill, nil
		}
	}
	return nil, fmt.Errorf("skill '%s' not found", name)
}

// SearchSkills searches for skills by keyword
func (r *Repository) SearchSkills(keyword string) []Skill {
	var results []Skill
	keywordLower := toLower(keyword)

	for _, skill := range r.skills {
		if contains(skill.Name, keywordLower) ||
			contains(skill.Description, keywordLower) ||
			containsAny(skill.Tags, keywordLower) {
			results = append(results, skill)
		}
	}

	return results
}

// Reload reloads the skills from file
func (r *Repository) Reload() error {
	return r.Load()
}

// Helper functions
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + ('a' - 'A')
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func containsAny(tags []string, keyword string) bool {
	for _, tag := range tags {
		if contains(toLower(tag), keyword) {
			return true
		}
	}
	return false
}