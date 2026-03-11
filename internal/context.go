package internal

import (
	"os/exec"
	"strings"
)

type RepoContext struct {
	InGitRepo bool
	RemoteURL string
	Branch    string
	IsDSCRepo bool
	RepoName  string
}

func GetRepoContext() RepoContext {
	ctx := RepoContext{}

	// Check if we're in a git repo
	if _, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		return ctx
	}
	ctx.InGitRepo = true

	// Get current branch
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		ctx.Branch = strings.TrimSpace(string(out))
	}

	// Get remote URL
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		remote := strings.TrimSpace(string(out))
		ctx.RemoteURL = remote
		if strings.Contains(remote, Org) {
			ctx.IsDSCRepo = true
			ctx.RepoName = extractRepoName(remote)
		}
	}

	return ctx
}

func extractRepoName(remote string) string {
	// Handle both SSH and HTTPS URLs
	remote = strings.TrimSuffix(remote, ".git")
	parts := strings.Split(remote, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
