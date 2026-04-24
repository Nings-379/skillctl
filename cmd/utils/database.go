package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/seekthought/skill/pkg/db"
)

func InitDB() (*db.Manager, error) {
	if err := db.EnsureDBExists(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	manager, err := db.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	return manager, nil
}

func CloseDB(manager *db.Manager) {
	if manager != nil {
		if err := manager.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to close database: %v\n", err)
		}
	}
}

func GetRepositoryByName(name string) (*db.Repository, error) {
	manager, err := InitDB()
	if err != nil {
		return nil, err
	}
	defer CloseDB(manager)

	repo, err := manager.GetQueries().GetRepositoryByName(context.Background(), name)
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func FindSkillInDatabase(name, repoName string) (db.Skill, error) {
	manager, err := InitDB()
	if err != nil {
		return db.Skill{}, err
	}
	defer CloseDB(manager)

	ctx := context.Background()
	skills, err := manager.GetQueries().ListAllSkills(ctx)
	if err != nil {
		return db.Skill{}, fmt.Errorf("failed to list skills: %w", err)
	}

	repoMap := make(map[int64]string)
	if repoName != "" {
		repos, err := manager.GetQueries().ListAllRepositories(ctx)
		if err != nil {
			return db.Skill{}, fmt.Errorf("failed to list repositories: %w", err)
		}

		targetRepoID := int64(-1)
		for _, repo := range repos {
			repoMap[repo.ID] = repo.Name
			if strings.EqualFold(repo.Name, repoName) {
				targetRepoID = repo.ID
			}
		}

		if targetRepoID == -1 {
			return db.Skill{}, fmt.Errorf("repository '%s' not found", repoName)
		}

		var filteredSkills []db.Skill
		for _, skill := range skills {
			if skill.RepositoryID == targetRepoID {
				filteredSkills = append(filteredSkills, skill)
			}
		}
		skills = filteredSkills
	}

	for _, skill := range skills {
		if strings.EqualFold(skill.Name, name) {
			return skill, nil
		}
	}

	for _, skill := range skills {
		if strings.EqualFold(skill.Slug, name) {
			return skill, nil
		}
	}

	var suggestions []string
	for _, skill := range skills {
		if strings.Contains(strings.ToLower(skill.Name), strings.ToLower(name)) {
			suggestion := skill.Name
			if repoName != "" && len(repoMap) > 0 {
				if repoNameDisplay, ok := repoMap[skill.RepositoryID]; ok {
					suggestion = fmt.Sprintf("%s (from repo: %s)", skill.Name, repoNameDisplay)
				}
			}
			suggestions = append(suggestions, suggestion)
		}
		if len(suggestions) >= 3 {
			break
		}
	}

	errMsg := fmt.Sprintf("skill '%s' not found", name)
	if repoName != "" {
		errMsg += fmt.Sprintf(" in repository '%s'", repoName)
	}
	if len(suggestions) > 0 {
		errMsg += fmt.Sprintf("\n\nDid you mean:\n  - %s", strings.Join(suggestions, "\n  - "))
	}

	return db.Skill{}, fmt.Errorf(errMsg)
}
