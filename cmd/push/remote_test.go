package pushcmd

import "testing"

func TestDetectRemoteProviderByHost(t *testing.T) {
	provider, err := detectRemoteProvider("https", "github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "github" {
		t.Fatalf("expected github, got %s", provider)
	}

	provider, err = detectRemoteProvider("https", "gitlab.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "gitlab" {
		t.Fatalf("expected gitlab, got %s", provider)
	}
}

func TestDetectRemoteProviderByProbe(t *testing.T) {
	originalProbe := remoteProviderProbe
	defer func() { remoteProviderProbe = originalProbe }()

	remoteProviderProbe = func(scheme, host string) (string, error) {
		if host == "code.cnworkshop.xyz" {
			return "gitlab", nil
		}
		return "", nil
	}

	provider, err := detectRemoteProvider("https", "code.cnworkshop.xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "gitlab" {
		t.Fatalf("expected gitlab, got %s", provider)
	}
}

func TestDetectDefaultBranchFromGitOutputParsing(t *testing.T) {
	originalRunner := gitCommandRunner
	defer func() { gitCommandRunner = originalRunner }()

	gitCommandRunner = func(repoDir string, args ...string) (string, error) {
		return "ref: refs/heads/master\tHEAD\n4050e3b441c40ba6687f88c1e2782ad7d1303b81\tHEAD\n", nil
	}

	branch, err := detectDefaultBranchFromGit("git@example.com:org/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Fatalf("expected master, got %s", branch)
	}
}
