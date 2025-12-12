package cli

import (
	"github.com/safeshell/safeshell/internal/wrapper"
	"github.com/spf13/cobra"
)

var wrapCmd = &cobra.Command{
	Use:   "wrap <command> [args...]",
	Short: "Execute a command with automatic checkpoint",
	Long: `Wraps a command with automatic checkpoint creation.
This is typically called via shell aliases set up by 'safeshell init'.

Before executing the command, safeshell will:
1. Parse the command to identify target files/directories
2. Create a checkpoint (backup) of those targets
3. Execute the actual command

If something goes wrong, use 'safeshell rollback' to restore.`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true, // Don't parse flags, pass them through to the wrapped command
	RunE:               runWrap,
}

func runWrap(cmd *cobra.Command, args []string) error {
	cmdName := args[0]
	cmdArgs := []string{}
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	return wrapper.Wrap(cmdName, cmdArgs)
}
