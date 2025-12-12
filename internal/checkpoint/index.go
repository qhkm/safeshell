package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qhkm/safeshell/internal/config"
)

// IndexEntry contains lightweight checkpoint metadata for fast lookups
type IndexEntry struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Sequence       int64     `json:"sequence"` // Monotonic sequence for ordering same-timestamp entries
	Command        string    `json:"command"`
	FileCount      int       `json:"file_count"`
	TotalSize      int64     `json:"total_size"`
	SessionID      string    `json:"session_id,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	RolledBack     bool      `json:"rolled_back"`
	Compressed     bool      `json:"compressed,omitempty"`
	CompressedSize int64     `json:"compressed_size,omitempty"`
}

// Index provides fast checkpoint lookups without loading full manifests
type Index struct {
	Entries      map[string]*IndexEntry `json:"entries"`
	NextSequence int64                  `json:"next_sequence"` // Monotonic counter for ordering
	UpdatedAt    time.Time              `json:"updated_at"`
	mu           sync.RWMutex
}

var (
	globalIndex     *Index
	globalIndexOnce sync.Once
	globalIndexMu   sync.Mutex
)

// GetIndex returns the global checkpoint index, loading or rebuilding as needed
func GetIndex() *Index {
	globalIndexMu.Lock()
	defer globalIndexMu.Unlock()

	globalIndexOnce.Do(func() {
		globalIndex = &Index{
			Entries: make(map[string]*IndexEntry),
		}
		globalIndex.Load()
	})
	return globalIndex
}

// ResetIndex resets the global index (for testing)
func ResetIndex() {
	globalIndexMu.Lock()
	defer globalIndexMu.Unlock()

	globalIndexOnce = sync.Once{}
	globalIndex = nil
}

// indexPath returns the path to the index file
func indexPath() string {
	return filepath.Join(config.GetCheckpointsDir(), ".index.json")
}

// Load reads the index from disk
func (idx *Index) Load() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	data, err := os.ReadFile(indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			// No index yet, rebuild it
			return idx.rebuildLocked()
		}
		return err
	}

	if err := json.Unmarshal(data, idx); err != nil {
		// Corrupted index, rebuild
		return idx.rebuildLocked()
	}

	// Check if index is stale (compare with directory)
	if idx.isStale() {
		return idx.rebuildLocked()
	}

	return nil
}

// isStale checks if the index needs rebuilding
func (idx *Index) isStale() bool {
	checkpointsDir := config.GetCheckpointsDir()
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		return true
	}

	// Count directories
	dirCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			dirCount++
		}
	}

	return dirCount != len(idx.Entries)
}

// rebuildLocked rebuilds the index from disk (must hold write lock)
func (idx *Index) rebuildLocked() error {
	idx.Entries = make(map[string]*IndexEntry)
	idx.NextSequence = 0

	checkpointsDir := config.GetCheckpointsDir()
	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect all entries first
	var tempEntries []*IndexEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		checkpointDir := filepath.Join(checkpointsDir, id)
		manifest, err := LoadManifest(checkpointDir)
		if err != nil {
			continue // Skip invalid checkpoints
		}

		// Count files and total size
		fileCount := 0
		var totalSize int64
		for _, f := range manifest.Files {
			if !f.IsDir {
				fileCount++
				totalSize += f.Size
			}
		}

		tempEntries = append(tempEntries, &IndexEntry{
			ID:             id,
			Timestamp:      manifest.Timestamp,
			Command:        manifest.Command,
			FileCount:      fileCount,
			TotalSize:      totalSize,
			SessionID:      manifest.SessionID,
			Tags:           manifest.Tags,
			RolledBack:     manifest.RolledBack,
			Compressed:     manifest.Compressed,
			CompressedSize: manifest.CompressedSize,
		})
	}

	// Sort by timestamp (oldest first), then by ID for same-timestamp entries
	for i := 0; i < len(tempEntries)-1; i++ {
		for j := i + 1; j < len(tempEntries); j++ {
			swap := false
			if tempEntries[j].Timestamp.Before(tempEntries[i].Timestamp) {
				swap = true
			} else if tempEntries[j].Timestamp.Equal(tempEntries[i].Timestamp) && tempEntries[j].ID < tempEntries[i].ID {
				swap = true
			}
			if swap {
				tempEntries[i], tempEntries[j] = tempEntries[j], tempEntries[i]
			}
		}
	}

	// Assign sequences in sorted order
	for _, e := range tempEntries {
		e.Sequence = idx.NextSequence
		idx.NextSequence++
		idx.Entries[e.ID] = e
	}

	idx.UpdatedAt = time.Now()
	return idx.saveLocked()
}

// saveLocked saves the index to disk (must hold write lock)
func (idx *Index) saveLocked() error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	// Ensure checkpoints directory exists
	if err := os.MkdirAll(config.GetCheckpointsDir(), 0755); err != nil {
		return err
	}

	return os.WriteFile(indexPath(), data, 0644)
}

// Save saves the index to disk
func (idx *Index) Save() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	return idx.saveLocked()
}

// Add adds a checkpoint to the index
func (idx *Index) Add(cp *Checkpoint) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	fileCount := 0
	var totalSize int64
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
			totalSize += f.Size
		}
	}

	// Assign monotonic sequence number for proper ordering
	seq := idx.NextSequence
	idx.NextSequence++

	idx.Entries[cp.ID] = &IndexEntry{
		ID:             cp.ID,
		Timestamp:      cp.Manifest.Timestamp,
		Sequence:       seq,
		Command:        cp.Manifest.Command,
		FileCount:      fileCount,
		TotalSize:      totalSize,
		SessionID:      cp.Manifest.SessionID,
		Tags:           cp.Manifest.Tags,
		RolledBack:     cp.Manifest.RolledBack,
		Compressed:     cp.Manifest.Compressed,
		CompressedSize: cp.Manifest.CompressedSize,
	}

	idx.UpdatedAt = time.Now()
	idx.saveLocked()
}

// Remove removes a checkpoint from the index
func (idx *Index) Remove(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.Entries, id)
	idx.UpdatedAt = time.Now()
	idx.saveLocked()
}

// Update updates a checkpoint's metadata in the index
func (idx *Index) Update(cp *Checkpoint) {
	idx.Add(cp) // Same operation
}

// GetEntry returns an index entry by ID
func (idx *Index) GetEntry(id string) *IndexEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.Entries[id]
}

// ListEntries returns all index entries sorted by timestamp (newest first)
// Uses sequence as tiebreaker when timestamps are equal
func (idx *Index) ListEntries() []*IndexEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	entries := make([]*IndexEntry, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		entries = append(entries, e)
	}

	// Sort by timestamp descending, then by sequence descending as tiebreaker
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			swap := false
			if entries[j].Timestamp.After(entries[i].Timestamp) {
				swap = true
			} else if entries[j].Timestamp.Equal(entries[i].Timestamp) && entries[j].Sequence > entries[i].Sequence {
				// Same timestamp, use sequence as tiebreaker
				swap = true
			}
			if swap {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries
}

// Rebuild forces a full index rebuild
func (idx *Index) Rebuild() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	return idx.rebuildLocked()
}
