package cli

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/qhkm/safeshell/internal/util"
	"github.com/spf13/cobra"
)

var (
	cleanOlderThan  string
	cleanDryRun     bool
	cleanCompress   bool
	cleanKeepCount  int
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up old checkpoints",
	Long: `Removes or compresses checkpoints older than the specified duration.

By default, uses the retention period from config (default: 7 days).

Options:
  --older-than    Duration threshold for cleanup (e.g., 7d, 24h)
  --compress      Compress instead of delete (saves 60-80% space)
  --keep          Keep at least N most recent checkpoints
  --dry-run       Show what would be done without doing it

Examples:
  safeshell clean                      # Delete checkpoints older than config retention
  safeshell clean --older-than 3d      # Delete checkpoints older than 3 days
  safeshell clean --compress           # Compress old checkpoints instead of deleting
  safeshell clean --older-than 1d --compress  # Compress checkpoints older than 1 day
  safeshell clean --keep 10            # Delete all but the 10 most recent
  safeshell clean --dry-run            # Show what would be deleted`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().StringVarP(&cleanOlderThan, "older-than", "o", "", "Duration (e.g., 7d, 24h, 30m)")
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "d", false, "Show what would be done without doing it")
	cleanCmd.Flags().BoolVarP(&cleanCompress, "compress", "c", false, "Compress old checkpoints instead of deleting")
	cleanCmd.Flags().IntVarP(&cleanKeepCount, "keep", "k", 0, "Keep at least N most recent checkpoints")
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

	// Handle --keep option
	if cleanKeepCount > 0 {
		return cleanKeepN(cleanKeepCount, cleanDryRun, cleanCompress)
	}

	// Handle --compress option
	if cleanCompress {
		return cleanWithCompress(duration, cleanDryRun)
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
				fmt.Printf("Would delete: %s (%s)\n", cp.ID, util.FormatTimeAgo(cp.CreatedAt))
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

func cleanWithCompress(duration time.Duration, dryRun bool) error {
	checkpoints, err := checkpoint.List()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-duration)
	toCompress := 0
	var totalOriginal, totalCompressed int64

	for _, cp := range checkpoints {
		if cp.CreatedAt.Before(cutoff) && !cp.Manifest.Compressed {
			if dryRun {
				fmt.Printf("Would compress: %s (%s)\n", cp.ID, util.FormatTimeAgo(cp.CreatedAt))
				toCompress++
			} else {
				fmt.Printf("Compressing: %s...\n", cp.ID)
				originalSize, compressedSize, err := checkpoint.Compress(cp.ID)
				if err != nil {
					color.Yellow("  Warning: %v\n", err)
					continue
				}
				totalOriginal += originalSize
				totalCompressed += compressedSize
				toCompress++
				ratio := float64(compressedSize) / float64(originalSize) * 100
				fmt.Printf("  %s → %s (%.1f%%)\n", util.FormatBytes(originalSize), util.FormatBytes(compressedSize), ratio)
			}
		}
	}

	if toCompress == 0 {
		fmt.Println("No checkpoints to compress.")
	} else if dryRun {
		fmt.Printf("\nWould compress %d checkpoint(s). Run without --dry-run to compress.\n", toCompress)
	} else {
		saved := totalOriginal - totalCompressed
		color.Green("✓ Compressed %d checkpoint(s), saved %s\n", toCompress, util.FormatBytes(saved))
	}

	return nil
}

func cleanKeepN(keepCount int, dryRun bool, compress bool) error {
	checkpoints, err := checkpoint.List()
	if err != nil {
		return err
	}

	if len(checkpoints) <= keepCount {
		fmt.Printf("Only %d checkpoint(s) exist, keeping all.\n", len(checkpoints))
		return nil
	}

	// Checkpoints are sorted newest first, so we skip the first N
	toProcess := checkpoints[keepCount:]
	processed := 0

	action := "delete"
	if compress {
		action = "compress"
	}

	for _, cp := range toProcess {
		if compress && cp.Manifest.Compressed {
			continue // Already compressed
		}

		if dryRun {
			fmt.Printf("Would %s: %s (%s)\n", action, cp.ID, util.FormatTimeAgo(cp.CreatedAt))
			processed++
		} else {
			if compress {
				fmt.Printf("Compressing: %s...\n", cp.ID)
				_, _, err := checkpoint.Compress(cp.ID)
				if err != nil {
					color.Yellow("  Warning: %v\n", err)
					continue
				}
			} else {
				if err := checkpoint.Delete(cp.ID); err != nil {
					color.Yellow("Warning: failed to delete %s: %v\n", cp.ID, err)
					continue
				}
			}
			processed++
		}
	}

	if processed == 0 {
		fmt.Printf("No checkpoints to %s.\n", action)
	} else if dryRun {
		fmt.Printf("\nWould %s %d checkpoint(s). Run without --dry-run to proceed.\n", action, processed)
	} else {
		if compress {
			color.Green("✓ Compressed %d checkpoint(s)\n", processed)
		} else {
			color.Green("✓ Deleted %d checkpoint(s), kept %d most recent\n", processed, keepCount)
		}
	}

	return nil
}

// parseDuration parses a duration string with support for days (d) and weeks (w)
func parseDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration")
	}

	// Handle day suffix (e.g., "7d")
	if s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	// Handle week suffix (e.g., "2w")
	if s[len(s)-1] == 'w' {
		var weeks int
		if _, err := fmt.Sscanf(s, "%dw", &weeks); err == nil {
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}

	// Fall back to standard Go duration parsing (h, m, s)
	return time.ParseDuration(s)
}
