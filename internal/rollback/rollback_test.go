package rollback

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
)

func setupTestEnv(t *testing.T) (string, func()) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "safeshell-rollback-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set up config to use temp directory
	os.Setenv("HOME", tmpDir)
	config.Init()

	// Reset index to ensure fresh state for each test
	checkpoint.ResetIndex()

	// Create test files directory
	testDir := filepath.Join(tmpDir, "testdata")
	os.MkdirAll(testDir, 0755)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestRollbackDeletedFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file
	testFile := filepath.Join(tmpDir, "testdata", "important.txt")
	originalContent := []byte("very important data")
	err := os.WriteFile(testFile, originalContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create checkpoint before "deletion"
	cp, err := checkpoint.Create("rm important.txt", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Simulate deletion
	os.Remove(testFile)

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("File should be deleted")
	}

	// Rollback
	err = Rollback(cp)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify file is restored
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File should be restored")
	}

	// Verify content
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != string(originalContent) {
		t.Errorf("Content mismatch: expected '%s', got '%s'", originalContent, restoredContent)
	}
}

func TestRollbackDeletedDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test directory with files
	testDir := filepath.Join(tmpDir, "testdata", "myproject")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(testDir, "go.mod"), []byte("module test"), 0644)

	// Create checkpoint before "deletion"
	cp, err := checkpoint.Create("rm -rf myproject", []string{testDir})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Simulate deletion
	os.RemoveAll(testDir)

	// Verify directory is gone
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("Directory should be deleted")
	}

	// Rollback
	err = Rollback(cp)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify files are restored
	if _, err := os.Stat(filepath.Join(testDir, "main.go")); os.IsNotExist(err) {
		t.Error("main.go should be restored")
	}

	if _, err := os.Stat(filepath.Join(testDir, "go.mod")); os.IsNotExist(err) {
		t.Error("go.mod should be restored")
	}

	// Verify content
	content, _ := os.ReadFile(filepath.Join(testDir, "main.go"))
	if string(content) != "package main" {
		t.Errorf("main.go content mismatch")
	}
}

func TestRollbackModifiedFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file
	testFile := filepath.Join(tmpDir, "testdata", "config.json")
	originalContent := []byte(`{"version": 1}`)
	os.WriteFile(testFile, originalContent, 0644)

	// Create checkpoint
	cp, err := checkpoint.Create("modify config.json", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Delete the file first, then create new one with different content
	// (simulating truncate + write, which breaks the hard link)
	os.Remove(testFile)
	os.WriteFile(testFile, []byte(`{"version": 2, "broken": true}`), 0644)

	// Rollback
	err = Rollback(cp)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify original content is restored
	restoredContent, _ := os.ReadFile(testFile)
	if string(restoredContent) != string(originalContent) {
		t.Errorf("Expected '%s', got '%s'", originalContent, restoredContent)
	}
}

func TestRollbackAlreadyRolledBack(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file and checkpoint
	testFile := filepath.Join(tmpDir, "testdata", "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cp, _ := checkpoint.Create("rm test.txt", []string{testFile})

	// Delete file
	os.Remove(testFile)

	// First rollback should succeed
	err := Rollback(cp)
	if err != nil {
		t.Fatalf("First rollback failed: %v", err)
	}

	// Reload checkpoint to get updated manifest
	cp, _ = checkpoint.Get(cp.ID)

	// Second rollback should fail
	err = Rollback(cp)
	if err == nil {
		t.Error("Second rollback should fail (already rolled back)")
	}
}

func TestRollbackByID(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file and checkpoint
	testFile := filepath.Join(tmpDir, "testdata", "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cp, _ := checkpoint.Create("rm test.txt", []string{testFile})

	// Delete file
	os.Remove(testFile)

	// Rollback by ID
	err := RollbackByID(cp.ID)
	if err != nil {
		t.Fatalf("RollbackByID failed: %v", err)
	}

	// Verify file is restored
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should be restored")
	}
}

func TestRollbackLatest(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create multiple files and checkpoints
	file1 := filepath.Join(tmpDir, "testdata", "file1.txt")
	file2 := filepath.Join(tmpDir, "testdata", "file2.txt")
	os.WriteFile(file1, []byte("file 1"), 0644)
	os.WriteFile(file2, []byte("file 2"), 0644)

	checkpoint.Create("rm file1.txt", []string{file1})
	checkpoint.Create("rm file2.txt", []string{file2})

	// Delete file2 (the one in the latest checkpoint)
	os.Remove(file2)

	// Rollback latest
	err := RollbackLatest()
	if err != nil {
		t.Fatalf("RollbackLatest failed: %v", err)
	}

	// file2 should be restored (it was in the latest checkpoint)
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("file2 should be restored")
	}
}

func TestRollbackPreservesPermissions(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file with specific permissions
	testFile := filepath.Join(tmpDir, "testdata", "script.sh")
	os.WriteFile(testFile, []byte("#!/bin/bash"), 0755)

	// Create checkpoint
	cp, _ := checkpoint.Create("rm script.sh", []string{testFile})

	// Delete file
	os.Remove(testFile)

	// Rollback
	err := Rollback(cp)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Check permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat restored file: %v", err)
	}

	// Should have executable permission
	if info.Mode().Perm()&0100 == 0 {
		t.Error("Restored file should be executable")
	}
}
