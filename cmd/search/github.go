package searchcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const githubSearchAPIBase = "https://api.github.com"

type githubRepositorySearchResponse struct {
	TotalCount int                    `json:"total_count"`
	Items      []githubRepositoryItem `json:"items"`
}

type githubCodeSearchResponse struct {
	TotalCount int                    `json:"total_count"`
	Items      []githubCodeSearchItem `json:"items"`
}

type githubCodeSearchItem struct {
	Path       string `json:"path"`
	HTMLURL    string `json:"html_url"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}

type githubRepositoryItem struct {
	Name            string   `json:"name"`
	FullName        string   `json:"full_name"`
	Description     string   `json:"description"`
	HTMLURL         string   `json:"html_url"`
	CloneURL        string   `json:"clone_url"`
	Language        string   `json:"language"`
	DefaultBranch   string   `json:"default_branch"`
	Topics          []string `json:"topics"`
	Archived        bool     `json:"archived"`
	Fork            bool     `json:"fork"`
	StargazersCount int      `json:"stargazers_count"`
	UpdatedAt       string   `json:"updated_at"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type githubRepoContentEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type githubRepoSignals struct {
	HasSkillsDir bool
	HasSkillMD   bool
	HasSkillYAML bool
	HasReadme    bool
}

type githubSkillCandidate struct {
	Repository githubRepositoryItem
	Signals    githubRepoSignals
	Score      int
	Reasons    []string
	Confidence string
}

type githubSkillDirectoryMatch struct {
	Repository githubRepositoryItem
	SkillName  string
	SkillURL   string
	Score      int
	MatchType  string
	MatchedBy  string
}

func searchGitHubGlobal(keyword string, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("--limit must be greater than 0")
	}
	if limit > 50 {
		limit = 50
	}

	token := os.Getenv("GITHUB_TOKEN")
	fmt.Printf("🌐 Source: GitHub global search\n")
	fmt.Printf("   Keyword: %s\n", keyword)
	fmt.Printf("   Limit: %d\n", limit)
	if token == "" {
		fmt.Println("⚠️  GITHUB_TOKEN not set. Using unauthenticated GitHub Search API, which has low rate limits.")
	}
	fmt.Println()

	matches, err := githubSearchSkillDirectoryMatches(keyword, limit, token)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		fmt.Println("❌ No matching skill names were found in GitHub repositories that contain a skills/ directory.")
		fmt.Println("💡 Try another keyword or use 'skillctl search -r <repo> <keyword>' for your indexed repositories.")
		return nil
	}

	sortGitHubSkillMatches(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}

	fmt.Printf("📊 GitHub Search Results:\n")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println()

	for i, match := range matches {
		repo := match.Repository
		fmt.Printf("%d. %s\n", i+1, match.SkillName)
		fmt.Printf("   📦 Repository: %s\n", repo.FullName)
		fmt.Printf("   🔗 Skill URL: %s\n", match.SkillURL)
		fmt.Printf("   📥 Clone: %s\n", repo.CloneURL)
		fmt.Printf("   ✅ Match: %s\n", match.MatchType)
		fmt.Printf("   🔎 Matched By: %s\n", match.MatchedBy)
		fmt.Printf("   🎯 Proof: repository contains skills/\n")
		if repo.Description != "" {
			fmt.Printf("   📝 Description: %s\n", repo.Description)
		}
		fmt.Printf("   ⭐ Stars: %d\n", repo.StargazersCount)
		if repo.Language != "" {
			fmt.Printf("   💻 Language: %s\n", repo.Language)
		}
		if repo.UpdatedAt != "" {
			fmt.Printf("   📅 Updated: %s\n", formatGitHubTimestamp(repo.UpdatedAt))
		}
		if len(repo.Topics) > 0 {
			fmt.Printf("   🏷️  Topics: %s\n", strings.Join(repo.Topics, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("✅ Found %d GitHub skill match(es) for '%s'\n\n", len(matches), keyword)
	fmt.Println("💡 Next steps:")
	fmt.Println("   - Use 'skillctl install <clone-url> <skill-name>' to install the matched skill from that repository")
	fmt.Println("   - Use 'skillctl install <clone-url>' if you want the entire repository")
	fmt.Println("   - Use 'skillctl add -r <repo-url>' and then 'skillctl install sync-repo' if you want to index a repository locally")
	return nil
}

func githubSearchSkillDirectoryMatches(keyword string, limit int, token string) ([]githubSkillDirectoryMatch, error) {
	matches, err := githubSearchSkillDirectoryMatchesByPath(keyword, limit, token)
	if err == nil && len(matches) > 0 {
		return matches, nil
	}

	return githubSearchSkillDirectoryMatchesByRepository(keyword, limit, token)
}

func githubSearchSkillDirectoryMatchesByPath(keyword string, limit int, token string) ([]githubSkillDirectoryMatch, error) {
	fetchLimit := githubSearchFetchLimit(limit)
	pathQueries := []string{
		fmt.Sprintf("%s SKILL.md in:path", strings.TrimSpace(keyword)),
		fmt.Sprintf("%s skill.yaml in:path", strings.TrimSpace(keyword)),
		fmt.Sprintf("%s in:path", strings.TrimSpace(keyword)),
	}
	contentQueries := []string{
		fmt.Sprintf("%s path:skills filename:SKILL.md", strings.TrimSpace(keyword)),
		fmt.Sprintf("%s path:skills filename:skill.yaml", strings.TrimSpace(keyword)),
	}

	repoCache := make(map[string]githubRepositoryItem)
	matchMap := make(map[string]githubSkillDirectoryMatch)
	var firstErr error

	for _, query := range pathQueries {
		items, err := githubSearchCodeForSkillPaths(query, fetchLimit, token)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, item := range items {
			skillName, ok := extractGitHubSkillNameFromPath(item.Path)
			if !ok {
				continue
			}

			repo, ok := repoCache[item.Repository.FullName]
			if !ok {
				repo, err = getGitHubRepositoryDetails(item.Repository.FullName, token)
				if err != nil {
					continue
				}
				repoCache[item.Repository.FullName] = repo
			}

			score, matchType := scoreGitHubSkillName(skillName, normalizeGitHubSkillSearchText(keyword))
			if score <= 0 {
				continue
			}

			upsertGitHubSkillMatch(matchMap, githubSkillDirectoryMatch{
				Repository: repo,
				SkillName:  skillName,
				SkillURL:   buildGitHubSkillURL(repo, skillName),
				Score:      score,
				MatchType:  matchType,
				MatchedBy:  "skill name",
			})
		}
	}

	for _, query := range contentQueries {
		items, err := githubSearchCodeForSkillPaths(query, fetchLimit, token)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, item := range items {
			skillName, ok := extractGitHubSkillNameFromPath(item.Path)
			if !ok {
				continue
			}

			repo, ok := repoCache[item.Repository.FullName]
			if !ok {
				repo, err = getGitHubRepositoryDetails(item.Repository.FullName, token)
				if err != nil {
					continue
				}
				repoCache[item.Repository.FullName] = repo
			}

			score, matchType := scoreGitHubSkillDescriptionMatch(skillName, normalizeGitHubSkillSearchText(keyword))
			if score <= 0 {
				continue
			}

			upsertGitHubSkillMatch(matchMap, githubSkillDirectoryMatch{
				Repository: repo,
				SkillName:  skillName,
				SkillURL:   buildGitHubSkillURL(repo, skillName),
				Score:      score,
				MatchType:  matchType,
				MatchedBy:  "skill description",
			})
		}
	}

	matches := make([]githubSkillDirectoryMatch, 0, len(matchMap))
	for _, match := range matchMap {
		matches = append(matches, match)
	}

	sortGitHubSkillMatches(matches)
	if len(matches) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return matches, nil
}

func githubSearchSkillDirectoryMatchesByRepository(keyword string, limit int, token string) ([]githubSkillDirectoryMatch, error) {
	repos, err := githubSearchRepositories(keyword, limit, token)
	if err != nil {
		return nil, err
	}

	matches := make([]githubSkillDirectoryMatch, 0, githubSearchFetchLimit(limit))
	var firstErr error
	for _, repo := range repos {
		signals, err := inspectGitHubRepository(repo.FullName, token)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !signals.HasSkillsDir {
			continue
		}

		skillEntries, err := listGitHubSkillEntries(repo.FullName, token)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		matches = append(matches, matchGitHubSkillEntries(repo, skillEntries, keyword)...)
		if len(matches) >= limit*2 {
			break
		}
	}

	sortGitHubSkillMatches(matches)
	if len(matches) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return matches, nil
}

func githubSearchCodeForSkillPaths(query string, limit int, token string) ([]githubCodeSearchItem, error) {
	apiURL := fmt.Sprintf("%s/search/code?q=%s&per_page=%d", githubSearchAPIBase, url.QueryEscape(query), limit)
	body, _, err := doGitHubAPIRequest(apiURL, token)
	if err != nil {
		return nil, err
	}

	var response githubCodeSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub code search response: %w", err)
	}

	return response.Items, nil
}

func extractGitHubSkillNameFromPath(path string) (string, bool) {
	normalized := strings.Trim(path, "/")
	if !strings.HasPrefix(normalized, "skills/") {
		return "", false
	}
	parts := strings.Split(normalized, "/")
	if len(parts) < 2 || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func getGitHubRepositoryDetails(fullName, token string) (githubRepositoryItem, error) {
	apiURL := fmt.Sprintf("%s/repos/%s", githubSearchAPIBase, fullName)
	body, _, err := doGitHubAPIRequest(apiURL, token)
	if err != nil {
		return githubRepositoryItem{}, err
	}

	var repo githubRepositoryItem
	if err := json.Unmarshal(body, &repo); err != nil {
		return githubRepositoryItem{}, fmt.Errorf("failed to parse GitHub repository details: %w", err)
	}
	return repo, nil
}

func githubSearchRepositories(keyword string, limit int, token string) ([]githubRepositoryItem, error) {
	fetchLimit := githubSearchFetchLimit(limit)
	queries := []string{
		"copilot skills archived:false fork:false",
		"agent skills archived:false fork:false",
		"claude skills archived:false fork:false",
		"mcp skills archived:false fork:false",
		strings.TrimSpace(keyword) + " skills archived:false fork:false",
	}

	itemsByFullName := make(map[string]githubRepositoryItem)
	var firstErr error
	for _, query := range queries {
		items, err := githubSearchRepositoriesForQuery(query, fetchLimit, token)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, item := range items {
			if _, ok := itemsByFullName[item.FullName]; !ok {
				itemsByFullName[item.FullName] = item
			}
		}
	}

	if len(itemsByFullName) == 0 {
		fallbackQuery := "skills archived:false fork:false"
		items, err := githubSearchRepositoriesForQuery(fallbackQuery, fetchLimit, token)
		if err != nil {
			if firstErr != nil {
				return nil, firstErr
			}
			return nil, err
		}
		return items, nil
	}

	items := make([]githubRepositoryItem, 0, len(itemsByFullName))
	for _, item := range itemsByFullName {
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].StargazersCount > items[j].StargazersCount
	})
	if len(items) > fetchLimit {
		items = items[:fetchLimit]
	}

	return items, nil
}

func githubSearchFetchLimit(limit int) int {
	fetchLimit := limit * 4
	if fetchLimit < 12 {
		fetchLimit = 12
	}
	if fetchLimit > 50 {
		fetchLimit = 50
	}
	return fetchLimit
}

func githubSearchRepositoriesForQuery(query string, limit int, token string) ([]githubRepositoryItem, error) {
	apiURL := fmt.Sprintf("%s/search/repositories?q=%s&sort=stars&order=desc&per_page=%d", githubSearchAPIBase, url.QueryEscape(query), limit)

	body, _, err := doGitHubAPIRequest(apiURL, token)
	if err != nil {
		return nil, err
	}

	var response githubRepositorySearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub search response: %w", err)
	}

	return response.Items, nil
}

func inspectGitHubRepository(fullName, token string) (githubRepoSignals, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/contents", githubSearchAPIBase, fullName)
	body, _, err := doGitHubAPIRequest(apiURL, token)
	if err != nil {
		return githubRepoSignals{}, err
	}

	var entries []githubRepoContentEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return githubRepoSignals{}, fmt.Errorf("failed to parse GitHub repository contents: %w", err)
	}

	signals := githubRepoSignals{}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name)
		switch {
		case entry.Type == "dir" && name == "skills":
			signals.HasSkillsDir = true
		case entry.Type == "file" && name == "skill.md":
			signals.HasSkillMD = true
		case entry.Type == "file" && name == "skill.yaml":
			signals.HasSkillYAML = true
		case entry.Type == "file" && strings.HasPrefix(name, "readme"):
			signals.HasReadme = true
		}
	}

	return signals, nil
}

func listGitHubSkillEntries(fullName, token string) ([]githubRepoContentEntry, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/contents/skills", githubSearchAPIBase, fullName)
	body, _, err := doGitHubAPIRequest(apiURL, token)
	if err != nil {
		return nil, err
	}

	var entries []githubRepoContentEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub skills directory contents: %w", err)
	}

	return entries, nil
}

func matchGitHubSkillEntries(repo githubRepositoryItem, entries []githubRepoContentEntry, keyword string) []githubSkillDirectoryMatch {
	normalizedKeyword := normalizeGitHubSkillSearchText(keyword)
	if normalizedKeyword == "" {
		return nil
	}

	matches := make([]githubSkillDirectoryMatch, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		score, matchType := scoreGitHubSkillName(entry.Name, normalizedKeyword)
		if score <= 0 {
			continue
		}

		matches = append(matches, githubSkillDirectoryMatch{
			Repository: repo,
			SkillName:  entry.Name,
			SkillURL:   buildGitHubSkillURL(repo, entry.Name),
			Score:      score,
			MatchType:  matchType,
			MatchedBy:  "skill name",
		})
	}

	return matches
}

func scoreGitHubSkillDescriptionMatch(skillName, normalizedKeyword string) (int, string) {
	if normalizedKeyword == "" {
		return 0, ""
	}
	if normalizeGitHubSkillSearchText(skillName) == normalizedKeyword {
		return 70, "exact skill name via description hit"
	}
	return 45, "keyword found in skill description"
}

func upsertGitHubSkillMatch(matchMap map[string]githubSkillDirectoryMatch, match githubSkillDirectoryMatch) {
	key := match.Repository.FullName + "#" + match.SkillName
	existing, exists := matchMap[key]
	if !exists || match.Score > existing.Score {
		matchMap[key] = match
		return
	}
	if match.Score == existing.Score && existing.MatchedBy != "skill name" && match.MatchedBy == "skill name" {
		matchMap[key] = match
	}
}

func scoreGitHubSkillName(skillName, normalizedKeyword string) (int, string) {
	normalizedName := normalizeGitHubSkillSearchText(skillName)
	if normalizedName == normalizedKeyword {
		return 100, "exact skill name"
	}
	if strings.HasPrefix(normalizedName, normalizedKeyword) {
		return 80, "skill name prefix"
	}
	if strings.Contains(normalizedName, normalizedKeyword) {
		return 60, "skill name contains keyword"
	}

	terms := strings.Fields(normalizedKeyword)
	if len(terms) > 1 {
		allTermsMatch := true
		for _, term := range terms {
			if !strings.Contains(normalizedName, term) {
				allTermsMatch = false
				break
			}
		}
		if allTermsMatch {
			return 50, "all keyword terms matched"
		}
	}

	return 0, ""
}

func sortGitHubSkillMatches(matches []githubSkillDirectoryMatch) {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Repository.StargazersCount != matches[j].Repository.StargazersCount {
			return matches[i].Repository.StargazersCount > matches[j].Repository.StargazersCount
		}
		if matches[i].Repository.FullName != matches[j].Repository.FullName {
			return matches[i].Repository.FullName < matches[j].Repository.FullName
		}
		return matches[i].SkillName < matches[j].SkillName
	})
}

func normalizeGitHubSkillSearchText(value string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ")
	normalized := strings.ToLower(strings.TrimSpace(replacer.Replace(value)))
	return strings.Join(strings.Fields(normalized), " ")
}

func buildGitHubSkillURL(repo githubRepositoryItem, skillName string) string {
	branch := repo.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	return fmt.Sprintf("%s/tree/%s/skills/%s", repo.HTMLURL, branch, skillName)
}

func scoreGitHubSkillCandidate(repo githubRepositoryItem, signals githubRepoSignals, keyword string) (int, []string) {
	nameLower := strings.ToLower(repo.Name)
	fullNameLower := strings.ToLower(repo.FullName)
	descLower := strings.ToLower(repo.Description)
	keywordLower := strings.ToLower(strings.TrimSpace(keyword))
	text := strings.Join([]string{nameLower, fullNameLower, descLower, strings.Join(repo.Topics, " ")}, " ")

	score := 0
	reasons := make([]string, 0, 6)

	hasSkillMetadata := containsAny(text, []string{"skill", "skills", "copilot", "agent", "prompt", "mcp", "claude"})
	hasSkillTopics := len(repo.Topics) > 0 && containsAny(strings.ToLower(strings.Join(repo.Topics, " ")), []string{"skill", "copilot", "agent", "prompt", "mcp"})
	hasSkillStructure := signals.HasSkillsDir || signals.HasSkillMD || signals.HasSkillYAML

	if keywordLower != "" {
		if strings.Contains(nameLower, keywordLower) || strings.Contains(fullNameLower, keywordLower) {
			score += 4
			reasons = append(reasons, "keyword in repo name")
		} else if strings.Contains(descLower, keywordLower) {
			score += 2
			reasons = append(reasons, "keyword in description")
		}
	}

	if hasSkillMetadata {
		score += 2
		reasons = append(reasons, "skill-related metadata")
	}
	if signals.HasSkillsDir {
		score += 5
		reasons = append(reasons, "has skills/ directory")
	}
	if signals.HasSkillMD {
		score += 5
		reasons = append(reasons, "has SKILL.md")
	}
	if signals.HasSkillYAML {
		score += 4
		reasons = append(reasons, "has skill.yaml")
	}
	if hasSkillTopics {
		score += 2
		reasons = append(reasons, "skill-related topics")
	}
	if repo.Archived {
		score -= 3
		reasons = append(reasons, "archived repository")
	}
	if repo.Fork {
		score -= 1
		reasons = append(reasons, "fork")
	}

	if !hasSkillStructure && !hasSkillMetadata && !hasSkillTopics {
		return 0, nil
	}

	if score < 2 {
		return 0, nil
	}

	return score, uniqueStrings(reasons)
}

func prioritizeGitHubCandidates(candidates []githubSkillCandidate) (strong []githubSkillCandidate, fallback []githubSkillCandidate) {
	strong = make([]githubSkillCandidate, 0, len(candidates))
	fallback = make([]githubSkillCandidate, 0, len(candidates))

	for _, candidate := range candidates {
		if hasPrimarySkillProof(candidate.Signals) {
			strong = append(strong, candidate)
			continue
		}
		fallback = append(fallback, candidate)
	}

	sortGitHubCandidates(strong)
	sortGitHubCandidates(fallback)
	return strong, fallback
}

func sortGitHubCandidates(candidates []githubSkillCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Repository.StargazersCount > candidates[j].Repository.StargazersCount
	})
}

func hasPrimarySkillProof(signals githubRepoSignals) bool {
	return signals.HasSkillsDir || signals.HasSkillMD
}

func confidenceLabel(score int) string {
	switch {
	case score >= 10:
		return "high"
	case score >= 6:
		return "medium"
	default:
		return "low"
	}
}

func doGitHubAPIRequest(apiURL, token string) ([]byte, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "skillctl")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, resp.Header, nil
	}

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		resetValue := resp.Header.Get("X-RateLimit-Reset")
		if remaining == "0" && resetValue != "" {
			if resetUnix, err := strconv.ParseInt(resetValue, 10, 64); err == nil {
				return nil, resp.Header, fmt.Errorf("GitHub API rate limit exceeded; resets at %s", time.Unix(resetUnix, 0).Local().Format(time.RFC3339))
			}
		}
	}

	return nil, resp.Header, fmt.Errorf("GitHub API request failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func formatGitHubTimestamp(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05")
}
