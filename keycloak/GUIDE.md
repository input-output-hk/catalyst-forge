# Keycloak Terraform Implementation Guide

**Version:** 1.0
**Author:** Gemini

## 1. Introduction

This guide provides instructions for configuring Keycloak using Terraform, based on the requirements outlined in the SRS document. It assumes that Keycloak has already been deployed via the Keycloak Operator into a Kubernetes cluster and that the provider is being configured to connect to that instance.

This guide is not an exhaustive Terraform tutorial but rather a specific set of instructions for implementing the required Keycloak configuration. It assumes proficiency in Terraform.

### 1.1. File Structure

To keep the configuration organized, we will use a flat module structure with files separated by logical function:

- `provider.tf`: Defines the Keycloak provider configuration.
- `realm.tf`: Configures the main `forge` realm, including token and session settings.
- `identity_providers.tf`: Configures Google Workspace and GitHub OIDC as identity providers.
- `roles_groups.tf`: Defines Keycloak roles and groups.
- `clients_oidc.tf`: Configures all OpenID Connect clients (frontend, backend, CLI, etc.).
- `clients_saml.tf`: Configures SAML clients, specifically for AWS integration.
- `variables.tf`: Contains variable definitions.
- `outputs.tf`: Defines outputs from the module.

---

## 2. Provider Configuration

The first step is to configure the Terraform provider to connect to your Keycloak instance. You will need the Keycloak URL, a client ID, and a client secret for a client with administrative privileges (e.g., `admin-cli`).

**File:** `provider.tf`

```terraform
# Example Provider Configuration
provider "keycloak" {
  url          = var.keycloak_url # e.g., "https://auth.projectcatalyst.dev"
  client_id    = "admin-cli"
  client_secret = var.keycloak_admin_cli_secret
  realm        = "master"
}
```

---

## 3. Realm Configuration

We will create a single realm named `forge` to house all configuration. This resource will also define global authentication policies, session management, and token lifespans as required by the SRS.

**File:** `realm.tf`

**SRS Requirements:**
- `REQ-AUTH-007`: SSO sessions for a maximum of 8 hours.
- `REQ-AUTH-008`: Access tokens with a 5-minute lifespan.
- `REQ-AUTH-014`: Refresh tokens with 8-hour lifespan for CLI authentication.
- `REQ-SEC-001` - `REQ-SEC-004`: Token lifespans, inactivity timeouts, and refresh token rotation.

Use the `keycloak_realm` resource. The settings below satisfy the SRS requirements. Note that refresh token rotation is enabled by default in modern Keycloak versions when a new refresh token is issued.

```terraform
# Example Realm Configuration
resource "keycloak_realm" "catalyst_forge" {
  realm   = "forge"
  enabled = true

  # SSL required for all external requests
  ssl_required = "all"

  # Disable user registration, password resets, etc.
  registration_allowed = false
  reset_password_allowed = false
  edit_username_allowed  = false

  # Token and Session Lifespans
  access_token_lifespan                = "5m"
  access_token_lifespan_for_implicit_flow = "5m"
  sso_session_idle_timeout             = "1h"
  sso_session_max_lifespan             = "8h"
  offline_session_idle_timeout         = "8h"

  # Enable refresh token rotation by revoking old tokens on refresh
  revoke_refresh_token = true

  # Brute force detection
  security_defenses {
    brute_force_detection {
      max_login_failures           = 10
      wait_increment_seconds       = 60
      minimum_quick_login_wait_seconds = 10
      max_failure_wait_seconds     = 3600
      failure_reset_time_seconds   = 43200
    }
  }
}
```

---

## 4. Identity Provider Configuration

We need to configure Google Workspace as the primary OIDC identity provider for user authentication and GitHub as an OIDC provider for token exchange in CI/CD workflows.

**File:** `identity_providers.tf`

### 4.1. Google Workspace (OIDC)

**SRS Requirements:**
- `REQ-AUTH-001`: Use Google Workspace as the sole identity provider.
- `REQ-AUTH-003`: Support Just-In-Time (JIT) provisioning.

Use the `keycloak_oidc_identity_provider` resource. Set `sync_mode = "FORCE"` to ensure user attributes are updated on each login. The `first_broker_login_flow_alias` handles JIT provisioning.

```terraform
# Example Google OIDC Identity Provider
resource "keycloak_oidc_identity_provider" "google" {
  realm              = keycloak_realm.catalyst_forge.id
  alias              = "google"
  enabled            = true

  # This enables JIT provisioning
  first_broker_login_flow_alias = "first broker login"

  # Update user profile info on each login
  sync_mode          = "FORCE"

  # Trust the email claim from Google
  trust_email        = true

  # Configuration from Google Cloud project
  client_id          = var.google_oidc_client_id
  client_secret      = var.google_oidc_client_secret

  # Optional: Restrict logins to a specific domain
  # hosted_domain    = "yourcompany.com"
}
```

### 4.2. GitHub Actions (OIDC for Token Exchange)

**SRS Requirements:**
- `REQ-AUTH-020`: Support OIDC federation for GitHub Actions.
- `REQ-AUTH-021`: Validate GitHub OIDC tokens.

This also uses the `keycloak_oidc_identity_provider` resource, but configured to trust GitHub's token issuer. We will later create a permission resource to allow token exchange.

```terraform
# Example GitHub OIDC Identity Provider for Token Exchange
resource "keycloak_oidc_identity_provider" "github" {
  realm              = keycloak_realm.catalyst_forge.id
  alias              = "github-oidc"
  enabled            = true

  authorization_url  = "https://github.com" # Placeholder
  token_url          = "https://github.com" # Placeholder
  issuer             = "https://token.actions.githubusercontent.com"
  jwks_url           = "https://token.actions.githubusercontent.com/.well-known/jwks"

  # Disable browser flows for this IdP
  gui_order          = ""
  hide_on_login_page = true

  # This IdP is only for token exchange
  sync_mode          = "IMPORT"

  # Client ID is not used in this flow
  client_id          = "github"
}
```

---

## 5. Client Configuration

We will define several clients for different applications and services.

**File:** `clients_oidc.tf`

### 5.1. Frontend Application

**SRS Requirements:**
- `REQ-AUTH-006`: Use OIDC Authorization Code Flow with PKCE.
- `REQ-SVC-005`: Use `keycloak-js` adapter.

Use the `keycloak_openid_client` resource. This is a public client with the standard flow enabled and PKCE configured.

```terraform
# Example Frontend Client
resource "keycloak_openid_client" "frontend_app" {
  realm_id    = keycloak_realm.catalyst_forge.id
  client_id   = "frontend-app"
  name        = "Frontend Application"
  enabled     = true

  access_type = "PUBLIC"

  standard_flow_enabled = true
  implicit_flow_enabled = false
  direct_access_grants_enabled = false

  pkce_code_challenge_method = "S256"

  valid_redirect_uris = [
    "https://forge.projectcatalyst.io/*",
    "http://localhost:3000/*"
  ]
  web_origins = [
    "https://forge.projectcatalyst.io",
    "http://localhost:3000"
  ]
}
```

### 5.2. Backend API

**SRS Requirements:**
- `REQ-SVC-001`: Perform local JWT validation.

This is a "bearer-only" client. It does not participate in login flows but is used to define the audience (`aud`) claim in tokens and apply client-specific roles or scopes.

```terraform
# Example Backend API Client (Bearer-Only)
resource "keycloak_openid_client" "backend_api" {
  realm_id    = keycloak_realm.catalyst_forge.id
  client_id   = "backend-api"
  name        = "Backend API"
  enabled     = true

  access_type = "BEARER-ONLY"

  # This client does not need login flows
  standard_flow_enabled = false
}
```

### 5.3. CLI Tools

**SRS Requirements:**
- `REQ-AUTH-011`: Implement OAuth 2.0 Device Authorization Grant.

This is a public client with the device authorization flow enabled.

```terraform
# Example CLI Client
resource "keycloak_openid_client" "cli_tool" {
  realm_id    = keycloak_realm.catalyst_forge.id
  client_id   = "cli-tool"
  name        = "Catalyst CLI"
  enabled     = true

  access_type = "PUBLIC"

  # Disable browser-based flows
  standard_flow_enabled = false

  # Enable the device authorization grant
  oauth2_device_authorization_grant_enabled = true
}
```

### 5.4. M2M Authentication (Build Systems)

**SRS Requirements:**
- `REQ-AUTH-016`: Implement OAuth2 Client Credentials Grant.

This is a confidential client with service accounts enabled, which allows it to authenticate with its own credentials.

```terraform
# Example M2M Client (Client Credentials)
resource "keycloak_openid_client" "build_system" {
  realm_id    = keycloak_realm.catalyst_forge.id
  client_id   = "build-system"
  name        = "Earthly Build System"
  enabled     = true

  access_type = "CONFIDENTIAL"

  # Enable the service account and client credentials grant
  service_accounts_enabled = true

  # Disable other flows
  standard_flow_enabled = false
}
```

### 5.5. GitHub Actions Token Exchange Permission

To complete the GitHub Actions integration, create a permission that allows the GitHub IdP to exchange its token for a token scoped to the `backend-api`.

```terraform
# Example Token Exchange Permission
resource "keycloak_identity_provider_token_exchange_scope_permission" "github_token_exchange" {
  realm_id     = keycloak_realm.catalyst_forge.id
  provider_alias = keycloak_oidc_identity_provider.github.alias

  # The client that is allowed to request the token exchange
  clients    = [
    keycloak_openid_client.backend_api.id
  ]

  # Policy to enforce checks on the incoming GitHub token (e.g., repository, organization)
  # This requires creating a client policy first.
  # policy = "my-github-policy"
}
```
**Note:** You must define client policies in the Keycloak UI under `Realms > Client Policies` to restrict which GitHub repositories or organizations can perform a token exchange. This is not fully manageable via the provider today.

---

## 6. AWS Integration (SAML)

**File:** `clients_saml.tf`

**SRS Requirements:**
- `REQ-INT-001`: Support SAML-based IAM role assumption.
- `REQ-INT-002`: Map Keycloak groups/roles to AWS IAM roles.

This requires a SAML client and protocol mappers to create the claims AWS expects.

```terraform
# Example AWS SAML Client
resource "keycloak_saml_client" "aws" {
  realm_id  = keycloak_realm.catalyst_forge.id
  client_id = "urn:amazon:webservices"
  name      = "AWS"
  enabled   = true

  # AWS requires the NameID format to be persistent
  name_id_format = "persistent"

  # The client signature is required by AWS
  sign_documents           = true
  signature_algorithm      = "RSA_SHA256"
  signature_key_name       = "KEY_ID"

  # The assertion consumer service URL for AWS
  valid_redirect_uris = [
    "https://signin.aws.amazon.com/saml"
  ]
}

# Mapper for Session Duration
resource "keycloak_saml_user_attribute_mapper" "aws_session_duration" {
  realm_id       = keycloak_realm.catalyst_forge.id
  client_id      = keycloak_saml_client.aws.id
  name           = "SessionDuration"
  user_attribute = "session.duration" # This attribute must be set on the user or group

  saml_attribute_name       = "https://aws.amazon.com/SAML/Attributes/SessionDuration"
  saml_attribute_name_format = "URI Reference"
}

# Mapper for Roles
resource "keycloak_saml_user_attribute_mapper" "aws_roles" {
  realm_id       = keycloak_realm.catalyst_forge.id
  client_id      = keycloak_saml_client.aws.id
  name           = "Role"
  user_attribute = "aws.roles" # This attribute must be set on the user or group

  saml_attribute_name       = "https://aws.amazon.com/SAML/Attributes/Role"
  saml_attribute_name_format = "URI Reference"
}
```
**Note:** For the role mapping to work, you must create groups in Keycloak that correspond to your AWS IAM roles. Then, add an attribute to each group where the key is `aws.roles` and the value is a comma-separated list of the IAM Role ARN and the SAML Provider ARN.

---

## 7. Roles and Groups

**File:** `roles_groups.tf`

**SRS Requirements:**
- `REQ-AUTHZ-002`: Embed roles and group memberships as JWT claims.

Define roles and groups using `keycloak_role` and `keycloak_group`. You can then create mappers to add this information to tokens.

```terraform
# Example Role
resource "keycloak_role" "developer" {
  realm_id    = keycloak_realm.catalyst_forge.id
  name        = "developer"
  description = "Default role for all authenticated users"
}

# Example Group
resource "keycloak_group" "platform_team" {
  realm_id = keycloak_realm.catalyst_forge.id
  name     = "platform-team"
}

# Example Mapper to add group membership to tokens
resource "keycloak_openid_group_membership_protocol_mapper" "group_mapper" {
  realm_id  = keycloak_realm.catalyst_forge.id
  client_id = keycloak_openid_client.backend_api.id # Add to a specific client's token
  name      = "groups"

  # The name of the claim in the token
  claim_name = "groups"

  add_to_id_token    = true
  add_to_access_token = true
}
```

---

## 8. Third-Party Application Integration

For services like **Argo CD, Grafana Cloud, Tailscale, and Swarmia**, the process is similar:

1.  Create a new `keycloak_openid_client` (for OIDC) or `keycloak_saml_client` (for SAML) in Terraform.
2.  Consult the documentation for the specific service to find the required redirect URIs and other settings.
3.  Configure the service to use Keycloak as the identity provider, providing it with your realm's OIDC discovery URL (`.../realms/forge/.well-known/openid-configuration`) or SAML metadata URL.
4.  Use protocol mappers to map Keycloak roles or groups to the permissions expected by the service, as you did for AWS.
