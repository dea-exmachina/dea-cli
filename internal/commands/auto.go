package commands

import (
	"encoding/json"
	"fmt"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

func newAutoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Automation management",
	}

	cmd.AddCommand(newAutoListCommand())
	cmd.AddCommand(newAutoRunCommand())
	cmd.AddCommand(newAutoInspectCommand())

	return cmd
}

func newAutoListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available automations",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Phase 1a stub.
			fmt.Println("Automation management available in Phase 1d.")
			return nil
		},
	}
}

func newAutoRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <automation-id>",
		Short: "Execute an automation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			automationID := args[0]
			mustLoadToken()

			data, err := apiClient.Post(api.AutomationRunPath(automationID), map[string]string{})
			if err != nil {
				if isNetworkErr(err) {
					return err
				}
				return fmt.Errorf("failed to run automation %s: %w", automationID, err)
			}

			var resp map[string]interface{}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &resp); err == nil {
					if msg, ok := resp["message"].(string); ok {
						fmt.Println(msg)
						return nil
					}
				}
			}

			fmt.Printf("Automation %s executed.\n", automationID)
			return nil
		},
	}
}

func newAutoInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <automation-id>",
		Short: "Inspect an automation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Phase 1a stub.
			fmt.Println("Inspect available in Phase 1d.")
			return nil
		},
	}
}
