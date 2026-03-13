package storage

import (
	"path/filepath"
	"testing"
)

func TestGetSkillsDir(t *testing.T) {
	dir, err := GetSkillsDir()
	if err != nil {
		t.Fatalf("GetSkillsDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GetSkillsDir() returned empty string")
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("GetSkillsDir() returned relative path: %s", dir)
	}
}

func TestEnsureSkillsDir(t *testing.T) {
	t.Skip("EnsureSkillsDir() modifies real user directory, skipping test")
}

func TestAddSkill(t *testing.T) {
	t.Skip("AddSkill() modifies real user directory, skipping test")
}

func TestGetSkill(t *testing.T) {
	t.Skip("GetSkill() reads from real user directory, skipping test")
}

func TestRemoveSkill(t *testing.T) {
	t.Skip("RemoveSkill() modifies real user directory, skipping test")
}

func TestListSkills(t *testing.T) {
	t.Skip("ListSkills() reads from real user directory, skipping test")
}

func TestSkillMeta(t *testing.T) {
	skillMeta := SkillMeta{
		Name:        "test-skill",
		Description: "Test skill description",
		Version:     "1.0.0",
		Author:      "Test Author",
		InstalledAt: "2024-01-01T00:00:00Z",
		Tags:        []string{"test", "example"},
	}

	if skillMeta.Name != "test-skill" {
		t.Errorf("Name = %s, expected test-skill", skillMeta.Name)
	}

	if skillMeta.Description != "Test skill description" {
		t.Errorf("Description = %s, expected Test skill description", skillMeta.Description)
	}

	if skillMeta.Version != "1.0.0" {
		t.Errorf("Version = %s, expected 1.0.0", skillMeta.Version)
	}

	if skillMeta.Author != "Test Author" {
		t.Errorf("Author = %s, expected Test Author", skillMeta.Author)
	}

	if len(skillMeta.Tags) != 2 {
		t.Errorf("Tags length = %d, expected 2", len(skillMeta.Tags))
	}

	if skillMeta.Tags[0] != "test" {
		t.Errorf("Tags[0] = %s, expected test", skillMeta.Tags[0])
	}

	if skillMeta.Tags[1] != "example" {
		t.Errorf("Tags[1] = %s, expected example", skillMeta.Tags[1])
	}
}