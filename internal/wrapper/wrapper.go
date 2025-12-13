package wrapper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/util"
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

// WrapDryRun shows what would be backed up without creating checkpoint or executing command
func WrapDryRun(cmdName string, args []string) error {
	fullCommand := cmdName + " " + strings.Join(args, " ")

	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("Dry Run - No changes will be made")
	fmt.Println()
	fmt.Printf("Command: %s\n", fullCommand)
	fmt.Println()

	// Check if command is supported
	cmdDef, ok := GetCommand(cmdName)
	if !ok {
		color.Yellow("⚠ Command '%s' is not wrapped by SafeShell\n", cmdName)
		fmt.Println("  This command will execute without creating a checkpoint.")
		fmt.Println()
		fmt.Println("Wrapped commands: rm, mv, cp, chmod, chown")
		return nil
	}

	fmt.Printf("Risk level: %s\n", cmdDef.RiskLevel)
	fmt.Println()

	// Parse arguments to get target paths
	targets, err := cmdDef.Parser(args)
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	if len(targets) == 0 {
		color.Yellow("⚠ No target files/directories detected\n")
		fmt.Println("  No checkpoint would be created.")
		return nil
	}

	color.New(color.FgWhite, color.Bold).Println("Files/directories to backup:")
	fmt.Println()

	var totalSize int64
	var totalFiles int
	existingCount := 0

	for _, target := range targets {
		info, err := os.Stat(target)
		if os.IsNotExist(err) {
			color.New(color.FgHiBlack).Printf("  ✗ %s (does not exist - will be skipped)\n", target)
			continue
		}
		if err != nil {
			color.New(color.FgRed).Printf("  ✗ %s (error: %v)\n", target, err)
			continue
		}

		existingCount++

		if info.IsDir() {
			// Count files in directory
			dirFiles := 0
			var dirSize int64
			filepath.Walk(target, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return err
				}
				dirFiles++
				dirSize += fi.Size()
				return nil
			})
			totalFiles += dirFiles
			totalSize += dirSize
			color.Green("  ✓ %s/ (directory, %d files, %s)\n", target, dirFiles, util.FormatBytes(dirSize))
		} else {
			totalFiles++
			totalSize += info.Size()
			color.Green("  ✓ %s (%s)\n", target, util.FormatBytes(info.Size()))
		}
	}

	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("Summary:")
	if existingCount > 0 {
		fmt.Printf("  • %d path(s) would be backed up\n", existingCount)
		fmt.Printf("  • %d total file(s)\n", totalFiles)
		fmt.Printf("  • %s total size\n", util.FormatBytes(totalSize))
		fmt.Println()
		color.Green("✓ A checkpoint would be created before executing this command\n")
	} else {
		color.Yellow("⚠ No existing files to backup - no checkpoint would be created\n")
	}

	fmt.Println()
	fmt.Println("To execute this command for real, run without --dry-run:")
	color.Cyan("  safeshell wrap %s\n", fullCommand)

	return nil
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
