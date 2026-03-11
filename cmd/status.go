package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DSC-PUCP/dsc-cli/internal"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current DSC-PUCP environment status",
	Example: `  dsc status
  dsc status --json`,
	Run: runStatus,
}

var jsonOutput bool

func init() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	if err := internal.CheckGHInstalled(); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}

	auth, err := internal.GetAuthStatus()
	if err != nil || !auth.Authenticated {
		fmt.Fprintln(os.Stderr, "error: not authenticated with GitHub")
		fmt.Fprintln(os.Stderr, "run: gh auth login")
		os.Exit(1)
	}

	membership, memberErr := internal.GetOrgMembership()
	repos, reposErr := internal.ListOrgRepos()
	repoCtx := internal.GetRepoContext()

	if jsonOutput {
		printJSON(auth, membership, repos, repoCtx)
		return
	}

	internal.PrintLogo()
	printHuman(auth, membership, memberErr, repos, reposErr, repoCtx)
}

func printHuman(
	auth internal.AuthStatus,
	membership internal.OrgMembership,
	memberErr error,
	repos []internal.Repo,
	reposErr error,
	ctx internal.RepoContext,
) {
	// GitHub auth
	fmt.Printf("GitHub     %s (%s)\n", check(true), auth.Username)

	// Org membership
	if memberErr != nil {
		fmt.Printf("Org        %s not a member of %s\n", cross(), internal.Org)
	} else {
		fmt.Printf("Org        %s %s  role: %s\n", check(true), internal.Org, membership.Role)
	}

	// Current repo context (only show if inside a DSC repo)
	if ctx.IsDSCRepo {
		fmt.Println()
		fmt.Printf("Context    %s  branch: %s\n", ctx.RepoName, ctx.Branch)
	}

	// Org repos
	fmt.Println()
	if reposErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch repos: %s\n", reposErr)
	} else {
		fmt.Printf("Repos in %s:\n", internal.Org)
		for _, r := range repos {
			desc := r.Description
			if desc == "" {
				desc = "-"
			}
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			visibility := "public"
			if r.Visibility == "private" {
				visibility = "private"
			}
			fmt.Printf("  %-30s  %-8s  %s\n", r.Name, visibility, desc)
		}
	}

}

func printJSON(auth internal.AuthStatus, membership internal.OrgMembership, repos []internal.Repo, ctx internal.RepoContext) {
	lines := []string{
		`{`,
		fmt.Sprintf(`  "github_user": %q,`, auth.Username),
		fmt.Sprintf(`  "org": %q,`, internal.Org),
		fmt.Sprintf(`  "role": %q,`, membership.Role),
		fmt.Sprintf(`  "in_dsc_repo": %v,`, ctx.IsDSCRepo),
		fmt.Sprintf(`  "repo": %q,`, ctx.RepoName),
		fmt.Sprintf(`  "branch": %q,`, ctx.Branch),
		fmt.Sprintf(`  "repo_count": %d`, len(repos)),
		`}`,
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func check(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func cross() string {
	return "✗"
}
