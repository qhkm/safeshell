package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type FileEntry struct {
	OriginalPath string      `json:"original_path"`
	BackupPath   string      `json:"backup_path"`
	Mode         os.FileMode `json:"mode"`
	Size         int64       `json:"size"`
	IsDir        bool        `json:"is_dir"`
}

type Manifest struct {
	ID             string      `json:"id"`
	SessionID      string      `json:"session_id,omitempty"`
	Timestamp      time.Time   `json:"timestamp"`
	Command        string      `json:"command"`
	WorkingDir     string      `json:"working_dir"`
	Files          []FileEntry `json:"files"`
	RolledBack     bool        `json:"rolled_back"`
	Tags           []string    `json:"tags,omitempty"`
	Note           string      `json:"note,omitempty"`
	Compressed     bool        `json:"compressed,omitempty"`
	CompressedSize int64       `json:"compressed_size,omitempty"`
	CompressedAt   time.Time   `json:"compressed_at,omitempty"`
}

func NewManifest(id, command, workingDir string) *Manifest {
	return &Manifest{
		ID:         id,
		Timestamp:  time.Now(),
		Command:    command,
		WorkingDir: workingDir,
		Files:      []FileEntry{},
		RolledBack: false,
	}
}

func (m *Manifest) AddFile(originalPath, backupPath string, mode os.FileMode, size int64, isDir bool) {
	m.Files = append(m.Files, FileEntry{
		OriginalPath: originalPath,
		BackupPath:   backupPath,
		Mode:         mode,
		Size:         size,
		IsDir:        isDir,
	})
}

func (m *Manifest) Save(checkpointDir string) error {
	manifestPath := filepath.Join(checkpointDir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func LoadManifest(checkpointDir string) (*Manifest, error) {
	manifestPath := filepath.Join(checkpointDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}
