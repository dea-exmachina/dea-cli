package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrUnauthorized is returned when the API responds with 401.
var ErrUnauthorized = fmt.Errorf("token expired. Run `dea auth refresh`")

// ErrRateLimited is returned when the API responds with 429.
var ErrRateLimited = fmt.Errorf("rate limited. Wait and retry")

// ErrNetwork is the sentinel for network-level failures.
var ErrNetwork = fmt.Errorf("network error")

// TokenProvider is implemented by auth.TokenStore. Using an interface here
// avoids an import cycle between the api and auth packages.
type TokenProvider interface {
	GetToken() string
}

// TokenResponse is returned by token-service endpoints.
type TokenResponse struct {
	WorkspaceToken string    `json:"workspace_token"`
	TokenType      string    `json:"token_type"`
	ExpiresAt      time.Time `json:"expires_at"`
	WorkspaceID    string    `json:"workspace_id"`
	AgentID        string    `json:"agent_id"`
}

// Client is the base HTTP client for the dea Edge Function API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	tokens     TokenProvider
}

// NewClient creates a new API client.
func NewClient(baseURL string, timeoutSeconds int, tokens TokenProvider) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		tokens: tokens,
	}
}

// Get performs an authenticated GET request.
func (c *Client) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil)
}

// Post performs an authenticated POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.do("POST", path, data)
}

// do executes an HTTP request with the workspace JWT in the Authorization header.
func (c *Client) do(method, path string, body []byte) ([]byte, error) {
	token := c.tokens.GetToken()
	if token == "" {
		return nil, fmt.Errorf("not authenticated. Run `dea auth login`")
	}

	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return respBody, nil
	default:
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
}

// RefreshToken calls the token-service/refresh endpoint.
func (c *Client) RefreshToken(currentToken string) (*TokenResponse, error) {
	url := c.baseURL + PathTokenRefresh

	body, err := json.Marshal(map[string]string{"token": currentToken})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+currentToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Edge function wraps response in { data: {...} }
	var wrapper struct {
		Data TokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		// Fall back to flat parse
		var tokenResp TokenResponse
		if err2 := json.Unmarshal(respBody, &tokenResp); err2 != nil {
			return nil, err
		}
		return &tokenResp, nil
	}
	return &wrapper.Data, nil
}

// IssueToken calls token-service/login with bootstrap credentials.
func (c *Client) IssueToken(credentials map[string]string) (*TokenResponse, error) {
	url := c.baseURL + PathTokenLogin

	body, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("login failed: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Edge function wraps response in { data: {...} }
	var wrapper struct {
		Data TokenResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		// Fall back to flat parse
		var tokenResp TokenResponse
		if err2 := json.Unmarshal(respBody, &tokenResp); err2 != nil {
			return nil, err
		}
		return &tokenResp, nil
	}
	return &wrapper.Data, nil
}

// IsNetworkError returns true if the error is a network connectivity error.
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, keyword := range []string{"network error", "connection refused", "no such host", "timeout", "dial"} {
		if containsStr(msg, keyword) {
			return true
		}
	}
	return false
}

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
