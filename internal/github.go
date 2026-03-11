package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

const Org = "DSC-PUCP"

type AuthStatus struct {
	Authenticated bool
	Username      string
}

type OrgMembership struct {
	Role  string `json:"role"`
	State string `json:"state"`
}

type Repo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	URL         string `json:"html_url"`
	UpdatedAt   string `json:"updatedAt"`
}

func CheckGHInstalled() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com")
	}
	return nil
}

func GetAuthStatus() (AuthStatus, error) {
	out, err := exec.Command("gh", "auth", "status", "--hostname", "github.com").CombinedOutput()
	if err != nil {
		return AuthStatus{}, nil
	}

	output := string(out)
	if strings.Contains(output, "Logged in") {
		username := extractUsername(output)
		return AuthStatus{Authenticated: true, Username: username}, nil
	}

	return AuthStatus{}, nil
}

func GetOrgMembership() (OrgMembership, error) {
	out, err := exec.Command("gh", "api",
		fmt.Sprintf("/user/memberships/orgs/%s", Org),
		"--jq", `{role: .role, state: .state}`,
	).Output()
	if err != nil {
		return OrgMembership{}, fmt.Errorf("not a member of %s", Org)
	}

	var m OrgMembership
	if err := json.Unmarshal(out, &m); err != nil {
		return OrgMembership{}, err
	}
	return m, nil
}

func ListOrgRepos() ([]Repo, error) {
	out, err := exec.Command("gh", "repo", "list", Org,
		"--json", "name,description,visibility,url,updatedAt",
		"--limit", "10",
	).Output()
	if err != nil {
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// FetchDocs retrieves all .md files from DSC-PUCP/docs repo via GitHub API
func FetchDocs(token string) (string, error) {
	// 1. Get repo tree recursively
	req, _ := http.NewRequest("GET",
		"https://api.github.com/repos/"+Org+"/docs/git/trees/main?recursive=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("could not access %s/docs repo (status %d)", Org, resp.StatusCode)
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return "", err
	}

	// 2. Fetch content of each .md file
	var docs strings.Builder
	for _, item := range tree.Tree {
		if item.Type != "blob" || !strings.HasSuffix(item.Path, ".md") {
			continue
		}

		req2, _ := http.NewRequest("GET",
			fmt.Sprintf("https://api.github.com/repos/%s/docs/contents/%s", Org, item.Path), nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("Accept", "application/vnd.github.raw+json")

		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()

		docs.WriteString(fmt.Sprintf("--- %s ---\n", item.Path))
		docs.Write(body)
		docs.WriteString("\n\n")
	}

	if docs.Len() == 0 {
		return "", fmt.Errorf("no docs found in %s/docs", Org)
	}

	return docs.String(), nil
}

func extractUsername(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Logged in") {
			parts := strings.Split(line, "account")
			if len(parts) > 1 {
				name := strings.TrimSpace(parts[1])
				name = strings.Split(name, " ")[0]
				return name
			}
		}
	}
	return "unknown"
}
