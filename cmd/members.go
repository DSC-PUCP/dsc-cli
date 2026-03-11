package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/DSC-PUCP/dsc-cli/internal"
	"github.com/spf13/cobra"
)

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage DSC-PUCP members",
}

var membersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all members of DSC-PUCP",
	Example: `  dsc members list
  dsc members list --json`,
	Run: runMembersList,
}

var membersJsonOutput bool

func init() {
	membersListCmd.Flags().BoolVar(&membersJsonOutput, "json", false, "Output as JSON")
	membersCmd.AddCommand(membersListCmd)
	rootCmd.AddCommand(membersCmd)
}

func runMembersList(cmd *cobra.Command, args []string) {
	creds, err := internal.LoadCredentials()
	if err != nil || creds.Token == "" {
		fmt.Fprintln(os.Stderr, "error: not authenticated")
		fmt.Fprintln(os.Stderr, "run: dsc auth login")
		os.Exit(1)
	}

	members, err := fetchMembers(creds.Token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}

	if membersJsonOutput {
		data, _ := json.MarshalIndent(members, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("%d members in DSC-PUCP\n\n", len(members))
	for _, m := range members {
		fmt.Printf("  @%-20s  %s\n", m.Login, m.Name)
	}
}

type member struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

func fetchMembers(token string) ([]member, error) {
	req, _ := http.NewRequest("GET",
		"https://api.github.com/orgs/"+internal.Org+"/members?per_page=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var members []member
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, err
	}

	return members, nil
}
