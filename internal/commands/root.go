package commands

import (
	"fmt"
	"os"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/dea-exmachina/dea-cli/internal/auth"
	"github.com/dea-exmachina/dea-cli/internal/config"
	"github.com/dea-exmachina/dea-cli/internal/queue"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	endpointFlag string

	// Shared instances (initialized in initGlobals)
	cfg        *config.Config
	tokenStore *auth.TokenStore
	apiClient  *api.Client
	offQueue   *queue.Queue
)

// Execute is the entry point called from main.go.
func Execute(version, commit, date string) {
	root := newRootCommand(version, commit, date)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:   "dea",
		Short: "dea CLI — governed interface for the dea-exmachina workspace",
		Long: `dea is the command-line interface for the dea-exmachina agent system.
It communicates exclusively with Edge Function endpoints using scoped workspace JWTs.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initGlobals()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	root.PersistentFlags().StringVar(&endpointFlag, "endpoint", "", "Override the API endpoint URL")

	// Register all subcommands
	root.AddCommand(newAuthCommand())
	root.AddCommand(newPullCommand())
	root.AddCommand(newClaimCommand())
	root.AddCommand(newTransitionCommand())
	root.AddCommand(newArtifactCommand())
	root.AddCommand(newSignalCommand())
	root.AddCommand(newDoneCommand())
	root.AddCommand(newWorkspaceCommand())
	root.AddCommand(newAutoCommand())
	root.AddCommand(newUpdateCommand(version, commit, date))

	return root
}

// initGlobals loads config and initializes shared API client + queue.
func initGlobals() error {
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Allow --endpoint flag to override config.
	if endpointFlag != "" {
		cfg.Endpoint = endpointFlag
	}

	tokenStore = auth.NewTokenStore()
	apiClient = api.NewClient(cfg.Endpoint, cfg.TimeoutSeconds, tokenStore)
	offQueue = queue.New()

	// Start background auto-refresh. Bridge api.TokenResponse -> auth.TokenData.
	auth.StartAutoRefresh(tokenStore, func(currentToken string) (*auth.TokenData, error) {
		resp, err := apiClient.RefreshToken(currentToken)
		if err != nil {
			return nil, err
		}
		existing := tokenStore.Load()
		endpoint := cfg.Endpoint
		if existing != nil {
			endpoint = existing.Endpoint
		}
		return &auth.TokenData{
			WorkspaceToken: resp.WorkspaceToken,
			TokenType:      resp.TokenType,
			ExpiresAt:      resp.ExpiresAt,
			WorkspaceID:    resp.WorkspaceID,
			AgentID:        resp.AgentID,
			Endpoint:       endpoint,
		}, nil
	})

	return nil
}

// apiPost wraps a POST call with offline queue support on network errors.
func apiPost(path string, body interface{}) ([]byte, error) {
	resp, err := apiClient.Post(path, body)
	if err != nil {
		if api.IsNetworkError(err) {
			if qErr := offQueue.Add("POST", path, body); qErr != nil {
				fmt.Fprintf(os.Stderr, "failed to queue request: %v\n", qErr)
			} else {
				fmt.Println("Queued offline. Will flush on next connection.")
			}
			return nil, err
		}
		return nil, err
	}
	return resp, nil
}
