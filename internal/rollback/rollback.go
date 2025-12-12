package rollback

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qhkm/safeshell/internal/checkpoint"
)

// Rollback restores files from a checkpoint
func Rollback(cp *checkpoint.Checkpoint) error {
	if cp.Manifest.RolledBack {
		return fmt.Errorf("checkpoint %s has already been rolled back", cp.ID)
	}

	// Auto-decompress if checkpoint is compressed
	if cp.Manifest.Compressed {
		fmt.Println("Decompressing checkpoint...")
		if err := checkpoint.EnsureDecompressed(cp); err != nil {
			return fmt.Errorf("failed to decompress checkpoint: %w", err)
		}
		// Reload checkpoint to get updated paths
		var err error
		cp, err = checkpoint.Get(cp.ID)
		if err != nil {
			return fmt.Errorf("failed to reload checkpoint: %w", err)
		}
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

// RollbackSelective restores only specific files from a checkpoint
func RollbackSelective(cp *checkpoint.Checkpoint, filePaths []string) error {
	if cp.Manifest.RolledBack {
		return fmt.Errorf("checkpoint %s has already been rolled back", cp.ID)
	}

	// Auto-decompress if checkpoint is compressed
	if cp.Manifest.Compressed {
		fmt.Println("Decompressing checkpoint...")
		if err := checkpoint.EnsureDecompressed(cp); err != nil {
			return fmt.Errorf("failed to decompress checkpoint: %w", err)
		}
		var err error
		cp, err = checkpoint.Get(cp.ID)
		if err != nil {
			return fmt.Errorf("failed to reload checkpoint: %w", err)
		}
	}

	// Build a map of files to restore for quick lookup
	toRestore := make(map[string]bool)
	for _, p := range filePaths {
		toRestore[p] = true
	}

	restored := 0
	failed := 0

	for _, file := range cp.Manifest.Files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Skip files not in our restore list
		if !toRestore[file.OriginalPath] {
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

	// Note: We don't mark the checkpoint as rolled back for selective restores
	// since not all files were restored

	if failed > 0 {
		return fmt.Errorf("restored %d files, %d failed", restored, failed)
	}

	fmt.Printf("Successfully restored %d files from checkpoint %s\n", restored, cp.ID)
	return nil
}

// RollbackToPath restores all files from a checkpoint to a different directory
func RollbackToPath(cp *checkpoint.Checkpoint, destPath string) error {
	// Auto-decompress if checkpoint is compressed
	if cp.Manifest.Compressed {
		fmt.Println("Decompressing checkpoint...")
		if err := checkpoint.EnsureDecompressed(cp); err != nil {
			return fmt.Errorf("failed to decompress checkpoint: %w", err)
		}
		var err error
		cp, err = checkpoint.Get(cp.ID)
		if err != nil {
			return fmt.Errorf("failed to reload checkpoint: %w", err)
		}
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	restored := 0
	failed := 0

	for _, file := range cp.Manifest.Files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Check if backup exists
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: backup file not found: %s\n", file.BackupPath)
			failed++
			continue
		}

		// Calculate destination path - preserve directory structure relative to working dir
		relPath := file.OriginalPath
		if strings.HasPrefix(file.OriginalPath, cp.Manifest.WorkingDir) {
			relPath = strings.TrimPrefix(file.OriginalPath, cp.Manifest.WorkingDir)
			relPath = strings.TrimPrefix(relPath, "/")
		} else {
			// For absolute paths outside working dir, use just the filename
			relPath = filepath.Base(file.OriginalPath)
		}

		targetPath := filepath.Join(destPath, relPath)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory for %s: %v\n", targetPath, err)
			failed++
			continue
		}

		// Restore the file to new location
		if err := checkpoint.RestoreFile(file.BackupPath, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore %s: %v\n", targetPath, err)
			failed++
			continue
		}

		// Restore original permissions
		if err := os.Chmod(targetPath, file.Mode); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore permissions for %s: %v\n", targetPath, err)
		}

		restored++
	}

	// Don't mark checkpoint as rolled back since we restored to a different location

	if failed > 0 {
		return fmt.Errorf("restored %d files to %s, %d failed", restored, destPath, failed)
	}

	fmt.Printf("Successfully restored %d files to %s\n", restored, destPath)
	return nil
}

// RollbackSelectiveToPath restores specific files to a different directory
func RollbackSelectiveToPath(cp *checkpoint.Checkpoint, filePaths []string, destPath string) error {
	// Auto-decompress if checkpoint is compressed
	if cp.Manifest.Compressed {
		fmt.Println("Decompressing checkpoint...")
		if err := checkpoint.EnsureDecompressed(cp); err != nil {
			return fmt.Errorf("failed to decompress checkpoint: %w", err)
		}
		var err error
		cp, err = checkpoint.Get(cp.ID)
		if err != nil {
			return fmt.Errorf("failed to reload checkpoint: %w", err)
		}
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Build a map of files to restore for quick lookup
	toRestore := make(map[string]bool)
	for _, p := range filePaths {
		toRestore[p] = true
	}

	restored := 0
	failed := 0

	for _, file := range cp.Manifest.Files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Skip files not in our restore list
		if !toRestore[file.OriginalPath] {
			continue
		}

		// Check if backup exists
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: backup file not found: %s\n", file.BackupPath)
			failed++
			continue
		}

		// Calculate destination path - preserve directory structure relative to working dir
		relPath := file.OriginalPath
		if strings.HasPrefix(file.OriginalPath, cp.Manifest.WorkingDir) {
			relPath = strings.TrimPrefix(file.OriginalPath, cp.Manifest.WorkingDir)
			relPath = strings.TrimPrefix(relPath, "/")
		} else {
			relPath = filepath.Base(file.OriginalPath)
		}

		targetPath := filepath.Join(destPath, relPath)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory for %s: %v\n", targetPath, err)
			failed++
			continue
		}

		// Restore the file to new location
		if err := checkpoint.RestoreFile(file.BackupPath, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore %s: %v\n", targetPath, err)
			failed++
			continue
		}

		// Restore original permissions
		if err := os.Chmod(targetPath, file.Mode); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore permissions for %s: %v\n", targetPath, err)
		}

		restored++
	}

	if failed > 0 {
		return fmt.Errorf("restored %d files to %s, %d failed", restored, destPath, failed)
	}

	fmt.Printf("Successfully restored %d files to %s\n", restored, destPath)
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
