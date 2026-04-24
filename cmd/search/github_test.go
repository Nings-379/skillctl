package searchcmd

import "testing"

func TestScoreGitHubSkillName(t *testing.T) {
	tests := []struct {
		name         string
		skillName    string
		keyword      string
		wantScore    int
		wantMatchTyp string
	}{
		{name: "exact match", skillName: "pdf", keyword: "pdf", wantScore: 100, wantMatchTyp: "exact skill name"},
		{name: "prefix match", skillName: "pdf-tools", keyword: "pdf", wantScore: 80, wantMatchTyp: "skill name prefix"},
		{name: "contains match", skillName: "advanced-pdf-tools", keyword: "pdf", wantScore: 60, wantMatchTyp: "skill name contains keyword"},
		{name: "multi term match", skillName: "agent governance toolkit", keyword: "agent toolkit", wantScore: 50, wantMatchTyp: "all keyword terms matched"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, matchType := scoreGitHubSkillName(tt.skillName, normalizeGitHubSkillSearchText(tt.keyword))
			if score != tt.wantScore {
				t.Fatalf("expected score %d, got %d", tt.wantScore, score)
			}
			if matchType != tt.wantMatchTyp {
				t.Fatalf("expected match type %q, got %q", tt.wantMatchTyp, matchType)
			}
		})
	}
}

func TestScoreGitHubSkillNameRejectWeakMatch(t *testing.T) {
	score, matchType := scoreGitHubSkillName("random-tools", normalizeGitHubSkillSearchText("pdf"))
	if score != 0 {
		t.Fatalf("expected weak match to be rejected, got score %d", score)
	}
	if matchType != "" {
		t.Fatalf("expected empty match type, got %q", matchType)
	}
}

func TestMatchGitHubSkillEntriesOnlyMatchesDirectories(t *testing.T) {
	repo := githubRepositoryItem{
		FullName:      "github/awesome-copilot",
		HTMLURL:       "https://github.com/github/awesome-copilot",
		DefaultBranch: "main",
	}
	entries := []githubRepoContentEntry{
		{Name: "pdf", Type: "dir"},
		{Name: "agent-governance", Type: "dir"},
		{Name: "README.md", Type: "file"},
	}

	matches := matchGitHubSkillEntries(repo, entries, "pdf")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].SkillName != "pdf" {
		t.Fatalf("expected matched skill name pdf, got %s", matches[0].SkillName)
	}
	if matches[0].SkillURL != "https://github.com/github/awesome-copilot/tree/main/skills/pdf" {
		t.Fatalf("unexpected skill url: %s", matches[0].SkillURL)
	}
}

func TestSortGitHubSkillMatches(t *testing.T) {
	matches := []githubSkillDirectoryMatch{
		{Repository: githubRepositoryItem{FullName: "repo/b", StargazersCount: 5}, SkillName: "pdf-tools", Score: 60},
		{Repository: githubRepositoryItem{FullName: "repo/a", StargazersCount: 100}, SkillName: "pdf", Score: 100},
		{Repository: githubRepositoryItem{FullName: "repo/c", StargazersCount: 200}, SkillName: "pdf-agent", Score: 80},
	}

	sortGitHubSkillMatches(matches)
	if matches[0].SkillName != "pdf" {
		t.Fatalf("expected highest score exact match first, got %s", matches[0].SkillName)
	}
	if matches[1].SkillName != "pdf-agent" {
		t.Fatalf("expected prefix match second, got %s", matches[1].SkillName)
	}
}

func TestGitHubSearchFetchLimit(t *testing.T) {
	if got := githubSearchFetchLimit(3); got != 12 {
		t.Fatalf("expected minimum fetch limit 12, got %d", got)
	}
	if got := githubSearchFetchLimit(10); got != 40 {
		t.Fatalf("expected expanded fetch limit 40, got %d", got)
	}
	if got := githubSearchFetchLimit(20); got != 50 {
		t.Fatalf("expected capped fetch limit 50, got %d", got)
	}
}

func TestNormalizeGitHubSkillSearchText(t *testing.T) {
	got := normalizeGitHubSkillSearchText("  agent_governance-toolkit ")
	if got != "agent governance toolkit" {
		t.Fatalf("unexpected normalized text: %q", got)
	}
}

func TestExtractGitHubSkillNameFromPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		want      string
		wantFound bool
	}{
		{name: "skill markdown path", path: "skills/agent-governance/SKILL.md", want: "agent-governance", wantFound: true},
		{name: "nested skill file path", path: "skills/agent-governance/examples/demo.md", want: "agent-governance", wantFound: true},
		{name: "non skills path", path: "docs/agent-governance/SKILL.md", wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := extractGitHubSkillNameFromPath(tt.path)
			if found != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, found)
			}
			if got != tt.want {
				t.Fatalf("expected skill name %q, got %q", tt.want, got)
			}
		})
	}
}

func TestScoreGitHubSkillDescriptionMatch(t *testing.T) {
	score, matchType := scoreGitHubSkillDescriptionMatch("agent-governance", normalizeGitHubSkillSearchText("governance"))
	if score != 45 {
		t.Fatalf("expected description match score 45, got %d", score)
	}
	if matchType != "keyword found in skill description" {
		t.Fatalf("unexpected match type: %q", matchType)
	}
}

func TestUpsertGitHubSkillMatchPrefersNameMatch(t *testing.T) {
	matchMap := map[string]githubSkillDirectoryMatch{}
	repo := githubRepositoryItem{FullName: "github/awesome-copilot"}

	upsertGitHubSkillMatch(matchMap, githubSkillDirectoryMatch{
		Repository: repo,
		SkillName:  "agent-governance",
		Score:      45,
		MatchType:  "keyword found in skill description",
		MatchedBy:  "skill description",
	})
	upsertGitHubSkillMatch(matchMap, githubSkillDirectoryMatch{
		Repository: repo,
		SkillName:  "agent-governance",
		Score:      80,
		MatchType:  "skill name prefix",
		MatchedBy:  "skill name",
	})

	match := matchMap["github/awesome-copilot#agent-governance"]
	if match.MatchedBy != "skill name" {
		t.Fatalf("expected skill-name match to win, got %q", match.MatchedBy)
	}
	if match.Score != 80 {
		t.Fatalf("expected score 80, got %d", match.Score)
	}
}
