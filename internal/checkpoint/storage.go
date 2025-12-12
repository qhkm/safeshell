package checkpoint

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BackupFile creates a backup of a file using hard links when possible.
// Falls back to copy if hard link fails (e.g., cross-filesystem).
func BackupFile(srcPath, dstPath string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Try hard link first (efficient, no extra disk space)
	err := os.Link(srcPath, dstPath)
	if err == nil {
		return nil
	}

	// Hard link failed, fall back to copy
	return copyFile(srcPath, dstPath)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// BackupDir recursively backs up a directory
func BackupDir(srcPath, dstPath string) error {
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dstPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return BackupFile(path, targetPath)
	})
}

// RestoreFile restores a file from backup to its original location
func RestoreFile(backupPath, originalPath string) error {
	// Ensure original directory exists
	originalDir := filepath.Dir(originalPath)
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		return fmt.Errorf("failed to create original directory: %w", err)
	}

	// Remove existing file if it exists
	if _, err := os.Stat(originalPath); err == nil {
		if err := os.Remove(originalPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Copy backup to original location
	return copyFile(backupPath, originalPath)
}

// RestoreDir restores a directory from backup
func RestoreDir(backupPath, originalPath string) error {
	return filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(backupPath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(originalPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return RestoreFile(path, targetPath)
	})
}

// GetDiskUsage returns the total size of a directory
func GetDiskUsage(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
