package cli

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/spf13/cobra"
)

var (
	listLimit int
	listAll   bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all checkpoints",
	Long:    `Lists all available checkpoints with their IDs, timestamps, and commands.`,
	RunE:    runList,
}

func init() {
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 10, "Number of checkpoints to show")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all checkpoints")
}

func runList(cmd *cobra.Command, args []string) error {
	checkpoints, err := checkpoint.List()
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		fmt.Println("No checkpoints found.")
		fmt.Println()
		fmt.Println("Checkpoints are created automatically when you use commands like rm, mv, cp.")
		fmt.Println("Run 'safeshell init' to set up the shell aliases.")
		return nil
	}

	// Apply limit
	displayCount := len(checkpoints)
	if !listAll && listLimit > 0 && displayCount > listLimit {
		displayCount = listLimit
	}

	fmt.Printf("Found %d checkpoint(s)", len(checkpoints))
	if displayCount < len(checkpoints) {
		fmt.Printf(" (showing %d)", displayCount)
	}
	fmt.Println()
	fmt.Println()

	// Header
	headerColor := color.New(color.FgWhite, color.Bold)
	headerColor.Printf("%-28s  %-20s  %-8s  %s\n", "ID", "TIME", "FILES", "COMMAND")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")

	for i, cp := range checkpoints[:displayCount] {
		// Format time relative to now
		timeStr := formatRelativeTime(cp.CreatedAt)

		// Count files (exclude directories)
		fileCount := 0
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}

		// Truncate command if too long
		command := cp.Manifest.Command
		if len(command) > 40 {
			command = command[:37] + "..."
		}

		// Color based on rolled back status
		if cp.Manifest.RolledBack {
			color.New(color.FgHiBlack).Printf("%-28s  %-20s  %-8d  %s (rolled back)\n",
				cp.ID, timeStr, fileCount, command)
		} else {
			fmt.Printf("%-28s  %-20s  %-8d  %s\n",
				cp.ID, timeStr, fileCount, command)
		}

		// Show a hint for the first item
		if i == 0 {
			color.New(color.FgHiBlack).Println("  └─ Use 'safeshell rollback --last' to restore")
		}
	}

	if displayCount < len(checkpoints) {
		fmt.Println()
		fmt.Printf("Use 'safeshell list --all' or 'safeshell list -n %d' to see more.\n", len(checkpoints))
	}

	return nil
}

func formatRelativeTime(t time.Time) string {
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
