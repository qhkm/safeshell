package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/qhkm/safeshell/internal/checkpoint"
	"github.com/spf13/cobra"
)

var (
	tagRemove bool
	tagNote   string
)

var tagCmd = &cobra.Command{
	Use:   "tag <checkpoint-id> [tag...]",
	Short: "Add tags or notes to a checkpoint",
	Long: `Add tags or notes to a checkpoint for better organization.

Tags help you categorize and find checkpoints later.
Notes provide additional context about why a checkpoint was created.

Examples:
  safeshell tag 2024-12-12T143022-a1b2c3 "before-refactor"
  safeshell tag 2024-12-12T143022-a1b2c3 important backup
  safeshell tag --last "pre-deploy"
  safeshell tag --last --note "Before major database migration"
  safeshell tag 2024-12-12T143022-a1b2c3 --remove old-tag`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTag,
}

var (
	tagLast bool
)

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.Flags().BoolVarP(&tagRemove, "remove", "r", false, "Remove the specified tag(s)")
	tagCmd.Flags().StringVarP(&tagNote, "note", "n", "", "Set a note for the checkpoint")
	tagCmd.Flags().BoolVarP(&tagLast, "last", "l", false, "Apply to the most recent checkpoint")
}

func runTag(cmd *cobra.Command, args []string) error {
	var cpID string
	var tags []string

	if tagLast {
		// All args are tags
		tags = args
		cp, err := checkpoint.GetLatest()
		if err != nil {
			return fmt.Errorf("no checkpoints found")
		}
		cpID = cp.ID
	} else {
		// First arg is checkpoint ID, rest are tags
		cpID = args[0]
		if len(args) > 1 {
			tags = args[1:]
		}
	}

	// Verify checkpoint exists
	cp, err := checkpoint.Get(cpID)
	if err != nil {
		return fmt.Errorf("checkpoint not found: %s", cpID)
	}

	// Set note if provided
	if tagNote != "" {
		if err := checkpoint.SetNote(cpID, tagNote); err != nil {
			return fmt.Errorf("failed to set note: %w", err)
		}
		color.Green("✓ Note set for checkpoint %s\n", cpID)
	}

	// Process tags
	if len(tags) > 0 {
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}

			if tagRemove {
				if err := checkpoint.RemoveTag(cpID, tag); err != nil {
					return fmt.Errorf("failed to remove tag '%s': %w", tag, err)
				}
				color.Yellow("- Removed tag '%s' from checkpoint %s\n", tag, cpID)
			} else {
				if err := checkpoint.AddTag(cpID, tag); err != nil {
					return fmt.Errorf("failed to add tag '%s': %w", tag, err)
				}
				color.Green("+ Added tag '%s' to checkpoint %s\n", tag, cpID)
			}
		}
	}

	// Show current state if no tags or note were provided
	if len(tags) == 0 && tagNote == "" {
		fmt.Println()
		color.New(color.FgCyan, color.Bold).Printf("Checkpoint: %s\n", cp.ID)
		fmt.Printf("Command:    %s\n", cp.Manifest.Command)
		fmt.Printf("Time:       %s\n", cp.Manifest.Timestamp.Format("2006-01-02 15:04:05"))

		if cp.Manifest.Note != "" {
			fmt.Printf("Note:       %s\n", cp.Manifest.Note)
		}

		if len(cp.Manifest.Tags) > 0 {
			fmt.Printf("Tags:       %s\n", strings.Join(cp.Manifest.Tags, ", "))
		} else {
			fmt.Println("Tags:       (none)")
		}

		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  safeshell tag <id> <tag>           Add a tag")
		fmt.Println("  safeshell tag <id> --note \"text\"   Set a note")
		fmt.Println("  safeshell tag <id> -r <tag>        Remove a tag")
	}

	return nil
}
