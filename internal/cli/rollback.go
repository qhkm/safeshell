package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/rollback"
	"github.com/spf13/cobra"
)

var (
	rollbackLast        bool
	rollbackFiles       string
	rollbackInteractive bool
	rollbackToPath      string
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [checkpoint-id]",
	Short: "Restore files from a checkpoint",
	Long: `Restores files from a checkpoint to their original locations.

You can either specify a checkpoint ID, or use --last to rollback the most recent checkpoint.

Options:
  --files    Restore only specific files (comma-separated paths)
  -i         Interactive mode - select which files to restore
  --to       Restore files to a different directory instead of original locations

Examples:
  safeshell rollback --last
  safeshell rollback 2024-12-12T143022-a1b2c3
  safeshell rollback --last --files "src/main.go,config.json"
  safeshell rollback --last -i
  safeshell rollback --last --to ./backup/       # Restore to different directory
  safeshell rollback --last --to ~/Desktop/old   # Restore to home directory`,
	RunE: runRollback,
}

func init() {
	rollbackCmd.Flags().BoolVarP(&rollbackLast, "last", "l", false, "Rollback the most recent checkpoint")
	rollbackCmd.Flags().StringVarP(&rollbackFiles, "files", "f", "", "Restore only specific files (comma-separated)")
	rollbackCmd.Flags().BoolVarP(&rollbackInteractive, "interactive", "i", false, "Interactive mode - select files to restore")
	rollbackCmd.Flags().StringVarP(&rollbackToPath, "to", "t", "", "Restore to a different directory")
}

func runRollback(cmd *cobra.Command, args []string) error {
	var cp *checkpoint.Checkpoint
	var err error

	if rollbackLast {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return fmt.Errorf("no checkpoints found")
		}
	} else if len(args) > 0 {
		cp, err = checkpoint.Get(args[0])
		if err != nil {
			return fmt.Errorf("checkpoint not found: %s", args[0])
		}
	} else {
		return fmt.Errorf("please specify a checkpoint ID or use --last")
	}

	// Show checkpoint info
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Printf("Checkpoint: %s\n", cp.ID)
	fmt.Printf("Command:    %s\n", cp.Manifest.Command)
	fmt.Printf("Time:       %s\n", cp.Manifest.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if cp.Manifest.RolledBack {
		return fmt.Errorf("checkpoint has already been rolled back")
	}

	// Determine which files to restore
	var filesToRestore []string

	if rollbackInteractive {
		filesToRestore, err = interactiveFileSelect(cp)
		if err != nil {
			return err
		}
		if len(filesToRestore) == 0 {
			printWarning("No files selected. Rollback cancelled.")
			return nil
		}
	} else if rollbackFiles != "" {
		// Parse comma-separated file list
		filesToRestore = parseFileList(rollbackFiles, cp)
		if len(filesToRestore) == 0 {
			return fmt.Errorf("none of the specified files found in checkpoint")
		}
	}

	// Count files
	fileCount := 0
	if len(filesToRestore) > 0 {
		fileCount = len(filesToRestore)
	} else {
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}
	}

	if rollbackToPath != "" {
		fmt.Printf("Restoring %d file(s) to %s...\n", fileCount, rollbackToPath)
	} else {
		fmt.Printf("Restoring %d file(s)...\n", fileCount)
	}
	fmt.Println()

	// Perform rollback
	if rollbackToPath != "" {
		// Restore to different directory
		if len(filesToRestore) > 0 {
			if err := rollback.RollbackSelectiveToPath(cp, filesToRestore, rollbackToPath); err != nil {
				return err
			}
		} else {
			if err := rollback.RollbackToPath(cp, rollbackToPath); err != nil {
				return err
			}
		}
	} else if len(filesToRestore) > 0 {
		// Selective rollback
		if err := rollback.RollbackSelective(cp, filesToRestore); err != nil {
			return err
		}
	} else {
		// Full rollback
		if err := rollback.Rollback(cp); err != nil {
			return err
		}
	}

	printSuccess("Rollback complete!")
	return nil
}

func interactiveFileSelect(cp *checkpoint.Checkpoint) ([]string, error) {
	var files []checkpoint.FileEntry
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			files = append(files, f)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files in checkpoint")
	}

	// Get current working directory for relative path display
	cwd, _ := os.Getwd()

	color.New(color.FgWhite, color.Bold).Println("Select files to restore:")
	fmt.Println("Enter file numbers (comma-separated), 'all' for all files, or 'q' to quit")
	fmt.Println()

	for i, f := range files {
		displayPath := f.OriginalPath
		if cwd != "" {
			if rel, err := filepath.Rel(cwd, f.OriginalPath); err == nil && !strings.HasPrefix(rel, "..") {
				displayPath = rel
			}
		}

		// Check current status
		status := ""
		if _, err := os.Stat(f.OriginalPath); os.IsNotExist(err) {
			status = color.RedString(" [deleted]")
		} else {
			status = color.YellowString(" [modified]")
		}

		fmt.Printf("  [%d] %s%s\n", i+1, displayPath, status)
	}

	fmt.Println()
	fmt.Print("Selection: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "q" || input == "quit" {
		return nil, nil
	}

	if input == "all" || input == "a" {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.OriginalPath)
		}
		return paths, nil
	}

	// Parse selection
	var selectedPaths []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(files) {
				selectedPaths = append(selectedPaths, files[idx-1].OriginalPath)
			}
		}
	}

	return selectedPaths, nil
}

func parseFileList(fileList string, cp *checkpoint.Checkpoint) []string {
	// Build a map of checkpoint files for quick lookup
	checkpointFiles := make(map[string]bool)
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			checkpointFiles[f.OriginalPath] = true
			// Also add relative paths
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, f.OriginalPath); err == nil {
					checkpointFiles[rel] = true
				}
			}
		}
	}

	var matched []string
	parts := strings.Split(fileList, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try as-is
		if checkpointFiles[part] {
			// Convert back to absolute path if needed
			if filepath.IsAbs(part) {
				matched = append(matched, part)
			} else {
				if cwd, err := os.Getwd(); err == nil {
					matched = append(matched, filepath.Join(cwd, part))
				}
			}
			continue
		}

		// Try as absolute path
		absPath, _ := filepath.Abs(part)
		if checkpointFiles[absPath] {
			matched = append(matched, absPath)
			continue
		}

		// Partial match - check if path ends with the given string
		for cpPath := range checkpointFiles {
			if strings.HasSuffix(cpPath, "/"+part) || strings.HasSuffix(cpPath, part) {
				matched = append(matched, cpPath)
				break
			}
		}
	}

	return matched
}
