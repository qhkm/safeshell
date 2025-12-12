package checkpoint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qhkm/safeshell/internal/config"
)

func setupTestEnv(t *testing.T) (string, func()) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "safeshell-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set up config to use temp directory
	os.Setenv("HOME", tmpDir)
	config.Init()

	// Create test files
	testDir := filepath.Join(tmpDir, "testdata")
	os.MkdirAll(testDir, 0755)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestCreateCheckpoint(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file
	testFile := filepath.Join(tmpDir, "testdata", "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create checkpoint
	cp, err := Create("rm test.txt", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verify checkpoint was created
	if cp.ID == "" {
		t.Error("Checkpoint ID should not be empty")
	}

	if cp.Manifest == nil {
		t.Fatal("Checkpoint manifest should not be nil")
	}

	if cp.Manifest.Command != "rm test.txt" {
		t.Errorf("Expected command 'rm test.txt', got '%s'", cp.Manifest.Command)
	}

	if len(cp.Manifest.Files) != 1 {
		t.Errorf("Expected 1 file in manifest, got %d", len(cp.Manifest.Files))
	}

	// Verify backup file exists
	if len(cp.Manifest.Files) > 0 {
		backupPath := cp.Manifest.Files[0].BackupPath
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Backup file should exist")
		}
	}
}

func TestCreateCheckpointWithDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test directory with files
	testDir := filepath.Join(tmpDir, "testdata", "mydir")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "a.txt"), []byte("file a"), 0644)
	os.WriteFile(filepath.Join(testDir, "b.txt"), []byte("file b"), 0644)

	// Create checkpoint
	cp, err := Create("rm -rf mydir", []string{testDir})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Should have backed up the directory and its files
	// 1 directory entry + 2 file entries = at least 2 files (excluding dir)
	fileCount := 0
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
		}
	}

	if fileCount != 2 {
		t.Errorf("Expected 2 files in manifest, got %d", fileCount)
	}
}

func TestCreateCheckpointNonExistentFile(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Try to create checkpoint for non-existent file
	cp, err := Create("rm nonexistent.txt", []string{"/nonexistent/path/file.txt"})
	if err != nil {
		t.Fatalf("Should not error for non-existent files: %v", err)
	}

	// Manifest should have no files
	if len(cp.Manifest.Files) != 0 {
		t.Errorf("Expected 0 files for non-existent path, got %d", len(cp.Manifest.Files))
	}
}

func TestListCheckpoints(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "testdata", "file1.txt")
	testFile2 := filepath.Join(tmpDir, "testdata", "file2.txt")
	os.WriteFile(testFile1, []byte("content1"), 0644)
	os.WriteFile(testFile2, []byte("content2"), 0644)

	// Create multiple checkpoints
	_, err := Create("rm file1.txt", []string{testFile1})
	if err != nil {
		t.Fatalf("Failed to create checkpoint 1: %v", err)
	}

	_, err = Create("rm file2.txt", []string{testFile2})
	if err != nil {
		t.Fatalf("Failed to create checkpoint 2: %v", err)
	}

	// List checkpoints
	checkpoints, err := List()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 2 {
		t.Errorf("Expected 2 checkpoints, got %d", len(checkpoints))
	}

	// Verify they're sorted by time (newest first)
	if len(checkpoints) >= 2 {
		if checkpoints[0].CreatedAt.Before(checkpoints[1].CreatedAt) {
			t.Error("Checkpoints should be sorted newest first")
		}
	}
}

func TestGetCheckpoint(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test file and checkpoint
	testFile := filepath.Join(tmpDir, "testdata", "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)

	created, err := Create("rm test.txt", []string{testFile})
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Get the checkpoint by ID
	retrieved, err := Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get checkpoint: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, retrieved.ID)
	}
}

func TestGetLatestCheckpoint(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test files
	testFile1 := filepath.Join(tmpDir, "testdata", "file1.txt")
	testFile2 := filepath.Join(tmpDir, "testdata", "file2.txt")
	os.WriteFile(testFile1, []byte("content1"), 0644)
	os.WriteFile(testFile2, []byte("content2"), 0644)

	// Create checkpoints
	Create("rm file1.txt", []string{testFile1})
	cp2, _ := Create("rm file2.txt", []string{testFile2})

	// Get latest
	latest, err := GetLatest()
	if err != nil {
		t.Fatalf("Failed to get latest checkpoint: %v", err)
	}

	if latest.ID != cp2.ID {
		t.Errorf("Expected latest to be '%s', got '%s'", cp2.ID, latest.ID)
	}
}

func TestDeleteCheckpoint(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a checkpoint
	testFile := filepath.Join(tmpDir, "testdata", "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)

	cp, _ := Create("rm test.txt", []string{testFile})

	// Delete it
	err := Delete(cp.ID)
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify it's gone
	_, err = Get(cp.ID)
	if err == nil {
		t.Error("Checkpoint should not exist after deletion")
	}
}
