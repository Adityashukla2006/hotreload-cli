package cmd

import (
	"fmt"
	"os"

	"hotreload/internal"

	"github.com/spf13/cobra"
)

var (
	rootDir  string
	buildCmd string
	execCmd  string
)

var rootCmd = &cobra.Command{
	Use:   "hotreload",
	Short: "Watch a project for changes and automatically rebuild and restart your server",
	Long: `hotreload watches a directory for file changes and automatically rebuilds
and restarts your server. Ideal for Go development with instant feedback.

Example:
  hotreload --root ./myproject --build "go build -o ./bin/server ./cmd/server" --exec "./bin/server"`,
	RunE: runHotReload,
}

func init() {
	rootCmd.Flags().StringVarP(&rootDir, "root", "r", ".", "Directory to watch for file changes (recursive)")
	rootCmd.Flags().StringVarP(&buildCmd, "build", "b", "", "Command to build the project when changes are detected")
	rootCmd.Flags().StringVarP(&execCmd, "exec", "e", "", "Command to run the server after a successful build")

	rootCmd.MarkFlagRequired("build")
	rootCmd.MarkFlagRequired("exec")
}

func runHotReload(cmd *cobra.Command, args []string) error {
	if buildCmd == "" || execCmd == "" {
		return fmt.Errorf("--build and --exec are required")
	}

	sup := internal.NewSupervisor(rootDir, buildCmd, execCmd)
	return sup.Run()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
