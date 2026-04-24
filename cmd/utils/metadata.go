package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/seekthought/skill/pkg/storage"
)

func ParseSkillMetadata(skillPath, skillName string) (storage.SkillMeta, error) {
	var meta storage.SkillMeta
	meta.Name = skillName

	if data, err := osReadFile(filepath.Join(skillPath, "SKILL.md")); err == nil {
		if err := parseSkillMetaFromMarkdown(&meta, string(data)); err == nil {
			return meta, nil
		}
	}

	if data, err := osReadFile(filepath.Join(skillPath, "skill.yaml")); err == nil {
		if err := parseSkillFromYAML(&meta, string(data)); err == nil {
			return meta, nil
		}
	}

	if gitRemote, err := GetGitRemote(skillPath); err == nil {
		if readme, err := FetchGitHubReadme(gitRemote); err == nil {
			meta.Description = ExtractDescriptionFromReadme(readme)
		}
	}

	return meta, nil
}

func ParseSkillContentMetadata(content string) (string, []string, error) {
	lines := strings.Split(content, "\n")
	var description string
	var tags []string
	inFrontMatter := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "---") {
			if i == 0 {
				inFrontMatter = true
				continue
			}
			if inFrontMatter {
				break
			}
		}

		if inFrontMatter {
			if strings.HasPrefix(trimmedLine, "description:") {
				description = trimMetaValue(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:")))
			}
			if strings.HasPrefix(trimmedLine, "tags:") {
				tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
				if tagsStr != "" && tagsStr != "[]" {
					tagsStr = strings.Trim(tagsStr, "[]")
					tagList := strings.Split(tagsStr, ",")
					for i, tag := range tagList {
						tagList[i] = trimMetaValue(strings.TrimSpace(tag))
					}
					tags = tagList
				}
			}
		}
	}

	if description == "" {
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") || strings.HasPrefix(trimmedLine, "---") {
				continue
			}
			description = trimmedLine
			if len(description) > 200 {
				description = description[:200] + "..."
			}
			break
		}
	}

	return description, tags, nil
}

func parseSkillMetaFromMarkdown(meta *storage.SkillMeta, markdown string) error {
	lines := strings.Split(markdown, "\n")
	inFrontMatter := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "---") {
			if i == 0 {
				inFrontMatter = true
				continue
			}
			if inFrontMatter {
				break
			}
		}

		if inFrontMatter {
			applyMetaToStorage(meta, trimmedLine)
		}
	}

	return nil
}

func parseSkillFromYAML(meta *storage.SkillMeta, yamlContent string) error {
	for _, line := range strings.Split(yamlContent, "\n") {
		applyMetaToStorage(meta, strings.TrimSpace(line))
	}
	return nil
}

func applyMetaToStorage(meta *storage.SkillMeta, trimmedLine string) {
	if strings.HasPrefix(trimmedLine, "name:") {
		meta.Name = trimMetaValue(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "name:")))
	}
	if strings.HasPrefix(trimmedLine, "description:") {
		meta.Description = trimMetaValue(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "description:")))
	}
	if strings.HasPrefix(trimmedLine, "version:") {
		meta.Version = trimMetaValue(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "version:")))
	}
	if strings.HasPrefix(trimmedLine, "author:") {
		meta.Author = trimMetaValue(strings.TrimSpace(strings.TrimPrefix(trimmedLine, "author:")))
	}
	if strings.HasPrefix(trimmedLine, "tags:") {
		tagsStr := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "tags:"))
		if tagsStr != "" && tagsStr != "[]" {
			tagsStr = strings.Trim(tagsStr, "[]")
			tags := strings.Split(tagsStr, ",")
			for i, tag := range tags {
				tags[i] = trimMetaValue(strings.TrimSpace(tag))
			}
			meta.Tags = tags
		}
	}
}

func trimMetaValue(value string) string {
	return strings.Trim(value, `"'`)
}

var osReadFile = os.ReadFile
