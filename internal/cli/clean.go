package cli

import (
	"fmt"
	"time"

	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/spf13/cobra"
)

var (
	cleanOlderThan string
	cleanDryRun    bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up old checkpoints",
	Long: `Removes checkpoints older than the specified duration.

By default, uses the retention period from config (default: 7 days).

Examples:
  safeshell clean                    # Use config retention period
  safeshell clean --older-than 3d    # Remove checkpoints older than 3 days
  safeshell clean --older-than 12h   # Remove checkpoints older than 12 hours
  safeshell clean --dry-run          # Show what would be deleted`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().StringVarP(&cleanOlderThan, "older-than", "o", "", "Duration (e.g., 7d, 24h, 30m)")
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "d", false, "Show what would be deleted without deleting")
}

func runClean(cmd *cobra.Command, args []string) error {
	var duration time.Duration

	if cleanOlderThan != "" {
		d, err := parseDuration(cleanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid duration: %s", cleanOlderThan)
		}
		duration = d
	} else {
		// Use config default
		cfg := config.Get()
		duration = time.Duration(cfg.RetentionDays) * 24 * time.Hour
	}

	if cleanDryRun {
		// Dry run - just show what would be deleted
		checkpoints, err := checkpoint.List()
		if err != nil {
			return err
		}

		cutoff := time.Now().Add(-duration)
		toDelete := 0

		for _, cp := range checkpoints {
			if cp.CreatedAt.Before(cutoff) {
				fmt.Printf("Would delete: %s (%s)\n", cp.ID, formatRelativeTime(cp.CreatedAt))
				toDelete++
			}
		}

		if toDelete == 0 {
			fmt.Println("No checkpoints to delete.")
		} else {
			fmt.Printf("\nWould delete %d checkpoint(s). Run without --dry-run to delete.\n", toDelete)
		}
		return nil
	}

	deleted, err := checkpoint.Clean(duration)
	if err != nil {
		return fmt.Errorf("failed to clean checkpoints: %w", err)
	}

	if deleted == 0 {
		fmt.Println("No checkpoints to clean.")
	} else {
		printSuccess(fmt.Sprintf("Deleted %d checkpoint(s)", deleted))
	}

	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Handle day suffix (not supported by time.ParseDuration)
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	return time.ParseDuration(s)
}
