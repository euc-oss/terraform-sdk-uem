package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AuthProvider defines the interface for authentication providers.
type AuthProvider interface {
	// GetToken returns a valid authentication token.
	GetToken() (string, error)

	// RefreshToken refreshes the authentication token if needed.
	RefreshToken() error
}

// OAuth2Auth implements OAuth2 client credentials flow for Workspace ONE.
type OAuth2Auth struct {
	tokenURL     string // Full token endpoint URL
	clientID     string
	clientSecret string
	token        string
	expiryTime   time.Time
	mu           sync.RWMutex
}

// NewOAuth2Auth creates a new OAuth2 authentication provider.
// TokenURL can be either:
// - Full URL: "https://na.uemauth.workspaceone.com/connect/token" (Workspace ONE cloud)
// - Instance URL: "https://your-instance.awmdm.com" (will append /oauth/token).
func NewOAuth2Auth(tokenURL, clientID, clientSecret string) *OAuth2Auth {
	return &OAuth2Auth{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// GetToken returns a valid OAuth2 token, refreshing if necessary.
func (a *OAuth2Auth) GetToken() (string, error) {
	a.mu.RLock()
	// Check if token is still valid (with 5-minute buffer)
	if a.token != "" && time.Now().Before(a.expiryTime.Add(-5*time.Minute)) {
		token := a.token
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	// Token needs refresh
	if err := a.RefreshToken(); err != nil {
		return "", err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token, nil
}

// RefreshToken obtains a new OAuth2 token from the Workspace ONE API.
func (a *OAuth2Auth) RefreshToken() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Build token endpoint URL
	tokenURL := a.tokenURL
	// If tokenURL doesn't contain a path, append the default OAuth2 path
	if !strings.Contains(tokenURL, "/oauth/token") && !strings.Contains(tokenURL, "/connect/token") {
		tokenURL = fmt.Sprintf("%s/oauth/token", strings.TrimSuffix(tokenURL, "/"))
	}
	// Ensure it has https:// prefix
	if !strings.HasPrefix(tokenURL, "http://") && !strings.HasPrefix(tokenURL, "https://") {
		tokenURL = "https://" + tokenURL
	}

	// Prepare request body
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log but don't override the main error
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Store token and expiry time
	a.token = tokenResp.AccessToken
	a.expiryTime = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// BasicAuth implements Basic Authentication for Workspace ONE.
type BasicAuth struct {
	username string
	password string
	token    string
	once     sync.Once
}

// NewBasicAuth creates a new Basic authentication provider.
func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{
		username: username,
		password: password,
	}
}

// GetToken returns the Base64-encoded credentials for Basic Auth.
func (a *BasicAuth) GetToken() (string, error) {
	a.once.Do(func() {
		credentials := fmt.Sprintf("%s:%s", a.username, a.password)
		a.token = base64.StdEncoding.EncodeToString([]byte(credentials))
	})
	return a.token, nil
}

// RefreshToken is a no-op for Basic Auth as credentials don't expire.
func (a *BasicAuth) RefreshToken() error {
	return nil
}
