package commands

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/dea-exmachina/dea-cli/internal/auth"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and workspace tokens",
	}

	cmd.AddCommand(newAuthLoginCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthRefreshCommand())
	cmd.AddCommand(newAuthRotateSSHCommand())

	return cmd
}

func newAuthLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the dea workspace and store a JWT",
		RunE: func(cmd *cobra.Command, args []string) error {
			scanner := bufio.NewScanner(os.Stdin)

			// Prompt for endpoint if not already set.
			endpoint := cfg.Endpoint
			fmt.Printf("Endpoint [%s]: ", endpoint)
			scanner.Scan()
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				endpoint = input
			}
			cfg.Endpoint = endpoint
			apiClient = api.NewClient(endpoint, cfg.TimeoutSeconds, tokenStore)

			fmt.Print("Agent ID: ")
			scanner.Scan()
			agentID := strings.TrimSpace(scanner.Text())

			fmt.Print("Secret key: ")
			scanner.Scan()
			secretKey := strings.TrimSpace(scanner.Text())

			if agentID == "" || secretKey == "" {
				return fmt.Errorf("agent ID and secret key are required")
			}

			tokenResp, err := apiClient.IssueToken(map[string]string{
				"agent_id":   agentID,
				"secret_key": secretKey,
			})
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			tokenData := &auth.TokenData{
				WorkspaceToken: tokenResp.WorkspaceToken,
				TokenType:      tokenResp.TokenType,
				ExpiresAt:      tokenResp.ExpiresAt,
				WorkspaceID:    tokenResp.WorkspaceID,
				AgentID:        tokenResp.AgentID,
				Endpoint:       endpoint,
			}

			if err := tokenStore.Save(tokenData); err != nil {
				return fmt.Errorf("failed to store token: %w", err)
			}

			fmt.Println("Authenticated. Token expires in 24h.")
			return nil
		},
	}
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := tokenStore.Load()
			if token == nil {
				fmt.Println("Not authenticated. Run `dea auth login`.")
				return nil
			}

			// Decode JWT claims (no verification — just read).
			claims, err := decodeJWTClaims(token.WorkspaceToken)
			if err != nil {
				claims = map[string]interface{}{}
			}

			agentID := token.AgentID
			if v, ok := claims["agent_id"].(string); ok && v != "" {
				agentID = v
			}

			workspaceID := token.WorkspaceID
			if v, ok := claims["workspace_id"].(string); ok && v != "" {
				workspaceID = v
			}

			scopes := ""
			if v, ok := claims["scopes"]; ok {
				switch s := v.(type) {
				case []interface{}:
					parts := make([]string, 0, len(s))
					for _, scope := range s {
						parts = append(parts, fmt.Sprintf("%v", scope))
					}
					scopes = strings.Join(parts, ", ")
				case string:
					scopes = s
				}
			}

			now := time.Now()
			timeUntil := token.ExpiresAt.Sub(now)
			hoursLeft := int(timeUntil.Hours())
			minutesLeft := int(timeUntil.Minutes()) % 60

			fmt.Printf("Authenticated\n")
			fmt.Printf("  Agent:      %s\n", agentID)
			fmt.Printf("  Workspace:  %s\n", workspaceID)
			if scopes != "" {
				fmt.Printf("  Scopes:     %s\n", scopes)
			}
			fmt.Printf("  Expires:    %s (in %dh %dm)\n",
				token.ExpiresAt.UTC().Format("2006-01-02 15:04:05 UTC"),
				hoursLeft, minutesLeft)

			return nil
		},
	}
}

func newAuthRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Manually refresh the workspace token",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := tokenStore.Load()
			if token == nil {
				return fmt.Errorf("not authenticated. Run `dea auth login`")
			}

			tokenResp, err := apiClient.RefreshToken(token.WorkspaceToken)
			if err != nil {
				return fmt.Errorf("refresh failed: %w", err)
			}

			newToken := &auth.TokenData{
				WorkspaceToken: tokenResp.WorkspaceToken,
				TokenType:      tokenResp.TokenType,
				ExpiresAt:      tokenResp.ExpiresAt,
				WorkspaceID:    tokenResp.WorkspaceID,
				AgentID:        tokenResp.AgentID,
				Endpoint:       token.Endpoint,
			}

			if err := tokenStore.Save(newToken); err != nil {
				return fmt.Errorf("failed to save refreshed token: %w", err)
			}

			fmt.Printf("Token refreshed. New expiry: %s\n",
				newToken.ExpiresAt.UTC().Format("2006-01-02 15:04:05 UTC"))
			return nil
		},
	}
}

func newAuthRotateSSHCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate-ssh",
		Short: "Rotate SSH key for the workspace agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("SSH key rotation available in Phase 1b.")
			return nil
		},
	}
}

// decodeJWTClaims decodes the payload of a JWT without verifying the signature.
func decodeJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Try RawURLEncoding (no padding) — standard for JWT.
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}
	return claims, nil
}

// mustLoadToken loads the token or exits with an error message.
func mustLoadToken() *auth.TokenData {
	token := tokenStore.Load()
	if token == nil {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run `dea auth login`.")
		os.Exit(1)
	}
	return token
}
