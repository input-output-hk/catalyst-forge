# Software Requirements Specification
## AWS Private Certificate Authority Implementation

**Document Version:** 1.0
**Date:** [TODO: Add date]
**Status:** Draft
**Authors:** [TODO: Add authors]

---

## Table of Contents
1. [Introduction](#1-introduction)
2. [Overall Description](#2-overall-description)
3. [Specific Requirements](#3-specific-requirements)
4. [Appendices](#4-appendices)

---

## 1. Introduction

### 1.1 Purpose
This Software Requirements Specification (SRS) documents the requirements for implementing an internal Certificate Authority (CA) infrastructure using AWS Private Certificate Authority (PCA) to replace manual certificate management processes and enable secure machine-to-machine communication across our infrastructure.

The intended audience includes:
- Engineering team members responsible for implementation
- Security team members responsible for review and approval
- Operations team members responsible for maintenance

### 1.2 Scope
**Product Name:** Internal Certificate Authority Infrastructure

**Product Goals:**
- Establish automated certificate lifecycle management using AWS PCA
- Enable secure mTLS authentication for build infrastructure
- Provide certificate infrastructure for Istio service mesh
- Support machine identity verification for OIDC/JWT authentication flows

**Benefits:**
- Elimination of manual certificate generation and distribution
- Improved security through automated rotation and shorter certificate lifetimes
- Centralized certificate management and audit capabilities
- Cost-optimized implementation suitable for startup scale

**Major Features:**
- Hierarchical CA structure with root and intermediate CAs
- API-based certificate issuance with RBAC controls
- Integration with existing Keycloak authentication
- Support for multiple certificate profiles and use cases

### 1.3 Definitions, Acronyms, and Abbreviations
- **CA** - Certificate Authority
- **PCA** - AWS Private Certificate Authority (AWS Private CA)
- **mTLS** - Mutual Transport Layer Security
- **EKS** - Amazon Elastic Kubernetes Service
- **RBAC** - Role-Based Access Control
- **SAN** - Subject Alternative Name
- **CRL** - Certificate Revocation List
- **OCSP** - Online Certificate Status Protocol
- **JWT** - JSON Web Token
- **OIDC** - OpenID Connect
- **TTL** - Time To Live
- **SM** - AWS Secrets Manager
- **STS** - AWS Security Token Service
- **KMS** - AWS Key Management Service
- **HSM** - Hardware Security Module
- **ACM** - AWS Certificate Manager
- **CSR** - Certificate Signing Request
- **Earthly** - Build automation tool for containerized builds
- **Buildkitd** - BuildKit daemon used by Earthly for remote builds
- **Istio Ambient Mode** - Sidecar-less service mesh architecture using ztunnel and waypoint proxies
- **Hetzner** - German data center provider hosting our bare-metal infrastructure
- **cert-manager** - Kubernetes certificate management controller
- **ztunnel** - Zero-trust tunnel, Istio's L4 proxy in ambient mode
- **waypoint** - Istio's L7 proxy in ambient mode

### 1.4 References
- IEEE Std 830-1998, IEEE Recommended Practice for Software Requirements Specifications
- ISO/IEC/IEEE 29148:2018, Systems and software engineering — Life cycle processes — Requirements engineering
- AWS Private Certificate Authority Documentation - https://docs.aws.amazon.com/privateca/latest/userguide/
- AWS Private CA API Reference - https://docs.aws.amazon.com/privateca/latest/APIReference/
- Istio Documentation (v1.27.1) - https://istio.io/latest/docs/
- Istio Ambient Mode Architecture - https://istio.io/latest/docs/ambient/
- cert-manager Documentation (v1.16.2) - https://cert-manager.io/docs/
- AWS Private CA Issuer for cert-manager - https://github.com/cert-manager/aws-privateca-issuer
- Keycloak Documentation (v26.3.0) - https://www.keycloak.org/documentation
- Earthly Documentation (v0.8.16) - https://earthly.dev/docs
- BuildKit Documentation - https://github.com/moby/buildkit
- AWS Certificate Manager Documentation - https://docs.aws.amazon.com/acm/latest/userguide/
- SOC 2 Trust Services Criteria - https://www.aicpa.org/resources/download/soc-2-reporting-on-an-examination-of-controls-at-a-service-organization-relevant-to-security-availability-processing-integrity-confidentiality-or-privacy

### 1.5 Overview
The remainder of this document is organized as follows:
- Section 2 provides an overall description of the system, including product perspective, functions, constraints, and assumptions
- Section 3 details specific functional and non-functional requirements
- Section 4 contains supporting appendices

---

## 2. Overall Description

### 2.1 Product Perspective

#### 2.1.1 System Context
The PCA implementation will operate within the following environment:
- Single AWS account infrastructure
- Four EKS clusters (dev, preprod, prod, shared-services)
- External bare-metal machines in Hetzner data centers
- Integration with existing AWS services (Secrets Manager, IAM, etc.)

#### 2.1.2 System Interfaces
- **AWS PCA API**: Direct API integration for certificate issuance and CA management
- **AWS Certificate Manager (ACM)**: Integration for AWS-managed certificate storage
- **Kubernetes cert-manager**: AWS PCA Issuer integration for Istio certificates
- **Custom Certificate API**: REST API for certificate issuance, internal-only access
- **Keycloak**: OAuth 2.0/OIDC provider for API authentication
- **Tailscale VPN**: Secure network connectivity between AWS and Hetzner infrastructure

#### 2.1.3 User Interfaces
- **Certificate API REST endpoints**: Internal REST API for certificate operations
- **Administrative CLI tools**: Command-line utilities for CA management
- **Monitoring dashboards**: CloudWatch dashboards for certificate metrics
- **Keycloak Admin Console**: For managing authentication and authorization

#### 2.1.4 Hardware Interfaces
- **Hetzner bare-metal machines**: Requiring certificate provisioning
- **AWS-managed HSMs**: Default FIPS 140-2 Level 2 certified HSMs provided by AWS PCA

#### 2.1.5 Software Interfaces
- **Earthly/Buildkitd**: Certificate consumption for mTLS (v0.8.16)
- **Istio Ambient Mesh**: Certificate provider integration (v1.27.1)
- **Keycloak**: Certificate validation for JWT exchange (v26.3.0)
- **Kubernetes**: Minimum version 1.30 for EKS clusters
- **cert-manager**: Version 1.16.2 with AWS PCA Issuer
- **AWS Services**: PCA, Secrets Manager, CloudWatch, IAM
- **Tailscale**: VPN for secure connectivity between AWS and Hetzner

#### 2.1.6 Communications Interfaces
- **TLS 1.2/1.3**: For all certificate-secured communications
- **REST API over HTTPS**: For certificate issuance API
- **Tailscale VPN**: Secure connectivity between AWS (eu-central-1) and Hetzner infrastructure
- **Internal network**: All API communication occurs over Tailscale-managed private network

#### 2.1.7 Memory Constraints
- No specific certificate caching requirements
- Standard memory allocation for API services running in EKS

#### 2.1.8 Operations
- Certificate issuance
- Certificate renewal
- Certificate revocation (future capability)
- CA key rotation
- Audit log generation

### 2.2 Product Functions

#### 2.2.1 Certificate Authority Hierarchy Management
- Establish and maintain root CA
- Provision and manage three intermediate CAs:
  - Client CA (for build clients)
  - Server CA (for Hetzner machines)
  - Istio CA (for service mesh)

#### 2.2.2 Certificate Lifecycle Management
- Automated certificate issuance
- Automated certificate renewal
- Certificate expiration monitoring
- Future: Certificate revocation support

#### 2.2.3 Authentication and Authorization
- Integration with Keycloak for API authentication
- RBAC-based certificate issuance policies
- Certificate-to-JWT exchange for machine authentication

#### 2.2.4 Use Case Support
- **UC1:** Earthly Remote Satellite Authentication
- **UC2:** Istio Ambient Mesh Certificate Provisioning
- **UC3:** Machine Identity for OIDC/JWT Exchange

### 2.3 User Characteristics

#### 2.3.1 DevOps Engineers
- Responsible for certificate deployment and automation
- Familiar with Kubernetes, AWS, and certificate concepts
- Requires streamlined certificate provisioning workflows

#### 2.3.2 Security Team
- Responsible for security policies and compliance
- Requires audit capabilities and security controls
- Future: SOC2 compliance requirements

#### 2.3.3 Application Developers
- Consumers of certificates through automated systems
- Limited direct interaction with CA infrastructure
- Requires transparent certificate management

### 2.4 Constraints

#### 2.4.1 Regulatory Constraints
- Future requirement for SOC2 compliance readiness
- Industry-standard cryptographic requirements
- No current regulatory compliance requirements

#### 2.4.2 Technical Constraints
- AWS PCA service limitations
- Maximum 7-day TTL for short-lived certificates
- Single AWS account architecture
- Cost optimization requirements for startup budget

#### 2.4.3 Design Constraints
- Must integrate with existing Earthly build system
- Must support Istio Ambient mode requirements
- Must maintain backward compatibility during migration

### 2.5 Assumptions and Dependencies

#### 2.5.1 Assumptions
- Keycloak deployment will be completed before full implementation
- Istio Ambient mesh deployment timeline aligns with CA implementation
- Current manual certificate management process will remain operational during migration
- Tailscale VPN provides reliable connectivity between AWS and Hetzner
- Certificate request volume will not exceed ~250 certificates per day during peak CI load
- AWS PCA service maintains standard availability SLAs
- Network latency over Tailscale is acceptable for certificate operations
- All infrastructure operates within eu-central-1 region

#### 2.5.2 Dependencies
- AWS PCA service availability in eu-central-1 region
- cert-manager v1.16.2 compatibility with AWS PCA Issuer
- Keycloak v26.3.0 OIDC implementation
- Tailscale VPN connectivity between Hetzner and AWS
- Kubernetes 1.30+ features in EKS clusters
- AWS IAM roles and policies for PCA access
- AWS KMS for key management in eu-central-1

---

## 3. Specific Requirements

### 3.1 Functional Requirements

#### 3.1.1 Certificate Authority Management

##### FR-CA-001: Root CA Establishment
- **Description:** System shall establish a single long-lived root CA in AWS PCA
- **Priority:** High
- **Acceptance Criteria:**
  - Root CA validity period: 10 years
  - Key algorithm: RSA 4096-bit
  - Storage: AWS-managed with default multi-AZ backup

##### FR-CA-002: Intermediate CA Provisioning
- **Description:** System shall provision three intermediate CAs under the root CA
- **Priority:** High
- **Acceptance Criteria:**
  - Client CA for non-server certificates
  - Server CA for Hetzner machine certificates
  - Istio CA for service mesh certificates
  - Validity period: 2 years for all intermediate CAs
  - Naming convention: `<company>-<type>-ca-<year>` (e.g., `company-client-ca-2025`)

##### FR-CA-003: CA Rotation Support
- **Description:** System shall support rotation of intermediate CAs without service disruption
- **Priority:** Medium
- **Acceptance Criteria:**
  - Automated notifications when CAs approach expiration
  - Manual trigger for rotation execution
  - 90-day overlap period between old and new CAs
  - Zero-downtime rotation process

#### 3.1.2 Certificate Issuance

##### FR-CI-001: API-Based Certificate Issuance
- **Description:** System shall provide REST API for certificate issuance
- **Priority:** High
- **Acceptance Criteria:**
  - API protected by Keycloak authentication
  - Support for different certificate profiles
  - API endpoints:
    - `POST /api/v1/certificates/issue` - Issue new certificate
    - `POST /api/v1/certificates/renew` - Renew existing certificate
    - `GET /api/v1/certificates/{serial}` - Get certificate details
    - `GET /api/v1/certificates/ca` - Get CA chain
  - PEM-encoded certificates in JSON responses to prevent encoding issues

##### FR-CI-002: Certificate Profile Management
- **Description:** System shall support multiple certificate profiles based on use case
- **Priority:** High
- **Acceptance Criteria:**
  - Client certificates: `keyUsage: digitalSignature, keyAgreement; extKeyUsage: clientAuth`
  - Server certificates: `keyUsage: digitalSignature, keyEncipherment; extKeyUsage: serverAuth`
  - Service certificates for Istio workloads: Standard TLS server certificate extensions
  - Machine identity certificates: Combined client and server extensions for dual-use

##### FR-CI-003: RBAC-Based Authorization
- **Description:** System shall enforce RBAC policies for certificate issuance
- **Priority:** High
- **Acceptance Criteria:**
  - Integration with Keycloak roles and permissions
  - Support for domain-based restrictions using glob patterns (e.g., `cert:*.mysite.com`)
  - Permission model:
    - `cert:issue:client` - Can issue client certificates
    - `cert:issue:server` - Can issue server certificates
    - `cert:issue:<domain-pattern>` - Domain-restricted issuance

##### FR-CI-004: Automated Certificate Renewal
- **Description:** System shall support automated certificate renewal before expiration
- **Priority:** High
- **Acceptance Criteria:**
  - Renewal triggered at 30% of certificate lifetime remaining
  - Notification system for renewal events
  - Retry policy: 3 attempts with exponential backoff (1min, 5min, 15min)
  - Support for both automated and manual renewal triggers

#### 3.1.3 Integration Requirements

##### FR-INT-001: Earthly Remote Satellite Integration
- **Description:** System shall provide certificates for Earthly remote satellite mTLS
- **Priority:** High
- **Acceptance Criteria:**
  - Client certificates for build clients
  - Server certificates for buildkitd satellites
  - Compatible with existing Earthly authentication flow
  - Certificate distribution:
    - Bespoke Go service for automatic rotation on build machines
    - Bootstrap with initial dual-purpose certificate
    - Service authenticates with Keycloak using cert to obtain JWT
    - Renewal via Certificate API using JWT authentication
    - Automatic local certificate replacement and service reload
    - Client certificates obtained via existing auth (GitHub OIDC, developer OIDC)

##### FR-INT-002: Istio cert-manager Integration
- **Description:** System shall integrate with cert-manager for Istio certificate provisioning
- **Priority:** High
- **Acceptance Criteria:**
  - AWS PCA Issuer CRD configuration in cert-manager
  - Certificate resources with appropriate `issuerRef`
  - Automatic certificate rotation 30 days before expiry
  - Certificates consumed by Istio ztunnel and waypoint proxies

##### FR-INT-003: Keycloak mTLS to JWT Exchange
- **Description:** System shall support certificate validation for JWT token exchange
- **Priority:** High
- **Acceptance Criteria:**
  - Hetzner machines can exchange valid certificates for JWTs
  - Certificate validation against Server CA
  - Implementation follows RFC 8705 (OAuth 2.0 Mutual-TLS Client Authentication)
  - Certificate tied to confidential OIDC client with service account
  - Client service account determines identity and access permissions

#### 3.1.4 Security Requirements

##### FR-SEC-001: Certificate Validation
- **Description:** System shall validate all certificate requests against defined policies
- **Priority:** High
- **Acceptance Criteria:**
  - Domain ownership validation
  - Request signature verification
  - Validation rules:
    - Request must come from authorized source for domain
    - Valid CSR format and signature
    - SAN entries match RBAC permissions
    - Certificate lifetime within policy limits
    - No duplicate serial numbers

##### FR-SEC-002: Audit Logging
- **Description:** System shall maintain comprehensive audit logs for all CA operations
- **Priority:** High
- **Acceptance Criteria:**
  - Log all certificate issuance events
  - Log all authentication attempts
  - Log all administrative operations
  - Log retention: 1 year minimum for compliance readiness
  - Log format: Structured JSON stored in dedicated PostgreSQL database
  - Required fields: timestamp, requester, certificate details, success/failure

##### FR-SEC-003: Key Protection
- **Description:** System shall protect all CA private keys using AWS KMS
- **Priority:** High
- **Acceptance Criteria:**
  - Hardware security module usage for root CA
  - Encryption at rest for all keys
  - Key access policies:
    - Root CA: No direct access (AWS managed only)
    - Intermediate CAs: Access only via PCA API with dedicated IAM roles
    - IAM roles assumable only by SRE team
    - Principle of least privilege for all access

### 3.2 Non-Functional Requirements

#### 3.2.1 Performance Requirements

##### NFR-PERF-001: Certificate Issuance Latency
- **Description:** Certificate issuance shall complete within acceptable time limits
- **Metric:** 95th percentile latency < 2 seconds
- **Priority:** Medium

##### NFR-PERF-002: API Throughput
- **Description:** Certificate API shall handle expected request volume
- **Metric:** Support 5 requests per second
- **Priority:** Medium

##### NFR-PERF-003: Renewal Processing
- **Description:** System shall process certificate renewals without service impact
- **Metric:** Process renewals within 1 hour of request
- **Priority:** Medium

#### 3.2.2 Reliability Requirements

##### NFR-REL-001: System Availability
- **Description:** Certificate issuance service shall maintain high availability
- **Metric:** 99.9% availability during business hours
- **Priority:** High
- **Notes:** Maintenance window: Tuesday 12:00 AM - 2:00 PM GMT

##### NFR-REL-002: Failover Capability
- **Description:** System shall handle AWS region failures gracefully
- **Metric:** RTO: 4 hours, RPO: 1 hour
- **Priority:** Medium

#### 3.2.3 Scalability Requirements

##### NFR-SCALE-001: Certificate Volume
- **Description:** System shall support growth in certificate requirements
- **Metric:** Support up to 1,000 active certificates
- **Priority:** Medium

##### NFR-SCALE-002: Multi-Region Support
- **Description:** System shall be designed for future multi-region expansion
- **Priority:** Low
- **Notes:** Not required for initial implementation

#### 3.2.4 Security Requirements

##### NFR-SEC-001: Encryption Standards
- **Description:** All certificates shall use industry-standard encryption
- **Requirements:**
  - Minimum RSA 2048-bit or ECDSA P-256
  - SHA-256 or stronger hash algorithms
  - TLS 1.2 minimum for all communications
- **Priority:** High

##### NFR-SEC-002: Compliance Readiness
- **Description:** System shall be designed for future SOC2 compliance
- **Requirements:**
  - Audit trail capabilities
  - Access control documentation
  - Security control implementation
  - Relevant SOC2 controls:
    - CC6.1 - Logical and Physical Access Controls
    - CC6.2 - Prior to Issuing System Credentials
    - CC6.3 - Entity Authentication
- **Priority:** Medium

##### NFR-SEC-003: Secret Management
- **Description:** System shall securely manage all sensitive configuration
- **Requirements:**
  - No hardcoded secrets
  - Integration with AWS Secrets Manager
  - Secret rotation policies:
    - API keys: 90 days
    - Service account credentials: 180 days
- **Priority:** High

#### 3.2.5 Maintainability Requirements

##### NFR-MAINT-001: Monitoring and Alerting
- **Description:** System shall provide comprehensive monitoring capabilities
- **Requirements:**
  - Certificate expiration monitoring (alerts at 30 and 7 days)
  - System health metrics
  - Integration with CloudWatch
  - Specific metrics:
    - Certificate expiration status
    - API latency and error rates
    - CA health status
    - Certificate issuance rate
  - Alert thresholds:
    - API error rate > 1%
    - API latency > 2 seconds (p95)
    - Certificate expiring within 7 days
- **Priority:** High

##### NFR-MAINT-002: Documentation
- **Description:** System shall be fully documented for operations team
- **Requirements:**
  - Architecture documentation
  - Operational runbooks
  - Troubleshooting guides
  - Documentation standards:
    - Markdown format for all documentation
    - Version controlled in Git
    - Updated with each release
- **Priority:** Medium

#### 3.2.6 Cost Requirements

##### NFR-COST-001: PCA Cost Optimization
- **Description:** System shall minimize AWS PCA operational costs
- **Requirements:**
  - Use long-lived root CA to minimize per-month charges
  - Optimize intermediate CA lifecycle
  - Cost targets:
    - Monthly budget: < €500
    - Expected monthly costs:
      - Root CA: €400/month
      - Intermediate CAs: €0 (waived if <1000 certificates/month per CA)
      - Total: ~€400/month
- **Priority:** High

##### NFR-COST-002: Operational Cost Monitoring
- **Description:** System shall track and report certificate infrastructure costs
- **Requirements:**
  - Monthly cost reporting
  - Cost allocation by use case
  - Cost tracking mechanisms:
    - AWS Cost Explorer tags for PCA resources
    - Monthly automated reports to engineering leadership
    - Alert if costs exceed €450/month
- **Priority:** Medium

### 3.3 Interface Requirements

#### 3.3.1 Certificate API Interface

##### API-001: Certificate Request Endpoint
```
POST /api/v1/certificates/issue
Authorization: Bearer <Keycloak JWT>
Content-Type: application/json

Request Body:
{
  "profile": "client|server|machine",  // Certificate profile type
  "common_name": "string",              // CN for the certificate
  "san": ["string"],                    // Subject Alternative Names
  "validity_days": number,              // Requested validity period (max 7)
  "csr": "string"                       // Optional: PEM-encoded CSR (if not provided, key pair generated)
}

Response (200 OK):
{
  "certificate": "string",              // PEM-encoded certificate
  "certificate_chain": "string",        // PEM-encoded certificate chain
  "private_key": "string",              // PEM-encoded private key (only if generated)
  "serial_number": "string",            // Certificate serial number
  "expires_at": "ISO-8601",             // Expiration timestamp
  "issuer_ca": "string"                 // Issuing CA identifier
}

Response (400 Bad Request):
{
  "error": "string",                    // Error code
  "message": "string"                   // Human-readable error message
}
```

##### API-002: Certificate Renewal Endpoint
```
POST /api/v1/certificates/renew
Authorization: Bearer <Keycloak JWT>
Content-Type: application/json

Request Body:
{
  "serial_number": "string",            // Serial number of certificate to renew
  "csr": "string"                       // Optional: New CSR (if not provided, reuse existing key)
}

Response (200 OK):
{
  "certificate": "string",              // PEM-encoded new certificate
  "certificate_chain": "string",        // PEM-encoded certificate chain
  "serial_number": "string",            // New certificate serial number
  "expires_at": "ISO-8601",             // New expiration timestamp
  "previous_serial": "string"           // Previous certificate serial number
}

Response (404 Not Found):
{
  "error": "certificate_not_found",
  "message": "Certificate with serial number not found or expired"
}
```

##### API-003: Certificate Status Endpoint
```
GET /api/v1/certificates/{serial_number}
Authorization: Bearer <Keycloak JWT>

Response (200 OK):
{
  "serial_number": "string",
  "common_name": "string",
  "san": ["string"],
  "status": "active|expired|revoked",
  "issued_at": "ISO-8601",
  "expires_at": "ISO-8601",
  "issuer_ca": "string",
  "profile": "string",
  "requested_by": "string"              // Identity that requested the certificate
}

Response (404 Not Found):
{
  "error": "certificate_not_found",
  "message": "Certificate not found"
}
```

##### API-004: CA Chain Endpoint
```
GET /api/v1/certificates/ca
Authorization: Bearer <Keycloak JWT>

Response (200 OK):
{
  "root_ca": "string",                  // PEM-encoded root CA certificate
  "intermediate_cas": {
    "client": "string",                 // PEM-encoded client CA certificate
    "server": "string",                 // PEM-encoded server CA certificate
    "istio": "string"                   // PEM-encoded Istio CA certificate
  }
}
```

#### 3.3.2 Monitoring Interface

##### MON-001: Metrics Export
```
GET /metrics
Format: Prometheus text-based exposition format

Metrics:
# Certificate issuance metrics
certificate_api_requests_total{method="issue|renew|status",status="success|failure"} counter
certificate_api_request_duration_seconds{method="issue|renew|status",quantile="0.5|0.95|0.99"} histogram
certificate_api_active_certificates{ca="client|server|istio"} gauge
certificate_api_certificates_expiring_soon{days="7|30",ca="client|server|istio"} gauge

# System health metrics
certificate_api_up{} gauge (1=healthy, 0=unhealthy)
certificate_api_database_connections_active{} gauge
certificate_api_pca_api_calls_total{operation="issue|get|list",status="success|failure"} counter
certificate_api_pca_api_latency_seconds{operation="issue|get|list",quantile="0.5|0.95|0.99"} histogram

# Business metrics
certificate_api_certificates_issued_total{profile="client|server|machine",ca="client|server|istio"} counter
certificate_api_certificates_renewed_total{profile="client|server|machine",ca="client|server|istio"} counter
certificate_api_authentication_failures_total{reason="invalid_token|expired_token|insufficient_permissions"} counter
```

##### MON-002: Health Check Endpoint
```
GET /health

Response (200 OK):
{
  "status": "healthy",
  "timestamp": "ISO-8601",
  "checks": {
    "database": "healthy",
    "aws_pca": "healthy",
    "keycloak": "healthy",
    "certificate_expiry": "healthy"
  },
  "version": "string"
}

Response (503 Service Unavailable):
{
  "status": "unhealthy",
  "timestamp": "ISO-8601",
  "checks": {
    "database": "healthy|unhealthy",
    "aws_pca": "healthy|unhealthy",
    "keycloak": "healthy|unhealthy",
    "certificate_expiry": "healthy|warning|critical"
  },
  "errors": ["string"]
}

Health Criteria:
- Database: Connection pool has available connections
- AWS PCA: Can reach PCA API and list CAs
- Keycloak: Can validate tokens
- Certificate Expiry: No CA certificates expiring within 30 days
```

### 3.4 Data Requirements

#### 3.4.1 Certificate Storage
- **Location:**
  - AWS Certificate Manager for AWS-integrated certificates (Istio)
  - PostgreSQL database for certificate metadata and audit trail
  - Local filesystem for Hetzner machine certificates
- **Format:** PEM encoding for all certificates
- **Retention:**
  - Active certificates: Retained until expiration + 30 days
  - Certificate metadata: 2 years for audit purposes
  - Expired certificates: 90 days after expiration

#### 3.4.2 Audit Log Storage
- **Location:** PostgreSQL database (dedicated audit schema)
- **Format:** JSON structured logging with the following schema:
```json
{
  "timestamp": "ISO-8601",
  "event_type": "certificate_issued|certificate_renewed|authentication_failed|ca_operation",
  "actor": {
    "identity": "string",
    "ip_address": "string",
    "auth_method": "jwt|certificate"
  },
  "resource": {
    "type": "certificate|ca|api",
    "identifier": "string"
  },
  "outcome": "success|failure",
  "details": {}
}
```
- **Retention:** 1 year minimum, extended to 3 years when SOC2 compliance is required

#### 3.4.3 Configuration Storage
- **Location:**
  - AWS Secrets Manager for sensitive configuration (API keys, credentials)
  - ConfigMap/Secrets in Kubernetes for application configuration
  - Environment variables for non-sensitive configuration
- **Format:**
```yaml
# Application configuration schema
api:
  port: 8443
  tls:
    enabled: true
    cert_path: /certs/tls.crt
    key_path: /certs/tls.key

database:
  host: postgresql.internal
  port: 5432
  database: certificates
  ssl_mode: require

aws:
  region: eu-central-1
  pca:
    root_ca_arn: "arn:aws:acm-pca:..."
    client_ca_arn: "arn:aws:acm-pca:..."
    server_ca_arn: "arn:aws:acm-pca:..."
    istio_ca_arn: "arn:aws:acm-pca:..."

keycloak:
  issuer_url: "https://keycloak.internal/realms/main"
  client_id: "certificate-api"

monitoring:
  metrics_port: 9090
  health_port: 8080
```

---

## 4. Appendices

### 4.1 Appendix A: Use Case Specifications

#### Use Case 1: Earthly Remote Satellite mTLS

**Actor:** Build Client (Developer workstation or CI system)

**Preconditions:**
- Build client has valid client certificate from Client CA
- Remote satellite has valid server certificate from Server CA
- Network connectivity established via Tailscale VPN
- Valid Earthly configuration pointing to remote satellite

**Main Flow:**
1. Build client initiates connection to remote satellite
2. TLS handshake initiated with satellite presenting server certificate
3. Client validates server certificate against CA chain
4. Client presents client certificate for mTLS
5. Server validates client certificate against CA chain
6. Secure channel established for build operations
7. Build commands executed on remote satellite
8. Build logs streamed back to client
9. Build artifacts transferred to client
10. Connection closed after build completion

**Alternative Flows:**
- **Certificate Expiration:** Auto-renewal triggered 72 hours before expiry
  1. Rotation service detects upcoming expiration
  2. New certificate requested from Certificate API
  3. Certificate replaced without interrupting active builds
  4. Old certificate remains valid during overlap period
- **Connection Retry:** 3 attempts with exponential backoff (1s, 5s, 15s)
  1. Initial connection attempt fails
  2. Wait for backoff period
  3. Retry connection with same credentials
  4. Alert generated after 3 failed attempts

**Postconditions:**
- Successful build execution over secure channel
- Audit log entry created with build details
- Metrics updated for monitoring

#### Use Case 2: Istio Certificate Provisioning

**Actor:** cert-manager in Kubernetes cluster

**Preconditions:**
- cert-manager configured with AWS PCA Issuer
- Istio configured to use cert-manager provider
- Istio ambient mode installed in cluster
- Proper RBAC configured for cert-manager service account

**Main Flow:**
1. Istio requests certificate for workload
2. cert-manager creates CertificateRequest resource
3. AWS PCA Issuer processes request
4. Certificate issued from Istio CA
5. Certificate delivered to Istio workload
6. Certificate mounted to pod filesystem
7. ztunnel/waypoint configured with new certificate
8. mTLS enabled for workload communications
9. Certificate status updated in cert-manager

**Alternative Flows:**
- **Renewal Flow:** Automatic renewal at 30% certificate lifetime
  1. cert-manager detects certificate nearing expiry
  2. New CertificateRequest created automatically
  3. New certificate issued and deployed
  4. Seamless rotation without service interruption
- **Failure Scenarios:** Retry with exponential backoff
  1. Certificate issuance fails (PCA API error)
  2. cert-manager retries after 30 seconds
  3. Exponential backoff up to 5 minutes
  4. Alert generated after 5 consecutive failures
  5. Manual intervention required if automated recovery fails

**Postconditions:**
- Workload has valid certificate for service mesh communication
- Certificate tracked in cert-manager with expiry monitoring
- Metrics exported for certificate lifecycle tracking

#### Use Case 3: Machine Certificate to JWT Exchange

**Actor:** Hetzner bare-metal machine

**Preconditions:**
- Machine has valid certificate from Server CA
- Keycloak configured for certificate authentication
- Machine registered in Keycloak with service account
- Confidential OIDC client configured with appropriate permissions

**Main Flow:**
1. Machine initiates mTLS connection to Keycloak
2. Machine presents certificate during TLS handshake
3. Keycloak validates certificate against Server CA
4. Keycloak extracts identity from certificate
5. Keycloak issues short-lived JWT token (1 hour validity)
6. Machine uses JWT for API authentication
7. Machine makes authorized API calls using JWT
8. JWT refreshed 10 minutes before expiry
9. Process repeats for continuous operation

**Alternative Flows:**
- **Certificate Renewal During JWT Validity:**
  1. New certificate obtained while JWT still valid
  2. Next JWT refresh uses new certificate
  3. Overlapping validity ensures no authentication gap
  4. Old certificate decommissioned after JWT expiry
- **Error Handling:** Fallback and retry mechanisms
  1. Certificate validation fails - retry with backoff
  2. JWT issuance fails - use cached JWT if valid
  3. Network error - retry up to 3 times
  4. Alert SRE team after repeated failures
  5. Fallback to manual intervention if automated recovery fails

**Postconditions:**
- Machine has valid JWT for API access
- Audit trail of authentication event logged
- Certificate-to-JWT mapping recorded for security analysis

### 4.2 Appendix B: Migration Strategy Considerations

While detailed migration steps are outside the scope of this SRS, the following requirements support migration:

- **Parallel operation** with existing manual process during transition period
- **Gradual migration** per use case (Earthly first, then Istio, then Keycloak)
- **Rollback capability** to manual process if issues arise
- **Minimal downtime** requirement (<1 hour) when migrating existing build agents to new PCA certificates
- **API compatibility layer** to support existing scripts during transition
- **Certificate format conversion** utilities for legacy systems
- **Dual-mode operation** supporting both manual and automated certificate issuance temporarily

### 4.3 Appendix C: Future Enhancements

The following capabilities are not required for initial implementation but should be considered in the design:

#### 4.3.1 Certificate Revocation (CRL/OCSP)
- **Estimated timeline:** >6 months
- **Design considerations:**
  - OCSP responder infrastructure required
  - Additional costs ($0.06/cert/month + $0.20/100k queries)
  - Integration with existing monitoring systems

#### 4.3.2 Multi-Region Support
- **Estimated timeline:** >2 years
- **Design considerations:**
  - Cross-region CA replication
  - Regional failover capabilities
  - Cost implications of multiple root CAs

#### 4.3.3 Hardware Security Module Integration
- **Estimated timeline:** >2 years
- **Design considerations:**
  - AWS CloudHSM integration
  - Additional monthly costs
  - Enhanced key protection for high-value certificates

#### 4.3.4 SOC2 Compliance Features
- **Estimated timeline:** ~1 year
- **Required enhancements:**
  - Extended audit log retention (3 years)
  - Formal access control procedures
  - Regular security assessments
  - Documented incident response procedures

### 4.4 Appendix D: Cost Analysis

#### 4.4.1 AWS PCA Pricing Structure (eu-central-1)
**CA Operation Costs:**
- General-purpose mode: $400/month per CA
- Short-lived certificate mode: $50/month per CA (certificates valid ≤7 days)

**Certificate Issuance Costs (General-purpose mode):**
- 1-1,000 certificates: $0.75 per certificate
- 1,001-10,000 certificates: $0.35 per certificate
- 10,001+ certificates: $0.001 per certificate

**OCSP (if enabled in future):**
- $0.06 per certificate per month if queried
- $0.20 per 100,000 OCSP queries

#### 4.4.2 Estimated Monthly Costs
Based on our architecture with a long-lived root CA and short-lived subordinate CAs:

- **Root CA:** $400/month (general-purpose mode, 10-year validity)
- **Client CA:** $50/month (short-lived certificate mode)
- **Server CA:** $50/month (short-lived certificate mode)
- **Istio CA:** $50/month (short-lived certificate mode)
- **Certificate issuance:**
  - ~7,500 certificates × $0.058 = $435/month
- **Total estimated:** ~$985/month

**Note:** The root CA must be in general-purpose mode due to its 10-year validity period. Subordinate CAs use short-lived certificate mode since they only issue certificates valid for ≤7 days.

#### 4.4.3 Cost Optimization Strategies
- Utilize 30-day free trial for initial testing
- Monitor certificate usage to stay within pricing tiers
- Consider short-lived certificate mode for frequently rotated certificates (saves $350/month on CA operation)
- Implement certificate caching to reduce unnecessary reissuance
- Regular review of certificate lifetime policies

### 4.5 Appendix E: Risk Analysis

#### 4.5.1 Technical Risks
- **Risk:** AWS PCA service limitations
  - **Mitigation:** Monitor AWS service limits, implement request throttling, maintain caching layer for frequently accessed certificates

- **Risk:** Integration complexity with existing systems
  - **Mitigation:** Phased rollout starting with non-critical systems, extensive testing in dev environment, maintain rollback procedures

#### 4.5.2 Security Risks
- **Risk:** Compromise of intermediate CA
  - **Mitigation:** HSM protection by default in AWS PCA, comprehensive audit logging, regular rotation every 2 years, immediate revocation capability

- **Risk:** Unauthorized certificate issuance
  - **Mitigation:** RBAC enforcement via Keycloak, detailed audit trails, anomaly detection alerts, regular access reviews

#### 4.5.3 Operational Risks
- **Risk:** Certificate expiration causing service outage
  - **Mitigation:** Automated monitoring with 30 and 7-day alerts, auto-renewal at 30% lifetime remaining, runbook for emergency certificate issuance

- **Risk:** Loss of connectivity between Hetzner and AWS
  - **Mitigation:** Local certificate caching, graceful degradation, Tailscale redundancy, extended certificate lifetimes for critical services

### 4.6 Appendix F: Glossary
- **Ambient Mode:** Istio's sidecar-less architecture using shared node proxies
- **Buildkitd:** The daemon component of BuildKit that executes container builds
- **Certificate Authority (CA):** Entity that issues digital certificates
- **Certificate Signing Request (CSR):** Message sent to CA to request a certificate
- **Earthly:** Build automation tool for reproducible builds
- **Hetzner:** German hosting provider for our bare-metal infrastructure
- **Intermediate CA:** CA certificate signed by a root CA, used to issue end-entity certificates
- **mTLS:** Mutual TLS where both client and server authenticate each other
- **Root CA:** Self-signed CA at the top of the certificate hierarchy
- **Subject Alternative Name (SAN):** X.509 extension allowing multiple hostnames in one certificate
- **Tailscale:** Zero-configuration VPN built on WireGuard
- **ztunnel:** Zero-trust tunnel, Istio's Layer 4 proxy in ambient mode

### 4.7 Appendix G: References and Resources
- AWS Private CA Documentation: https://docs.aws.amazon.com/privateca/latest/userguide/
- AWS Private CA Pricing: https://aws.amazon.com/private-ca/pricing/
- cert-manager AWS PCA Issuer: https://cert-manager.github.io/aws-privateca-issuer/
- Istio Ambient Mode Guide: https://istio.io/latest/docs/ambient/
- Keycloak mTLS Documentation: https://www.keycloak.org/docs/latest/server_admin/#_mutual_tls
- RFC 8705 - OAuth 2.0 Mutual-TLS: https://datatracker.ietf.org/doc/html/rfc8705
- SOC 2 Compliance Guide: https://www.aicpa.org/resources/download/soc-2-reporting

---

## Document History

| Version | Date | Author | Description |
|---------|------|--------|-------------|
| 1.0 | 2025-09-03 | Engineering Team | Initial draft |

## Review and Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Engineering Lead | TBD | _______ | _______ |
| Security Lead | TBD | _______ | _______ |
| Operations Lead | TBD | _______ | _______ |