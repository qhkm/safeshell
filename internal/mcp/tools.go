package mcp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/qhkm/safeshell/internal/rollback"
	"github.com/qhkm/safeshell/internal/util"
)

func (s *Server) registerTools() {
	s.tools["checkpoint_create"] = s.toolCheckpointCreate
	s.tools["checkpoint_list"] = s.toolCheckpointList
	s.tools["checkpoint_rollback"] = s.toolCheckpointRollback
	s.tools["checkpoint_status"] = s.toolCheckpointStatus
	s.tools["checkpoint_delete"] = s.toolCheckpointDelete
	s.tools["checkpoint_diff"] = s.toolCheckpointDiff
	s.tools["checkpoint_tag"] = s.toolCheckpointTag
	s.tools["checkpoint_search"] = s.toolCheckpointSearch
	s.tools["checkpoint_compress"] = s.toolCheckpointCompress
	s.tools["checkpoint_decompress"] = s.toolCheckpointDecompress
}

func (s *Server) toolCheckpointCreate(args map[string]interface{}) (string, error) {
	// Parse paths
	pathsRaw, ok := args["paths"]
	if !ok {
		return "", fmt.Errorf("missing required argument: paths")
	}

	pathsArray, ok := pathsRaw.([]interface{})
	if !ok {
		return "", fmt.Errorf("paths must be an array of strings")
	}

	var paths []string
	for _, p := range pathsArray {
		if str, ok := p.(string); ok {
			paths = append(paths, str)
		}
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("paths array is empty")
	}

	// Get reason
	reason := "MCP checkpoint"
	if r, ok := args["reason"].(string); ok && r != "" {
		reason = r
	}

	// Create checkpoint
	cp, err := checkpoint.Create(reason, paths)
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// Count files
	fileCount := 0
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
		}
	}

	return fmt.Sprintf(`Checkpoint created successfully!

ID: %s
Time: %s
Reason: %s
Files backed up: %d
Paths: %s

To rollback, use: checkpoint_rollback with id="%s" or id="latest"`,
		cp.ID,
		cp.CreatedAt.Format("2006-01-02 15:04:05"),
		reason,
		fileCount,
		strings.Join(paths, ", "),
		cp.ID,
	), nil
}

func (s *Server) toolCheckpointList(args map[string]interface{}) (string, error) {
	limit := 10
	if l, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	// Check for session filter
	sessionOnly := false
	if s, ok := args["session"].(bool); ok && s {
		sessionOnly = true
	}

	var checkpoints []*checkpoint.Checkpoint
	var err error

	if sessionOnly {
		checkpoints, err = checkpoint.GetCurrentSession()
	} else {
		checkpoints, err = checkpoint.List()
	}
	if err != nil {
		return "", fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		if sessionOnly {
			return fmt.Sprintf("No checkpoints found in current session (%s).\n\nUse checkpoint_create to create a checkpoint before destructive operations.", checkpoint.GetSessionID()), nil
		}
		return "No checkpoints found.\n\nUse checkpoint_create to create a checkpoint before destructive operations.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d checkpoint(s)\n\n", len(checkpoints)))
	sb.WriteString("| ID | Time | Files | Reason |\n")
	sb.WriteString("|---|---|---|---|\n")

	displayCount := len(checkpoints)
	if displayCount > limit {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		cp := checkpoints[i]
		fileCount := 0
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}

		timeAgo := util.FormatTimeAgo(cp.CreatedAt)
		reason := cp.Manifest.Command
		if len(reason) > 30 {
			reason = reason[:27] + "..."
		}

		status := ""
		if cp.Manifest.RolledBack {
			status = " (rolled back)"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s%s |\n",
			cp.ID, timeAgo, fileCount, reason, status))
	}

	if len(checkpoints) > limit {
		sb.WriteString(fmt.Sprintf("\n... and %d more. Use limit parameter to see more.", len(checkpoints)-limit))
	}

	sb.WriteString("\n\nTo rollback: use checkpoint_rollback with the checkpoint ID")

	return sb.String(), nil
}

func (s *Server) toolCheckpointRollback(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing required argument: id")
	}

	var cp *checkpoint.Checkpoint
	var err error

	if id == "latest" {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return "", fmt.Errorf("no checkpoints found")
		}
	} else {
		cp, err = checkpoint.Get(id)
		if err != nil {
			return "", fmt.Errorf("checkpoint not found: %s", id)
		}
	}

	if cp.Manifest.RolledBack {
		return "", fmt.Errorf("checkpoint %s has already been rolled back", cp.ID)
	}

	// Check for selective file restore
	var filesToRestore []string
	if filesRaw, ok := args["files"]; ok && filesRaw != nil {
		if filesArray, ok := filesRaw.([]interface{}); ok {
			for _, f := range filesArray {
				if str, ok := f.(string); ok {
					filesToRestore = append(filesToRestore, str)
				}
			}
		}
	}

	var fileCount int
	var rollbackErr error

	if len(filesToRestore) > 0 {
		// Selective rollback
		fileCount = len(filesToRestore)
		rollbackErr = rollback.RollbackSelective(cp, filesToRestore)
	} else {
		// Full rollback - count files
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}
		rollbackErr = rollback.Rollback(cp)
	}

	if rollbackErr != nil {
		return "", fmt.Errorf("rollback failed: %w", rollbackErr)
	}

	restoreType := "All files have"
	if len(filesToRestore) > 0 {
		restoreType = "Selected files have"
	}

	return fmt.Sprintf(`Rollback successful!

Checkpoint: %s
Reason: %s
Files restored: %d
Original time: %s

%s been restored to their original locations.`,
		cp.ID,
		cp.Manifest.Command,
		fileCount,
		cp.CreatedAt.Format("2006-01-02 15:04:05"),
		restoreType,
	), nil
}

func (s *Server) toolCheckpointStatus(args map[string]interface{}) (string, error) {
	cfg := config.Get()

	checkpoints, err := checkpoint.List()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	var totalSize int64
	var totalFiles int
	rolledBack := 0

	for _, cp := range checkpoints {
		size, _ := checkpoint.GetDiskUsage(cp.FilesDir)
		totalSize += size

		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				totalFiles++
			}
		}

		if cp.Manifest.RolledBack {
			rolledBack++
		}
	}

	var sb strings.Builder
	sb.WriteString("SafeShell Status\n")
	sb.WriteString("================\n\n")
	sb.WriteString(fmt.Sprintf("Config directory: %s\n", cfg.SafeShellDir))
	sb.WriteString(fmt.Sprintf("Retention: %d days\n", cfg.RetentionDays))
	sb.WriteString(fmt.Sprintf("Max checkpoints: %d\n\n", cfg.MaxCheckpoints))
	sb.WriteString(fmt.Sprintf("Total checkpoints: %d\n", len(checkpoints)))
	sb.WriteString(fmt.Sprintf("Total files backed up: %d\n", totalFiles))
	sb.WriteString(fmt.Sprintf("Storage used: %s\n", util.FormatBytes(totalSize)))
	sb.WriteString(fmt.Sprintf("Rolled back: %d\n", rolledBack))

	if len(checkpoints) > 0 {
		latest := checkpoints[0]
		sb.WriteString(fmt.Sprintf("\nLatest checkpoint:\n"))
		sb.WriteString(fmt.Sprintf("  ID: %s\n", latest.ID))
		sb.WriteString(fmt.Sprintf("  Reason: %s\n", latest.Manifest.Command))
		sb.WriteString(fmt.Sprintf("  Time: %s\n", util.FormatTimeAgo(latest.CreatedAt)))
	}

	return sb.String(), nil
}

func (s *Server) toolCheckpointDelete(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing required argument: id")
	}

	// Verify checkpoint exists
	cp, err := checkpoint.Get(id)
	if err != nil {
		return "", fmt.Errorf("checkpoint not found: %s", id)
	}

	// Delete
	if err := checkpoint.Delete(id); err != nil {
		return "", fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	return fmt.Sprintf("Checkpoint %s deleted successfully.\n\nReason was: %s", cp.ID, cp.Manifest.Command), nil
}

func (s *Server) toolCheckpointDiff(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing required argument: id")
	}

	var cp *checkpoint.Checkpoint
	var err error

	if id == "latest" {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return "", fmt.Errorf("no checkpoints found")
		}
	} else {
		cp, err = checkpoint.Get(id)
		if err != nil {
			return "", fmt.Errorf("checkpoint not found: %s", id)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Diff for checkpoint: %s\n", cp.ID))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", cp.Manifest.Command))
	sb.WriteString(fmt.Sprintf("Time: %s\n\n", cp.CreatedAt.Format("2006-01-02 15:04:05")))

	if cp.Manifest.RolledBack {
		sb.WriteString("⚠ This checkpoint has already been rolled back\n\n")
	}

	deleted := 0
	modified := 0
	unchanged := 0

	var deletedFiles []string
	var modifiedFiles []string

	for _, f := range cp.Manifest.Files {
		if f.IsDir {
			continue
		}

		info, err := os.Stat(f.OriginalPath)
		if os.IsNotExist(err) {
			deleted++
			deletedFiles = append(deletedFiles, f.OriginalPath)
		} else if err != nil {
			deleted++
			deletedFiles = append(deletedFiles, f.OriginalPath)
		} else {
			// Compare sizes as quick check
			if info.Size() != f.Size {
				modified++
				modifiedFiles = append(modifiedFiles, f.OriginalPath)
			} else {
				unchanged++
			}
		}
	}

	sb.WriteString("Summary:\n")
	if deleted > 0 {
		sb.WriteString(fmt.Sprintf("  • %d file(s) deleted - will be restored\n", deleted))
	}
	if modified > 0 {
		sb.WriteString(fmt.Sprintf("  • %d file(s) modified - will be reverted\n", modified))
	}
	if unchanged > 0 {
		sb.WriteString(fmt.Sprintf("  • %d file(s) unchanged - no action needed\n", unchanged))
	}
	sb.WriteString("\n")

	if deleted+modified > 0 {
		sb.WriteString("Files to restore:\n")
		for _, f := range deletedFiles {
			sb.WriteString(fmt.Sprintf("  + %s (deleted)\n", f))
		}
		for _, f := range modifiedFiles {
			sb.WriteString(fmt.Sprintf("  ~ %s (modified)\n", f))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("To restore, use: checkpoint_rollback with id=\"%s\"\n", cp.ID))
	} else {
		sb.WriteString("✓ All files are already in sync with checkpoint\n")
	}

	return sb.String(), nil
}

func (s *Server) toolCheckpointTag(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing required argument: id")
	}

	var cpID string
	if id == "latest" {
		cp, err := checkpoint.GetLatest()
		if err != nil {
			return "", fmt.Errorf("no checkpoints found")
		}
		cpID = cp.ID
	} else {
		cpID = id
	}

	// Verify checkpoint exists
	cp, err := checkpoint.Get(cpID)
	if err != nil {
		return "", fmt.Errorf("checkpoint not found: %s", cpID)
	}

	var actions []string

	// Handle note
	if note, ok := args["note"].(string); ok && note != "" {
		if err := checkpoint.SetNote(cpID, note); err != nil {
			return "", fmt.Errorf("failed to set note: %w", err)
		}
		actions = append(actions, fmt.Sprintf("Set note: %s", note))
	}

	// Handle tag
	if tag, ok := args["tag"].(string); ok && tag != "" {
		remove := false
		if r, ok := args["remove"].(bool); ok {
			remove = r
		}

		if remove {
			if err := checkpoint.RemoveTag(cpID, tag); err != nil {
				return "", fmt.Errorf("failed to remove tag: %w", err)
			}
			actions = append(actions, fmt.Sprintf("Removed tag: %s", tag))
		} else {
			if err := checkpoint.AddTag(cpID, tag); err != nil {
				return "", fmt.Errorf("failed to add tag: %w", err)
			}
			actions = append(actions, fmt.Sprintf("Added tag: %s", tag))
		}
	}

	if len(actions) == 0 {
		// Just show current tags and note
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Checkpoint: %s\n", cp.ID))
		sb.WriteString(fmt.Sprintf("Command: %s\n", cp.Manifest.Command))

		if cp.Manifest.Note != "" {
			sb.WriteString(fmt.Sprintf("Note: %s\n", cp.Manifest.Note))
		}

		if len(cp.Manifest.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(cp.Manifest.Tags, ", ")))
		} else {
			sb.WriteString("Tags: (none)\n")
		}

		return sb.String(), nil
	}

	return fmt.Sprintf("Checkpoint %s updated:\n%s", cpID, strings.Join(actions, "\n")), nil
}

func (s *Server) toolCheckpointSearch(args map[string]interface{}) (string, error) {
	opts := checkpoint.SearchOptions{}

	if file, ok := args["file"].(string); ok {
		opts.FileName = file
	}
	if tag, ok := args["tag"].(string); ok {
		opts.Tag = tag
	}
	if cmd, ok := args["command"].(string); ok {
		opts.Command = cmd
	}

	if opts.FileName == "" && opts.Tag == "" && opts.Command == "" {
		return "", fmt.Errorf("please provide at least one search criteria: file, tag, or command")
	}

	results, err := checkpoint.Search(opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return "No checkpoints found matching your search criteria.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d checkpoint(s)\n\n", len(results)))
	sb.WriteString("| ID | Time | Files | Command |\n")
	sb.WriteString("|---|---|---|---|\n")

	for _, cp := range results {
		fileCount := 0
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}

		timeAgo := util.FormatTimeAgo(cp.CreatedAt)
		command := cp.Manifest.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}

		status := ""
		if cp.Manifest.RolledBack {
			status = " (rolled back)"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s%s |\n",
			cp.ID, timeAgo, fileCount, command, status))

		// Show tags if present
		if len(cp.Manifest.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("| | tags: %s | | |\n", strings.Join(cp.Manifest.Tags, ", ")))
		}
	}

	return sb.String(), nil
}

func (s *Server) toolCheckpointCompress(args map[string]interface{}) (string, error) {
	// Handle older_than parameter (takes precedence)
	if olderThan, ok := args["older_than"].(string); ok && olderThan != "" {
		duration, err := parseDuration(olderThan)
		if err != nil {
			return "", fmt.Errorf("invalid duration: %s", olderThan)
		}

		count, saved, err := checkpoint.CompressOlderThan(duration)
		if err != nil {
			return "", fmt.Errorf("compression failed: %w", err)
		}

		if count == 0 {
			return "No checkpoints to compress.", nil
		}

		return fmt.Sprintf("Compressed %d checkpoint(s), saved %s", count, util.FormatBytes(saved)), nil
	}

	// Handle id parameter
	id, _ := args["id"].(string)
	if id == "" {
		return "", fmt.Errorf("please provide 'id' or 'older_than' parameter")
	}

	// Compress all
	if id == "all" {
		checkpoints, err := checkpoint.List()
		if err != nil {
			return "", fmt.Errorf("failed to list checkpoints: %w", err)
		}

		compressed := 0
		var totalSaved int64

		for _, cp := range checkpoints {
			if cp.Manifest.Compressed {
				continue
			}

			originalSize, compressedSize, err := checkpoint.Compress(cp.ID)
			if err != nil {
				continue
			}

			totalSaved += originalSize - compressedSize
			compressed++
		}

		if compressed == 0 {
			return "No checkpoints to compress (all already compressed).", nil
		}

		return fmt.Sprintf("Compressed %d checkpoint(s), total saved: %s", compressed, util.FormatBytes(totalSaved)), nil
	}

	// Single checkpoint
	var cp *checkpoint.Checkpoint
	var err error

	if id == "latest" {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return "", fmt.Errorf("no checkpoints found")
		}
	} else {
		cp, err = checkpoint.Get(id)
		if err != nil {
			return "", fmt.Errorf("checkpoint not found: %s", id)
		}
	}

	if cp.Manifest.Compressed {
		return fmt.Sprintf("Checkpoint %s is already compressed (%s)", cp.ID, util.FormatBytes(cp.Manifest.CompressedSize)), nil
	}

	originalSize, compressedSize, err := checkpoint.Compress(cp.ID)
	if err != nil {
		return "", fmt.Errorf("compression failed: %w", err)
	}

	saved := originalSize - compressedSize
	ratio := float64(compressedSize) / float64(originalSize) * 100

	return fmt.Sprintf(`Checkpoint compressed successfully!

ID: %s
Original: %s
Compressed: %s (%.1f%%)
Saved: %s

The checkpoint will be automatically decompressed when you rollback.`,
		cp.ID,
		util.FormatBytes(originalSize),
		util.FormatBytes(compressedSize),
		ratio,
		util.FormatBytes(saved),
	), nil
}

func (s *Server) toolCheckpointDecompress(args map[string]interface{}) (string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing required argument: id")
	}

	var cp *checkpoint.Checkpoint
	var err error

	if id == "latest" {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return "", fmt.Errorf("no checkpoints found")
		}
	} else {
		cp, err = checkpoint.Get(id)
		if err != nil {
			return "", fmt.Errorf("checkpoint not found: %s", id)
		}
	}

	if !cp.Manifest.Compressed {
		return fmt.Sprintf("Checkpoint %s is not compressed", cp.ID), nil
	}

	if err := checkpoint.Decompress(cp.ID); err != nil {
		return "", fmt.Errorf("decompression failed: %w", err)
	}

	return fmt.Sprintf("Checkpoint %s decompressed successfully", cp.ID), nil
}

// parseDuration parses a duration string with support for days (d) and weeks (w)
func parseDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration")
	}

	// Handle day suffix (e.g., "7d")
	if s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}

	// Handle week suffix (e.g., "2w")
	if s[len(s)-1] == 'w' {
		var weeks int
		if _, err := fmt.Sscanf(s, "%dw", &weeks); err == nil {
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}

	// Fall back to standard Go duration parsing (h, m, s)
	return time.ParseDuration(s)
}
