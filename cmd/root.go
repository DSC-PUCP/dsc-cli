package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dsc",
	Short: "CLI for DSC-PUCP",
	Long:  "Developer Student Club PUCP - Command line tool for managing projects and resources.",
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		checkUpdate()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkUpdate() {
	if Version == "dev" {
		return
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/DSC-PUCP/dsc-cli/releases/latest")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest != "" && latest != Version {
		fmt.Fprintf(os.Stderr, "\n  Update available: %s → %s\n", Version, latest)
		fmt.Fprintf(os.Stderr, "  Run: curl -fsSL https://raw.githubusercontent.com/DSC-PUCP/dsc-cli/main/install.sh | sh\n\n")
	}
}
