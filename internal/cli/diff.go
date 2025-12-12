package cli

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/spf13/cobra"
)

var (
	diffLast    bool
	diffContent bool
	diffFile    string
)

var diffCmd = &cobra.Command{
	Use:   "diff [checkpoint-id]",
	Short: "Show what would be restored from a checkpoint",
	Long: `Shows the differences between a checkpoint and current filesystem state.

This helps you understand what a rollback would do before executing it.

Options:
  --content    Show actual content differences for modified text files
  --file       Show diff for a specific file only

Examples:
  safeshell diff --last                        # Compare with most recent checkpoint
  safeshell diff --last --content              # Show content changes
  safeshell diff --last --file src/main.go     # Diff specific file
  safeshell diff 2024-12-12T143022             # Compare with specific checkpoint`,
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVarP(&diffLast, "last", "l", false, "Compare with most recent checkpoint")
	diffCmd.Flags().BoolVarP(&diffContent, "content", "c", false, "Show actual content differences")
	diffCmd.Flags().StringVarP(&diffFile, "file", "f", "", "Show diff for specific file only")
}

type FileDiff struct {
	Path         string
	Status       string // "deleted", "modified", "unchanged"
	BackupSize   int64
	CurrentSize  int64
	BackupPath   string
}

func runDiff(cmd *cobra.Command, args []string) error {
	var cp *checkpoint.Checkpoint
	var err error

	if diffLast {
		cp, err = checkpoint.GetLatest()
		if err != nil {
			return fmt.Errorf("no checkpoints found")
		}
	} else if len(args) > 0 {
		cp, err = checkpoint.Get(args[0])
		if err != nil {
			return fmt.Errorf("checkpoint not found: %s", args[0])
		}
	} else {
		return fmt.Errorf("please specify a checkpoint ID or use --last")
	}

	// Analyze differences
	diffs := analyzeDiffs(cp)

	// Print header
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Printf("Checkpoint: %s\n", cp.ID)
	fmt.Printf("Command:    %s\n", cp.Manifest.Command)
	fmt.Printf("Time:       %s\n", cp.Manifest.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if cp.Manifest.RolledBack {
		color.Yellow("⚠ This checkpoint has already been rolled back\n\n")
	}

	// Count by status
	deleted := 0
	modified := 0
	unchanged := 0
	var totalRestoreSize int64

	for _, d := range diffs {
		switch d.Status {
		case "deleted":
			deleted++
			totalRestoreSize += d.BackupSize
		case "modified":
			modified++
			totalRestoreSize += d.BackupSize
		case "unchanged":
			unchanged++
		}
	}

	// Summary
	color.New(color.FgWhite, color.Bold).Println("Summary:")
	if deleted > 0 {
		color.Red("  • %d file(s) deleted - will be restored\n", deleted)
	}
	if modified > 0 {
		color.Yellow("  • %d file(s) modified - will be reverted\n", modified)
	}
	if unchanged > 0 {
		color.Green("  • %d file(s) unchanged - no action needed\n", unchanged)
	}
	fmt.Printf("  • Total restore size: %s\n", formatBytes(totalRestoreSize))
	fmt.Println()

	// Filter by specific file if requested
	if diffFile != "" {
		var filteredDiffs []FileDiff
		absFile, _ := filepath.Abs(diffFile)
		for _, d := range diffs {
			if d.Path == diffFile || d.Path == absFile || strings.HasSuffix(d.Path, "/"+diffFile) {
				filteredDiffs = append(filteredDiffs, d)
			}
		}
		if len(filteredDiffs) == 0 {
			return fmt.Errorf("file '%s' not found in checkpoint", diffFile)
		}
		diffs = filteredDiffs
	}

	// Detailed file list
	if deleted+modified > 0 {
		color.New(color.FgWhite, color.Bold).Println("Files to restore:")
		fmt.Println()

		for _, d := range diffs {
			if d.Status == "unchanged" {
				continue
			}

			// Shorten path for display
			displayPath := d.Path
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, d.Path); err == nil && !strings.HasPrefix(rel, "..") {
					displayPath = rel
				}
			}

			switch d.Status {
			case "deleted":
				color.Red("  + %s", displayPath)
				color.New(color.FgHiBlack).Printf(" (%s)\n", formatBytes(d.BackupSize))
				if diffContent {
					showFileContent(d.BackupPath, "backup")
				}
			case "modified":
				color.Yellow("  ~ %s", displayPath)
				color.New(color.FgHiBlack).Printf(" (%s → %s)\n", formatBytes(d.CurrentSize), formatBytes(d.BackupSize))
				if diffContent {
					showContentDiff(d.BackupPath, d.Path)
				}
			}
		}
		fmt.Println()
	}

	// Instructions
	if deleted+modified > 0 {
		fmt.Println("To restore these files, run:")
		color.Cyan("  safeshell rollback %s\n", cp.ID)
		fmt.Println()
		fmt.Println("To restore specific files only:")
		color.Cyan("  safeshell rollback %s --files \"path/to/file\"\n", cp.ID)
	} else {
		color.Green("✓ All files are already in sync with checkpoint\n")
	}

	return nil
}

func analyzeDiffs(cp *checkpoint.Checkpoint) []FileDiff {
	var diffs []FileDiff

	for _, f := range cp.Manifest.Files {
		if f.IsDir {
			continue
		}

		diff := FileDiff{
			Path:       f.OriginalPath,
			BackupSize: f.Size,
			BackupPath: f.BackupPath,
		}

		// Check if file exists
		info, err := os.Stat(f.OriginalPath)
		if os.IsNotExist(err) {
			diff.Status = "deleted"
			diff.CurrentSize = 0
		} else if err != nil {
			diff.Status = "deleted" // Treat errors as deleted
			diff.CurrentSize = 0
		} else {
			diff.CurrentSize = info.Size()

			// Compare content (using hash for efficiency)
			if filesMatch(f.BackupPath, f.OriginalPath) {
				diff.Status = "unchanged"
			} else {
				diff.Status = "modified"
			}
		}

		diffs = append(diffs, diff)
	}

	return diffs
}

func filesMatch(path1, path2 string) bool {
	// Quick check: compare file sizes first (much faster than hashing)
	info1, err1 := os.Stat(path1)
	info2, err2 := os.Stat(path2)

	if err1 != nil || err2 != nil {
		return false
	}

	// Different sizes = definitely different files
	if info1.Size() != info2.Size() {
		return false
	}

	// Same size = need to compare content via hash
	hash1, err1 := fileHash(path1)
	hash2, err2 := fileHash(path2)

	if err1 != nil || err2 != nil {
		return false
	}

	return hash1 == hash2
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// showFileContent displays the content of a file (for deleted files)
func showFileContent(path string, label string) {
	if !isTextFile(path) {
		color.New(color.FgHiBlack).Println("    (binary file)")
		return
	}

	lines, err := readFileLines(path, 20)
	if err != nil {
		color.New(color.FgHiBlack).Printf("    (unable to read: %v)\n", err)
		return
	}

	fmt.Println()
	color.New(color.FgHiBlack).Printf("    --- %s content ---\n", label)
	for i, line := range lines {
		color.Green("    %3d: %s\n", i+1, truncateLine(line, 80))
	}
	if len(lines) == 20 {
		color.New(color.FgHiBlack).Println("    ... (truncated)")
	}
	fmt.Println()
}

// showContentDiff displays a unified diff between two files
func showContentDiff(backupPath, currentPath string) {
	if !isTextFile(backupPath) || !isTextFile(currentPath) {
		color.New(color.FgHiBlack).Println("    (binary file - content diff not available)")
		return
	}

	backupLines, err1 := readFileLines(backupPath, 500)
	currentLines, err2 := readFileLines(currentPath, 500)

	if err1 != nil || err2 != nil {
		color.New(color.FgHiBlack).Println("    (unable to read files for diff)")
		return
	}

	fmt.Println()
	color.New(color.FgHiBlack).Println("    --- content diff (current → backup) ---")

	// Compute diff using LCS-based algorithm
	diff := computeDiff(currentLines, backupLines)

	changesShown := 0
	maxChanges := 30

	for _, d := range diff {
		if changesShown >= maxChanges {
			color.New(color.FgHiBlack).Println("    ... (diff truncated)")
			break
		}

		switch d.Op {
		case diffDelete: // Line removed (was in current, not in backup)
			color.Red("    -%3d: %s\n", d.LineNum, truncateLine(d.Text, 70))
			changesShown++
		case diffInsert: // Line added (in backup, not in current)
			color.Green("    +%3d: %s\n", d.LineNum, truncateLine(d.Text, 70))
			changesShown++
		}
	}

	if changesShown == 0 {
		color.New(color.FgHiBlack).Println("    (no differences)")
	}
	fmt.Println()
}

type diffOp int

const (
	diffEqual diffOp = iota
	diffDelete
	diffInsert
)

type diffLine struct {
	Op      diffOp
	Text    string
	LineNum int
}

// computeDiff computes the diff between two line slices using a simple LCS approach
func computeDiff(a, b []string) []diffLine {
	// Build LCS table
	m, n := len(a), len(b)

	// For very large files, use simpler approach
	if m > 500 || n > 500 {
		return computeSimpleDiff(a, b)
	}

	// LCS dynamic programming
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = maxInt(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find diff
	var result []diffLine
	i, j := m, n

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, diffLine{Op: diffInsert, Text: b[j-1], LineNum: j})
			j--
		} else if i > 0 {
			result = append(result, diffLine{Op: diffDelete, Text: a[i-1], LineNum: i})
			i--
		}
	}

	// Reverse to get correct order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// computeSimpleDiff falls back to simple comparison for large files
func computeSimpleDiff(a, b []string) []diffLine {
	var result []diffLine

	// Build set of lines in each
	aSet := make(map[string]bool)
	bSet := make(map[string]bool)
	for _, line := range a {
		aSet[line] = true
	}
	for _, line := range b {
		bSet[line] = true
	}

	// Find deletions (in a but not b)
	for i, line := range a {
		if !bSet[line] {
			result = append(result, diffLine{Op: diffDelete, Text: line, LineNum: i + 1})
		}
	}

	// Find insertions (in b but not a)
	for i, line := range b {
		if !aSet[line] {
			result = append(result, diffLine{Op: diffInsert, Text: line, LineNum: i + 1})
		}
	}

	return result
}

// isTextFile checks if a file appears to be text (not binary)
func isTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first 512 bytes
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}

	return true
}

// readFileLines reads up to maxLines from a file
func readFileLines(path string, maxLines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < maxLines {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// truncateLine truncates a line to maxLen characters
func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}
