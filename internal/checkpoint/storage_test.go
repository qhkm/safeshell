package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content for backup")
	err = os.WriteFile(srcPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Backup file
	dstPath := filepath.Join(tmpDir, "backup", "source.txt")
	err = BackupFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// Verify content matches
	backupContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	if string(backupContent) != string(content) {
		t.Errorf("Backup content mismatch: expected '%s', got '%s'", content, backupContent)
	}
}

func TestBackupFileHardLink(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-hardlink-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	err = os.WriteFile(srcPath, []byte("hardlink test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Get source inode
	srcInfo, _ := os.Stat(srcPath)

	// Backup file (should use hard link on same filesystem)
	dstPath := filepath.Join(tmpDir, "backup.txt")
	err = BackupFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("BackupFile failed: %v", err)
	}

	// Get dest inode - on same filesystem, should be same inode (hard link)
	dstInfo, _ := os.Stat(dstPath)

	// Both should have same size at minimum
	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("File sizes should match: src=%d, dst=%d", srcInfo.Size(), dstInfo.Size())
	}
}

func TestBackupDir(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-dir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source directory with files
	srcDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file 1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file 2"), 0644)

	// Backup directory
	dstDir := filepath.Join(tmpDir, "backup")
	err = BackupDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("BackupDir failed: %v", err)
	}

	// Verify structure
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt should exist in backup")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.txt")); os.IsNotExist(err) {
		t.Error("subdir/file2.txt should exist in backup")
	}

	// Verify content
	content, _ := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if string(content) != "file 1" {
		t.Errorf("file1.txt content mismatch")
	}
}

func TestRestoreFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-restore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backup file
	backupPath := filepath.Join(tmpDir, "backup", "file.txt")
	os.MkdirAll(filepath.Dir(backupPath), 0755)
	originalContent := []byte("original content")
	os.WriteFile(backupPath, originalContent, 0644)

	// Original location (empty)
	originalPath := filepath.Join(tmpDir, "original", "file.txt")

	// Restore
	err = RestoreFile(backupPath, originalPath)
	if err != nil {
		t.Fatalf("RestoreFile failed: %v", err)
	}

	// Verify restored file
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Error("Restored file should exist")
	}

	content, _ := os.ReadFile(originalPath)
	if string(content) != string(originalContent) {
		t.Errorf("Restored content mismatch")
	}
}

func TestRestoreFileOverwrite(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-overwrite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backup file with original content
	backupPath := filepath.Join(tmpDir, "backup.txt")
	os.WriteFile(backupPath, []byte("original"), 0644)

	// Create existing file with different content
	originalPath := filepath.Join(tmpDir, "original.txt")
	os.WriteFile(originalPath, []byte("modified"), 0644)

	// Restore (should overwrite)
	err = RestoreFile(backupPath, originalPath)
	if err != nil {
		t.Fatalf("RestoreFile failed: %v", err)
	}

	// Verify content was restored
	content, _ := os.ReadFile(originalPath)
	if string(content) != "original" {
		t.Errorf("Expected 'original', got '%s'", content)
	}
}

func TestGetDiskUsage(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "safeshell-diskusage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with known sizes
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("12345"), 0644)     // 5 bytes
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("1234567890"), 0644) // 10 bytes

	size, err := GetDiskUsage(tmpDir)
	if err != nil {
		t.Fatalf("GetDiskUsage failed: %v", err)
	}

	if size != 15 {
		t.Errorf("Expected 15 bytes, got %d", size)
	}
}
