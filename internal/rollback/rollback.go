package rollback

import (
	"fmt"
	"os"

	"github.com/safeshell/safeshell/internal/checkpoint"
)

// Rollback restores files from a checkpoint
func Rollback(cp *checkpoint.Checkpoint) error {
	if cp.Manifest.RolledBack {
		return fmt.Errorf("checkpoint %s has already been rolled back", cp.ID)
	}

	restored := 0
	failed := 0

	for _, file := range cp.Manifest.Files {
		// Skip directories (we handle files individually)
		if file.IsDir {
			continue
		}

		// Check if backup exists
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: backup file not found: %s\n", file.BackupPath)
			failed++
			continue
		}

		// Restore the file
		if err := checkpoint.RestoreFile(file.BackupPath, file.OriginalPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore %s: %v\n", file.OriginalPath, err)
			failed++
			continue
		}

		// Restore original permissions
		if err := os.Chmod(file.OriginalPath, file.Mode); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore permissions for %s: %v\n", file.OriginalPath, err)
		}

		restored++
	}

	// Mark checkpoint as rolled back
	cp.Manifest.RolledBack = true
	if err := cp.Manifest.Save(cp.Dir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update manifest: %v\n", err)
	}

	if failed > 0 {
		return fmt.Errorf("restored %d files, %d failed", restored, failed)
	}

	fmt.Printf("Successfully restored %d files from checkpoint %s\n", restored, cp.ID)
	return nil
}

// RollbackByID finds and rolls back a checkpoint by ID
func RollbackByID(id string) error {
	cp, err := checkpoint.Get(id)
	if err != nil {
		return err
	}
	return Rollback(cp)
}

// RollbackLatest rolls back the most recent checkpoint
func RollbackLatest() error {
	cp, err := checkpoint.GetLatest()
	if err != nil {
		return err
	}
	return Rollback(cp)
}
