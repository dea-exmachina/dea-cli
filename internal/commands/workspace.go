package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWorkspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Workspace status and management",
	}

	cmd.AddCommand(newWorkspaceStatusCommand())
	cmd.AddCommand(newWorkspaceSyncCommand())
	cmd.AddCommand(newWorkspaceScopeCommand())

	return cmd
}

func newWorkspaceStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show workspace status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := mustLoadToken()

			// Phase 1a: read workspace info from token claims + config.
			claims, err := decodeJWTClaims(token.WorkspaceToken)
			if err != nil {
				claims = map[string]interface{}{}
			}

			workspaceID := token.WorkspaceID
			if v, ok := claims["workspace_id"].(string); ok && v != "" {
				workspaceID = v
			}

			tier := "standard"
			if v, ok := claims["tier"].(string); ok && v != "" {
				tier = v
			}

			agentID := token.AgentID
			if v, ok := claims["agent_id"].(string); ok && v != "" {
				agentID = v
			}

			endpoint := token.Endpoint
			if endpoint == "" {
				endpoint = cfg.Endpoint
			}

			fmt.Printf("Workspace Status\n")
			fmt.Printf("  Workspace ID: %s\n", workspaceID)
			fmt.Printf("  Agent ID:     %s\n", agentID)
			fmt.Printf("  Tier:         %s\n", tier)
			fmt.Printf("  Endpoint:     %s\n", endpoint)
			fmt.Printf("  Token expiry: %s\n",
				token.ExpiresAt.UTC().Format("2006-01-02 15:04:05 UTC"))

			return nil
		},
	}
}

func newWorkspaceSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync workspace files",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("File sync available in Phase 1b.")
			return nil
		},
	}
}

func newWorkspaceScopeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scope",
		Short: "Manage workspace scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Scope management available in Phase 1b (Dashboard UI).")
			return nil
		},
	}
}
