package checkpoint

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultExclusions contains directory names that are excluded by default.
// These are typically generated/cached directories that can be regenerated.
var DefaultExclusions = []string{
	// Build outputs
	".build",      // Swift Package Manager
	"build",       // Generic build folder
	"dist",        // Distribution builds
	"out",         // Common output folder
	"target",      // Rust/Maven builds

	// Dependencies
	"node_modules", // Node.js
	"vendor",       // Go vendor, PHP composer
	".venv",        // Python virtual env
	"venv",         // Python virtual env
	"__pycache__",  // Python cache
	".pytest_cache",// Python pytest

	// IDE/Editor
	".idea",        // JetBrains
	".vscode",      // VS Code (usually safe to exclude)

	// Version control
	".git",         // Git internals
	".svn",         // Subversion
	".hg",          // Mercurial

	// OS files
	".DS_Store",    // macOS
	"Thumbs.db",    // Windows

	// Caches
	".cache",       // Generic cache
	".npm",         // npm cache
	".yarn",        // Yarn cache
	".cargo",       // Rust cargo
	"DerivedData",  // Xcode

	// SafeShell's own directory (prevent recursive backup)
	".safeshell",
}

// shouldExclude checks if a path should be excluded from backup
func shouldExclude(path string) bool {
	base := filepath.Base(path)
	for _, excluded := range DefaultExclusions {
		if base == excluded {
			return true
		}
	}
	return false
}

// isSymlink checks if a path is a symbolic link
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// shouldSkipPath checks if a path should be skipped (symlink or excluded)
func shouldSkipPath(path string, info os.FileInfo) (skip bool, skipDir bool) {
	// Check if it's a symlink
	if isSymlink(path) {
		return true, false
	}

	// Check exclusion list
	if shouldExclude(path) {
		if info.IsDir() {
			return true, true // Skip entire directory
		}
		return true, false
	}

	// Skip framework bundles on macOS (contain symlinks)
	if strings.HasSuffix(path, ".framework") && info.IsDir() {
		return true, true
	}

	return false, false
}

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

// copyBufferSize is 32KB - optimal for most filesystems
const copyBufferSize = 32 * 1024

// Reusable buffer pool to reduce allocations during file copies
var copyBufferPool = make(chan []byte, 4)

func getCopyBuffer() []byte {
	select {
	case buf := <-copyBufferPool:
		return buf
	default:
		return make([]byte, copyBufferSize)
	}
}

func putCopyBuffer(buf []byte) {
	select {
	case copyBufferPool <- buf:
	default:
		// Pool full, let GC handle it
	}
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

	// Use buffered copy for better performance
	buf := getCopyBuffer()
	defer putCopyBuffer(buf)

	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// BackupDir recursively backs up a directory, skipping excluded paths and symlinks
func BackupDir(srcPath, dstPath string) error {
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip permission errors gracefully
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Check if path should be skipped
		skip, skipDir := shouldSkipPath(path, info)
		if skip {
			if skipDir {
				return filepath.SkipDir
			}
			return nil
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

// CompressDir compresses a directory into a .tar.gz file and removes the original
func CompressDir(srcDir, archivePath string) (int64, error) {
	// Create the archive file
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create archive: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(archiveFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk the source directory and add files to archive
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to create archive: %w", err)
	}

	// Close writers to flush data
	tarWriter.Close()
	gzWriter.Close()
	archiveFile.Close()

	// Get compressed size
	info, err := os.Stat(archivePath)
	if err != nil {
		return 0, err
	}
	compressedSize := info.Size()

	// Remove original directory
	if err := os.RemoveAll(srcDir); err != nil {
		return compressedSize, fmt.Errorf("failed to remove original directory: %w", err)
	}

	return compressedSize, nil
}

// DecompressDir decompresses a .tar.gz file into a directory
func DecompressDir(archivePath, dstDir string) error {
	// Open archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		targetPath := filepath.Join(dstDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			file.Close()
		}
	}

	return nil
}

// IsCompressed checks if a checkpoint directory has been compressed
func IsCompressed(checkpointDir string) bool {
	archivePath := filepath.Join(checkpointDir, "files.tar.gz")
	_, err := os.Stat(archivePath)
	return err == nil
}

// GetArchivePath returns the path to the compressed archive
func GetArchivePath(checkpointDir string) string {
	return filepath.Join(checkpointDir, "files.tar.gz")
}

// GetFilesDir returns the path to the files directory
func GetFilesDir(checkpointDir string) string {
	return filepath.Join(checkpointDir, "files")
}
