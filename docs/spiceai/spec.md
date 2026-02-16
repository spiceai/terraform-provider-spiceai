# Spice AI Management API Specification

**Version:** 1.0.0  
**Base URL:** `https://api.spice.cloud`

This document provides the complete specification for the Spice AI Management API. The API is designed to support infrastructure-as-code workflows, including Terraform providers, CI/CD pipelines, and programmatic management of Spice AI resources.

> **Note:** The OpenAPI specification is auto-generated from API route handlers. See `/v1/docs` endpoint or `apps/api/.schema/openapi.json` for the generated spec.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Authorization Scopes](#authorization-scopes)
4. [Rate Limiting](#rate-limiting)
5. [Error Handling](#error-handling)
6. [Resources](#resources)
   - [Apps](#apps)
   - [Deployments](#deployments)
   - [API Keys](#api-keys)
   - [Secrets](#secrets)
   - [Members](#members)
   - [Regions](#regions)
   - [Container Images](#container-images)
   - [Health](#health)
7. [Common Patterns](#common-patterns)
8. [Terraform Provider Usage](#terraform-provider-usage)

---

## Overview

The Spice AI Management API provides programmatic access to manage Spice AI applications and deployments. It follows RESTful conventions with JSON request/response bodies.

### Key Concepts

| Concept | Description |
|---------|-------------|
| **App** | A Spice AI application containing configuration, datasets, and models |
| **Deployment** | An instance of an app running in a specific region |
| **Spicepod** | The configuration manifest defining an app's data sources and behavior |
| **API Key** | Credentials for authenticating with the Spice runtime |

### API Versioning

The API is versioned via the URL path (e.g., `/v1/`). Breaking changes will result in a new version.

---

## Authentication

The API uses OAuth 2.0 Client Credentials flow for authentication.

### Obtaining Access Tokens

1. Create an OAuth client in the Spice AI Portal
2. Exchange client credentials for an access token
3. Include the token in the `Authorization` header

### Request Headers

```http
Authorization: Bearer <access_token>
Content-Type: application/json
```

### Token Format

Tokens are JWTs containing:
- `client_id`: The OAuth client identifier (UUIDv7)
- `org_id`: The organization ID the token is scoped to
- `scopes`: Array of granted scopes
- `jti`: Unique token identifier (for revocation)
- `exp`: Token expiration timestamp

### Token Validation

The API validates tokens by:
1. Verifying JWT signature (HS256)
2. Checking token expiration
3. Validating required claims
4. Confirming client is active and not revoked
5. Ensuring token scopes are authorized for the client

---

## Authorization Scopes

Scopes control what operations a token can perform.

| Scope | Description | Implied Scopes |
|-------|-------------|----------------|
| `apps:read` | Read app information | - |
| `apps:write` | Create and update apps | `apps:read` |
| `apps:delete` | Delete apps | `apps:read`, `apps:write` |
| `deployments:read` | Read deployment information | - |
| `deployments:write` | Create deployments | `deployments:read` |
| `config:read` | Read app configuration | - |
| `config:write` | Update app configuration | `config:read` |
| `secrets:read` | Read secrets (sensitive) | - |
| `secrets:write` | Create and update secrets (sensitive) | `secrets:read` |
| `members:read` | Read organization members | - |
| `members:write` | Add and update members | `members:read` |
| `members:delete` | Remove members | `members:read`, `members:write` |
| `*` | Full access to all operations | All scopes |

### Scope Hierarchy

Write scopes implicitly include read scopes. For example, `apps:write` grants `apps:read` access.

---

## Rate Limiting

| Limit Type | Value |
|------------|-------|
| Requests per minute | 1000 |
| Requests per hour | 10000 |
| Concurrent deployments | 10 |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining in window
- `X-RateLimit-Reset`: Unix timestamp when limit resets

---

## Error Handling

### Error Response Format

```json
{
  "error": "Human-readable error message",
  "details": {
    "fieldErrors": {},
    "formErrors": []
  }
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `201` | Created |
| `202` | Accepted (async operation started) |
| `204` | No Content (successful delete) |
| `400` | Bad Request - Invalid input |
| `401` | Unauthorized - Missing or invalid token |
| `403` | Forbidden - Insufficient scope |
| `404` | Not Found - Resource doesn't exist |
| `409` | Conflict - Resource already exists or concurrent operation |
| `500` | Internal Server Error |

### Common Error Scenarios

| Scenario | Status | Error Message |
|----------|--------|---------------|
| Missing Authorization header | 401 | `Missing Authorization header` |
| Invalid token format | 401 | `Invalid token format` |
| Token expired | 401 | `Token expired` |
| Insufficient scope | 403 | `Insufficient scope. Required: <scope>` |
| App not found | 404 | `App with ID <id> not found` |
| App name exists | 409 | `App "<name>" already exists in this organization` |
| App limit exceeded | 403 | `Organization has reached the maximum app limit` |

---

## Resources

### Apps

An **App** represents a Spice AI application. Apps contain configuration, are deployed to regions, and have associated API keys for runtime authentication.

#### App Object

```json
{
  "id": 12345,
  "name": "my-app",
  "description": "My Spice AI application",
  "visibility": "private",
  "created_at": "2024-01-15T10:30:00Z",
  "region": "us-east-2",
  "cluster_id": "cluster-abc123",
  "api_key": "sk_live_xxxxx",
  "tags": {
    "environment": "production",
    "team": "data"
  },
  "config": {
    "spicepod": { ... },
    "registry": "ghcr.io/spiceai",
    "image": "spiceai-enterprise",
    "image_tag": "1.5.0-models",
    "update_channel": "stable",
    "replicas": 2,
    "node_group": "standard",
    "storage_claim_size_gb": 10
  }
}
```

#### App Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | integer | Read-only | Unique app identifier |
| `name` | string | Yes | App name (4+ chars, alphanumeric and hyphens) |
| `description` | string | No | Human-readable description |
| `visibility` | string | No | `public` or `private` (default: `private`) |
| `created_at` | string | Read-only | ISO 8601 creation timestamp |
| `region` | string | No | Deployment region (e.g., `us-east-2`) |
| `cluster_id` | string | Read-only | Kubernetes cluster identifier |
| `api_key` | string | Read-only | Primary API key for runtime |
| `tags` | object | No | Key-value tags for the app |
| `config` | object | No | App configuration (see Config Object) |

#### Config Object

| Field | Type | Description |
|-------|------|-------------|
| `spicepod` | object | Spicepod configuration manifest |
| `registry` | string | Registry for the spiced image (e.g., `ghcr.io/spiceai`) |
| `image` | string | Image name for the spiced container (e.g., `spiceai-enterprise`) |
| `image_tag` | string | Spice runtime container image tag |
| `update_channel` | string | Update channel: `stable`, `nightly`, `internal`, or `internal-sandbox` |
| `replicas` | integer | Number of runtime replicas (1-10) |
| `node_group` | string | Compute node group |
| `storage_claim_size_gb` | number | Persistent storage size in GB |

---

#### List Apps

```
GET /v1/apps
```

Returns all apps in the authenticated organization.

**Required Scope:** `apps:read`

**Response:**

```json
{
  "apps": [
    {
      "id": 12345,
      "name": "my-app",
      "description": "My application",
      "visibility": "private",
      "created_at": "2024-01-15T10:30:00Z",
      "region": "us-east-2",
      "cluster_id": "cluster-abc123",
      "api_key": "sk_live_xxxxx",
      "tags": {
        "environment": "production"
      }
    }
  ]
}
```

---

#### Create App

```
POST /v1/apps
```

Creates a new app in the authenticated organization.

**Required Scope:** `apps:write`

**Request Body:**

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `name` | string | Yes | 4+ chars, pattern: `^[a-zA-Z0-9-]+$` | Unique app name |
| `description` | string | No | - | App description |
| `visibility` | string | No | `public` \| `private` | Default: `private` |
| `tags` | object | No | Key-value pairs | Custom tags for the app |

**Example Request:**

```json
{
  "name": "my-new-app",
  "description": "A new Spice AI application",
  "visibility": "private",
  "tags": {
    "environment": "staging",
    "team": "engineering"
  }
}
```

**Response:** `201 Created`

```json
{
  "id": 12346,
  "name": "my-new-app",
  "description": "A new Spice AI application",
  "visibility": "private",
  "created_at": "2024-01-15T12:00:00Z",
  "region": null,
  "cluster_id": null,
  "api_key": "sk_live_yyyyy"
}
```

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400` | Invalid name format or missing required fields |
| `403` | Organization app limit reached |
| `409` | App with this name already exists |

---

#### Get App

```
GET /v1/apps/{appId}
```

Returns details for a specific app, including its configuration.

**Required Scope:** `apps:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Response:** `200 OK`

```json
{
  "id": 12345,
  "name": "my-app",
  "description": "My application",
  "visibility": "private",
  "created_at": "2024-01-15T10:30:00Z",
  "production_branch": "main",
  "api_key": "sk_live_xxxxx",
  "tags": {
    "environment": "production",
    "team": "data"
  },
  "config": {
    "spicepod": {
      "version": "v1",
      "kind": "Spicepod",
      "name": "my-app",
      "datasets": [...]
    },
    "registry": "ghcr.io/spiceai",
    "image": "spiceai-enterprise",
    "image_tag": "1.5.0-models",
    "update_channel": "stable",
    "replicas": 2,
    "region": "us-east-2",
    "node_group": "standard",
    "storage_claim_size_gb": 10
  }
}
```

---

#### Update App

```
PUT /v1/apps/{appId}
```

Updates an app's metadata and configuration.

**Required Scope:** `apps:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Request Body:**

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | App description |
| `visibility` | string | `public` or `private` |
| `production_branch` | string | Git branch for production |
| `tags` | object | Key-value tags for the app |
| `spicepod` | string \| object | Spicepod config (YAML string or JSON object) |
| `registry` | string | Registry for the spiced image |
| `image` | string | Image name for the spiced container |
| `image_tag` | string | Runtime container image tag |
| `update_channel` | string | Update channel: `stable`, `nightly`, `internal`, or `internal-sandbox` |
| `replicas` | integer | Number of replicas (1-10) |
| `node_group` | string | Compute node group |
| `region` | string | Deployment region |
| `storage_claim_size_gb` | number | Storage size in GB |

**Example Request:**

```json
{
  "description": "Updated description",
  "tags": {
    "environment": "production",
    "version": "2.0"
  },
  "replicas": 3,
  "spicepod": {
    "version": "v1",
    "kind": "Spicepod",
    "name": "my-app",
    "datasets": [
      {
        "name": "my_dataset",
        "from": "s3://bucket/path",
        "acceleration": {
          "enabled": true
        }
      }
    ]
  }
}
```

**Response:** `200 OK`

Returns the updated app object.

---

#### Delete App

```
DELETE /v1/apps/{appId}
```

Soft deletes an app (sets `deleted_at` timestamp). Apps can be reactivated by creating a new app with the same name.

**Required Scope:** `apps:delete`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Response:** `204 No Content`

---

### Deployments

A **Deployment** represents a running instance of an app's configuration. Deployments are created from the app's current spicepod configuration.

#### Deployment Object

```json
{
  "id": 67890,
  "status": "succeeded",
  "created_at": "2024-01-15T12:00:00Z",
  "updated_at": "2024-01-15T12:05:00Z",
  "image_tag": "1.5.0-models",
  "replicas": 2,
  "branch": "main",
  "commit_sha": "abc123def456",
  "commit_message": "Update dataset configuration",
  "error_message": null,
  "creation_source": "api",
  "created_by": 123
}
```

#### Deployment Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique deployment identifier |
| `status` | string | Deployment status (see Status Values) |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last update timestamp |
| `image_tag` | string | Runtime container image tag |
| `replicas` | integer | Number of runtime replicas |
| `branch` | string | Git branch name |
| `commit_sha` | string | Git commit SHA |
| `commit_message` | string | Git commit message |
| `error_message` | string | Error details (if failed) |
| `creation_source` | string | How deployment was created |
| `created_by` | integer | User ID who created deployment |

#### Deployment Status Values

| Status | Description |
|--------|-------------|
| `queued` | Deployment is queued for processing |
| `in_progress` | Deployment is being applied |
| `succeeded` | Deployment completed successfully |
| `failed` | Deployment failed (see `error_message`) |
| `created` | Deployment record created |

---

#### List Deployments

```
GET /v1/apps/{appId}/deployments
```

Returns deployments for the specified app.

**Required Scope:** `deployments:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Max deployments to return (max: 100) |
| `status` | string | - | Filter by status |

**Response:** `200 OK`

```json
{
  "deployments": [
    {
      "id": 67890,
      "status": "succeeded",
      "created_at": "2024-01-15T12:00:00Z",
      "updated_at": "2024-01-15T12:05:00Z",
      "image_tag": "1.5.0-models",
      "replicas": 2,
      "branch": "main",
      "commit_sha": "abc123def456",
      "commit_message": "Update dataset configuration",
      "error_message": null,
      "creation_source": "api",
      "created_by": 123
    }
  ]
}
```

---

#### Create Deployment

```
POST /v1/apps/{appId}/deployments
```

Creates a new deployment using the app's current spicepod configuration.

**Required Scope:** `deployments:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Request Body (all fields optional):**

| Field | Type | Description |
|-------|------|-------------|
| `image_tag` | string | Override runtime image tag |
| `replicas` | integer | Override replica count (1-10) |
| `branch` | string | Git branch name |
| `commit_sha` | string | Git commit SHA |
| `commit_message` | string | Git commit message |
| `debug` | boolean | Enable debug mode |

**Example Request:**

```json
{
  "image_tag": "1.5.0-models",
  "replicas": 2,
  "branch": "main",
  "commit_sha": "abc123def456",
  "commit_message": "Deploy new dataset configuration"
}
```

**Response:** `202 Accepted`

```json
{
  "id": 67891,
  "status": "queued",
  "created_at": "2024-01-15T14:00:00Z",
  "updated_at": null,
  "image_tag": "1.5.0-models",
  "replicas": 2,
  "branch": "main",
  "commit_sha": "abc123def456",
  "commit_message": "Deploy new dataset configuration",
  "error_message": null,
  "creation_source": "api",
  "created_by": null
}
```

**Prerequisites:**
- App must have a spicepod configuration
- App must have an API key
- Spicepod must not be deleted or paused

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400` | App has no spicepod configuration |
| `400` | Invalid spicepod configuration |
| `400` | Spicepod is deleted or paused |

---

### API Keys

API keys authenticate requests to the Spice runtime. Each app has two keys for rotation.

#### API Keys Object

```json
{
  "api_key": "sk_live_xxxxx",
  "api_key_2": "sk_live_yyyyy"
}
```

---

#### Get API Keys

```
GET /v1/apps/{appId}/api-keys
```

Returns the API keys for an app.

**Required Scope:** `apps:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Response:** `200 OK`

```json
{
  "api_key": "sk_live_xxxxx",
  "api_key_2": "sk_live_yyyyy"
}
```

---

#### Regenerate API Key

```
POST /v1/apps/{appId}/api-keys
```

Regenerates an API key. The previous key is immediately invalidated.

**Required Scope:** `apps:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Request Body:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `key_number` | integer | 1 | Which key to regenerate: `0` = both, `1` = primary, `2` = secondary |

**Example Request:**

```json
{
  "key_number": 1
}
```

**Response:** `200 OK`

```json
{
  "api_key": "sk_live_new_xxxxx",
  "api_key_2": "sk_live_yyyyy",
  "regenerated_key": 1
}
```

---

### Secrets

Secrets store sensitive configuration values (API keys, passwords, connection strings) for use in your Spicepod configuration. Secret values are encrypted at rest and masked in API responses.

#### Secret Object

```json
{
  "id": 456,
  "name": "DATABASE_PASSWORD",
  "value": "********",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### Secret Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique secret identifier |
| `name` | string | Secret name (must start with letter or underscore) |
| `value` | string | Secret value (always masked in responses) |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last update timestamp |

---

#### List Secrets

```
GET /v1/apps/{appId}/secrets
```

Returns all secrets for the specified app. Secret values are masked.

**Required Scope:** `secrets:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Response:** `200 OK`

```json
{
  "secrets": [
    {
      "id": 456,
      "name": "DATABASE_PASSWORD",
      "value": "********",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": 457,
      "name": "API_TOKEN",
      "value": "********",
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  ]
}
```

---

#### Get Secret

```
GET /v1/apps/{appId}/secrets/{secretName}
```

Returns a specific secret by name. The value is masked.

**Required Scope:** `secrets:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |
| `secretName` | string | The secret name |

**Response:** `200 OK`

```json
{
  "id": 456,
  "name": "DATABASE_PASSWORD",
  "value": "********",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

#### Create or Update Secret

```
POST /v1/apps/{appId}/secrets
```

Creates a new secret or updates an existing one with the same name.

**Required Scope:** `secrets:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |

**Request Body:**

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `name` | string | Yes | Pattern: `^[a-zA-Z_][a-zA-Z0-9_]*$` | Secret name |
| `value` | string | Yes | - | Secret value (will be encrypted) |

**Example Request:**

```json
{
  "name": "DATABASE_PASSWORD",
  "value": "my-secure-password-123"
}
```

**Response:** `200 OK`

```json
{
  "id": 456,
  "name": "DATABASE_PASSWORD",
  "value": "********",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400` | Invalid secret name format |
| `404` | App not found |

---

#### Delete Secret

```
DELETE /v1/apps/{appId}/secrets/{secretName}
```

Deletes a secret by name.

**Required Scope:** `secrets:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `appId` | integer | The app ID |
| `secretName` | string | The secret name |

**Response:** `204 No Content`

---

### Members

Members represent users belonging to an organization. The Members API allows you to manage organization membership and roles programmatically.

#### Member Object

```json
{
  "user_id": 123,
  "username": "johndoe",
  "roles": ["admin", "member"],
  "is_owner": false,
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### Member Fields

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | integer | Unique user identifier |
| `username` | string | User's username |
| `roles` | array | List of assigned roles |
| `is_owner` | boolean | Whether the member is the organization owner |
| `created_at` | string | ISO 8601 timestamp when member joined |

---

#### List Members

```
GET /v1/members
```

Returns all members in the authenticated organization.

**Required Scope:** `members:read`

**Response:** `200 OK`

```json
{
  "members": [
    {
      "user_id": 123,
      "username": "johndoe",
      "roles": ["admin", "member"],
      "is_owner": true,
      "created_at": "2024-01-10T08:00:00Z"
    },
    {
      "user_id": 456,
      "username": "janedoe",
      "roles": ["member"],
      "is_owner": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

#### Get Member

```
GET /v1/members/{memberId}
```

Returns details for a specific organization member.

**Required Scope:** `members:read`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `memberId` | integer | The user ID of the member |

**Response:** `200 OK`

```json
{
  "user_id": 456,
  "username": "janedoe",
  "roles": ["member"],
  "is_owner": false,
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

#### Add Member

```
POST /v1/members
```

Adds a new member to the organization with specified roles.

**Required Scope:** `members:write`

**Request Body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `username` | string | Yes | - | Username of the user to add |
| `roles` | array | No | `["member"]` | Roles to assign to the member |

**Example Request:**

```json
{
  "username": "newuser",
  "roles": ["member"]
}
```

**Response:** `201 Created`

```json
{
  "user_id": 789,
  "username": "newuser",
  "roles": ["member"],
  "created_at": "2024-01-20T14:00:00Z"
}
```

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400` | Invalid request body |
| `404` | User not found |
| `409` | User is already a member |

---

#### Update Member Roles

```
PATCH /v1/members/{memberId}
```

Updates the roles of a specific organization member.

**Required Scope:** `members:write`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `memberId` | integer | The user ID of the member |

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `roles` | array | Yes | New roles to assign to the member |

**Example Request:**

```json
{
  "roles": ["admin", "member"]
}
```

**Response:** `200 OK`

```json
{
  "user_id": 456,
  "username": "janedoe",
  "roles": ["admin", "member"],
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `400` | Invalid request body |
| `403` | Cannot modify organization owner |
| `404` | Member not found |

---

#### Remove Member

```
DELETE /v1/members/{memberId}
```

Removes a member from the organization (soft delete).

**Required Scope:** `members:delete`

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `memberId` | integer | The user ID of the member |

**Response:** `204 No Content`

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `403` | Cannot remove organization owner |
| `404` | Member not found |

---

### Regions

Regions define where apps can be deployed.

#### Region Object

```json
{
  "name": "US East (Ohio)",
  "region": "us-east-2",
  "provider": "aws",
  "providerName": "AWS",
  "isDefault": true,
  "disabled": false,
  "cname": "us-west-2-prod-aws-data"
}
```

---

#### List Regions

```
GET /v1/regions
```

Returns available deployment regions.

**Required Scope:** `apps:read`

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `env` | string | Filter by environment: `prod` \| `dev` |

**Response:** `200 OK`

```json
{
  "regions": [
    {
      "name": "US East (Ohio)",
      "region": "us-east-2",
      "provider": "aws",
      "providerName": "AWS",
      "isDefault": true,
      "disabled": false,
      "cname": "us-west-2-prod-aws-data"
    },
    {
      "name": "US West (Oregon)",
      "region": "us-west-2",
      "provider": "aws",
      "providerName": "AWS",
      "isDefault": false,
      "disabled": false,
      "cname": "us-west-2-prod-aws-data"
    }
  ],
  "default": "us-east-2"
}
```

---

### Container Images

Container images are the Spice runtime versions available for deployments.

#### Container Image Object

```json
{
  "name": "spiceai/spiceai:1.5.0-models",
  "tag": "1.5.0-models",
  "channel": "stable"
}
```

---

#### List Container Images

```
GET /v1/container-images
```

Returns available Spice runtime container images.

**Required Scope:** `apps:read`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `channel` | string | `stable` | Release channel: `stable` \| `enterprise` |

**Response:** `200 OK`

```json
{
  "images": [
    {
      "name": "spiceai/spiceai:1.5.0-models",
      "tag": "1.5.0-models",
      "channel": "stable"
    },
    {
      "name": "spiceai/spiceai:1.4.0-models",
      "tag": "1.4.0-models",
      "channel": "stable"
    }
  ],
  "default": "1.5.0-models"
}
```

---

### Health

#### Health Check

```
GET /v1/health
```

Returns the API health status. No authentication required.

**Response:** `200 OK`

```json
{
  "status": "ok",
  "timestamp": "2024-01-15T14:30:00Z"
}
```

---

## Common Patterns

### Pagination

List endpoints support pagination via `limit` and `offset` query parameters:

```
GET /v1/apps/{appId}/deployments?limit=10&offset=20
```

### Idempotency

Create operations are idempotent when using the same name:
- Creating an app with an existing name returns `409 Conflict`
- Creating an app with a soft-deleted name reactivates it

### Async Operations

Deployment creation is asynchronous:
1. `POST /v1/apps/{appId}/deployments` returns `202 Accepted`
2. Poll `GET /v1/apps/{appId}/deployments` to check status
3. Deployment progresses: `queued` → `in_progress` → `succeeded` | `failed`

### Soft Deletes

Apps are soft-deleted (set `deleted_at` timestamp):
- Deleted apps are excluded from list responses
- Creating an app with a deleted name reactivates it
- Deployments for deleted apps are stopped

---

## Terraform Provider Usage

The API is designed to support Terraform provider workflows.

### Example Terraform Configuration

```hcl
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 1.0"
    }
  }
}

provider "spiceai" {
  # Uses SPICEAI_CLIENT_ID and SPICEAI_CLIENT_SECRET env vars
}

resource "spiceai_app" "example" {
  name        = "my-terraform-app"
  description = "Managed by Terraform"
  visibility  = "private"
  region      = "us-east-2"
  cname       = "us-west-2-prod-aws-data"

  config {
    replicas  = 2
    image_tag = "1.5.0-models"
    
    spicepod = jsonencode({
      version = "v1"
      kind    = "Spicepod"
      name    = "my-terraform-app"
      datasets = [
        {
          name = "my_dataset"
          from = "s3://bucket/path"
          acceleration = {
            enabled = true
          }
        }
      ]
    })
  }
}

resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id
  
  # Optional overrides
  replicas  = 3
  image_tag = "1.5.0-models"
}
```

### Terraform Resource Mapping

| Terraform Resource | API Endpoints |
|-------------------|---------------|
| `spiceai_app` | `POST/GET/PUT/DELETE /v1/apps/{appId}` |
| `spiceai_deployment` | `POST/GET /v1/apps/{appId}/deployments` |
| `spiceai_app_api_key` | `GET/POST /v1/apps/{appId}/api-keys` |
| `spiceai_secret` | `GET/POST/DELETE /v1/apps/{appId}/secrets` |
| `spiceai_member` | `GET/POST/PATCH/DELETE /v1/members` |

### Import Existing Resources

```bash
# Import existing app
terraform import spiceai_app.example 12345

# Import by name
terraform import spiceai_app.example my-existing-app
```

### State Refresh

The provider refreshes state by calling:
1. `GET /v1/apps/{appId}` - Current app configuration
2. `GET /v1/apps/{appId}/deployments?limit=1&status=succeeded` - Latest successful deployment

---

## Spicepod Configuration Reference

The spicepod is the core configuration for a Spice AI app.

### Minimal Spicepod

```json
{
  "version": "v1",
  "kind": "Spicepod",
  "name": "my-app"
}
```

### Full Spicepod Example

```json
{
  "version": "v1",
  "kind": "Spicepod",
  "name": "my-app",
  "datasets": [
    {
      "name": "my_dataset",
      "from": "s3://my-bucket/data/",
      "acceleration": {
        "enabled": true,
        "mode": "memory",
        "refresh_mode": "full",
        "refresh_check_interval": "10m"
      },
      "columns": [
        {
          "name": "id",
          "type": "integer"
        },
        {
          "name": "value",
          "type": "string"
        }
      ]
    }
  ],
  "models": [
    {
      "name": "my_model",
      "from": "huggingface:huggingface.co/model-name"
    }
  ],
  "embeddings": [
    {
      "name": "my_embedding",
      "from": "openai"
    }
  ]
}
```

### Spicepod Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Schema version (`v1` or `v1beta1`) |
| `kind` | string | Yes | Must be `Spicepod` |
| `name` | string | Yes | App name |
| `datasets` | array | No | Data source configurations |
| `models` | array | No | ML model configurations |
| `embeddings` | array | No | Embedding model configurations |
| `catalogs` | array | No | External catalog configurations |
| `views` | array | No | SQL view definitions |

---

## OpenAPI Specification

The complete OpenAPI 3.1 specification is auto-generated from API route handlers and available at:

```
GET /v1/docs
```

This returns the JSON OpenAPI document that can be used with code generators and API tools.

**Source locations:**
- Generated spec: `apps/api/.schema/openapi.json`
- Route handlers: `apps/api/app/v1/*/route.ts`

The OpenAPI spec is generated from JSDoc `@swagger` annotations in the route handlers.

### Component Schemas

The OpenAPI spec defines reusable schemas in `components.schemas`:

| Schema | Description |
|--------|-------------|
| `App` | Basic app object (used in list/create responses) |
| `AppWithConfig` | Full app object with config (used in get/update responses) |
| `Secret` | Secret object with masked value |
| `Deployment` | Deployment object with status and metadata |
| `ApiKeys` | API keys object with primary and secondary keys |
| `ApiKeysRegenerated` | API keys response after regeneration |

### Security Scheme

The spec defines a `BearerAuth` security scheme for OAuth 2.0 JWT authentication:

```yaml
securitySchemes:
  BearerAuth:
    type: http
    scheme: bearer
    bearerFormat: JWT
```

All protected endpoints reference this scheme via `security: [{ BearerAuth: [] }]`.

### Regenerating the Spec

To regenerate the OpenAPI spec after modifying route handlers:

```bash
cd apps/api
yarn openapi:generate
```

---

## Changelog

### 2025-12-15

- Added reusable component schemas to OpenAPI spec: `App`, `AppWithConfig`, `Secret`, `Deployment`, `ApiKeys`, `ApiKeysRegenerated`
- Added `BearerAuth` security scheme definition to OpenAPI components
- Updated API endpoints to reference component schemas via `$ref` for better code generation support

### 2025-12-12

- Added `registry`, `image`, and `update_channel` fields to App config
- `update_channel` supports values: `stable`, `nightly`, `internal`, `internal-sandbox`
- Added `tags` field to App object for custom key-value metadata

### 2025-12-11

- Added Members API for organization member management
- Added Secrets API for app secret management
- New scopes: `members:read`, `members:write`, `members:delete`

### 2025-12-10

- Initial API release
- App CRUD operations
- Deployment management
- API key management
- Region and container image listing
- OAuth 2.0 authentication

---

## Support

- Documentation: https://docs.spice.ai
- GitHub Issues: https://github.com/spiceai/spiceai/issues
