package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/qhkm/safeshell/internal/rollback"
	"github.com/qhkm/safeshell/internal/wrapper"
)

// Integration tests for the full SafeShell workflow

func setupIntegrationEnv(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "safeshell-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	os.Setenv("HOME", tmpDir)
	config.Init()

	workDir := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(workDir, 0755)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return workDir, cleanup
}

func TestFullWorkflow_DeleteAndRestore(t *testing.T) {
	workDir, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Step 1: Create project files
	projectDir := filepath.Join(workDir, "myproject")
	os.MkdirAll(projectDir, 0755)

	files := map[string]string{
		"main.go":    "package main\n\nfunc main() {}",
		"go.mod":     "module myproject\n\ngo 1.20",
		"README.md":  "# My Project",
		"config.yml": "debug: true",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(projectDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	// Step 2: Parse rm command arguments
	args := []string{"-rf", projectDir}
	targets, err := wrapper.ParseRmArgs(args)
	if err != nil {
		t.Fatalf("Failed to parse args: %v", err)
	}

	if len(targets) != 1 || targets[0] != projectDir {
		t.Fatalf("Expected [%s], got %v", projectDir, targets)
	}

	// Step 3: Create checkpoint
	cp, err := checkpoint.Create("rm -rf myproject", targets)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	t.Logf("Created checkpoint: %s", cp.ID)

	// Verify checkpoint has all files
	fileCount := 0
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
		}
	}
	if fileCount != 4 {
		t.Errorf("Expected 4 files in checkpoint, got %d", fileCount)
	}

	// Step 4: Simulate deletion
	os.RemoveAll(projectDir)

	// Verify deletion
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Fatal("Project should be deleted")
	}

	// Step 5: List checkpoints
	checkpoints, err := checkpoint.List()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	// Step 6: Rollback
	err = rollback.Rollback(cp)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Step 7: Verify all files are restored
	for name, expectedContent := range files {
		filePath := filepath.Join(projectDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read restored %s: %v", name, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected '%s', got '%s'", name, expectedContent, content)
		}
	}

	t.Log("Full workflow completed successfully")
}

func TestWorkflow_MultipleCheckpoints(t *testing.T) {
	workDir, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Create multiple files
	file1 := filepath.Join(workDir, "file1.txt")
	file2 := filepath.Join(workDir, "file2.txt")
	file3 := filepath.Join(workDir, "file3.txt")

	os.WriteFile(file1, []byte("content 1"), 0644)
	os.WriteFile(file2, []byte("content 2"), 0644)
	os.WriteFile(file3, []byte("content 3"), 0644)

	// Create checkpoints for each deletion
	cp1, _ := checkpoint.Create("rm file1.txt", []string{file1})
	os.Remove(file1)

	cp2, _ := checkpoint.Create("rm file2.txt", []string{file2})
	os.Remove(file2)

	cp3, _ := checkpoint.Create("rm file3.txt", []string{file3})
	os.Remove(file3)

	// Verify all files are deleted
	for _, f := range []string{file1, file2, file3} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("File %s should be deleted", f)
		}
	}

	// List checkpoints
	checkpoints, _ := checkpoint.List()
	if len(checkpoints) != 3 {
		t.Errorf("Expected 3 checkpoints, got %d", len(checkpoints))
	}

	// Rollback middle checkpoint (file2)
	rollback.Rollback(cp2)

	// Only file2 should be restored
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("file2 should be restored")
	}

	// file1 and file3 should still be deleted
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("file1 should still be deleted")
	}
	if _, err := os.Stat(file3); !os.IsNotExist(err) {
		t.Error("file3 should still be deleted")
	}

	// Rollback remaining files
	rollback.Rollback(cp1)
	rollback.Rollback(cp3)

	// All files should now exist
	for _, f := range []string{file1, file2, file3} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("File %s should exist after rollback", f)
		}
	}
}

func TestWorkflow_CleanOldCheckpoints(t *testing.T) {
	workDir, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Create some checkpoints
	for i := 0; i < 5; i++ {
		file := filepath.Join(workDir, "file.txt")
		os.WriteFile(file, []byte("content"), 0644)
		checkpoint.Create("rm file.txt", []string{file})
		os.Remove(file)
	}

	// Verify checkpoints exist
	checkpoints, _ := checkpoint.List()
	if len(checkpoints) != 5 {
		t.Errorf("Expected 5 checkpoints, got %d", len(checkpoints))
	}

	// Clean checkpoints older than 0 (all of them)
	deleted, err := checkpoint.Clean(0)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if deleted != 5 {
		t.Errorf("Expected to delete 5 checkpoints, deleted %d", deleted)
	}

	// Verify no checkpoints remain
	checkpoints, _ = checkpoint.List()
	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints after clean, got %d", len(checkpoints))
	}
}

func TestWorkflow_ChmodRestore(t *testing.T) {
	workDir, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Create a script file with executable permission
	scriptFile := filepath.Join(workDir, "script.sh")
	os.WriteFile(scriptFile, []byte("#!/bin/bash\necho hello"), 0755)

	// Verify it's executable
	info, _ := os.Stat(scriptFile)
	if info.Mode().Perm() != 0755 {
		t.Fatalf("Script should have 0755 permissions")
	}

	// Parse chmod args
	args := []string{"644", scriptFile}
	targets, _ := wrapper.ParseChmodArgs(args)

	// Create checkpoint before chmod
	cp, _ := checkpoint.Create("chmod 644 script.sh", targets)

	// Change permissions
	os.Chmod(scriptFile, 0644)

	// Verify permissions changed
	info, _ = os.Stat(scriptFile)
	if info.Mode().Perm() != 0644 {
		t.Error("Permissions should be 0644 after chmod")
	}

	// Rollback
	rollback.Rollback(cp)

	// Verify permissions restored
	info, _ = os.Stat(scriptFile)
	if info.Mode().Perm()&0100 == 0 {
		t.Error("Script should be executable after rollback")
	}
}

func TestWorkflow_MvRestore(t *testing.T) {
	workDir, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Create source file
	srcFile := filepath.Join(workDir, "original.txt")
	os.WriteFile(srcFile, []byte("original content"), 0644)

	// Parse mv args
	dstFile := filepath.Join(workDir, "moved.txt")
	args := []string{srcFile, dstFile}
	targets, _ := wrapper.ParseMvArgs(args)

	// targets should be the source file
	if len(targets) != 1 || targets[0] != srcFile {
		t.Fatalf("Expected source file as target, got %v", targets)
	}

	// Create checkpoint before mv
	cp, _ := checkpoint.Create("mv original.txt moved.txt", targets)

	// Simulate mv (delete source, create dest)
	content, _ := os.ReadFile(srcFile)
	os.Remove(srcFile)
	os.WriteFile(dstFile, content, 0644)

	// Verify move happened
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("Source should not exist after mv")
	}
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Error("Destination should exist after mv")
	}

	// Rollback
	rollback.Rollback(cp)

	// Source should be restored
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		t.Error("Source should be restored after rollback")
	}

	restoredContent, _ := os.ReadFile(srcFile)
	if string(restoredContent) != "original content" {
		t.Errorf("Content mismatch after rollback")
	}
}
