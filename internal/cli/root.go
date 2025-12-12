package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "safeshell",
		Short: "Safe shell operations with automatic checkpoints",
		Long: `SafeShell creates automatic filesystem checkpoints before destructive
operations, enabling safe autonomous agent execution with easy rollback.

Let agents run freely. Everything is reversible.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return config.Init()
		},
	}

	version = "0.1.2"
)

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(wrapCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("safeshell v%s\n", version)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

// Helper functions for colored output
func printSuccess(msg string) {
	color.Green("✓ %s", msg)
}

func printWarning(msg string) {
	color.Yellow("! %s", msg)
}

func printError(msg string) {
	color.Red("✗ %s", msg)
}

func printInfo(msg string) {
	color.Cyan("→ %s", msg)
}

func exitWithError(msg string) {
	printError(msg)
	os.Exit(1)
}
