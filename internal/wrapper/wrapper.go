package wrapper

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/safeshell/safeshell/internal/checkpoint"
)

// Wrap executes a command with automatic checkpoint creation
func Wrap(cmdName string, args []string) error {
	// Check if command is supported
	cmdDef, ok := GetCommand(cmdName)
	if !ok {
		// Not a wrapped command, just execute it directly
		return executeCommand(cmdName, args)
	}

	// Parse arguments to get target paths
	targets, err := cmdDef.Parser(args)
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Filter targets to only existing paths
	var existingTargets []string
	for _, target := range targets {
		if _, err := os.Stat(target); err == nil {
			existingTargets = append(existingTargets, target)
		}
	}

	// Create checkpoint if there are targets to backup
	if len(existingTargets) > 0 {
		fullCommand := cmdName + " " + strings.Join(args, " ")
		cp, err := checkpoint.Create(fullCommand, existingTargets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create checkpoint: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[safeshell] Checkpoint created: %s\n", cp.ID)
		}
	}

	// Execute the actual command
	return executeCommand(cmdName, args)
}

func executeCommand(cmdName string, args []string) error {
	// Find the real command (not our alias)
	cmdPath, err := findRealCommand(cmdName)
	if err != nil {
		return fmt.Errorf("command not found: %s", cmdName)
	}

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// findRealCommand finds the actual binary path, skipping any safeshell wrappers
func findRealCommand(cmdName string) (string, error) {
	// Common binary locations
	searchPaths := []string{
		"/bin/" + cmdName,
		"/usr/bin/" + cmdName,
		"/usr/local/bin/" + cmdName,
		"/opt/homebrew/bin/" + cmdName,
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Fall back to PATH lookup
	return exec.LookPath(cmdName)
}
