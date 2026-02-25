package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

func newClaimCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "claim <card-id>",
		Short: "Claim a card and set it as in-progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			token := mustLoadToken()

			body := map[string]string{
				"agent_id": token.AgentID,
			}

			data, err := apiClient.Post(api.CardClaimPath(cardID), body)
			if err != nil {
				if isNetworkErr(err) {
					return err
				}
				return fmt.Errorf("failed to claim card %s: %w", cardID, err)
			}

			// Parse response if available.
			var resp map[string]interface{}
			if len(data) > 0 {
				_ = json.Unmarshal(data, &resp)
			}

			// Write current card to .dea-context/.current-card
			if err := os.MkdirAll(".dea-context", 0755); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not create .dea-context dir: %v\n", err)
			} else {
				if err := os.WriteFile(".dea-context/.current-card", []byte(cardID), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not write .current-card: %v\n", err)
				}
			}

			fmt.Printf("Claimed %s. Card is now in-progress.\n", cardID)
			return nil
		},
	}
}

func isNetworkErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, keyword := range []string{"network error", "connection", "no such host", "timeout", "dial"} {
		if containsIgnoreCase(msg, keyword) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	sLower := toLower(s)
	subLower := toLower(sub)
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
