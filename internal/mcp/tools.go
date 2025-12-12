package mcp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/config"
	"github.com/qhkm/safeshell/internal/rollback"
)

func (s *Server) registerTools() {
	s.tools["checkpoint_create"] = s.toolCheckpointCreate
	s.tools["checkpoint_list"] = s.toolCheckpointList
	s.tools["checkpoint_rollback"] = s.toolCheckpointRollback
	s.tools["checkpoint_status"] = s.toolCheckpointStatus
	s.tools["checkpoint_delete"] = s.toolCheckpointDelete
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

	checkpoints, err := checkpoint.List()
	if err != nil {
		return "", fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
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

		timeAgo := formatTimeAgo(cp.CreatedAt)
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

	// Count files
	fileCount := 0
	for _, f := range cp.Manifest.Files {
		if !f.IsDir {
			fileCount++
		}
	}

	// Perform rollback
	if err := rollback.Rollback(cp); err != nil {
		return "", fmt.Errorf("rollback failed: %w", err)
	}

	return fmt.Sprintf(`Rollback successful!

Checkpoint: %s
Reason: %s
Files restored: %d
Original time: %s

All files have been restored to their original locations.`,
		cp.ID,
		cp.Manifest.Command,
		fileCount,
		cp.CreatedAt.Format("2006-01-02 15:04:05"),
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
	sb.WriteString(fmt.Sprintf("Storage used: %s\n", formatBytes(totalSize)))
	sb.WriteString(fmt.Sprintf("Rolled back: %d\n", rolledBack))

	if len(checkpoints) > 0 {
		latest := checkpoints[0]
		sb.WriteString(fmt.Sprintf("\nLatest checkpoint:\n"))
		sb.WriteString(fmt.Sprintf("  ID: %s\n", latest.ID))
		sb.WriteString(fmt.Sprintf("  Reason: %s\n", latest.Manifest.Command))
		sb.WriteString(fmt.Sprintf("  Time: %s\n", formatTimeAgo(latest.CreatedAt)))
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

// Helper functions
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 15:04")
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
