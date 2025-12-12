package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/safeshell/safeshell/internal/config"
)

type Checkpoint struct {
	ID        string
	Dir       string
	FilesDir  string
	Manifest  *Manifest
	CreatedAt time.Time
}

// Create creates a new checkpoint for the given files before executing a command
func Create(command string, targetPaths []string) (*Checkpoint, error) {
	// Generate unique ID
	timestamp := time.Now().Format("2006-01-02T150405")
	shortUUID := uuid.New().String()[:8]
	id := fmt.Sprintf("%s-%s", timestamp, shortUUID)

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create checkpoint directory
	checkpointDir := filepath.Join(config.GetCheckpointsDir(), id)
	filesDir := filepath.Join(checkpointDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Create manifest
	manifest := NewManifest(id, command, workingDir)

	// Backup each target path
	for _, targetPath := range targetPaths {
		// Resolve to absolute path
		absPath := targetPath
		if !filepath.IsAbs(targetPath) {
			absPath = filepath.Join(workingDir, targetPath)
		}

		// Check if path exists
		info, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			// Path doesn't exist, skip it
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", absPath, err)
		}

		// Calculate backup path (preserve directory structure)
		relPath := strings.TrimPrefix(absPath, "/")
		backupPath := filepath.Join(filesDir, relPath)

		if info.IsDir() {
			// Backup directory recursively
			if err := BackupDir(absPath, backupPath); err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to backup directory %s: %v\n", absPath, err)
				continue
			}
			manifest.AddFile(absPath, backupPath, info.Mode(), 0, true)

			// Also add individual files within the directory
			filepath.Walk(absPath, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return err
				}
				relFilePath := strings.TrimPrefix(path, "/")
				backupFilePath := filepath.Join(filesDir, relFilePath)
				manifest.AddFile(path, backupFilePath, fi.Mode(), fi.Size(), false)
				return nil
			})
		} else {
			// Backup single file
			if err := BackupFile(absPath, backupPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to backup file %s: %v\n", absPath, err)
				continue
			}
			manifest.AddFile(absPath, backupPath, info.Mode(), info.Size(), false)
		}
	}

	// Save manifest
	if err := manifest.Save(checkpointDir); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	return &Checkpoint{
		ID:        id,
		Dir:       checkpointDir,
		FilesDir:  filesDir,
		Manifest:  manifest,
		CreatedAt: manifest.Timestamp,
	}, nil
}

// List returns all checkpoints sorted by creation time (newest first)
func List() ([]*Checkpoint, error) {
	checkpointsDir := config.GetCheckpointsDir()
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Checkpoint{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoints directory: %w", err)
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		checkpointDir := filepath.Join(checkpointsDir, entry.Name())
		manifest, err := LoadManifest(checkpointDir)
		if err != nil {
			// Skip invalid checkpoints
			continue
		}

		checkpoints = append(checkpoints, &Checkpoint{
			ID:        manifest.ID,
			Dir:       checkpointDir,
			FilesDir:  filepath.Join(checkpointDir, "files"),
			Manifest:  manifest,
			CreatedAt: manifest.Timestamp,
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
	})

	return checkpoints, nil
}

// Get retrieves a specific checkpoint by ID
func Get(id string) (*Checkpoint, error) {
	checkpointDir := filepath.Join(config.GetCheckpointsDir(), id)
	manifest, err := LoadManifest(checkpointDir)
	if err != nil {
		return nil, fmt.Errorf("checkpoint not found: %s", id)
	}

	return &Checkpoint{
		ID:        manifest.ID,
		Dir:       checkpointDir,
		FilesDir:  filepath.Join(checkpointDir, "files"),
		Manifest:  manifest,
		CreatedAt: manifest.Timestamp,
	}, nil
}

// GetLatest returns the most recent checkpoint
func GetLatest() (*Checkpoint, error) {
	checkpoints, err := List()
	if err != nil {
		return nil, err
	}
	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found")
	}
	return checkpoints[0], nil
}

// Delete removes a checkpoint
func Delete(id string) error {
	checkpointDir := filepath.Join(config.GetCheckpointsDir(), id)
	return os.RemoveAll(checkpointDir)
}

// Clean removes checkpoints older than the specified duration
func Clean(olderThan time.Duration) (int, error) {
	checkpoints, err := List()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-olderThan)
	deleted := 0

	for _, cp := range checkpoints {
		if cp.CreatedAt.Before(cutoff) {
			if err := Delete(cp.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete checkpoint %s: %v\n", cp.ID, err)
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}
