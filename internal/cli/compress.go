package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/spf13/cobra"
)

var (
	compressAll       bool
	compressOlderThan string
	compressLast      bool
	decompressFlag    bool
)

var compressCmd = &cobra.Command{
	Use:   "compress [checkpoint-id]",
	Short: "Compress checkpoints to save disk space",
	Long: `Compress checkpoint files into .tar.gz archives to save disk space.

Compressed checkpoints are automatically decompressed when you rollback.
Typical space savings: 60-80% for text files.

Options:
  --all              Compress all uncompressed checkpoints
  --older-than       Compress checkpoints older than duration (e.g., "7d", "24h")
  --decompress       Decompress instead of compress

Examples:
  safeshell compress --last                    # Compress most recent checkpoint
  safeshell compress 2024-12-12T143022-a1b2c3  # Compress specific checkpoint
  safeshell compress --all                     # Compress all checkpoints
  safeshell compress --older-than 3d           # Compress checkpoints older than 3 days
  safeshell compress --last --decompress       # Decompress most recent checkpoint`,
	RunE: runCompress,
}

func init() {
	rootCmd.AddCommand(compressCmd)
	compressCmd.Flags().BoolVarP(&compressLast, "last", "l", false, "Compress most recent checkpoint")
	compressCmd.Flags().BoolVarP(&compressAll, "all", "a", false, "Compress all uncompressed checkpoints")
	compressCmd.Flags().StringVar(&compressOlderThan, "older-than", "", "Compress checkpoints older than duration")
	compressCmd.Flags().BoolVarP(&decompressFlag, "decompress", "d", false, "Decompress instead of compress")
}

func runCompress(cmd *cobra.Command, args []string) error {
	// Handle --older-than
	if compressOlderThan != "" {
		duration, err := parseDuration(compressOlderThan)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}

		fmt.Printf("Compressing checkpoints older than %s...\n", compressOlderThan)
		count, saved, err := checkpoint.CompressOlderThan(duration)
		if err != nil {
			return err
		}

		if count == 0 {
			fmt.Println("No checkpoints to compress.")
		} else {
			color.Green("✓ Compressed %d checkpoint(s), saved %s\n", count, formatBytes(saved))
		}
		return nil
	}

	// Handle --all
	if compressAll {
		return compressAllCheckpoints()
	}

	// Handle specific checkpoint or --last
	var cp *checkpoint.Checkpoint
	var err error

	if compressLast {
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
		return fmt.Errorf("please specify a checkpoint ID, use --last, --all, or --older-than")
	}

	// Decompress or compress
	if decompressFlag {
		return decompressCheckpoint(cp)
	}
	return compressCheckpoint(cp)
}

func compressCheckpoint(cp *checkpoint.Checkpoint) error {
	if cp.Manifest.Compressed {
		color.Yellow("Checkpoint %s is already compressed (%s)\n", cp.ID, formatBytes(cp.Manifest.CompressedSize))
		return nil
	}

	fmt.Printf("Compressing checkpoint %s...\n", cp.ID)

	originalSize, compressedSize, err := checkpoint.Compress(cp.ID)
	if err != nil {
		return err
	}

	saved := originalSize - compressedSize
	ratio := float64(compressedSize) / float64(originalSize) * 100

	color.Green("✓ Compressed checkpoint %s\n", cp.ID)
	fmt.Printf("  Original:   %s\n", formatBytes(originalSize))
	fmt.Printf("  Compressed: %s (%.1f%%)\n", formatBytes(compressedSize), ratio)
	fmt.Printf("  Saved:      %s\n", formatBytes(saved))

	return nil
}

func decompressCheckpoint(cp *checkpoint.Checkpoint) error {
	if !cp.Manifest.Compressed {
		color.Yellow("Checkpoint %s is not compressed\n", cp.ID)
		return nil
	}

	fmt.Printf("Decompressing checkpoint %s...\n", cp.ID)

	if err := checkpoint.Decompress(cp.ID); err != nil {
		return err
	}

	color.Green("✓ Decompressed checkpoint %s\n", cp.ID)
	return nil
}

func compressAllCheckpoints() error {
	checkpoints, err := checkpoint.List()
	if err != nil {
		return err
	}

	compressed := 0
	var totalSaved int64

	for _, cp := range checkpoints {
		if cp.Manifest.Compressed {
			continue
		}

		fmt.Printf("Compressing %s...\n", cp.ID)
		originalSize, compressedSize, err := checkpoint.Compress(cp.ID)
		if err != nil {
			color.Yellow("  Warning: %v\n", err)
			continue
		}

		saved := originalSize - compressedSize
		totalSaved += saved
		compressed++

		ratio := float64(compressedSize) / float64(originalSize) * 100
		fmt.Printf("  %s → %s (%.1f%%)\n", formatBytes(originalSize), formatBytes(compressedSize), ratio)
	}

	fmt.Println()
	if compressed == 0 {
		fmt.Println("No checkpoints to compress.")
	} else {
		color.Green("✓ Compressed %d checkpoint(s), total saved: %s\n", compressed, formatBytes(totalSaved))
	}

	return nil
}
