package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

func newPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull card, board, or context from the workspace",
	}

	cmd.AddCommand(newPullCardCommand())
	cmd.AddCommand(newPullBoardCommand())
	cmd.AddCommand(newPullContextCommand())

	return cmd
}

func newPullCardCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "card <card-id>",
		Short: "Pull context for a specific card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cardID := args[0]
			mustLoadToken()

			data, err := apiClient.Get(api.CardContextPath(cardID))
			if err != nil {
				return handleAPIError(err, "card", cardID, "context")
			}

			// Write to .dea-context/card-<id>.json in the current directory.
			if err := os.MkdirAll(".dea-context", 0755); err != nil {
				return fmt.Errorf("failed to create .dea-context directory: %w", err)
			}

			outPath := fmt.Sprintf(".dea-context/card-%s.json", cardID)
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write context file: %w", err)
			}

			// Parse and print summary — handle { data: { card: {...} } } wrapper.
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err == nil {
				card := parsed
				// Unwrap { data: ... }
				if d, ok := parsed["data"].(map[string]interface{}); ok {
					card = d
				}
				// Unwrap { card: ... } if present
				if c, ok := card["card"].(map[string]interface{}); ok {
					printCardSummary(c)
				} else {
					printCardSummary(card)
				}
			} else {
				fmt.Printf("Context written to %s\n", outPath)
			}

			return nil
		},
	}
}

func newPullBoardCommand() *cobra.Command {
	var projectSlug string

	cmd := &cobra.Command{
		Use:   "board",
		Short: "List active cards on the board",
		RunE: func(cmd *cobra.Command, args []string) error {
			mustLoadToken()

			projectID := projectSlug
			if projectID == "" {
				projectID = cfg.DefaultProject
			}
			if projectID == "" {
				return fmt.Errorf("project ID required. Use --project <slug> or set default_project in config")
			}

			path := api.PathCards + "?project_id=" + projectID
			data, err := apiClient.Get(path)
			if err != nil {
				return handleAPIError(err, "board", projectID, "list")
			}

			var cards []map[string]interface{}
			if err := json.Unmarshal(data, &cards); err != nil {
				// Try { data: [...] } wrapper or { data: { cards: [...] } }.
				var resp map[string]interface{}
				if err2 := json.Unmarshal(data, &resp); err2 == nil {
					// { data: [...] }
					if arr, ok := resp["data"].([]interface{}); ok {
						for _, item := range arr {
							if card, ok := item.(map[string]interface{}); ok {
								cards = append(cards, card)
							}
						}
					} else if d, ok := resp["data"].(map[string]interface{}); ok {
						if arr, ok := d["cards"].([]interface{}); ok {
							for _, item := range arr {
								if card, ok := item.(map[string]interface{}); ok {
									cards = append(cards, card)
								}
							}
						}
					}
				}
			}

			if len(cards) == 0 {
				fmt.Println("No active cards found.")
				return nil
			}

			printCardTable(cards)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectSlug, "project", "", "Project slug or ID")
	return cmd
}

func newPullContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Pull context for the current working card",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read current card from .dea-context/.current-card
			data, err := os.ReadFile(".dea-context/.current-card")
			if err != nil {
				return fmt.Errorf("no current card set. Use `dea claim <card-id>` first")
			}

			cardID := string(data)
			if cardID == "" {
				return fmt.Errorf("no current card set. Use `dea claim <card-id>` first")
			}

			// Delegate to pull card.
			pullCmd := newPullCardCommand()
			return pullCmd.RunE(pullCmd, []string{cardID})
		},
	}
}

func printCardSummary(card map[string]interface{}) {
	title := strField(card, "title", "(no title)")
	lane := strField(card, "lane", strField(card, "status", "unknown"))
	priority := strField(card, "priority", "normal")
	summary := strField(card, "summary", strField(card, "description", ""))

	fmt.Printf("Card: %s\n", title)
	fmt.Printf("  Lane:     %s\n", lane)
	fmt.Printf("  Priority: %s\n", priority)
	if summary != "" {
		fmt.Printf("  Summary:  %s\n", summary)
	}
}

func printCardTable(cards []map[string]interface{}) {
	fmt.Printf("%-20s  %-30s  %-12s  %-8s\n", "ID", "TITLE", "LANE", "PRIORITY")
	fmt.Printf("%-20s  %-30s  %-12s  %-8s\n",
		"--------------------", "------------------------------", "------------", "--------")

	for _, card := range cards {
		id := strField(card, "id", strField(card, "card_id", "?"))
		title := strField(card, "title", "(no title)")
		lane := strField(card, "lane", strField(card, "status", "?"))
		priority := strField(card, "priority", "normal")

		if len(title) > 30 {
			title = title[:27] + "..."
		}

		fmt.Printf("%-20s  %-30s  %-12s  %-8s\n", id, title, lane, priority)
	}
}

func strField(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func handleAPIError(err error, resource, id, op string) error {
	return fmt.Errorf("failed to %s %s %s: %w", op, resource, id, err)
}
