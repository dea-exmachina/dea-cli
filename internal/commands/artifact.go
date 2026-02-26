package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dea-exmachina/dea-cli/internal/api"
	"github.com/spf13/cobra"
)

// StagedArtifact represents a locally staged file awaiting push.
type StagedArtifact struct {
	FilePath string `json:"file_path"`
	CardID   string `json:"card_id"`
}

const stagedArtifactsPath = ".dea-context/staged-artifacts.json"

func newArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Stage or push artifacts for a card",
	}

	cmd.AddCommand(newArtifactStageCommand())
	cmd.AddCommand(newArtifactPushCommand())

	return cmd
}

func newArtifactStageCommand() *cobra.Command {
	var cardID string

	cmd := &cobra.Command{
		Use:   "stage <file>",
		Short: "Stage a file for a card (does not upload yet)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			if cardID == "" {
				data, err := os.ReadFile(".dea-context/.current-card")
				if err != nil || len(data) == 0 {
					return fmt.Errorf("--card is required (or run `dea claim <card-id>` first)")
				}
				cardID = strings.TrimSpace(string(data))
			}

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", filePath)
			}

			staged, err := loadStagedArtifacts()
			if err != nil {
				staged = []StagedArtifact{}
			}

			staged = append(staged, StagedArtifact{
				FilePath: filePath,
				CardID:   cardID,
			})

			if err := saveStagedArtifacts(staged); err != nil {
				return fmt.Errorf("failed to save staged artifacts: %w", err)
			}

			fmt.Printf("Staged: %s -> card %s\n", filePath, cardID)
			return nil
		},
	}

	cmd.Flags().StringVar(&cardID, "card", "", "Card ID to stage the artifact for")
	return cmd
}

func newArtifactPushCommand() *cobra.Command {
	var cardID string

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push staged artifacts for a card to the workspace API",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := mustLoadToken()

			if cardID == "" {
				data, err := os.ReadFile(".dea-context/.current-card")
				if err != nil || len(data) == 0 {
					return fmt.Errorf("--card is required (or run `dea claim <card-id>` first)")
				}
				cardID = strings.TrimSpace(string(data))
			}

			staged, err := loadStagedArtifacts()
			if err != nil || len(staged) == 0 {
				fmt.Println("No staged artifacts to push.")
				return nil
			}

			// Partition: items for this card vs. items for other cards.
			var toPush []StagedArtifact
			var remaining []StagedArtifact
			for _, a := range staged {
				if a.CardID == cardID {
					toPush = append(toPush, a)
				} else {
					remaining = append(remaining, a)
				}
			}

			if len(toPush) == 0 {
				fmt.Printf("No staged artifacts for card %s.\n", cardID)
				return nil
			}

			pushedCount := 0
			for _, artifact := range toPush {
				if err := pushArtifact(artifact.FilePath, cardID, token.WorkspaceID); err != nil {
					return fmt.Errorf("failed to push %s: %w", artifact.FilePath, err)
				}
				pushedCount++
			}

			// Update staging list — keep items for other cards.
			if err := saveStagedArtifacts(remaining); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to update staged artifacts: %v\n", err)
			}

			fmt.Printf("Pushed %d artifact(s) for card %s.\n", pushedCount, cardID)
			return nil
		},
	}

	cmd.Flags().StringVar(&cardID, "card", "", "Card ID to push artifacts for")
	return cmd
}

func pushArtifact(filePath, cardID, workspaceID string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	fileData, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	h := sha256.Sum256(fileData)
	fileHash := hex.EncodeToString(h[:])

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot stat file: %w", err)
	}

	filename := filepath.Base(filePath)
	fileType := inferFileType(filename)

	body := map[string]interface{}{
		"workspace_id": workspaceID,
		"card_id":      cardID,
		"filename":     filename,
		"file_type":    fileType,
		"file_hash":    fileHash,
		"storage_path": filePath, // local path for Phase 1a; GCS in Phase 1b
		"file_size":    info.Size(),
	}

	_, err = apiClient.Post(api.PathArtifacts, body)
	if err != nil {
		if isNetworkErr(err) {
			if qErr := offQueue.Add("POST", api.PathArtifacts, body); qErr == nil {
				fmt.Println("Queued offline. Will flush on next connection.")
			}
			return err
		}
		return err
	}

	fmt.Printf("  Pushed: %s (%s, %d bytes)\n", filename, fileType, info.Size())
	return nil
}

func inferFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".sh":
		return "shell"
	case ".sql":
		return "sql"
	case ".txt":
		return "text"
	case ".pdf":
		return "pdf"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		return "image"
	default:
		if ext == "" {
			return "binary"
		}
		return strings.TrimPrefix(ext, ".")
	}
}

func loadStagedArtifacts() ([]StagedArtifact, error) {
	data, err := os.ReadFile(stagedArtifactsPath)
	if os.IsNotExist(err) {
		return []StagedArtifact{}, nil
	}
	if err != nil {
		return nil, err
	}

	var items []StagedArtifact
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveStagedArtifacts(items []StagedArtifact) error {
	if err := os.MkdirAll(".dea-context", 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stagedArtifactsPath, data, 0644)
}
