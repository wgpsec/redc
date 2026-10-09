package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"red-cloud/mod/mcp"

	"github.com/spf13/cobra"
)

// generateMCPToken returns a fresh random token for the CLI SSE server.
func generateMCPToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate token: %v\n", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

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

var (
	mcpSSEToken       string
	mcpSSEAllowRemote bool
)

var mcpSSECmd = &cobra.Command{
	Use:   "sse [address]",
	Short: "Start MCP server with SSE transport",
	Long: `Start an MCP server using SSE (Server-Sent Events) transport.
The server listens on the specified address (default: localhost:8089).

The SSE transport exposes the full tool set, including exec_command and
start_case, so every request must carry a bearer token (Authorization: Bearer
<token> or X-RedC-Token: <token>). A token is generated and printed on startup
unless --token is supplied.

Binding to a non-loopback address additionally requires --allow-remote.

Example:
  redc mcp sse localhost:8089
  redc mcp sse 0.0.0.0:9000 --token <token> --allow-remote`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr := "localhost:8089"
		if len(args) > 0 {
			addr = args[0]
		}

		token := mcpSSEToken
		generated := false
		if token == "" {
			token = generateMCPToken()
			generated = true
		}

		manager := mcp.NewMCPServerManager(redcProject, nil)
		manager.SetToken(token)
		manager.SetAllowRemote(mcpSSEAllowRemote)

		if generated {
			fmt.Printf("MCP SSE token (generated): %s\n", token)
		}
		if err := manager.Start(mcp.TransportSSE, addr); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start MCP SSE server: %v\n", err)
			os.Exit(1)
		}
		// Block until stopped
		select {}
	},
}

func init() {
	mcpSSECmd.Flags().StringVar(&mcpSSEToken, "token", "", "bearer token required from SSE clients (generated if empty)")
	mcpSSECmd.Flags().BoolVar(&mcpSSEAllowRemote, "allow-remote", false, "allow binding to a non-loopback address")

	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpStdioCmd)
	mcpCmd.AddCommand(mcpSSECmd)
}
