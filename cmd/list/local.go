package listcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seekthought/skill/pkg/storage"

	cmdutils "github.com/seekthought/skill/cmd/utils"
)

func listLocal(opt *options) error {
	printSectionHeader("Installed Skills")

	skillsDir, err := storage.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get skills directory: %w", err)
	}

	skills, err := loadInstalledSkills(skillsDir)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("No skills installed.")
		fmt.Println()
		printTips(
			"Install a skill using: skillctl install <git-url>",
			"Add a skill from local path: skillctl add <local-path>",
			"List remote skills using: skillctl list -R <url>",
		)
		return nil
	}

	renderCustomSkills(skills, customSkillRenderOptions{
		PathLabel:     "Path",
		ShowPath:      true,
		ShowInstalled: true,
		ShowTags:      true,
	})

	fmt.Println()
	fmt.Printf("Total: %d skill(s) installed\n", len(skills))
	fmt.Println()
	printTips("Use 'skillctl add <skill-name> -p <location>' to add a skill to your project")

	return nil
}

func listProject(opt *options) error {
	printSectionHeader("Project Skills")

	baseDir := opt.projectDir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	skills := loadProjectSkills(baseDir)
	if len(skills) == 0 {
		fmt.Println("No skills found in project directory.")
		fmt.Println()
		fmt.Printf("Project directory: %s\n", baseDir)
		fmt.Println()
		printTips(
			"Add a skill to project using: skillctl add <skill-name> -p <location>",
			"List installed skills using: skillctl list",
			"List remote skills using: skillctl list -R <url>",
		)
		return nil
	}

	renderCustomSkills(skills, customSkillRenderOptions{
		PathLabel:   "Location",
		BasePath:    baseDir,
		ShowPath:    true,
		ShowVersion: true,
		ShowAuthor:  true,
		ShowTags:    true,
	})

	fmt.Println()
	fmt.Printf("Total: %d skill(s) in project\n", len(skills))
	fmt.Printf("Project directory: %s\n", baseDir)
	fmt.Println()
	printTips("Use 'skillctl add <skill-name> -p <location>' to add more skills to your project")

	return nil
}

func listScanDir(dirPath string) error {
	absPath := dirPath
	if !filepath.IsAbs(dirPath) {
		var err error
		absPath, err = filepath.Abs(dirPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	printSectionHeader("Skills in Directory")
	fmt.Printf("Directory: %s\n\n", absPath)

	skills, err := cmdutils.ScanCustomSkills(absPath)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in this directory.")
		fmt.Println()
		printTips("Make sure the directory contains skill subdirectories with SKILL.md or skill.yaml files")
		return nil
	}

	renderCustomSkills(skills, customSkillRenderOptions{
		PathLabel:   "Location",
		BasePath:    absPath,
		ShowPath:    true,
		ShowVersion: true,
		ShowAuthor:  true,
		ShowTags:    true,
	})

	fmt.Println()
	fmt.Printf("Total: %d skill(s) found\n", len(skills))
	fmt.Println()

	return nil
}

func loadInstalledSkills(skillsDir string) ([]cmdutils.CustomSkill, error) {
	installedSkills, err := storage.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed skills: %w", err)
	}

	var skills []cmdutils.CustomSkill
	for _, skillMeta := range installedSkills {
		skills = append(skills, cmdutils.SkillMetaToCustomSkill(skillMeta, filepath.Join(skillsDir, skillMeta.Name)))
	}

	if len(skills) > 0 {
		return skills, nil
	}

	skills, err = cmdutils.ScanCustomSkills(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan skills directory: %w", err)
	}
	return skills, nil
}

func loadProjectSkills(baseDir string) []cmdutils.CustomSkill {
	return scanSkillsAcrossDirs(cmdutils.ProjectSkillDirs(baseDir))
}

func scanSkillsAcrossDirs(dirs []string) []cmdutils.CustomSkill {
	var allSkills []cmdutils.CustomSkill
	for _, dir := range dirs {
		skills, err := cmdutils.ScanCustomSkills(dir)
		if err != nil {
			continue
		}
		allSkills = append(allSkills, skills...)
	}
	return allSkills
}
