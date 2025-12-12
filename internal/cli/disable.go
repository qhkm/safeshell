package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Remove shell aliases and revert to normal binaries",
	Long: `Removes SafeShell aliases from your shell configuration.

After running this command, rm/mv/cp/chmod/chown will use the
original system binaries without SafeShell protection.

Your checkpoints and SafeShell installation remain intact.
Run 'safeshell enable' or 'safeshell init' to re-enable protection.

Examples:
  safeshell disable     # Remove aliases from shell config
  safeshell enable      # Re-enable protection later`,
	RunE: runDisable,
}

func init() {
	rootCmd.AddCommand(disableCmd)
}

func runDisable(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Detect shell (same logic as init.go)
	shell := os.Getenv("SHELL")
	var rcFile string
	var shellName string

	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = filepath.Join(homeDir, ".zshrc")
		shellName = "zsh"
	case strings.Contains(shell, "bash"):
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			rcFile = bashProfile
		} else {
			rcFile = filepath.Join(homeDir, ".bashrc")
		}
		shellName = "bash"
	default:
		rcFile = filepath.Join(homeDir, ".bashrc")
		shellName = "bash"
	}

	// Check if SafeShell is installed
	if !containsSafeShell(rcFile) {
		fmt.Printf("SafeShell aliases not found in %s\n", rcFile)
		fmt.Println("Nothing to disable.")
		return nil
	}

	// Remove the alias block
	if err := removeAliasBlock(rcFile); err != nil {
		return fmt.Errorf("failed to remove aliases: %w", err)
	}

	printSuccess(fmt.Sprintf("SafeShell aliases removed from %s", rcFile))
	fmt.Println()
	fmt.Println("To apply changes, run:")
	fmt.Printf("  source %s\n", rcFile)
	fmt.Println()
	fmt.Println("Or restart your terminal.")
	fmt.Println()
	fmt.Printf("Your %s shell will now use the original system binaries.\n", shellName)
	fmt.Println("Your checkpoints are still available via 'safeshell list'.")
	fmt.Println()
	fmt.Println("To re-enable protection:")
	fmt.Println("  safeshell enable")

	return nil
}

func removeAliasBlock(rcFile string) error {
	// Read the file
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inBlock := false

	for _, line := range lines {
		// Check for start marker
		if strings.Contains(line, "# SafeShell") && !strings.Contains(line, "# End SafeShell") {
			inBlock = true
			continue
		}

		// Check for end marker
		if strings.Contains(line, "# End SafeShell") {
			inBlock = false
			continue
		}

		// Keep lines outside the block
		if !inBlock {
			newLines = append(newLines, line)
		}
	}

	// Remove trailing empty lines that might have been left
	for len(newLines) > 0 && newLines[len(newLines)-1] == "" {
		// Keep at least one trailing newline check
		if len(newLines) > 1 && newLines[len(newLines)-2] == "" {
			newLines = newLines[:len(newLines)-1]
		} else {
			break
		}
	}

	// Write the file back
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(rcFile, []byte(newContent), 0644)
}
