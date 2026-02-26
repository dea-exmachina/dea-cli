package commands

import (
	"encoding/json"
	"fmt"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

var validStages = []string{
	"backlog", "ready", "in-progress", "review", "done", "blocked",
}

func newTransitionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "transition <card-id> <stage>",
		Short: "Transition a card to a new stage",
		Long: fmt.Sprintf("Transition a card to a new stage.\nValid stages: %v", validStages),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			stage := args[1]
			mustLoadToken()

			// Normalize stage name: CLI uses "in-progress" but DB uses "in_progress"
			lane := stage
			if lane == "in-progress" {
				lane = "in_progress"
			}

			body := map[string]string{
				"target_lane": lane,
			}

			data, err := apiClient.Post(api.CardTransitionPath(cardID), body)
			if err != nil {
				if isNetworkErr(err) {
					return err
				}
				// Check if it looks like a governance rejection.
				if isGovernanceRejection(err.Error()) {
					fmt.Printf("Governance rejection: transition to %q denied for card %s.\n", stage, cardID)
					fmt.Printf("Reason: %v\n", err)
					return nil
				}
				return fmt.Errorf("failed to transition card %s: %w", cardID, err)
			}

			// Parse response.
			var resp map[string]interface{}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &resp); err == nil {
					if msg, ok := resp["message"].(string); ok {
						fmt.Println(msg)
						return nil
					}
				}
			}

			fmt.Printf("Card %s transitioned to %s.\n", cardID, stage)
			return nil
		},
	}
}

func isGovernanceRejection(errMsg string) bool {
	for _, keyword := range []string{"governance", "rejected", "forbidden", "not allowed", "policy"} {
		if containsIgnoreCase(errMsg, keyword) {
			return true
		}
	}
	return false
}
