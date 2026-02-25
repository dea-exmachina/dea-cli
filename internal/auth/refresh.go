package auth

import (
	"fmt"
	"os"
	"time"
)

// RefreshFunc is a function that refreshes a workspace token given the current
// raw JWT. Returns the new TokenData on success.
// Implemented as a function type to avoid import cycles between auth and api.
type RefreshFunc func(currentToken string) (*TokenData, error)

// StartAutoRefresh starts a background goroutine that refreshes the token at
// the 20hr mark (4hr before a 24hr token expiry). Call this from main() after
// successful authentication.
//
// If refresh fails: logs to stderr but does not exit — the CLI continues with
// the existing token until expiry.
func StartAutoRefresh(store *TokenStore, refresh RefreshFunc) {
	go func() {
		for {
			token := store.Load()
			if token == nil {
				time.Sleep(5 * time.Minute)
				continue
			}

			expiresAt := token.ExpiresAt
			// Refresh 4hr before expiry (at ~20hr mark for 24hr tokens).
			refreshAt := expiresAt.Add(-4 * time.Hour)

			now := time.Now()
			if now.Before(refreshAt) {
				time.Sleep(refreshAt.Sub(now))
			}

			// Perform refresh.
			newToken, err := refresh(token.WorkspaceToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "token refresh failed: %v\n", err)
				time.Sleep(5 * time.Minute) // retry in 5 minutes
				continue
			}

			if err := store.Save(newToken); err != nil {
				fmt.Fprintf(os.Stderr, "failed to save refreshed token: %v\n", err)
			}
		}
	}()
}
