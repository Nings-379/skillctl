package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

var ProjectSkillSubdirs = []string{
	filepath.Join(".github", "skills"),
	filepath.Join(".agents", "skills"),
	filepath.Join(".claude", "skills"),
	filepath.Join(".copilot", "skills"),
	filepath.Join("examples", "skills"),
}

func ResolveProjectDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	return os.Getwd()
}

func ProjectSkillDirs(baseDir string) []string {
	dirs := make([]string, 0, len(ProjectSkillSubdirs))
	for _, subdir := range ProjectSkillSubdirs {
		dirs = append(dirs, filepath.Join(baseDir, subdir))
	}
	return dirs
}

func FindSkillInProject(baseDir, skillName string) (string, error) {
	for _, dir := range ProjectSkillDirs(baseDir) {
		skillPath := filepath.Join(dir, skillName)
		if _, err := os.Stat(skillPath); err == nil {
			return skillPath, nil
		}
	}

	return "", fmt.Errorf("skill '%s' not found in project directory: %s", skillName, baseDir)
}
