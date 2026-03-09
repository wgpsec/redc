package cmd

import (
	"red-cloud/mod/mcp"

	"github.com/spf13/cobra"
)

// MCP commands
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) server",
	Long: `Start an MCP server to expose redc functionality via the Model Context Protocol.
Supports both STDIO and SSE (Server-Sent Events) transport modes.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var mcpStdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start MCP server with STDIO transport",
	Long: `Start an MCP server using STDIO transport.
The server reads JSON-RPC requests from stdin and writes responses to stdout.
This mode is suitable for integration with AI assistants and tools.`,
	Run: func(cmd *cobra.Command, args []string) {
		manager := mcp.NewMCPServerManager(redcProject, nil)
		if err := manager.Start(mcp.TransportSTDIO, ""); err != nil {
			return
		}
		// Block until stopped (STDIO mode handles its own loop)
		select {}
	},
}

var mcpSSECmd = &cobra.Command{
	Use:   "sse [address]",
	Short: "Start MCP server with SSE transport",
	Long: `Start an MCP server using SSE (Server-Sent Events) transport.
The server listens on the specified address (default: localhost:8080).

Example:
  redc mcp sse localhost:8080
  redc mcp sse :9000`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr := "localhost:8080"
		if len(args) > 0 {
			addr = args[0]
		}

		manager := mcp.NewMCPServerManager(redcProject, nil)
		if err := manager.Start(mcp.TransportSSE, addr); err != nil {
			return
		}
		// Block until stopped
		select {}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpStdioCmd)
	mcpCmd.AddCommand(mcpSSECmd)
}
