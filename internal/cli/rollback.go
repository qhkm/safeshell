package cli

import (
	"fmt"

	"github.com/safeshell/safeshell/internal/checkpoint"
	"github.com/safeshell/safeshell/internal/rollback"
	"github.com/spf13/cobra"
)

var (
	rollbackLast bool
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [checkpoint-id]",
	Short: "Restore files from a checkpoint",
	Long: `Restores files from a checkpoint to their original locations.

You can either specify a checkpoint ID, or use --last to rollback the most recent checkpoint.

Examples:
  safeshell rollback 2024-12-12T143022-a1b2c3
  safeshell rollback --last`,
	RunE: runRollback,
}

func init() {
	rollbackCmd.Flags().BoolVarP(&rollbackLast, "last", "l", false, "Rollback the most recent checkpoint")
}

func runRollback(cmd *cobra.Command, args []string) error {
	var cp *checkpoint.Checkpoint
	var err error

	if rollbackLast {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return fmt.Errorf("failed to get latest checkpoint: %w", err)
		}
	} else if len(args) > 0 {
		cp, err = checkpoint.Get(args[0])
		if err != nil {
			return fmt.Errorf("checkpoint not found: %s", args[0])
		}
	} else {
		return fmt.Errorf("please specify a checkpoint ID or use --last")
	}

	// Show what will be restored
	fmt.Printf("Checkpoint: %s\n", cp.ID)
	fmt.Printf("Command:    %s\n", cp.Manifest.Command)
	fmt.Printf("Time:       %s\n", cp.Manifest.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	fileCount := 0
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
		}
	}
	fmt.Printf("Restoring %d file(s)...\n", fileCount)
	fmt.Println()

	// Perform rollback
	if err := rollback.Rollback(cp); err != nil {
		return err
	}

	printSuccess("Rollback complete!")
	return nil
}
