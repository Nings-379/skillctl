package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	skillsDir = ".skills"
	indexFile = "index.json"
)

// SkillMeta represents skill metadata
type SkillMeta struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Author      string   `json:"author"`
	InstalledAt string   `json:"installed_at"`
}

// SkillIndex manages the index of installed skills
type SkillIndex struct {
	Skills map[string]SkillMeta `json:"skills"`
}

// NewSkillIndex creates a new skill index
func NewSkillIndex() *SkillIndex {
	return &SkillIndex{
		Skills: make(map[string]SkillMeta),
	}
}

// GetSkillsDir returns the skills directory path
func GetSkillsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, skillsDir), nil
}

// GetIndexPath returns the index file path
func GetIndexPath() (string, error) {
	skillsDir, err := GetSkillsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(skillsDir, indexFile), nil
}

// EnsureSkillsDir ensures the skills directory exists
func EnsureSkillsDir() error {
	skillsDir, err := GetSkillsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	return nil
}

// LoadIndex loads the skill index from disk
func LoadIndex() (*SkillIndex, error) {
	indexPath, err := GetIndexPath()
	if err != nil {
		return nil, err
	}

	// If index doesn't exist, return empty index
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return NewSkillIndex(), nil
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index SkillIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index file: %w", err)
	}

	if index.Skills == nil {
		index.Skills = make(map[string]SkillMeta)
	}

	return &index, nil
}

// SaveIndex saves the skill index to disk
func SaveIndex(index *SkillIndex) error {
	if err := EnsureSkillsDir(); err != nil {
		return err
	}

	indexPath, err := GetIndexPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// AddSkill adds a skill to the index
func AddSkill(name string, meta SkillMeta) error {
	index, err := LoadIndex()
	if err != nil {
		return err
	}

	index.Skills[name] = meta

	return SaveIndex(index)
}

// RemoveSkill removes a skill from the index
func RemoveSkill(name string) error {
	index, err := LoadIndex()
	if err != nil {
		return err
	}

	if _, exists := index.Skills[name]; !exists {
		return fmt.Errorf("skill '%s' is not installed", name)
	}

	delete(index.Skills, name)

	return SaveIndex(index)
}

// GetSkill retrieves a skill from the index
func GetSkill(name string) (*SkillMeta, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, err
	}

	meta, exists := index.Skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' is not installed", name)
	}

	return &meta, nil
}

// ListSkills returns all installed skills
func ListSkills() ([]SkillMeta, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, err
	}

	skills := make([]SkillMeta, 0, len(index.Skills))
	for _, meta := range index.Skills {
		skills = append(skills, meta)
	}

	return skills, nil
}