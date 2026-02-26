package commands

import (
	"fmt"
	"strings"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

var validSignalTypes = []string{"discovery", "correction", "friction", "pattern"}

func newSignalCommand() *cobra.Command {
	var (
		cardID      string
		signalType  string
		content     string
	)

	cmd := &cobra.Command{
		Use:   "signal",
		Short: "Emit a learning signal for a card",
		Long: fmt.Sprintf("Emit a learning signal. Valid types: %s",
			strings.Join(validSignalTypes, ", ")),
		RunE: func(cmd *cobra.Command, args []string) error {
			mustLoadToken()

			if cardID == "" {
				return fmt.Errorf("--card is required")
			}
			if signalType == "" {
				return fmt.Errorf("--type is required")
			}
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			if !isValidSignalType(signalType) {
				return fmt.Errorf("invalid signal type %q. Valid types: %s",
					signalType, strings.Join(validSignalTypes, ", "))
			}

			// API expects { signals: [...] } wrapper.
			body := map[string]interface{}{
				"signals": []map[string]string{
					{
						"card_id":     cardID,
						"signal_type": signalType,
						"content":     content,
					},
				},
			}

			_, err := apiClient.Post(api.PathSignals, body)
			if err != nil {
				if isNetworkErr(err) {
					if qErr := offQueue.Add("POST", api.PathSignals, body); qErr == nil {
						fmt.Println("Queued offline. Will flush on next connection.")
					}
					return nil
				}
				return fmt.Errorf("failed to emit signal: %w", err)
			}

			fmt.Printf("Signal emitted: [%s] on card %s\n", signalType, cardID)
			return nil
		},
	}

	cmd.Flags().StringVar(&cardID, "card", "", "Card ID to emit the signal for")
	cmd.Flags().StringVar(&signalType, "type", "", "Signal type (discovery|correction|friction|pattern)")
	cmd.Flags().StringVar(&content, "content", "", "Signal content text")

	return cmd
}

func isValidSignalType(t string) bool {
	for _, valid := range validSignalTypes {
		if t == valid {
			return true
		}
	}
	return false
}
