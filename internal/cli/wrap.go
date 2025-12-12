package cli

import (
	"github.com/qhkm/safeshell/internal/wrapper"
	"github.com/spf13/cobra"
)

var wrapCmd = &cobra.Command{
	Use:   "wrap [--dry-run] <command> [args...]",
	Short: "Execute a command with automatic checkpoint",
	Long: `Wraps a command with automatic checkpoint creation.
This is typically called via shell aliases set up by 'safeshell init'.

Before executing the command, safeshell will:
1. Parse the command to identify target files/directories
2. Create a checkpoint (backup) of those targets
3. Execute the actual command

If something goes wrong, use 'safeshell rollback' to restore.

Options:
  --dry-run    Show what would be backed up without creating checkpoint or executing command

Examples:
  safeshell wrap rm -rf ./build           # Normal execution with checkpoint
  safeshell wrap --dry-run rm -rf ./build # Preview what would be backed up`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true, // Don't parse flags, pass them through to the wrapped command
	RunE:               runWrap,
}

func runWrap(cmd *cobra.Command, args []string) error {
	// Check for --dry-run flag (must handle manually since DisableFlagParsing is true)
	dryRun := false
	actualArgs := args

	if len(args) > 0 && args[0] == "--dry-run" {
		dryRun = true
		actualArgs = args[1:]
	}

	if len(actualArgs) == 0 {
		return cmd.Help()
	}

	cmdName := actualArgs[0]
	cmdArgs := []string{}
	if len(actualArgs) > 1 {
		cmdArgs = actualArgs[1:]
	}

	if dryRun {
		return wrapper.WrapDryRun(cmdName, cmdArgs)
	}

	return wrapper.Wrap(cmdName, cmdArgs)
}
