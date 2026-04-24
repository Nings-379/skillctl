package listcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/seekthought/skill/pkg/db"

	cmdutils "github.com/seekthought/skill/cmd/utils"
)

func printSectionHeader(title string) {
	fmt.Println(title + ":")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

func printTips(tips ...string) {
	for _, tip := range tips {
		fmt.Printf("💡 %s\n", tip)
	}
}

func renderCustomSkills(skills []cmdutils.CustomSkill, opts customSkillRenderOptions) {
	for i, skill := range skills {
		fmt.Printf("\n%d. %s\n", i+1, skill.Name)

		if skill.Description != "" {
			fmt.Printf("   Description: %s\n", skill.Description)
		}
		if opts.ShowPath && skill.Path != "" {
			fmt.Printf("   %s: %s\n", opts.PathLabel, displayPath(skill.Path, opts.BasePath))
		}
		if opts.ShowVersion && skill.Version != "" {
			fmt.Printf("   Version: %s\n", skill.Version)
		}
		if opts.ShowAuthor && skill.Author != "" {
			fmt.Printf("   Author: %s\n", skill.Author)
		}
		if opts.ShowInstalled && skill.InstalledAt != "" {
			fmt.Printf("   Installed: %s\n", skill.InstalledAt)
		}
		if opts.ShowTags && len(skill.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
		}
	}
}

func renderRepositorySkills(manager *db.Manager, ctx context.Context, skills []db.Skill) {
	for i, skill := range skills {
		fmt.Printf("\n%d. %s\n", i+1, skill.Name)

		if skill.Description.Valid && skill.Description.String != "" {
			fmt.Printf("   📝 Description: %s\n", skill.Description.String)
		}
		if skill.Version.Valid && skill.Version.String != "" {
			fmt.Printf("   📌 Version: %s\n", skill.Version.String)
		}
		if skill.Author.Valid && skill.Author.String != "" {
			fmt.Printf("   👤 Author: %s\n", skill.Author.String)
		}
		if skill.Category.Valid && skill.Category.String != "" {
			fmt.Printf("   📂 Category: %s\n", skill.Category.String)
		}
		if skill.RepoUrl.Valid && skill.RepoUrl.String != "" {
			fmt.Printf("   🔗 URL: %s\n", skill.RepoUrl.String)
		}
		if skill.Stars.Valid && skill.Stars.Int64 > 0 {
			fmt.Printf("   ⭐ Stars: %d\n", skill.Stars.Int64)
		}

		tags, err := manager.GetQueries().GetTagsBySkillID(ctx, skill.ID)
		if err == nil && len(tags) > 0 {
			tagNames := make([]string, len(tags))
			for j, tag := range tags {
				tagNames[j] = tag.Tag
			}
			fmt.Printf("   🏷️  Tags: %s\n", strings.Join(tagNames, ", "))
		}
	}
}

func displayPath(pathValue, basePath string) string {
	if basePath == "" {
		return pathValue
	}
	relPath, err := filepath.Rel(basePath, pathValue)
	if err != nil {
		return pathValue
	}
	return relPath
}
