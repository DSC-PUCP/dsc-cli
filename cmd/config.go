package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DSC-PUCP/dsc-cli/internal"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Set a configuration value",
	Example: `  dsc config set gemini-key
  dsc config set openai-key`,
	Args: cobra.ExactArgs(1),
	Run:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	Run:   runConfigList,
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration value",
	Example: `  dsc config unset gemini-key`,
	Args: cobra.ExactArgs(1),
	Run:  runConfigUnset,
}

func init() {
	configCmd.AddCommand(configSetCmd, configListCmd, configUnsetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]

	creds, _ := internal.LoadCredentials()

	switch key {
	case "gemini-key":
		fmt.Fprintf(os.Stderr, "Enter your Gemini API key: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading key")
			os.Exit(1)
		}
		value := strings.TrimSpace(string(raw))
		if value == "" {
			fmt.Fprintln(os.Stderr, "error: key cannot be empty")
			os.Exit(1)
		}
		creds.GeminiKey = value
		if err := internal.SaveCredentials(creds); err != nil {
			fmt.Fprintln(os.Stderr, "error saving config: "+err.Error())
			os.Exit(1)
		}
		fmt.Println("gemini-key set")

	default:
		fmt.Fprintf(os.Stderr, "error: unknown key %q\n", key)
		fmt.Fprintln(os.Stderr, "available keys: gemini-key")
		os.Exit(1)
	}
}

func runConfigList(cmd *cobra.Command, args []string) {
	creds, err := internal.LoadCredentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "not configured yet — run: dsc auth login")
		os.Exit(1)
	}

	geminiStatus := "not set"
	if creds.GeminiKey != "" {
		geminiStatus = "set"
	}

	fmt.Printf("github-user   %s\n", creds.Username)
	fmt.Printf("gemini-key    %s\n", geminiStatus)
}

func runConfigUnset(cmd *cobra.Command, args []string) {
	key := args[0]
	creds, err := internal.LoadCredentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "no config found")
		os.Exit(1)
	}

	switch key {
	case "gemini-key":
		creds.GeminiKey = ""
		if err := internal.SaveCredentials(creds); err != nil {
			fmt.Fprintln(os.Stderr, "error: "+err.Error())
			os.Exit(1)
		}
		fmt.Println("gemini-key removed")
	default:
		fmt.Fprintf(os.Stderr, "error: unknown key %q\n", key)
		os.Exit(1)
	}
}
