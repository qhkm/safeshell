package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/qhkm/safeshell/internal/util"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show safeshell status and statistics",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Header
	color.New(color.FgCyan, color.Bold).Println("SafeShell Status")
	fmt.Println("────────────────────────────────")

	// Configuration
	fmt.Printf("Config directory: %s\n", cfg.SafeShellDir)
	fmt.Printf("Retention:        %d days\n", cfg.RetentionDays)
	fmt.Printf("Max checkpoints:  %d\n", cfg.MaxCheckpoints)
	fmt.Println()

	// Checkpoint statistics
	checkpoints, err := checkpoint.List()
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	fmt.Printf("Total checkpoints: %d\n", len(checkpoints))

	if len(checkpoints) > 0 {
		// Calculate total size
		var totalSize int64
		var totalFiles int
		rolledBack := 0

		for _, cp := range checkpoints {
			size, _ := checkpoint.GetDiskUsage(cp.FilesDir)
			totalSize += size

			for _, f := range cp.Manifest.Files {
				if !f.IsDir {
					totalFiles++
				}
			}

			if cp.Manifest.RolledBack {
				rolledBack++
			}
		}

		fmt.Printf("Total files backed up: %d\n", totalFiles)
		fmt.Printf("Storage used: %s\n", util.FormatBytes(totalSize))
		fmt.Printf("Rolled back: %d\n", rolledBack)
		fmt.Println()

		// Latest checkpoint
		latest := checkpoints[0]
		color.New(color.FgWhite, color.Bold).Println("Latest checkpoint:")
		fmt.Printf("  ID:      %s\n", latest.ID)
		fmt.Printf("  Command: %s\n", latest.Manifest.Command)
		fmt.Printf("  Time:    %s\n", util.FormatTimeAgo(latest.CreatedAt))
	} else {
		fmt.Println()
		fmt.Println("No checkpoints yet. Run 'safeshell init' to set up automatic checkpoints.")
	}

	return nil
}
