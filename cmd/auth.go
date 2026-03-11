package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/DSC-PUCP/dsc-cli/internal"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Run:   runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Run:   runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Run:   runAuthStatus,
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) {
	if internal.IsAuthenticated() {
		creds, _ := internal.LoadCredentials()
		fmt.Printf("already logged in as @%s\n", creds.Username)
		fmt.Println("run: dsc auth logout  to switch accounts")
		return
	}

	// Zero-friction: reuse gh CLI token if available
	if token, err := internal.GHToken(); err == nil && token != "" {
		fmt.Println("detected gh CLI session, verifying membership...")
		if username, err := internal.VerifyOrgMembership(token); err == nil {
			internal.SaveCredentials(internal.Credentials{Token: token, Username: username})
			fmt.Printf("logged in as @%s\n", username)
			return
		}
	}

	// Fallback: Device Flow
	device, err := internal.RequestDeviceCode()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: could not connect to GitHub")
		os.Exit(1)
	}

	fmt.Printf("\nCopy your one-time code: %s\n\n", device.UserCode)
	fmt.Printf("Press Enter to open github.com in your browser...")
	fmt.Scanln()

	openBrowser(device.VerificationURI)
	fmt.Println("Waiting for authorization...")

	token, err := internal.PollForToken(device.DeviceCode, device.Interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}

	username, err := internal.VerifyOrgMembership(token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		fmt.Fprintln(os.Stderr, "only members of DSC-PUCP can use this tool")
		os.Exit(1)
	}

	if err := internal.SaveCredentials(internal.Credentials{Token: token, Username: username}); err != nil {
		fmt.Fprintln(os.Stderr, "error saving credentials: "+err.Error())
		os.Exit(1)
	}

	fmt.Printf("\nlogged in as @%s\n", username)
}

func runAuthLogout(cmd *cobra.Command, args []string) {
	if !internal.IsAuthenticated() {
		fmt.Fprintln(os.Stderr, "not logged in")
		os.Exit(1)
	}
	if err := internal.RemoveCredentials(); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
	fmt.Println("logged out")
}

func runAuthStatus(cmd *cobra.Command, args []string) {
	if !internal.IsAuthenticated() {
		fmt.Fprintln(os.Stderr, "not logged in")
		fmt.Fprintln(os.Stderr, "run: dsc auth login")
		os.Exit(1)
	}
	creds, _ := internal.LoadCredentials()
	fmt.Printf("logged in as @%s\n", creds.Username)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "linux":
		cmd, args = "xdg-open", []string{url}
	default:
		fmt.Printf("open this URL in your browser: %s\n", url)
		return
	}
	exec.Command(cmd, args...).Start()
}
