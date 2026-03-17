// Copyright (c) Spice AI, Inc. 2025, 2026
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
	httpClient             *http.Client
	apiEndpoint            string
	oauthEndpoint          string
	clientID               string
	clientSecret           string
	vercelProtectionBypass string

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
func NewSpiceAIClient(clientID, clientSecret, apiEndpoint, oauthEndpoint, vercelProtectionBypass string) *SpiceAIClient {
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
		apiEndpoint:            strings.TrimSuffix(apiEndpoint, "/"),
		oauthEndpoint:          oauthEndpoint,
		clientID:               clientID,
		clientSecret:           clientSecret,
		vercelProtectionBypass: vercelProtectionBypass,
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
	if c.vercelProtectionBypass != "" {
		req.Header.Set("x-vercel-protection-bypass", c.vercelProtectionBypass)
	}

	return c.httpClient.Do(req)
}

// App represents a Spice.ai app.
type App struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description,omitempty"`
	Visibility       string            `json:"visibility,omitempty"`
	CreatedAt        string            `json:"created_at,omitempty"`
	Cname            string            `json:"cname,omitempty"`
	ClusterID        string            `json:"cluster_id,omitempty"`
	ProductionBranch string            `json:"production_branch,omitempty"`
	APIKey           string            `json:"api_key,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	Config           *AppConfig        `json:"config,omitempty"`
}

// AppConfig represents the configuration of an app.
type AppConfig struct {
	Spicepod           interface{}         `json:"spicepod,omitempty"`
	Registry           string              `json:"registry,omitempty"`
	Image              string              `json:"image,omitempty"`
	ImageTag           string              `json:"image_tag,omitempty"`
	UpdateChannel      string              `json:"update_channel,omitempty"`
	Replicas           int                 `json:"replicas,omitempty"`
	Resources          *ContainerResources `json:"resources,omitempty"`
	Executor           *ExecutorConfig     `json:"executor,omitempty"`
	Region             string              `json:"region,omitempty"`
	NodeGroup          string              `json:"node_group,omitempty"`
	StorageClaimSizeGB float64             `json:"storage_claim_size_gb,omitempty"`
}

// CreateAppRequest represents the request to create an app.
type CreateAppRequest struct {
	Name        string              `json:"name"`
	Cname       string              `json:"cname"`
	Description string              `json:"description,omitempty"`
	Visibility  string              `json:"visibility,omitempty"`
	Tags        map[string]string   `json:"tags,omitempty"`
	Replicas    *int                `json:"replicas,omitempty"`
	Resources   *ContainerResources `json:"resources,omitempty"`
	Executor    *ExecutorConfig     `json:"executor,omitempty"`
}

// UpdateAppRequest represents the request to update an app.
type UpdateAppRequest struct {
	Description        string              `json:"description,omitempty"`
	Visibility         string              `json:"visibility,omitempty"`
	ProductionBranch   string              `json:"production_branch,omitempty"`
	Tags               map[string]string   `json:"tags,omitempty"`
	Spicepod           interface{}         `json:"spicepod,omitempty"`
	Registry           string              `json:"registry,omitempty"`
	Image              string              `json:"image,omitempty"`
	ImageTag           string              `json:"image_tag,omitempty"`
	UpdateChannel      string              `json:"update_channel,omitempty"`
	Replicas           *int                `json:"replicas,omitempty"`
	Resources          *ContainerResources `json:"resources,omitempty"`
	Executor           *ExecutorConfig     `json:"executor,omitempty"`
	NodeGroup          string              `json:"node_group,omitempty"`
	Region             string              `json:"region,omitempty"`
	StorageClaimSizeGB *float64            `json:"storage_claim_size_gb,omitempty"`
}

// Deployment represents a Spice.ai deployment.
type Deployment struct {
	ID             int64  `json:"id"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	ImageTag       string `json:"image_tag,omitempty"`
	Replicas       int    `json:"replicas,omitempty"`
	Branch         string `json:"branch,omitempty"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	CommitMessage  string `json:"commit_message,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreationSource string `json:"creation_source,omitempty"`
	CreatedBy      int64  `json:"created_by,omitempty"`
}

// ResourceRequests represents requested container resources.
type ResourceRequests struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// ResourceLimits represents container resource limits.
type ResourceLimits struct {
	CPU              string `json:"cpu,omitempty"`
	Memory           string `json:"memory,omitempty"`
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`
}

// ContainerResources represents resource requests and limits for a container.
type ContainerResources struct {
	Limits   *ResourceLimits   `json:"limits,omitempty"`
	Requests *ResourceRequests `json:"requests,omitempty"`
}

// ExecutorConfig represents executor container configuration.
type ExecutorConfig struct {
	Replicas  *int                `json:"replicas,omitempty"`
	Resources *ContainerResources `json:"resources,omitempty"`
}

// CreateDeploymentRequest represents the request to create a deployment.
type CreateDeploymentRequest struct {
	ImageTag      string `json:"image_tag,omitempty"`
	Channel       string `json:"channel,omitempty"`
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

// Region represents a deployment region.
type Region struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	Provider     string `json:"provider"`
	ProviderName string `json:"providerName"`
	IsDefault    bool   `json:"isDefault"`
	Disabled     bool   `json:"disabled"`
	CName        string `json:"cname"`
}

// RegionsResponse represents the response from listing regions.
type RegionsResponse struct {
	Regions []Region `json:"regions"`
	Default string   `json:"default"`
}

// ContainerImage represents a container image.
type ContainerImage struct {
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Channel string `json:"channel"`
}

// ContainerImagesResponse represents the response from listing container images.
type ContainerImagesResponse struct {
	Images  []ContainerImage `json:"images"`
	Default string           `json:"default"`
}

// APIKeys represents the API keys for an app.
type APIKeys struct {
	APIKey  string `json:"api_key"`
	APIKey2 string `json:"api_key_2"`
}

// Secret represents a Spice.ai app secret.
type Secret struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// SecretsResponse represents the response from listing secrets.
type SecretsResponse struct {
	Secrets []Secret `json:"secrets"`
}

// CreateSecretRequest represents the request to create or update a secret.
type CreateSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Member represents a Spice.ai organization member.
type Member struct {
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	IsOwner   bool     `json:"is_owner,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// MembersResponse represents the response from listing members.
type MembersResponse struct {
	Members []Member `json:"members"`
}

// AddMemberRequest represents the request to add a member.
type AddMemberRequest struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles,omitempty"`
}

// UpdateMemberRequest represents the request to update a member's roles.
type UpdateMemberRequest struct {
	Roles []string `json:"roles"`
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

// ListRegions lists available deployment regions.
func (c *SpiceAIClient) ListRegions(ctx context.Context, env string) (*RegionsResponse, error) {
	path := "/v1/regions"
	if env != "" {
		path += "?env=" + url.QueryEscape(env)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list regions: status %d: %s", resp.StatusCode, string(body))
	}

	var regionsResp RegionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&regionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode regions response: %w", err)
	}

	return &regionsResp, nil
}

// ListContainerImages lists available container images.
func (c *SpiceAIClient) ListContainerImages(ctx context.Context, channel string) (*ContainerImagesResponse, error) {
	path := "/v1/container-images"
	if channel != "" {
		path += "?channel=" + url.QueryEscape(channel)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list container images: status %d: %s", resp.StatusCode, string(body))
	}

	var imagesResp ContainerImagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&imagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode container images response: %w", err)
	}

	return &imagesResp, nil
}

// GetAPIKeys retrieves the API keys for an app.
func (c *SpiceAIClient) GetAPIKeys(ctx context.Context, appID int64) (*APIKeys, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/apps/%d/api-keys", appID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get API keys: status %d: %s", resp.StatusCode, string(body))
	}

	var apiKeys APIKeys
	if err := json.NewDecoder(resp.Body).Decode(&apiKeys); err != nil {
		return nil, fmt.Errorf("failed to decode API keys response: %w", err)
	}

	return &apiKeys, nil
}

// ListSecrets lists all secrets for an app.
func (c *SpiceAIClient) ListSecrets(ctx context.Context, appID int64) ([]Secret, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/apps/%d/secrets", appID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list secrets: status %d: %s", resp.StatusCode, string(body))
	}

	var secretsResp SecretsResponse
	if err := json.NewDecoder(resp.Body).Decode(&secretsResp); err != nil {
		return nil, fmt.Errorf("failed to decode secrets response: %w", err)
	}

	return secretsResp.Secrets, nil
}

// GetSecret retrieves a secret by name.
func (c *SpiceAIClient) GetSecret(ctx context.Context, appID int64, secretName string) (*Secret, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/apps/%d/secrets/%s", appID, url.PathEscape(secretName)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Secret not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get secret: status %d: %s", resp.StatusCode, string(body))
	}

	var secret Secret
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("failed to decode secret response: %w", err)
	}

	return &secret, nil
}

// CreateOrUpdateSecret creates or updates a secret.
func (c *SpiceAIClient) CreateOrUpdateSecret(ctx context.Context, appID int64, req *CreateSecretRequest) (*Secret, error) {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/v1/apps/%d/secrets", appID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create/update secret: status %d: %s", resp.StatusCode, string(body))
	}

	var secret Secret
	if err := json.NewDecoder(resp.Body).Decode(&secret); err != nil {
		return nil, fmt.Errorf("failed to decode secret response: %w", err)
	}

	return &secret, nil
}

// DeleteSecret deletes a secret by name.
func (c *SpiceAIClient) DeleteSecret(ctx context.Context, appID int64, secretName string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/v1/apps/%d/secrets/%s", appID, url.PathEscape(secretName)), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete secret: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListMembers lists all members in the organization.
func (c *SpiceAIClient) ListMembers(ctx context.Context) ([]Member, error) {
	resp, err := c.doRequest(ctx, "GET", "/v1/members", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list members: status %d: %s", resp.StatusCode, string(body))
	}

	var membersResp MembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&membersResp); err != nil {
		return nil, fmt.Errorf("failed to decode members response: %w", err)
	}

	return membersResp.Members, nil
}

// GetMember retrieves a member by user ID.
func (c *SpiceAIClient) GetMember(ctx context.Context, memberID int64) (*Member, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/v1/members/%d", memberID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Member not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get member: status %d: %s", resp.StatusCode, string(body))
	}

	var member Member
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode member response: %w", err)
	}

	return &member, nil
}

// AddMember adds a new member to the organization.
func (c *SpiceAIClient) AddMember(ctx context.Context, req *AddMemberRequest) (*Member, error) {
	resp, err := c.doRequest(ctx, "POST", "/v1/members", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add member: status %d: %s", resp.StatusCode, string(body))
	}

	var member Member
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode member response: %w", err)
	}

	return &member, nil
}

// UpdateMember updates a member's roles.
func (c *SpiceAIClient) UpdateMember(ctx context.Context, memberID int64, req *UpdateMemberRequest) (*Member, error) {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/v1/members/%d", memberID), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update member: status %d: %s", resp.StatusCode, string(body))
	}

	var member Member
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode member response: %w", err)
	}

	return &member, nil
}

// DeleteMember removes a member from the organization.
func (c *SpiceAIClient) DeleteMember(ctx context.Context, memberID int64) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/v1/members/%d", memberID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete member: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
