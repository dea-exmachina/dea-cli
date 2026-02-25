package api

const (
	// PathCards is the base path for card operations.
	PathCards = "/workspace-api/api/cards"

	// PathArtifacts is the path for artifact registration.
	PathArtifacts = "/workspace-api/api/artifacts"

	// PathSignals is the path for emitting signals.
	PathSignals = "/workspace-api/api/signals"

	// PathVault is the path for vault entries.
	PathVault = "/workspace-api/api/vault"

	// PathWorkspaces is the path for workspace registration.
	PathWorkspaces = "/workspace-api/api/workspaces"

	// PathAutomations is the base path for automation operations.
	PathAutomations = "/workspace-api/api/automations"

	// PathTokenIssue is the token-service issue endpoint.
	PathTokenIssue = "/token-service/issue"

	// PathTokenRefresh is the token-service refresh endpoint.
	PathTokenRefresh = "/token-service/refresh"

	// PathTokenRevoke is the token-service revoke endpoint.
	PathTokenRevoke = "/token-service/revoke"
)

// CardPath returns the path for a specific card.
func CardPath(cardID string) string {
	return PathCards + "/" + cardID
}

// CardTransitionPath returns the path for transitioning a card.
func CardTransitionPath(cardID string) string {
	return PathCards + "/" + cardID + "/transition"
}

// CardClaimPath returns the path for claiming a card.
func CardClaimPath(cardID string) string {
	return PathCards + "/" + cardID + "/claim"
}

// CardContextPath returns the path for getting card context.
func CardContextPath(cardID string) string {
	return PathCards + "/" + cardID + "/context"
}

// AutomationRunPath returns the path for running an automation.
func AutomationRunPath(automationID string) string {
	return PathAutomations + "/" + automationID + "/run"
}
