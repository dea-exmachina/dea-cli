package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

func newDoneCommand() *cobra.Command {
	var summary string

	cmd := &cobra.Command{
		Use:   "done <card-id>",
		Short: "Mark a card as done: push artifacts, transition to review, emit signal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			mustLoadToken()

			// Step 1: Push staged artifacts if any exist.
			staged, err := loadStagedArtifacts()
			if err == nil {
				var hasStagedForCard bool
				for _, a := range staged {
					if a.CardID == cardID {
						hasStagedForCard = true
						break
					}
				}

				if hasStagedForCard {
					fmt.Printf("Pushing staged artifacts for card %s...\n", cardID)
					var toPush []StagedArtifact
					var remaining []StagedArtifact
					for _, a := range staged {
						if a.CardID == cardID {
							toPush = append(toPush, a)
						} else {
							remaining = append(remaining, a)
						}
					}

					pushedCount := 0
					for _, artifact := range toPush {
						if err := pushArtifact(artifact.FilePath, cardID); err != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to push %s: %v\n", artifact.FilePath, err)
							continue
						}
						pushedCount++
					}

					if pushedCount > 0 {
						_ = saveStagedArtifacts(remaining)
						fmt.Printf("Pushed %d artifact(s).\n", pushedCount)
					}
				}
			}

			// Step 2: Transition card to review.
			fmt.Printf("Transitioning card %s to review...\n", cardID)
			transitionBody := map[string]string{"stage": "review"}
			_, err = apiClient.Post(api.CardTransitionPath(cardID), transitionBody)
			if err != nil {
				if isNetworkErr(err) {
					if qErr := offQueue.Add("POST", api.CardTransitionPath(cardID), transitionBody); qErr == nil {
						fmt.Println("Queued transition offline. Will flush on next connection.")
					}
				} else {
					return fmt.Errorf("failed to transition card to review: %w", err)
				}
			} else {
				fmt.Printf("Card %s is now in review.\n", cardID)
			}

			// Step 3: Optionally emit a pattern signal with the summary.
			if summary != "" {
				summary = strings.TrimSpace(summary)
				signals := []map[string]string{
					{
						"card_id":     cardID,
						"signal_type": "pattern",
						"content":     summary,
					},
				}
				_, err = apiClient.Post(api.PathSignals, signals)
				if err != nil {
					if isNetworkErr(err) {
						if qErr := offQueue.Add("POST", api.PathSignals, signals); qErr == nil {
							fmt.Println("Queued signal offline. Will flush on next connection.")
						}
					} else {
						fmt.Fprintf(os.Stderr, "warning: failed to emit signal: %v\n", err)
					}
				} else {
					fmt.Println("Pattern signal emitted.")
				}
			}

			fmt.Printf("\nDone. Card %s submitted for review.\n", cardID)
			return nil
		},
	}

	cmd.Flags().StringVar(&summary, "summary", "", "Summary text to emit as a pattern signal")
	return cmd
}
