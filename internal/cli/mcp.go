package cli

import (
	"fmt"

	"github.com/qhkm/safeshell/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server",
	Long: `Starts a Model Context Protocol server over stdio.

This allows AI agents like Claude Code to interact with SafeShell directly,
creating checkpoints and rolling back programmatically.

MCP Tools available:
  - checkpoint_create  Create a checkpoint for specific files
  - checkpoint_list    List all available checkpoints
  - checkpoint_rollback Rollback to a previous checkpoint
  - checkpoint_status  Get SafeShell status
  - checkpoint_delete  Delete a specific checkpoint

To use with Claude Code, add to your MCP settings:
  {
    "mcpServers": {
      "safeshell": {
        "command": "safeshell",
        "args": ["mcp"]
      }
    }
  }`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	server := mcp.NewServer()

	// Suppress any output that might interfere with MCP protocol
	// The server communicates via stdio JSON-RPC

	if err := server.Run(); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}
