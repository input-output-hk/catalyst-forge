# Keycloak Integration Requirements Document

**Document Version:** 1.0
**Status:** Draft
**Last Updated:** 2025-01-02
**Author(s):** Joshua Gilman
**Reviewers:** Steven Johnson, Sasha Prokhorenko

## Document Control

| Version | Date | Author | Description |
|---------|------|--------|-------------|
| 1.0 | 2025-01-02 | Joshua Gilman | Initial restructured version |

## Table of Contents

1. **Introduction**
   - 1.1 Purpose
   - 1.2 Scope
   - 1.3 Definitions, Acronyms, and Abbreviations
   - 1.4 References
   - 1.5 Document Overview

2. **Overall Description**
   - 2.1 Product Perspective
   - 2.2 Business Context and Justification
   - 2.3 User Classes and Characteristics
   - 2.4 Operating Environment
   - 2.5 Assumptions and Dependencies
   - 2.6 Constraints

3. **Functional Requirements**
   - 3.1 Authentication Requirements
   - 3.2 Authorization Requirements
   - 3.3 Integration Requirements
   - 3.4 Service Protection Requirements
   - 3.5 Session Management Requirements
   - 3.6 Delegation and Impersonation Requirements

4. **Non-Functional Requirements**
   - 4.1 Performance Requirements
   - 4.2 Security Requirements
   - 4.3 Availability Requirements
   - 4.4 Scalability Requirements
   - 4.5 Maintainability Requirements
   - 4.6 Compatibility Requirements
   - 4.7 Usability Requirements

5. **External Interface Requirements**
   - 5.1 User Interfaces
   - 5.2 Hardware Interfaces
   - 5.3 Software Interfaces
   - 5.4 Communication Interfaces

6. **System Architecture**
   - 6.1 Architectural Overview
   - 6.2 Component Descriptions
   - 6.3 Data Flow Diagrams

7. **Implementation Planning**
   - 7.1 Migration Strategy
   - 7.2 Risk Assessment and Mitigation
   - 7.3 Training Requirements
   - 7.4 Acceptance Criteria

8. **Operations and Maintenance**
   - 8.1 Monitoring and Observability
   - 8.2 Backup and Recovery
   - 8.3 Update and Patch Management

9. **Appendices**
   - Appendix A: Configuration Examples
   - Appendix B: API Specifications
   - Appendix C: Decision Log
   - Appendix D: Open Issues
   - Appendix E: Implementation Details

---

## 1. Introduction

### 1.1 Purpose

This document outlines the requirements for implementing Keycloak as the centralized authentication and authorization service for our development platform. This initiative will standardize authentication across all internal and third-party tools.

### 1.2 Scope

This document covers:
- Integration of Keycloak with Google Workspace as the sole identity provider
- Authentication and authorization for platform services (Backend API, Frontend Application)
- Integration with third-party services (AWS, Argo CD, Grafana Cloud, Tailscale, Swarmia)
- Infrastructure and operational requirements for Keycloak deployment

**Out of Scope:**
- Support for non-Google identity providers (Azure AD, Okta, etc.)
- User management interfaces beyond Keycloak's built-in admin console
- Custom user registration or self-service account creation
- Migration of existing user passwords or credentials
- Integration with legacy authentication systems
- Mobile application authentication
- External customer/partner authentication (customer-facing systems retain their own authentication)
- Developer authentication to GitHub (developers continue using GitHub accounts)
- Changes to existing customer-facing authentication systems

### 1.3 Definitions, Acronyms, and Abbreviations

- **API**: Application Programming Interface
- **Argo CD**: GitOps continuous delivery tool for Kubernetes
- **AWS**: Amazon Web Services
- **CN**: Common Name (certificate field)
- **EKS**: Elastic Kubernetes Service (AWS managed Kubernetes)
- **Earthly**: Build automation tool for reproducible builds
- **Envoy Gateway**: API gateway built on Envoy proxy
- **HPA**: Horizontal Pod Autoscaler
- **IAM**: Identity and Access Management (AWS service)
- **IdP**: Identity Provider
- **Istio**: Service mesh for microservices
- **JIT**: Just-In-Time provisioning
- **JWT**: JSON Web Token
- **JWKS**: JSON Web Key Set
- **M2M**: Machine-to-Machine
- **mTLS**: Mutual TLS (Transport Layer Security)
- **OIDC**: OpenID Connect
- **PCA**: Private Certificate Authority (AWS service)
- **PITR**: Point-In-Time Recovery
- **PKCE**: Proof Key for Code Exchange
- **PyInfra**: Python-based infrastructure automation tool
- **RBAC**: Role-Based Access Control
- **RDS**: Relational Database Service (AWS)
- **RPO**: Recovery Point Objective
- **RTO**: Recovery Time Objective
- **SAN**: Subject Alternative Name (certificate field)
- **SAML**: Security Assertion Markup Language
- **SP**: Service Provider
- **SRE**: Site Reliability Engineering/Engineer
- **SSO**: Single Sign-On
- **TLS**: Transport Layer Security
- **UMA**: User-Managed Access
- **VPC**: Virtual Private Cloud

### 1.4 References

- OAuth 2.0 Framework - RFC 6749: https://datatracker.ietf.org/doc/html/rfc6749
- OpenID Connect Core 1.0: https://openid.net/specs/openid-connect-core-1_0.html
- JSON Web Token (JWT) - RFC 7519: https://datatracker.ietf.org/doc/html/rfc7519
- OAuth 2.0 Device Authorization Grant - RFC 8628: https://datatracker.ietf.org/doc/html/rfc8628
- Proof Key for Code Exchange (PKCE) - RFC 7636: https://datatracker.ietf.org/doc/html/rfc7636
- SAML 2.0 Technical Overview: http://docs.oasis-open.org/security/saml/Post2.0/
- Keycloak Documentation: https://www.keycloak.org/documentation
- AWS IAM SAML Identity Providers: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_saml.html
- Argo CD Documentation: https://argo-cd.readthedocs.io/
- Envoy Gateway Documentation: https://gateway.envoyproxy.io/
- Istio Documentation: https://istio.io/latest/docs/
- RBAC Design Document (Internal) - Detailed role definitions and permission model

### 1.5 Document Overview

This document is organized into the following major sections:
- Section 2: Overall system description and context
- Section 3: Functional requirements for authentication and authorization
- Section 4: Non-functional requirements including performance and security
- Section 5: External interface specifications
- Section 6: System architecture details
- Section 7: Implementation planning and migration strategy
- Section 8: Operations and maintenance requirements

---

## 2. Overall Description

### 2.1 Product Perspective

Keycloak will serve as the centralized authentication and authorization service within the following infrastructure:
- **Infrastructure**: AWS EKS with Envoy Gateway and Istio service mesh
- **Identity Provider**: Google Workspace (exclusive)
- **Protected Services**: Backend API (Go/Gin/GORM), Frontend Application (Vue.js)
- **Integration Targets**: AWS, Argo CD, [Additional services TBD]

### 2.2 Business Context and Justification

**Current State Challenges:**
- Fragmented authentication across different tools and services
- Manual user provisioning and deprovisioning processes
- No centralized access control mechanism
- Security risks from inconsistent authentication methods
- Difficult to audit access across systems
- Limited developer self-service capabilities due to security concerns
- SRE team acting as intermediary for system access requests

**Strategic Initiative:**
This Keycloak integration is part of a broader introduction of a unified in-house Internal Development Platform (IDP) called Catalyst Forge. The centralized authentication service will enable:

**Expected Benefits:**
- **Risk Reduction**: Centralized authentication reduces security vulnerabilities and provides consistent security policies
- **Time Savings**: Developers gain self-service access to systems, reducing delays and dependencies
- **Cost Savings**: Reduced SRE overhead by eliminating manual access management and middle-man activities
- **Improved Security**: Consistent authentication standards and centralized audit logging
- **Enhanced Productivity**: Single sign-on improves user experience and reduces context switching
- **Automated Lifecycle**: User provisioning/deprovisioning automatically tied to Google Workspace status

### 2.3 User Classes and Characteristics

#### Human Users

1. **Software Engineers** (Primary users)
   - **Count**: 20-30 users
   - **Needs**: Access to development tools, APIs, cloud resources, and deployment systems
   - **Characteristics**: Technical users comfortable with both CLI and web interfaces
   - **Access Patterns**: Daily usage, 8-hour sessions, heavy API usage, frequent authentication

2. **QA Engineers**
   - **Count**: Included in 20-30 total users
   - **Needs**: Access to testing environments, deployment systems, and monitoring tools
   - **Characteristics**: Technical users with similar profile to Software Engineers
   - **Access Patterns**: Daily usage with slightly different tool access patterns than Software Engineers

3. **DevX Engineers**
   - **Count**: Included in total user count
   - **Needs**: Platform development and maintenance access
   - **Characteristics**: Technical users requiring elevated privileges (but less than SRE)
   - **Access Patterns**: Daily platform development work, infrastructure access

4. **SRE Team Members**
   - **Count**: 2 users
   - **Needs**: Full administrative access to infrastructure, monitoring, and platform management
   - **Characteristics**: Highly technical, acting as both platform administrators and engineers
   - **Access Patterns**: 24/7 on-call rotation, emergency access requirements, highest privilege level

#### System Users

5. **Build Systems** (Machine-to-Machine)
   - **Count**: <5 Hetzner bare metal machines
   - **Needs**: Automated API access for build processes
   - **Characteristics**: Earthly satellite systems requiring continuous authentication
   - **Access Patterns**: Continuous operation, high-frequency API calls

6. **External CI/CD Services**
   - **Count**: Variable (based on GitHub Actions workflows)
   - **Needs**: Temporary access for deployment and automation tasks
   - **Characteristics**: GitHub Actions workflows requiring federated authentication
   - **Access Patterns**: Event-driven (triggered by commits), short-lived access tokens

7. **Internal Services** (Machine-to-Machine)
   - **Count**: Variable (microservices within EKS)
   - **Needs**: Service-to-service authentication within cluster
   - **Characteristics**: Kubernetes services requiring authenticated API access
   - **Access Patterns**: Continuous inter-service communication within EKS cluster

**Growth Projection**: User base expected to grow by no more than 10% annually

### 2.4 Operating Environment

- **Deployment Platform**: AWS EKS
- **Network Layer**: Envoy Gateway and Istio service mesh
- **Database**: AWS RDS PostgreSQL
- **Monitoring**: Grafana Cloud
- **Current Authentication State**:
  - **Argo CD**: Currently using Google OIDC (to be migrated to Keycloak)
  - **Backend API**: In development, no current authentication (Keycloak will be first implementation)
  - **Frontend Application**: In development, no current authentication (Keycloak will be first implementation)
  - **Build Systems**: Currently using mTLS with AWS PCA certificates (5-day rotation)
  - **User Provisioning**: Just-In-Time (JIT) provisioning via Google Workspace authentication

### 2.5 Assumptions and Dependencies

**Assumptions:**
- Google Workspace will remain the organization's primary identity provider
- All users have active Google Workspace accounts
- Kubernetes cluster with Envoy Gateway is operational
- Network connectivity between all components is reliable

**Dependencies:**
- Google Workspace availability for authentication
- AWS services (EKS, RDS) availability
- Keycloak Operator for deployment management
- Token Exchange feature (preview) for GitHub Actions integration

### 2.6 Constraints

- No support for username/password authentication
- No support for identity providers other than Google Workspace
- Token Exchange feature is in preview status (risk for GitHub Actions integration)
- Build machines limited to <5 bare metal Hetzner machines

---

## 3. Functional Requirements

### 3.1 Authentication Requirements

#### 3.1.1 Google Workspace Integration

**REQ-AUTH-001:** The system SHALL use Google Workspace as the sole identity provider
**REQ-AUTH-002:** The system SHALL implement OpenID Connect (OIDC) for federation with Google Workspace
**REQ-AUTH-003:** The system SHALL support Just-In-Time (JIT) provisioning of user accounts upon first successful authentication
**REQ-AUTH-004:** The system SHALL NOT support username/password authentication
**REQ-AUTH-005:** The system SHALL NOT support any identity providers other than Google Workspace

#### 3.1.2 Browser-Based User Authentication

**REQ-AUTH-006:** The system SHALL implement OIDC Authorization Code Flow with PKCE for web applications
**REQ-AUTH-007:** The system SHALL maintain SSO sessions for a maximum of 8 hours
**REQ-AUTH-008:** The system SHALL issue access tokens with a 5-minute lifespan
**REQ-AUTH-009:** The system SHALL support silent token refresh using refresh tokens
**REQ-AUTH-010:** The system SHALL provide seamless SSO across all integrated web applications

#### 3.1.3 CLI Authentication

**REQ-AUTH-011:** The system SHALL implement OAuth 2.0 Device Authorization Grant for CLI tools
**REQ-AUTH-012:** The system SHALL store tokens in the operating system's native secure storage when available
**REQ-AUTH-013:** The system SHALL fall back to file-based storage with 0600 permissions when native storage is unavailable
**REQ-AUTH-014:** The system SHALL issue refresh tokens with 8-hour lifespan for CLI authentication
**REQ-AUTH-015:** The system SHALL automatically refresh access tokens in CLI tools without user interaction

#### 3.1.4 Machine-to-Machine Authentication

**REQ-AUTH-016:** The system SHALL implement OAuth2 Client Credentials Grant for M2M authentication
**REQ-AUTH-017:** The system SHALL support credential management via AWS Secrets Manager OR mTLS certificates
**REQ-AUTH-018:** The system SHALL enforce credential rotation (quarterly for secrets OR 5-day for certificates)
**REQ-AUTH-019:** The system SHALL restrict M2M access through scope-based permissions

#### 3.1.5 External Service Authentication

**REQ-AUTH-020:** The system SHALL support OIDC federation for GitHub Actions workflows
**REQ-AUTH-021:** The system SHALL validate GitHub OIDC tokens and exchange them for platform tokens
**REQ-AUTH-022:** The system SHALL enforce repository and organization-based access controls
**REQ-AUTH-023:** The system SHALL map GitHub token claims to platform token claims

### 3.2 Authorization Requirements

#### 3.2.1 Role-Based Access Control

**REQ-AUTHZ-001:** The system SHALL implement dual-layer RBAC (coarse-grained at gateway, fine-grained at service)
**REQ-AUTHZ-002:** The system SHALL embed roles and group memberships as JWT claims
**REQ-AUTHZ-003:** The system SHALL support role and permission assignment by platform administrators
**REQ-AUTHZ-004:** The system SHALL support project/team-based access control as defined in the RBAC Design Document
**REQ-AUTHZ-005:** The system SHALL implement role hierarchies and permission inheritance per the RBAC Design Document

*Note: Detailed RBAC model, role definitions, and permission boundaries are documented in the separate RBAC Design Document.*

#### 3.2.2 Gateway-Level Authorization

**REQ-AUTHZ-006:** Envoy Gateway SHALL enforce coarse-grained authorization based on roles
**REQ-AUTHZ-007:** Envoy Gateway SHALL extract and validate JWT claims
**REQ-AUTHZ-008:** Envoy Gateway SHALL pass validated claims to downstream services via headers

#### 3.2.3 Service-Level Authorization

**REQ-AUTHZ-009:** Services SHALL perform fine-grained authorization as specified in the RBAC Design Document
**REQ-AUTHZ-010:** Services SHALL filter responses based on user permissions per the RBAC Design Document
**REQ-AUTHZ-011:** Services SHALL log all authorization decisions for audit purposes
**REQ-AUTHZ-012:** Services SHALL enforce resource-level permissions as defined in the RBAC Design Document

*Note: Specific authorization rules and data filtering requirements are documented in the separate RBAC Design Document.*

### 3.3 Integration Requirements

#### 3.3.1 AWS Integration

**REQ-INT-001:** The system SHALL support SAML-based IAM role assumption
**REQ-INT-002:** The system SHALL map Keycloak groups/roles to AWS IAM roles
**REQ-INT-003:** The system SHALL issue temporary AWS credentials with 4-hour maximum duration
**REQ-INT-004:** The system SHALL support both console and CLI/SDK access patterns

#### 3.3.2 Argo CD Integration

**REQ-INT-005:** The system SHALL replace existing Google OIDC with Keycloak OIDC
**REQ-INT-006:** The system SHALL maintain transparent user experience during migration
**REQ-INT-007:** The system SHALL align Argo CD session timeout with Keycloak SSO session (8 hours)
**REQ-INT-008:** The system SHALL support Argo CD CLI authentication via OIDC

#### 3.3.3 Grafana Cloud Integration

**REQ-INT-009:** The system SHALL provide SSO integration with Grafana Cloud via OIDC or SAML
**REQ-INT-010:** The system SHALL map Keycloak roles to Grafana Cloud permissions
**REQ-INT-011:** The system SHALL support automatic user provisioning in Grafana Cloud

#### 3.3.4 Tailscale Integration

**REQ-INT-012:** The system SHALL provide SSO integration with Tailscale via OIDC or SAML
**REQ-INT-013:** The system SHALL enforce network access policies based on Keycloak authentication
**REQ-INT-014:** The system SHALL support automatic user provisioning in Tailscale

#### 3.3.5 Swarmia Integration

**REQ-INT-015:** The system SHALL provide SSO integration with Swarmia via OIDC or SAML
**REQ-INT-016:** The system SHALL map Keycloak roles to Swarmia permissions
**REQ-INT-017:** The system SHALL support automatic user provisioning in Swarmia

### 3.4 Service Protection Requirements

#### 3.4.1 Backend API Protection

**REQ-SVC-001:** The Backend API SHALL perform local JWT validation
**REQ-SVC-002:** The Backend API SHALL cache Keycloak's public keys (JWKS)
**REQ-SVC-003:** The Backend API SHALL use github.com/golang-jwt/jwt library for token validation
**REQ-SVC-004:** The Backend API SHALL implement custom Gin middleware for route protection

#### 3.4.2 Frontend Application Protection

**REQ-SVC-005:** The Frontend SHALL use keycloak-js adapter for authentication
**REQ-SVC-006:** The Frontend SHALL store tokens exclusively in-memory
**REQ-SVC-007:** The Frontend SHALL implement Vue Router navigation guards for route protection
**REQ-SVC-008:** The Frontend SHALL automatically attach bearer tokens to API requests

### 3.5 Session Management Requirements

**REQ-SESS-001:** The system SHALL allow users to view all their active sessions
**REQ-SESS-002:** The system SHALL allow users to terminate individual sessions remotely
**REQ-SESS-003:** The system SHALL enforce concurrent session limits per user as configured by administrators
**REQ-SESS-004:** The system SHALL display session metadata including login time, IP address, and client type
**REQ-SESS-005:** The system SHALL automatically terminate sessions that exceed the idle timeout

### 3.6 Delegation and Impersonation Requirements

**REQ-DELEG-001:** The system SHOULD support user impersonation by authorized administrators (if supported by Keycloak)
**REQ-DELEG-002:** All impersonation actions SHALL be logged with the original administrator identity
**REQ-DELEG-003:** Impersonation sessions SHALL be clearly marked in audit logs
**REQ-DELEG-004:** Impersonation privileges SHALL be restricted to platform-admin role only

*Note: Implementation dependent on Keycloak's native impersonation capabilities.*

---

## 4. Non-Functional Requirements

### 4.1 Performance Requirements

**REQ-PERF-001:** Token validation and authorization SHALL NOT add more than 5ms to request latency
**REQ-PERF-002:** The system SHALL handle 20 concurrent users during peak usage
**REQ-PERF-003:** The system SHALL support up to 1 request per second sustained load
**REQ-PERF-004:** JWT token size SHALL NOT exceed 8KB to prevent HTTP header size issues and performance degradation
**REQ-PERF-005:** The system SHALL maintain sub-second response times for authentication operations under normal load

### 4.2 Security Requirements

#### 4.2.1 Token Security

**REQ-SEC-001:** Access tokens SHALL have a maximum lifespan of 5 minutes
**REQ-SEC-002:** Refresh tokens SHALL have a maximum lifespan of 8 hours
**REQ-SEC-003:** SSO sessions SHALL timeout after 1 hour of inactivity
**REQ-SEC-004:** The system SHALL implement refresh token rotation
**REQ-SEC-005:** The system SHALL support token revocation [Future requirement]

#### 4.2.2 Access Management

**REQ-SEC-006:** User access SHALL be revoked within 1 hour of offboarding notification
**REQ-SEC-007:** The system SHALL provide break-glass emergency access procedures
**REQ-SEC-008:** All break-glass access SHALL be logged and reviewed within 24 hours
**REQ-SEC-009:** Initial role assignment SHALL require manual approval by authorized personnel

#### 4.2.3 Audit and Compliance

**REQ-SEC-010:** The system SHALL log all authentication attempts (successful and failed)
**REQ-SEC-011:** The system SHALL log all administrative actions
**REQ-SEC-012:** The system SHALL retain audit logs for 30 days
**REQ-SEC-013:** The system SHALL be designed for future SOC 2 compliance
**REQ-SEC-014:** All authentication data SHALL be stored within AWS US regions

### 4.3 Availability Requirements

**REQ-AVAIL-001:** The system SHALL be deployed in high-availability active-active configuration
**REQ-AVAIL-002:** The system SHALL maintain minimum three replicas
**REQ-AVAIL-003:** Recovery Time Objective (RTO) SHALL be 4 hours
**REQ-AVAIL-004:** Recovery Point Objective (RPO) SHALL be 12 hours
**REQ-AVAIL-005:** The system SHALL support zero-downtime updates

### 4.4 Scalability Requirements

**REQ-SCALE-001:** The system SHALL auto-scale based on CPU and memory utilization via HPA
**REQ-SCALE-002:** The system SHALL scale up when CPU utilization exceeds 80% or memory exceeds 80%
**REQ-SCALE-003:** The system SHALL scale down when CPU utilization drops below 20% and memory below 20%
**REQ-SCALE-004:** The system SHALL maintain a minimum of 3 replicas for high availability
**REQ-SCALE-005:** The system SHALL NOT enforce a maximum replica limit (scale as needed)
**REQ-SCALE-006:** Initial resource allocation SHALL be 1 CPU and 2Gi memory per replica
**REQ-SCALE-007:** The system SHALL integrate with Karpenter for node-level scaling
**REQ-SCALE-008:** The system SHALL accommodate 10% annual growth in transaction volume

### 4.5 Maintainability Requirements

**REQ-MAINT-001:** The system SHALL use Keycloak Operator for lifecycle management
**REQ-MAINT-002:** The deployed version SHALL NOT be more than two minor versions behind latest stable
**REQ-MAINT-003:** The system SHALL support rolling updates without downtime
**REQ-MAINT-004:** Configuration backups SHALL be performed daily and retained for 30 days
**REQ-MAINT-005:** All Keycloak configuration SHALL be managed as Infrastructure as Code (IaC)
**REQ-MAINT-006:** Configuration changes SHALL be deployed via pull requests with at least one SRE reviewer
**REQ-MAINT-007:** The system SHALL maintain configuration version history in Git
**REQ-MAINT-008:** Disaster recovery procedures SHALL be tested quarterly

### 4.6 Compatibility Requirements

**REQ-COMPAT-001:** The system SHALL support modern web browsers (specific versions not enforced)
**REQ-COMPAT-002:** CLI tools SHALL support Linux and macOS operating systems
**REQ-COMPAT-003:** CLI tools SHOULD provide best-effort support for Windows
**REQ-COMPAT-004:** The system SHALL maintain backward compatibility for API versions for at least 6 months

### 4.7 Usability Requirements

**REQ-USAB-001:** The login page SHALL be branded with internal IDP branding (Catalyst Forge)
**REQ-USAB-002:** The system SHALL provide clear error messages for authentication failures
**REQ-USAB-003:** The system SHALL provide user-friendly session timeout warnings
**REQ-USAB-004:** All user-facing interfaces SHALL follow established internal branding guidelines

---

## 5. External Interface Requirements

### 5.1 User Interfaces

**REQ-UI-001:** The Keycloak admin console SHALL be accessible to SRE team members and engineering managers
**REQ-UI-002:** The admin console SHALL be exposed both internally and via VPN/Tailscale for remote access
**REQ-UI-003:** The system SHALL provide a user self-service portal for profile management
**REQ-UI-004:** Users SHALL be able to view and update their profile information
**REQ-UI-005:** Users SHALL be able to manage API tokens and credentials through the self-service portal
**REQ-UI-006:** Users SHALL be able to configure two-factor authentication (TOTP or WebAuthn) devices
**REQ-UI-007:** The SSO login page SHALL be branded with Catalyst Forge branding
**REQ-UI-008:** All user interfaces SHALL be in English
**REQ-UI-009:** The SSO login page SHALL be mobile responsive

### 5.2 Hardware Interfaces

Build Machine Requirements:
- Bare metal Hetzner machines (<5 total)
- Network connectivity to Keycloak endpoints
- Storage for credential/certificate files

### 5.3 Software Interfaces

#### 5.3.1 Google Workspace Interface
- **Protocol**: OpenID Connect
- **Project**: Existing Google Cloud project `catalyst-forge` with OIDC configured
- **Endpoints**: Standard Google OAuth 2.0 endpoints (accounts.google.com)
- **Required Claims**:
  - email
  - name
- **Group Mapping**: Not required (permissions managed within Keycloak)

#### 5.3.2 AWS IAM Interface
- **Protocol**: SAML 2.0
- **Trust relationship configuration required**
- **Session duration**: 4 hours maximum

#### 5.3.3 Backend API Interface
- **Token validation**: Local JWT validation
- **JWKS endpoint**: Keycloak must expose public keys for token validation
- **Header propagation**: Envoy Gateway SHALL pass the Authorization header containing the JWT token
- **CORS**: Backend API SHALL support CORS for cross-origin requests
- **Service introspection**: Services may perform additional token introspection as needed

### 5.4 Communication Interfaces

**REQ-COMM-001:** All external communications SHALL use HTTPS with TLS 1.2 or higher
**REQ-COMM-002:** Internal service-to-service communication SHALL use mTLS within the service mesh
**REQ-COMM-003:** The system SHALL implement rate limiting on sensitive endpoints (login, token endpoints)
**REQ-COMM-004:** Rate limits SHALL be configurable per endpoint
**REQ-COMM-005:** The system SHALL log rate limit violations for security monitoring
**REQ-COMM-006:** WebSocket support is NOT required for initial implementation

---

## 6. System Architecture

### 6.1 Architectural Overview

#### 6.1.1 Core Components
- **Identity Provider**: Google Workspace (exclusive)
- **Authentication Service**: Keycloak
- **Infrastructure**: AWS EKS with Envoy Gateway and Istio service mesh
- **Protected Services**: Backend API (Go/Gin/GORM), Frontend Application (Vue.js)

### 6.2 Component Descriptions

#### 6.2.1 Keycloak Deployment
- **Deployment Model**: High-availability, active-active configuration
- **Minimum Replicas**: 3
- **Database**: AWS RDS PostgreSQL (db.t3.medium initial)
- **Namespace**: Dedicated `keycloak` namespace
- **Environment Strategy**: Single instance with separate realms for staging/production

#### 6.2.2 Network Integration
- **Envoy Gateway**: OAuth2 filter for edge authentication, coarse-grained RBAC
- **Service Mesh**: Istio with mTLS between services
- **Load Balancing**: Round-robin load balancing with health-check based routing via Kubernetes Service

### 6.3 Data Flow Diagrams

The following sections describe the authentication flows in sequence format for later conversion to visual diagrams.

#### 6.3.1 User Authentication Flow (Browser)
```
1. User → Frontend Application: Access protected resource
2. Frontend Application → User: Redirect to Keycloak login
3. User → Keycloak: Redirect to Google OAuth
4. User → Google: Authenticate with credentials
5. Google → Keycloak: Return with OIDC token
6. Keycloak → Keycloak: JIT provision user (if first login)
7. Keycloak → User: Issue session cookie and redirect with auth code
8. User → Frontend Application: Return with auth code
9. Frontend Application → Keycloak: Exchange code for tokens
10. Keycloak → Frontend Application: Return access token and refresh token
11. Frontend Application → Backend API: API call with bearer token
12. Backend API → Backend API: Validate JWT locally using cached JWKS
13. Backend API → Frontend Application: Return protected resource
```

#### 6.3.2 CLI Authentication Flow
```
1. CLI Tool → Keycloak: Request device code
2. Keycloak → CLI Tool: Return device code and verification URL
3. CLI Tool → User: Display verification URL
4. User → Browser: Open verification URL
5. User → Keycloak: Enter device code
6. User → Google: Authenticate (if not already)
7. Google → Keycloak: Return authentication
8. Keycloak → Browser: Confirmation of authorization
9. CLI Tool → Keycloak: Poll for token (using device code)
10. Keycloak → CLI Tool: Return access and refresh tokens
11. CLI Tool → Local Storage: Store tokens securely
12. CLI Tool → Backend API: API calls with bearer token
```

#### 6.3.3 M2M Authentication Flow (Build Machines)
```
1. Build Machine → Keycloak: Client credentials grant (client_id + secret/certificate)
2. Keycloak → Keycloak: Validate credentials
3. Keycloak → Build Machine: Return access token
4. Build Machine → Backend API: API call with bearer token
5. Backend API → Backend API: Validate JWT
6. Backend API → Build Machine: Return response
```

#### 6.3.4 GitHub Actions Authentication Flow
```
1. GitHub Actions → GitHub: Request OIDC token for Keycloak audience
2. GitHub → GitHub Actions: Return GitHub OIDC token
3. GitHub Actions → Keycloak: Token exchange request with GitHub token
4. Keycloak → Keycloak: Validate GitHub token signature
5. Keycloak → Keycloak: Check authorization policies (repo, org, branch)
6. Keycloak → GitHub Actions: Return platform access token
7. GitHub Actions → Backend API: API call with platform token
8. Backend API → Backend API: Validate JWT
9. Backend API → GitHub Actions: Return response
```

#### 6.3.5 Token Refresh Flow
```
1. Client → Keycloak: POST refresh token to token endpoint
2. Keycloak → Keycloak: Validate refresh token
3. Keycloak → Keycloak: Check session validity
4. Keycloak → Keycloak: Rotate refresh token (invalidate old)
5. Keycloak → Client: Return new access token and refresh token
```

---

## 7. Implementation Planning

### 7.1 Migration Strategy

#### 7.1.1 Phased Approach

**Phase 1: Keycloak Deployment and Google Workspace Integration** (~1 week)
- Deploy Keycloak in AWS EKS with HA configuration
- Configure Google Workspace as OIDC provider
- Set up JIT provisioning
- Configure realms for staging and production
- Verify Google authentication flow
- **Success Criteria**: Users can authenticate via Google, JIT provisioning works

**Phase 2: Backend API and Frontend Integration** (~1 week, parallel execution)
- Implement JWT validation in Backend API
- Integrate keycloak-js in Frontend application
- Configure Envoy Gateway OAuth2 filter
- Test end-to-end authentication flow
- **Success Criteria**: Frontend can authenticate users and call protected Backend APIs

**Phase 3: Argo CD Migration** (~1 day)
- Configure Keycloak client for Argo CD
- Update Argo CD OIDC configuration
- Test CLI and web authentication
- Migrate without downtime using rolling update
- **Success Criteria**: Seamless transition from Google OIDC to Keycloak OIDC

**Phase 4: AWS Integration** (~3 days)
- Configure Keycloak as SAML IdP in AWS
- Create IAM roles with trust relationships
- Map Keycloak roles to AWS IAM roles
- Configure and test saml2aws CLI tool
- Enable for all users
- **Success Criteria**: All users can assume AWS roles via Keycloak

**Phase 5: External Services and M2M Authentication** (~1 week)
- **5a: GitHub Actions** (Priority 1)
  - Enable Token Exchange feature
  - Configure GitHub OIDC provider
  - Set up authorization policies
  - Test workflow authentication
- **5b: Build Machines** (Priority 2)
  - Deploy credentials via PyInfra
  - Configure service accounts
  - Test build system authentication
- **5c: SaaS Integrations** (Grafana Cloud, Tailscale, Swarmia)
  - Configure OIDC/SAML for each service
  - Test SSO and provisioning
- **Success Criteria**: All external services and M2M flows operational

#### 7.1.2 Rollback Plan

Each phase includes rollback procedures:
- **Phase 1**: Remove Keycloak deployment, no user impact
- **Phase 2**: Disable authentication in development services
- **Phase 3**: Revert Argo CD configuration to Google OIDC
- **Phase 4**: Remove SAML trust relationships, revert to previous access method
- **Phase 5**: Disable specific integrations without affecting core services

### 7.2 Risk Assessment and Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Token Exchange feature breaking changes | Medium | High | Thorough testing in staging, automated tests, maintain documentation of exact configuration |
| Google Workspace outage | Low | High | Implement break-glass procedures, maintain active sessions during outage, document manual override process |
| Keycloak service outage | Low | High | HA deployment with 3+ replicas, automated failover, monitoring and alerting |
| Performance degradation under load | Medium | Medium | Load testing before production, HPA configuration, performance monitoring |
| JWT token size exceeding limits | Low | Medium | Monitor token size, optimize claims, implement pagination for group memberships if needed |
| User adoption challenges | Medium | Low | Comprehensive training program, clear documentation, phased rollout |
| Integration compatibility issues | Medium | Medium | Thorough testing in staging environment, vendor documentation review |
| Misconfiguration leading to security issues | Low | High | Infrastructure as Code, peer review process, security scanning, principle of least privilege |
| Database corruption or loss | Low | High | Automated backups, PITR enabled, tested recovery procedures |
| Network connectivity issues | Low | Medium | Multi-AZ deployment, retry logic in clients, circuit breakers |

### 7.3 Training Requirements

#### 7.3.1 Developer Training
- **Delivery Methods**: Documentation, tutorial videos, live workshops
- **Content**:
  - Overview of new authentication flow
  - Using CLI tools with device authorization
  - Troubleshooting common authentication issues
  - API token management
  - Best practices for token handling

#### 7.3.2 SRE Training
- **Delivery Methods**: Hands-on workshops, runbooks, documentation
- **Content**:
  - Keycloak administration and configuration
  - Common operational scenarios
  - Monitoring and alerting interpretation
  - Disaster recovery procedures
  - Performance tuning
  - Security incident response
  - Backup and restore procedures

#### 7.3.3 Engineering Manager Training
- **Delivery Methods**: Workshop sessions, documentation
- **Content**:
  - Admin console navigation
  - User management basics
  - Viewing audit logs
  - Basic troubleshooting
  - Access request approval workflow

#### 7.3.4 Security Team Briefing
- **Content**:
  - Architecture overview
  - Security controls and audit capabilities
  - Incident response procedures
  - Compliance considerations

### 7.4 Acceptance Criteria

The following criteria must be met before production deployment:

#### Technical Criteria
- ✓ All functional requirements implemented and tested
- ✓ Performance benchmarks achieved:
  - Token validation <5ms overhead
  - System handles 1 RPS sustained load
  - 20 concurrent users supported
- ✓ High availability verified (3+ replicas operational)
- ✓ All integrations functional:
  - Google Workspace authentication working
  - Backend API protected and functional
  - Frontend authentication working
  - Argo CD migrated successfully
  - AWS role assumption operational
  - GitHub Actions authentication tested
  - Build machine authentication verified

#### Operational Criteria
- ✓ Monitoring and alerting configured and tested
- ✓ Backup and restore procedures tested
- ✓ Disaster recovery procedure documented and tested
- ✓ Runbooks created for common scenarios
- ✓ Infrastructure as Code reviewed and deployed

#### Security Criteria
- ✓ Security review completed
- ✓ Audit logging enabled and verified
- ✓ Break-glass procedures documented
- ✓ Token rotation verified
- ✓ Rate limiting configured on sensitive endpoints

#### Documentation and Training
- ✓ User documentation complete
- ✓ Administrator documentation complete
- ✓ Training delivered to all user groups
- ✓ Support procedures established

#### Sign-off
- **Final Approval**: SRE team
- **Stakeholder Review**: Engineering managers and team leads
- **Go-Live Decision**: Based on successful completion of all criteria

---

## 8. Operations and Maintenance

### 8.1 Monitoring and Observability

#### 8.1.1 Metrics
- **Collection**: Prometheus metrics endpoint enabled
- **Storage**: Grafana Cloud
- **Key Metrics**:
  - Authentication success/failure rates
  - Token issuance counts
  - JVM memory and CPU usage
  - Database connection pool status
  - API endpoint latency and error rates
  - Active users and sessions count

#### 8.1.2 Alerting
Critical alerts shall be configured to notify the on-call SRE engineer for:
- **Critical Alerts**:
  - Failed login rate exceeding 10 failures/minute from same IP or 50 failures/minute overall
  - API response time p95 latency exceeding 500ms for 5 minutes
  - Error rate exceeding 5% for 5 consecutive minutes
  - JVM heap usage exceeding 90%
  - Keycloak service unresponsive for 3 consecutive health checks
  - Database connection pool exhaustion
- **Warning Alerts**:
  - JVM heap usage exceeding 75%
  - Failed login attempts from new geographic locations
  - Certificate expiration within 30 days
  - Database storage exceeding 80% capacity
- **Info Alerts**:
  - Configuration changes detected
  - New admin console access
  - Successful disaster recovery test completion

#### 8.1.3 Tracing
- Integration with platform distributed tracing system
- Trace header propagation required for all requests
- Span generation for internal Keycloak operations
- Correlation of traces across service boundaries

### 8.2 Backup and Recovery

#### 8.2.1 Backup Strategy
- **Primary**: AWS RDS automated snapshots (twice daily)
- **Retention**: 30 days for all backups
- **Storage**: AWS-managed RDS snapshot storage (same region)
- **Point-In-Time Recovery (PITR)**: Enabled for granular restoration
- **Configuration Backups**: Keycloak realm configuration exported daily and stored in Git
- **Initial deployment**: Single-region, multi-AZ

#### 8.2.2 Recovery Procedures
Detailed recovery procedures will be documented in a separate disaster recovery runbook. High-level recovery scenarios include:
- Complete Keycloak service failure
- Database corruption or loss
- Realm configuration corruption
- Certificate expiration recovery
- Network partition recovery

### 8.3 Update and Patch Management

- **Update Method**: Keycloak Operator managed rolling updates
- **Testing Requirements**: Staging validation before production
- **Version Policy**: Maximum two minor versions behind latest stable
- **Security Patch SLA**:
  - Critical vulnerabilities: Apply within 24 hours (may bypass standard change control)
  - High severity: Apply within 7 days
  - Medium severity: Apply within 30 days
  - Low severity: Apply in next scheduled update
- **Change Control**: All non-critical updates require PR with SRE review
- **Update Windows**: Updates performed during business hours with zero-downtime deployment

---

## 9. Appendices

### Appendix A: Configuration Examples

#### A.1 Keycloak Realm Configuration Example
```json
{
  "realm": "catalyst-forge",
  "enabled": true,
  "sslRequired": "all",
  "registrationAllowed": false,
  "registrationEmailAsUsername": true,
  "rememberMe": false,
  "verifyEmail": false,
  "loginWithEmailAllowed": true,
  "duplicateEmailsAllowed": false,
  "resetPasswordAllowed": false,
  "editUsernameAllowed": false,
  "bruteForceProtected": true,
  "permanentLockout": false,
  "maxFailureWaitSeconds": 900,
  "minimumQuickLoginWaitSeconds": 60,
  "waitIncrementSeconds": 60,
  "quickLoginCheckMilliSeconds": 1000,
  "maxDeltaTimeSeconds": 43200,
  "failureFactor": 10,
  "defaultRole": {
    "name": "developer",
    "description": "Default role for all authenticated users"
  }
}
```

#### A.2 Client Configuration Example (Backend API)
```json
{
  "clientId": "backend-api",
  "enabled": true,
  "protocol": "openid-connect",
  "publicClient": true,
  "standardFlowEnabled": false,
  "implicitFlowEnabled": false,
  "directAccessGrantsEnabled": false,
  "serviceAccountsEnabled": false,
  "authorizationServicesEnabled": false,
  "bearerOnly": true
}
```

#### A.3 Client Configuration Example (Frontend Application)
```json
{
  "clientId": "frontend-app",
  "enabled": true,
  "protocol": "openid-connect",
  "publicClient": true,
  "redirectUris": [
    "https://app.catalyst-forge.io/*",
    "http://localhost:3000/*"
  ],
  "webOrigins": [
    "https://app.catalyst-forge.io",
    "http://localhost:3000"
  ],
  "standardFlowEnabled": true,
  "implicitFlowEnabled": false,
  "directAccessGrantsEnabled": false,
  "attributes": {
    "pkce.code.challenge.method": "S256"
  }
}
```

#### A.4 Google OIDC Provider Configuration
```json
{
  "alias": "google",
  "providerId": "google",
  "enabled": true,
  "trustEmail": true,
  "storeToken": false,
  "addReadTokenRoleOnCreate": false,
  "authenticateByDefault": false,
  "linkOnly": false,
  "firstBrokerLoginFlowAlias": "first broker login",
  "config": {
    "clientId": "${GOOGLE_CLIENT_ID}",
    "clientSecret": "${GOOGLE_CLIENT_SECRET}",
    "hostedDomain": "yourcompany.com",
    "useJwksUrl": "true",
    "hideOnLoginPage": "false"
  }
}
```

#### A.5 SAML Configuration for AWS
```xml
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge">
  <IDPSSODescriptor WantAuthnRequestsSigned="true"
                    protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                         Location="https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge/protocol/saml"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge/protocol/saml"/>
  </IDPSSODescriptor>
</EntityDescriptor>
```

### Appendix B: API Specifications

#### B.1 Token Endpoint
```
POST /auth/realms/catalyst-forge/protocol/openid-connect/token
Content-Type: application/x-www-form-urlencoded

Request Parameters:
- grant_type: authorization_code | refresh_token | client_credentials | urn:ietf:params:oauth:grant-type:device_code
- client_id: <client_identifier>
- client_secret: <client_secret> (for confidential clients)
- code: <authorization_code> (for authorization_code grant)
- refresh_token: <refresh_token> (for refresh_token grant)
- device_code: <device_code> (for device grant)

Response:
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJ...",
  "token_type": "Bearer",
  "expires_in": 300,
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICI...",
  "refresh_expires_in": 28800,
  "scope": "openid profile email"
}
```

#### B.2 JWKS Endpoint
```
GET /auth/realms/catalyst-forge/protocol/openid-connect/certs

Response:
{
  "keys": [
    {
      "kid": "vdaec4Br3ZnRFtZN-VIbTuMVVSmhvNu8YmYrDCaUJOo",
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "n": "modulus_value_base64",
      "e": "AQAB"
    }
  ]
}
```

#### B.3 Custom JWT Claims Structure
```json
{
  "exp": 1234567890,
  "iat": 1234567890,
  "jti": "unique-token-id",
  "iss": "https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge",
  "aud": ["backend-api", "account"],
  "sub": "user-uuid",
  "typ": "Bearer",
  "azp": "frontend-app",
  "session_state": "session-id",
  "acr": "1",
  "realm_access": {
    "roles": ["developer", "default-roles"]
  },
  "resource_access": {
    "backend-api": {
      "roles": ["api-user"]
    }
  },
  "scope": "openid profile email",
  "email_verified": true,
  "name": "John Doe",
  "preferred_username": "john.doe@company.com",
  "given_name": "John",
  "family_name": "Doe",
  "email": "john.doe@company.com",
  "groups": ["engineering", "platform-team"],
  "custom_claims": {
    "department": "engineering",
    "employee_id": "EMP123",
    "projects": ["project-a", "project-b"]
  }
}
```

#### B.4 Device Authorization Endpoint
```
POST /auth/realms/catalyst-forge/protocol/openid-connect/auth/device
Content-Type: application/x-www-form-urlencoded

Request:
client_id=cli-tool&scope=openid profile

Response:
{
  "device_code": "Xg0u5pPZxHRn6P5IwKRmE4gJLQqxXCz",
  "user_code": "HJKL-MNOP",
  "verification_uri": "https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge/device",
  "verification_uri_complete": "https://keycloak.catalyst-forge.io/auth/realms/catalyst-forge/device?user_code=HJKL-MNOP",
  "expires_in": 600,
  "interval": 5
}
```

#### B.5 Token Exchange Endpoint (GitHub Actions)
```
POST /auth/realms/catalyst-forge/protocol/openid-connect/token
Content-Type: application/x-www-form-urlencoded

Request:
grant_type=urn:ietf:params:oauth:grant-type:token-exchange&
subject_token=<github_oidc_token>&
subject_token_type=urn:ietf:params:oauth:token-type:jwt&
requested_token_type=urn:ietf:params:oauth:token-type:access_token&
audience=backend-api

Response:
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJ...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 300,
  "scope": "deploy:prod",
  "refresh_token": null
}
```

#### B.6 User Info Endpoint
```
GET /auth/realms/catalyst-forge/protocol/openid-connect/userinfo
Authorization: Bearer <access_token>

Response:
{
  "sub": "user-uuid",
  "email_verified": true,
  "name": "John Doe",
  "preferred_username": "john.doe@company.com",
  "given_name": "John",
  "family_name": "Doe",
  "email": "john.doe@company.com"
}
```

### Appendix C: Decision Log

#### C.1 M2M Authentication Decision

**Options Evaluated:**

**Option A: Traditional Client Credentials**
- ✅ Simple, well-understood
- ✅ Works with existing PyInfra flow
- ❌ Quarterly rotation (manual process)
- ❌ Long-lived secrets on disk

**Option B: mTLS Authentication**
- ✅ Automatic 5-day rotation
- ✅ Leverages existing PCA infrastructure
- ✅ No long-lived secrets
- ❌ Requires Keycloak mTLS configuration
- ❌ More complex initial setup

**Recommendation**: Option B (mTLS) - provides superior security with less operational overhead given existing PCA infrastructure.

**Decision**: [Pending final decision]

### Appendix D: Open Issues

1. **Token Exchange Feature Stability**: Preview feature risk for GitHub Actions integration
2. **RBAC Model Definition**: Specific roles and permissions need to be defined
3. **Additional Integration Targets**: List of services beyond AWS and Argo CD
4. **Performance Benchmarks**: Specific targets for concurrent users and requests/second
5. **Disaster Recovery Testing**: DR procedures need to be documented and tested
6. **Compliance Requirements**: Specific SOC 2 controls to be implemented

### Appendix E: Implementation Details

#### E.1 Build Machine Security Implementation

**PyInfra Deployment Flow:**
```bash
# PyInfra deployment flow
1. SRE authenticates with AWS (local credentials)
2. PyInfra uses boto3 to fetch from AWS Secrets Manager:
   - client_id for build machine
   - client_secret for build machine
3. Deploy to /etc/earthly/keycloak-creds (0600, earthly:earthly)
4. Configure systemd service to read credentials
```

**Network Restrictions:**
```bash
# iptables/nftables outbound rules
ALLOW -> Keycloak (port 443)
ALLOW -> Backend API (port 443)
ALLOW -> AWS PCA endpoints
ALLOW -> Package repositories
ALLOW -> Earthly registry (if used)
DENY -> Everything else
```

**Service Configuration:**
```ini
# /etc/systemd/system/earthly-satellite.service
[Service]
User=earthly
Group=earthly
PrivateTmp=true
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=/etc/earthly/keycloak-creds.json
# Or for mTLS approach:
ReadOnlyPaths=/etc/earthly/certs/
```

#### E.2 GitHub Actions Workflow Example

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      id-token: write # Required to request the OIDC token
      contents: read
    steps:
      - name: 'Get GitHub OIDC Token'
        id: get_token
        uses: actions/github-script@v6
        with:
          script: |
            const token = await core.getIDToken('keycloak-client-id');
            core.setOutput('token', token);

      - name: 'Exchange Token with Keycloak'
        id: exchange_token
        run: |
          # Script to POST to Keycloak's token endpoint
          # and get the platform access token
```

---

**Document Status**: This document is complete and ready for review by stakeholders.