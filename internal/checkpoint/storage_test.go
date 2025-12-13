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

// === Exclusion Tests ===

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Should exclude
		{"node_modules dir", "/project/node_modules", true},
		{"nested node_modules", "/project/packages/app/node_modules", true},
		{".git dir", "/project/.git", true},
		{".safeshell dir", "/Users/test/.safeshell", true},
		{"build dir", "/project/build", true},
		{".build dir", "/project/.build", true},
		{"vendor dir", "/project/vendor", true},
		{"DerivedData", "/Users/test/Library/Developer/Xcode/DerivedData", true},
		{".DS_Store file", "/project/.DS_Store", true},
		{"__pycache__", "/project/src/__pycache__", true},
		{"dist dir", "/project/dist", true},
		{"target dir", "/project/target", true},
		{".venv dir", "/project/.venv", true},
		{".cache dir", "/project/.cache", true},

		// Should NOT exclude
		{"src dir", "/project/src", false},
		{"regular file", "/project/main.go", false},
		{"file named node_modules.txt", "/project/node_modules.txt", false},
		{"dir containing node_modules in name", "/project/my_node_modules_backup", false},
		{"build in middle of path", "/project/build_scripts/run.sh", false},
		{"vendor in filename", "/project/vendor.json", false},
		{".git in filename", "/project/.gitignore", false},
		{"nested src", "/project/packages/core/src", false},
		{"buildfile", "/project/buildfile", false},
		{"distribute", "/project/distribute", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestShouldExcludeCaseSensitivity(t *testing.T) {
	// Exclusions should be case-sensitive (Unix convention)
	tests := []struct {
		path     string
		expected bool
	}{
		{"/project/node_modules", true},
		{"/project/Node_Modules", false}, // Different case = not excluded
		{"/project/NODE_MODULES", false},
		{"/project/.git", true},
		{"/project/.Git", false},
		{"/project/.GIT", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExclude(%q) = %v, want %v (case sensitivity)", tt.path, result, tt.expected)
			}
		})
	}
}

func TestBackupDirWithExclusions(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure with excluded directories
	dirs := []string{
		"src/app",
		"src/node_modules/package",
		"src/.git/objects",
		"src/vendor",
		"src/real_code",
		"src/.safeshell/checkpoints",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create files
	files := map[string]string{
		"src/main.go":                        "package main",
		"src/app/app.go":                     "package app",
		"src/node_modules/package/index.js":  "module.exports = {}",
		"src/.git/config":                    "[core]",
		"src/vendor/lib.go":                  "package vendor",
		"src/real_code/util.go":              "package real",
		"src/.gitignore":                     "*.log",
		"src/.safeshell/checkpoints/test.json": "{}",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Run backup
	if err := BackupDir(srcDir, dstDir); err != nil {
		t.Fatalf("BackupDir failed: %v", err)
	}

	// Check what was backed up
	shouldExist := []string{
		"main.go",
		"app/app.go",
		"real_code/util.go",
		".gitignore",
	}

	shouldNotExist := []string{
		"node_modules",
		"node_modules/package/index.js",
		".git",
		".git/config",
		"vendor",
		"vendor/lib.go",
		".safeshell",
		".safeshell/checkpoints/test.json",
	}

	for _, path := range shouldExist {
		fullPath := filepath.Join(dstDir, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected %s to exist in backup, but it doesn't", path)
		}
	}

	for _, path := range shouldNotExist {
		fullPath := filepath.Join(dstDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			t.Errorf("Expected %s to NOT exist in backup, but it does", path)
		}
	}
}

func TestSymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	regularFile := filepath.Join(tmpDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Skipf("Symlinks not supported: %v", err)
	}

	// Test isSymlink
	if !isSymlink(symlinkPath) {
		t.Error("isSymlink should return true for symlink")
	}

	if isSymlink(regularFile) {
		t.Error("isSymlink should return false for regular file")
	}

	if isSymlink(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("isSymlink should return false for nonexistent path")
	}
}

func TestBackupDirSkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source structure
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	// Create regular file
	regularFile := filepath.Join(srcDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(srcDir, "link.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Skipf("Symlinks not supported: %v", err)
	}

	// Run backup
	if err := BackupDir(srcDir, dstDir); err != nil {
		t.Fatalf("BackupDir failed: %v", err)
	}

	// Regular file should exist
	if _, err := os.Stat(filepath.Join(dstDir, "regular.txt")); os.IsNotExist(err) {
		t.Error("regular.txt should exist in backup")
	}

	// Symlink should NOT exist (skipped)
	if _, err := os.Stat(filepath.Join(dstDir, "link.txt")); err == nil {
		t.Error("link.txt (symlink) should not exist in backup")
	}
}

func TestFrameworkExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create framework structure
	frameworkDir := filepath.Join(srcDir, "Sparkle.framework")
	if err := os.MkdirAll(filepath.Join(frameworkDir, "Headers"), 0755); err != nil {
		t.Fatalf("Failed to create framework dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(frameworkDir, "Sparkle"), []byte("binary"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create regular file
	if err := os.WriteFile(filepath.Join(srcDir, "main.swift"), []byte("import Sparkle"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Run backup
	if err := BackupDir(srcDir, dstDir); err != nil {
		t.Fatalf("BackupDir failed: %v", err)
	}

	// Framework should not exist
	if _, err := os.Stat(filepath.Join(dstDir, "Sparkle.framework")); err == nil {
		t.Error("Sparkle.framework should not exist in backup")
	}

	// Regular file should exist
	if _, err := os.Stat(filepath.Join(dstDir, "main.swift")); os.IsNotExist(err) {
		t.Error("main.swift should exist in backup")
	}
}

func TestDeepNestedExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create deeply nested node_modules (monorepo style)
	paths := []string{
		"packages/app1/node_modules/dep/index.js",
		"packages/app2/node_modules/dep/index.js",
		"packages/app1/src/main.ts",
		"packages/app2/src/main.ts",
	}

	for _, p := range paths {
		fullPath := filepath.Join(srcDir, p)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Run backup
	if err := BackupDir(srcDir, dstDir); err != nil {
		t.Fatalf("BackupDir failed: %v", err)
	}

	// Source files should exist
	if _, err := os.Stat(filepath.Join(dstDir, "packages/app1/src/main.ts")); os.IsNotExist(err) {
		t.Error("packages/app1/src/main.ts should exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "packages/app2/src/main.ts")); os.IsNotExist(err) {
		t.Error("packages/app2/src/main.ts should exist")
	}

	// node_modules should not exist
	if _, err := os.Stat(filepath.Join(dstDir, "packages/app1/node_modules")); err == nil {
		t.Error("packages/app1/node_modules should not exist")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "packages/app2/node_modules")); err == nil {
		t.Error("packages/app2/node_modules should not exist")
	}
}

func TestEmptyDirectoryBackup(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	if err := BackupDir(srcDir, dstDir); err != nil {
		t.Fatalf("BackupDir failed on empty dir: %v", err)
	}

	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Error("Destination directory should exist")
	}
}
