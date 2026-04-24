package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var VSCodeSkillsDirs = map[string]string{
	"github":  filepath.Join(".github", "skills"),
	"agents":  filepath.Join(".agents", "skills"),
	"claude":  filepath.Join(".claude", "skills"),
	"copilot": filepath.Join(".copilot", "skills"),
}

func GetVSCodeInstallPath(location, projectDir, repoName string) (string, error) {
	skillsDir := location
	if dir, ok := VSCodeSkillsDirs[location]; ok {
		skillsDir = dir
	}

	var baseDir string
	if filepath.IsAbs(skillsDir) {
		baseDir = ""
	} else {
		baseDir = projectDir
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
		}
	}

	if baseDir != "" {
		return filepath.Join(baseDir, skillsDir, repoName), nil
	}
	return filepath.Join(skillsDir, repoName), nil
}

func RefreshVSCode() error {
	if os.Getenv("VSCODE_PID") == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "code", ".", "--goto", cwd)
		} else {
			cmd = exec.Command("code", ".", "--goto", cwd)
		}

		if err := cmd.Start(); err != nil {
			return nil
		}
		return nil
	}

	fmt.Println("   (Running in VS Code - workspace will refresh automatically)")
	return nil
}

func IsLocalPath(source string) bool {
	if strings.Contains(source, "://") {
		if strings.HasPrefix(source, "git://") || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "ssh://") {
			return false
		}
	}
	if strings.Contains(source, "git@") && strings.Contains(source, ":") {
		return false
	}
	if strings.HasSuffix(source, ".git") && !filepath.IsAbs(source) {
		return false
	}
	if strings.Contains(source, "/") || strings.Contains(source, string(filepath.Separator)) {
		if _, err := os.Stat(source); err == nil {
			return true
		}
	}
	if !strings.Contains(source, "://") && !strings.Contains(source, "@") {
		if _, err := os.Stat(source); err == nil {
			return true
		}
	}
	return false
}
