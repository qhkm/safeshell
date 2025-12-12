package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/spf13/cobra"
)

var (
	listLimit   int
	listAll     bool
	listSession bool
	listGrouped bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all checkpoints",
	Long: `Lists all available checkpoints with their IDs, timestamps, and commands.

Options:
  --session   Show only checkpoints from the current terminal session
  --grouped   Group checkpoints by session

Examples:
  safeshell list                # Show recent checkpoints
  safeshell list --session      # Show only current session's checkpoints
  safeshell list --grouped      # Group by session`,
	RunE: runList,
}

func init() {
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 10, "Number of checkpoints to show")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all checkpoints")
	listCmd.Flags().BoolVarP(&listSession, "session", "s", false, "Show only current session's checkpoints")
	listCmd.Flags().BoolVar(&listGrouped, "grouped", false, "Group checkpoints by session")
}

func runList(cmd *cobra.Command, args []string) error {
	// Handle grouped display
	if listGrouped {
		return runListGrouped()
	}

	var checkpoints []*checkpoint.Checkpoint
	var err error

	// Get checkpoints based on session flag
	if listSession {
		checkpoints, err = checkpoint.GetCurrentSession()
		if err != nil {
			return fmt.Errorf("failed to list checkpoints: %w", err)
		}
	} else {
		checkpoints, err = checkpoint.List()
		if err != nil {
			return fmt.Errorf("failed to list checkpoints: %w", err)
		}
	}

	if len(checkpoints) == 0 {
		if listSession {
			fmt.Println("No checkpoints found in current session.")
			fmt.Println()
			currentSession := checkpoint.GetSessionID()
			color.New(color.FgHiBlack).Printf("Current session: %s\n", currentSession)
		} else {
			fmt.Println("No checkpoints found.")
			fmt.Println()
			fmt.Println("Checkpoints are created automatically when you use commands like rm, mv, cp.")
			fmt.Println("Run 'safeshell init' to set up the shell aliases.")
		}
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
	if listSession {
		currentSession := checkpoint.GetSessionID()
		fmt.Printf(" in session %s", currentSession)
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

		// Build status suffix
		suffix := ""
		if cp.Manifest.RolledBack {
			suffix = " (rolled back)"
		}
		if cp.Manifest.Compressed {
			suffix += " [compressed]"
		}

		// Color based on rolled back status
		if cp.Manifest.RolledBack {
			color.New(color.FgHiBlack).Printf("%-28s  %-20s  %-8d  %s%s\n",
				cp.ID, timeStr, fileCount, command, suffix)
		} else if cp.Manifest.Compressed {
			color.New(color.FgCyan).Printf("%-28s  %-20s  %-8d  %s%s\n",
				cp.ID, timeStr, fileCount, command, suffix)
		} else {
			fmt.Printf("%-28s  %-20s  %-8d  %s\n",
				cp.ID, timeStr, fileCount, command)
		}

		// Show tags if any
		if len(cp.Manifest.Tags) > 0 {
			color.New(color.FgMagenta).Printf("  └─ tags: %s\n", strings.Join(cp.Manifest.Tags, ", "))
		} else if cp.Manifest.Note != "" {
			// Show note if no tags
			note := cp.Manifest.Note
			if len(note) > 50 {
				note = note[:47] + "..."
			}
			color.New(color.FgHiBlack).Printf("  └─ %s\n", note)
		} else if i == 0 {
			// Show a hint for the first item only if no tags/note
			color.New(color.FgHiBlack).Println("  └─ Use 'safeshell rollback --last' to restore")
		}
	}

	if displayCount < len(checkpoints) {
		fmt.Println()
		fmt.Printf("Use 'safeshell list --all' or 'safeshell list -n %d' to see more.\n", len(checkpoints))
	}

	return nil
}

func runListGrouped() error {
	grouped, err := checkpoint.ListBySession()
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if len(grouped) == 0 {
		fmt.Println("No checkpoints found.")
		return nil
	}

	// Count total checkpoints
	total := 0
	for _, cps := range grouped {
		total += len(cps)
	}

	fmt.Printf("Found %d checkpoint(s) across %d session(s)\n\n", total, len(grouped))

	currentSession := checkpoint.GetSessionID()

	for sessionID, checkpoints := range grouped {
		// Session header
		sessionLabel := sessionID
		if sessionID == currentSession {
			sessionLabel = sessionID + " (current)"
		}

		color.New(color.FgCyan, color.Bold).Printf("Session: %s\n", sessionLabel)
		fmt.Printf("─────────────────────────────────────────────\n")

		for _, cp := range checkpoints {
			timeStr := formatRelativeTime(cp.CreatedAt)

			fileCount := 0
			for _, f := range cp.Manifest.Files {
				if !f.IsDir {
					fileCount++
				}
			}

			command := cp.Manifest.Command
			if len(command) > 30 {
				command = command[:27] + "..."
			}

			if cp.Manifest.RolledBack {
				color.New(color.FgHiBlack).Printf("  %s  %-15s  %d files  %s (rolled back)\n",
					cp.ID, timeStr, fileCount, command)
			} else {
				fmt.Printf("  %s  %-15s  %d files  %s\n",
					cp.ID, timeStr, fileCount, command)
			}
		}
		fmt.Println()
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
