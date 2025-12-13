package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/qhkm/safeshell/internal/util"
	"github.com/spf13/cobra"
)

var (
	searchFile    string
	searchTag     string
	searchCommand string
	searchAfter   string
	searchBefore  string
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for checkpoints",
	Long: `Search for checkpoints by file name, tag, command, or date.

Search Options:
  --file      Search by file name or path (partial match)
  --tag       Search by tag
  --command   Search by command (partial match)
  --after     Show checkpoints created after this date (YYYY-MM-DD)
  --before    Show checkpoints created before this date (YYYY-MM-DD)

You can also provide a general query that searches across files, tags, and commands.

Examples:
  safeshell search main.go                    # Search for file name
  safeshell search --file "src/config"        # Search by file path
  safeshell search --tag important            # Search by tag
  safeshell search --command "rm -rf"         # Search by command
  safeshell search --after 2024-12-01         # Checkpoints after date
  safeshell search --tag backup --after 2024-12-01  # Combined search`,
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVarP(&searchFile, "file", "f", "", "Search by file name/path")
	searchCmd.Flags().StringVarP(&searchTag, "tag", "t", "", "Search by tag")
	searchCmd.Flags().StringVarP(&searchCommand, "command", "c", "", "Search by command")
	searchCmd.Flags().StringVar(&searchAfter, "after", "", "Show checkpoints after this date (YYYY-MM-DD)")
	searchCmd.Flags().StringVar(&searchBefore, "before", "", "Show checkpoints before this date (YYYY-MM-DD)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	opts := checkpoint.SearchOptions{}

	// If a positional argument is provided, use it as a general query
	if len(args) > 0 {
		query := strings.Join(args, " ")
		// Search across file names if no specific flag is set
		if searchFile == "" && searchTag == "" && searchCommand == "" {
			searchFile = query
		}
	}

	opts.FileName = searchFile
	opts.Tag = searchTag
	opts.Command = searchCommand

	// Parse dates
	if searchAfter != "" {
		t, err := time.Parse("2006-01-02", searchAfter)
		if err != nil {
			return fmt.Errorf("invalid --after date format (use YYYY-MM-DD): %w", err)
		}
		opts.After = t
	}

	if searchBefore != "" {
		t, err := time.Parse("2006-01-02", searchBefore)
		if err != nil {
			return fmt.Errorf("invalid --before date format (use YYYY-MM-DD): %w", err)
		}
		// Add a day to include the entire day
		opts.Before = t.Add(24 * time.Hour)
	}

	// Check if any search criteria provided
	if opts.FileName == "" && opts.Tag == "" && opts.Command == "" && opts.After.IsZero() && opts.Before.IsZero() {
		return fmt.Errorf("please provide search criteria (--file, --tag, --command, --after, --before)")
	}

	results, err := checkpoint.Search(opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No checkpoints found matching your search criteria.")
		return nil
	}

	fmt.Printf("Found %d checkpoint(s)\n\n", len(results))

	// Header
	headerColor := color.New(color.FgWhite, color.Bold)
	headerColor.Printf("%-28s  %-20s  %-8s  %s\n", "ID", "TIME", "FILES", "COMMAND")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────────")

	for _, cp := range results {
		timeStr := util.FormatTimeAgo(cp.CreatedAt)

		// Count files
		fileCount := 0
		for _, f := range cp.Manifest.Files {
			if !f.IsDir {
				fileCount++
			}
		}

		// Truncate command
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

		// Show tags if any
		if len(cp.Manifest.Tags) > 0 {
			color.New(color.FgMagenta).Printf("  └─ tags: %s\n", strings.Join(cp.Manifest.Tags, ", "))
		}

		// If searching by file, show matching files
		if opts.FileName != "" {
			showMatchingFiles(cp, opts.FileName)
		}
	}

	return nil
}

func showMatchingFiles(cp *checkpoint.Checkpoint, search string) {
	searchLower := strings.ToLower(search)
	var matches []string

	for _, f := range cp.Manifest.Files {
		if f.IsDir {
			continue
		}
		if strings.Contains(strings.ToLower(f.OriginalPath), searchLower) {
			matches = append(matches, f.OriginalPath)
		}
	}

	if len(matches) > 0 {
		// Show first few matching files
		shown := 0
		for _, m := range matches {
			if shown >= 3 {
				color.New(color.FgHiBlack).Printf("  └─ ... and %d more files\n", len(matches)-3)
				break
			}
			color.New(color.FgHiBlack).Printf("  └─ %s\n", m)
			shown++
		}
	}
}
