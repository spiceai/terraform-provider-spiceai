// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAPIEndpoint   = "https://api.spice.ai"
	DefaultOAuthEndpoint = "https://spice.ai/api/oauth/token"
)

// SpiceAIClient is the client for interacting with the Spice.ai API.
type SpiceAIClient struct {
	httpClient    *http.Client
	apiEndpoint   string
	oauthEndpoint string
	clientID      string
	clientSecret  string

	// Token management
	accessToken string
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex
}

// TokenResponse represents the OAuth token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewSpiceAIClient creates a new Spice.ai API client.
func NewSpiceAIClient(clientID, clientSecret, apiEndpoint, oauthEndpoint string) *SpiceAIClient {
	if apiEndpoint == "" {
		apiEndpoint = DefaultAPIEndpoint
	}
	if oauthEndpoint == "" {
		oauthEndpoint = DefaultOAuthEndpoint
	}

	return &SpiceAIClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiEndpoint:   strings.TrimSuffix(apiEndpoint, "/"),
		oauthEndpoint: oauthEndpoint,
		clientID:      clientID,
		clientSecret:  clientSecret,
	}
}

// getAccessToken retrieves or refreshes the OAuth access token.
func (c *SpiceAIClient) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMutex.RUnlock()
		return token, nil
	}
	c.tokenMutex.RUnlock()

	// Need to refresh token
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", c.oauthEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	// Set expiry with 1 minute buffer
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return c.accessToken, nil
}

// doRequest performs an authenticated HTTP request.
func (c *SpiceAIClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.apiEndpoint + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// App represents a Spice.ai app.
type App struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description,omitempty"`
	Visibility       string     `json:"visibility,omitempty"`
	CreatedAt        string     `json:"created_at,omitempty"`
	Region           string     `json:"region,omitempty"`
	ProductionBranch string     `json:"production_branch,omitempty"`
	APIKey           string     `json:"api_key,omitempty"`
	Config           *AppConfig `json:"config,omitempty"`
}

// AppConfig represents the configuration of an app.
type AppConfig struct {
	Spicepod           interface{} `json:"spicepod,omitempty"`
	ImageTag           string      `json:"image_tag,omitempty"`
	Replicas           int         `json:"replicas,omitempty"`
	NodeGroup          string      `json:"node_group,omitempty"`
	StorageClaimSizeGB float64     `json:"storage_claim_size_gb,omitempty"`
}

// CreateAppRequest represents the request to create an app.
type CreateAppRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

// UpdateAppRequest represents the request to update an app.
type UpdateAppRequest struct {
	Description        string      `json:"description,omitempty"`
	Visibility         string      `json:"visibility,omitempty"`
	ProductionBranch   string      `json:"production_branch,omitempty"`
	Spicepod           interface{} `json:"spicepod,omitempty"`
	ImageTag           string      `json:"image_tag,omitempty"`
	Replicas           *int        `json:"replicas,omitempty"`
	NodeGroup          string      `json:"node_group,omitempty"`
	Region             string      `json:"region,omitempty"`
	StorageClaimSizeGB *float64    `json:"storage_claim_size_gb,omitempty"`
}

// Deployment represents a Spice.ai deployment.
type Deployment struct {
	ID             int64  `json:"id"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	ImageTag       string `json:"image_tag,omitempty"`
	Replicas       int    `json:"replicas,omitempty"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	CommitMessage  string `json:"commit_message,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreationSource string `json:"creation_source,omitempty"`
}

// CreateDeploymentRequest represents the request to create a deployment.
type CreateDeploymentRequest struct {
	ImageTag      string `json:"image_tag,omitempty"`
	Replicas      *int   `json:"replicas,omitempty"`
	Branch        string `json:"branch,omitempty"`
	CommitSHA     string `json:"commit_sha,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	Debug         *bool  `json:"debug,omitempty"`
}

// AppsResponse represents the response from listing apps.
type AppsResponse struct {
	Apps []App `json:"apps"`
}

// DeploymentsResponse represents the response from listing deployments.
type DeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
}

// CreateApp creates a new app.
func (c *SpiceAIClient) CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error) {
	resp, err := c.doRequest(ctx, "POST", "/v1/apps", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create app: status %d: %s", resp.StatusCode, string(body))
	}

	var app App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode app response: %w", err)
	}

	return &app, nil
}

// GetApp retrieves an app by ID.
func (c *SpiceAIClient) GetApp(ctx context.Context, appID int64) (*App, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/apps/%d", appID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // App not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get app: status %d: %s", resp.StatusCode, string(body))
	}

	var app App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode app response: %w", err)
	}

	return &app, nil
}

// UpdateApp updates an app.
func (c *SpiceAIClient) UpdateApp(ctx context.Context, appID int64, req *UpdateAppRequest) (*App, error) {
	resp, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/v1/apps/%d", appID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update app: status %d: %s", resp.StatusCode, string(body))
	}

	var app App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode app response: %w", err)
	}

	return &app, nil
}

// DeleteApp deletes an app.
func (c *SpiceAIClient) DeleteApp(ctx context.Context, appID int64) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/v1/apps/%d", appID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete app: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListApps lists all apps.
func (c *SpiceAIClient) ListApps(ctx context.Context) ([]App, error) {
	resp, err := c.doRequest(ctx, "GET", "/v1/apps", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list apps: status %d: %s", resp.StatusCode, string(body))
	}

	var appsResp AppsResponse
	if err := json.NewDecoder(resp.Body).Decode(&appsResp); err != nil {
		return nil, fmt.Errorf("failed to decode apps response: %w", err)
	}

	return appsResp.Apps, nil
}

// CreateDeployment creates a new deployment for an app.
func (c *SpiceAIClient) CreateDeployment(ctx context.Context, appID int64, req *CreateDeploymentRequest) (*Deployment, error) {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/v1/apps/%d/deployments", appID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create deployment: status %d: %s", resp.StatusCode, string(body))
	}

	var deployment Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode deployment response: %w", err)
	}

	return &deployment, nil
}

// GetDeployment retrieves a specific deployment.
func (c *SpiceAIClient) GetDeployment(ctx context.Context, appID int64, deploymentID int64) (*Deployment, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/apps/%d/deployments?limit=100", appID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list deployments: status %d: %s", resp.StatusCode, string(body))
	}

	var deploymentsResp DeploymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&deploymentsResp); err != nil {
		return nil, fmt.Errorf("failed to decode deployments response: %w", err)
	}

	for _, d := range deploymentsResp.Deployments {
		if d.ID == deploymentID {
			return &d, nil
		}
	}

	return nil, nil // Deployment not found
}

// ListDeployments lists deployments for an app.
func (c *SpiceAIClient) ListDeployments(ctx context.Context, appID int64, limit int, status string) ([]Deployment, error) {
	path := fmt.Sprintf("/v1/apps/%d/deployments?limit=%d", appID, limit)
	if status != "" {
		path += "&status=" + url.QueryEscape(status)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list deployments: status %d: %s", resp.StatusCode, string(body))
	}

	var deploymentsResp DeploymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&deploymentsResp); err != nil {
		return nil, fmt.Errorf("failed to decode deployments response: %w", err)
	}

	return deploymentsResp.Deployments, nil
}
