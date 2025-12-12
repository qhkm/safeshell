package checkpoint

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qhkm/safeshell/internal/config"
)

// GetSessionID returns a session identifier for grouping checkpoints.
// It uses SAFESHELL_SESSION env var if set, otherwise derives from terminal/process.
func GetSessionID() string {
	// Check for explicit session ID
	if sessionID := os.Getenv("SAFESHELL_SESSION"); sessionID != "" {
		return sessionID
	}

	// Try to get parent process ID (terminal session)
	ppid := os.Getppid()

	// Create a short hash of date + ppid for a consistent but readable ID
	dateStr := time.Now().Format("2006-01-02")
	hash := md5.Sum([]byte(dateStr + strconv.Itoa(ppid)))
	return fmt.Sprintf("%x", hash[:4])
}

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

	// Create manifest with session ID
	manifest := NewManifest(id, command, workingDir)
	manifest.SessionID = GetSessionID()

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

	cp := &Checkpoint{
		ID:        id,
		Dir:       checkpointDir,
		FilesDir:  filesDir,
		Manifest:  manifest,
		CreatedAt: manifest.Timestamp,
	}

	// Add to index for faster future lookups
	GetIndex().Add(cp)

	return cp, nil
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
// Optimized to use the index for accurate timestamp comparison
func GetLatest() (*Checkpoint, error) {
	idx := GetIndex()
	entries := idx.ListEntries() // Already sorted by timestamp (newest first)

	if len(entries) == 0 {
		return nil, fmt.Errorf("no checkpoints found")
	}

	return Get(entries[0].ID)
}

// Delete removes a checkpoint
func Delete(id string) error {
	checkpointDir := filepath.Join(config.GetCheckpointsDir(), id)
	if err := os.RemoveAll(checkpointDir); err != nil {
		return err
	}
	// Remove from index
	GetIndex().Remove(id)
	return nil
}

// ListBySession returns checkpoints grouped by session ID
func ListBySession() (map[string][]*Checkpoint, error) {
	checkpoints, err := List()
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]*Checkpoint)
	for _, cp := range checkpoints {
		sessionID := cp.Manifest.SessionID
		if sessionID == "" {
			sessionID = "default"
		}
		grouped[sessionID] = append(grouped[sessionID], cp)
	}

	return grouped, nil
}

// GetCurrentSession returns checkpoints from the current session only
func GetCurrentSession() ([]*Checkpoint, error) {
	checkpoints, err := List()
	if err != nil {
		return nil, err
	}

	currentSession := GetSessionID()
	var sessionCheckpoints []*Checkpoint
	for _, cp := range checkpoints {
		if cp.Manifest.SessionID == currentSession {
			sessionCheckpoints = append(sessionCheckpoints, cp)
		}
	}

	return sessionCheckpoints, nil
}

// AddTag adds a tag to a checkpoint
func AddTag(id string, tag string) error {
	cp, err := Get(id)
	if err != nil {
		return err
	}

	// Check if tag already exists
	for _, t := range cp.Manifest.Tags {
		if t == tag {
			return nil // Already has this tag
		}
	}

	cp.Manifest.Tags = append(cp.Manifest.Tags, tag)
	if err := cp.Manifest.Save(cp.Dir); err != nil {
		return err
	}
	// Update index
	GetIndex().Update(cp)
	return nil
}

// RemoveTag removes a tag from a checkpoint
func RemoveTag(id string, tag string) error {
	cp, err := Get(id)
	if err != nil {
		return err
	}

	var newTags []string
	for _, t := range cp.Manifest.Tags {
		if t != tag {
			newTags = append(newTags, t)
		}
	}

	cp.Manifest.Tags = newTags
	if err := cp.Manifest.Save(cp.Dir); err != nil {
		return err
	}
	// Update index
	GetIndex().Update(cp)
	return nil
}

// SetNote sets the note for a checkpoint
func SetNote(id string, note string) error {
	cp, err := Get(id)
	if err != nil {
		return err
	}

	cp.Manifest.Note = note
	if err := cp.Manifest.Save(cp.Dir); err != nil {
		return err
	}
	// Update index
	GetIndex().Update(cp)
	return nil
}

// ListByTag returns all checkpoints with a specific tag
func ListByTag(tag string) ([]*Checkpoint, error) {
	checkpoints, err := List()
	if err != nil {
		return nil, err
	}

	var tagged []*Checkpoint
	for _, cp := range checkpoints {
		for _, t := range cp.Manifest.Tags {
			if t == tag {
				tagged = append(tagged, cp)
				break
			}
		}
	}

	return tagged, nil
}

// Search finds checkpoints matching the given criteria
type SearchOptions struct {
	FileName string // Search by file name/path (partial match)
	Tag      string // Search by tag
	Command  string // Search by command (partial match)
	Before   time.Time
	After    time.Time
}

func Search(opts SearchOptions) ([]*Checkpoint, error) {
	checkpoints, err := List()
	if err != nil {
		return nil, err
	}

	var results []*Checkpoint

	for _, cp := range checkpoints {
		match := true

		// Filter by tag
		if opts.Tag != "" {
			tagFound := false
			for _, t := range cp.Manifest.Tags {
				if strings.EqualFold(t, opts.Tag) {
					tagFound = true
					break
				}
			}
			if !tagFound {
				match = false
			}
		}

		// Filter by command
		if match && opts.Command != "" {
			if !strings.Contains(strings.ToLower(cp.Manifest.Command), strings.ToLower(opts.Command)) {
				match = false
			}
		}

		// Filter by file name
		if match && opts.FileName != "" {
			fileFound := false
			searchLower := strings.ToLower(opts.FileName)
			for _, f := range cp.Manifest.Files {
				if strings.Contains(strings.ToLower(f.OriginalPath), searchLower) {
					fileFound = true
					break
				}
			}
			if !fileFound {
				match = false
			}
		}

		// Filter by date range
		if match && !opts.After.IsZero() {
			if cp.CreatedAt.Before(opts.After) {
				match = false
			}
		}

		if match && !opts.Before.IsZero() {
			if cp.CreatedAt.After(opts.Before) {
				match = false
			}
		}

		if match {
			results = append(results, cp)
		}
	}

	return results, nil
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

// Compress compresses a checkpoint to save disk space
func Compress(id string) (int64, int64, error) {
	cp, err := Get(id)
	if err != nil {
		return 0, 0, err
	}

	if cp.Manifest.Compressed {
		return 0, cp.Manifest.CompressedSize, fmt.Errorf("checkpoint already compressed")
	}

	filesDir := GetFilesDir(cp.Dir)
	archivePath := GetArchivePath(cp.Dir)

	// Get original size
	originalSize, err := GetDiskUsage(filesDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get original size: %w", err)
	}

	// Compress
	compressedSize, err := CompressDir(filesDir, archivePath)
	if err != nil {
		return originalSize, 0, fmt.Errorf("failed to compress: %w", err)
	}

	// Update manifest
	cp.Manifest.Compressed = true
	cp.Manifest.CompressedSize = compressedSize
	cp.Manifest.CompressedAt = time.Now()

	if err := cp.Manifest.Save(cp.Dir); err != nil {
		return originalSize, compressedSize, fmt.Errorf("failed to update manifest: %w", err)
	}

	// Update index
	GetIndex().Update(cp)

	return originalSize, compressedSize, nil
}

// Decompress decompresses a checkpoint for access
func Decompress(id string) error {
	cp, err := Get(id)
	if err != nil {
		return err
	}

	if !cp.Manifest.Compressed {
		return nil // Already decompressed
	}

	filesDir := GetFilesDir(cp.Dir)
	archivePath := GetArchivePath(cp.Dir)

	// Decompress
	if err := DecompressDir(archivePath, filesDir); err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}

	// Remove archive
	if err := os.Remove(archivePath); err != nil {
		return fmt.Errorf("failed to remove archive: %w", err)
	}

	// Update manifest
	cp.Manifest.Compressed = false
	cp.Manifest.CompressedSize = 0

	if err := cp.Manifest.Save(cp.Dir); err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}

	// Update index
	GetIndex().Update(cp)

	return nil
}

// CompressOlderThan compresses checkpoints older than the specified duration
func CompressOlderThan(olderThan time.Duration) (int, int64, error) {
	checkpoints, err := List()
	if err != nil {
		return 0, 0, err
	}

	cutoff := time.Now().Add(-olderThan)
	compressed := 0
	var totalSaved int64

	for _, cp := range checkpoints {
		if cp.CreatedAt.Before(cutoff) && !cp.Manifest.Compressed {
			originalSize, compressedSize, err := Compress(cp.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to compress checkpoint %s: %v\n", cp.ID, err)
				continue
			}
			compressed++
			totalSaved += originalSize - compressedSize
		}
	}

	return compressed, totalSaved, nil
}

// EnsureDecompressed ensures a checkpoint is decompressed before access
func EnsureDecompressed(cp *Checkpoint) error {
	if cp.Manifest.Compressed {
		return Decompress(cp.ID)
	}
	return nil
}
