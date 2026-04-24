package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seekthought/skill/pkg/storage"
)

type CustomSkill struct {
	Name        string
	Description string
	Version     string
	Tags        []string
	Author      string
	Path        string
	InstalledAt string
}

func ScanCustomSkills(skillsDir string) ([]CustomSkill, error) {
	var skills []CustomSkill

	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return skills, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skill, err := ParseCustomSkill(skillPath, entry.Name())
		if err != nil {
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

func ParseCustomSkill(skillPath, skillName string) (CustomSkill, error) {
	meta, err := ParseSkillMetadata(skillPath, skillName)
	if err != nil {
		return CustomSkill{}, err
	}

	skill := SkillMetaToCustomSkill(meta, skillPath)
	if skill.Name == "" {
		skill.Name = skillName
	}
	if skill.Description == "" {
		skill.Description = fmt.Sprintf("Skill: %s", skillName)
	}

	return skill, nil
}

func SkillMetaToCustomSkill(meta storage.SkillMeta, path string) CustomSkill {
	return CustomSkill{
		Name:        meta.Name,
		Description: meta.Description,
		Version:     meta.Version,
		Tags:        meta.Tags,
		Author:      meta.Author,
		Path:        path,
		InstalledAt: meta.InstalledAt,
	}
}

func CopyDirectory(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := CopyDirectory(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := CopyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, ", ")
}
