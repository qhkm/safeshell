package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"enable"},
	Short:   "Setup shell aliases for safeshell",
	Long: `Adds shell aliases to your shell configuration file (.zshrc or .bashrc).
This makes rm, mv, cp, chmod, and chown automatically create checkpoints.

Use 'safeshell disable' to remove the aliases and revert to normal binaries.`,
	RunE: runInit,
}

const aliasBlock = `
# SafeShell - Automatic filesystem checkpoints
# Added by 'safeshell init'
alias rm='safeshell wrap rm'
alias mv='safeshell wrap mv'
alias cp='safeshell wrap cp'
alias chmod='safeshell wrap chmod'
alias chown='safeshell wrap chown'
# End SafeShell
`

func runInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Detect shell
	shell := os.Getenv("SHELL")
	var rcFile string
	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = filepath.Join(homeDir, ".zshrc")
	case strings.Contains(shell, "bash"):
		// Check for .bash_profile on macOS
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			rcFile = bashProfile
		} else {
			rcFile = filepath.Join(homeDir, ".bashrc")
		}
	default:
		rcFile = filepath.Join(homeDir, ".bashrc")
	}

	// Check if already initialized
	if containsSafeShell(rcFile) {
		printWarning(fmt.Sprintf("SafeShell aliases already exist in %s", rcFile))
		fmt.Println("To re-initialize, first remove the existing SafeShell block from your shell config.")
		return nil
	}

	// Append aliases to shell config
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcFile, err)
	}
	defer f.Close()

	if _, err := f.WriteString(aliasBlock); err != nil {
		return fmt.Errorf("failed to write aliases: %w", err)
	}

	printSuccess(fmt.Sprintf("Added SafeShell aliases to %s", rcFile))
	fmt.Println()
	fmt.Println("To activate, run:")
	fmt.Printf("  source %s\n", rcFile)
	fmt.Println()
	fmt.Println("Or start a new terminal session.")
	fmt.Println()
	fmt.Println("The following commands will now create automatic checkpoints:")
	fmt.Println("  rm, mv, cp, chmod, chown")
	fmt.Println()
	fmt.Println("Use 'safeshell list' to view checkpoints")
	fmt.Println("Use 'safeshell rollback <id>' to restore files")

	return nil
}

func containsSafeShell(rcFile string) bool {
	f, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "SafeShell") {
			return true
		}
	}
	return false
}
