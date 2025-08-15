package main

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addRunFlags defines all flags for the run command with their defaults and help text.
func addRunFlags(cmd *cobra.Command) {
	// Server config
	cmd.Flags().Int("http-port", 8080, "HTTP port to listen on")
	cmd.Flags().Duration("server-timeout", 30*time.Second, "Server timeout")
	cmd.Flags().String("public-base-url", "", "Public base URL for generating links")
	cmd.Flags().String("cookie-samesite", "Strict", "Cookie SameSite policy (Strict|Lax|None)")

	// Auth config (AuthKit)
	cmd.Flags().Duration("invite-ttl", 72*time.Hour, "Default invite TTL")
	cmd.Flags().Duration("auth-access-ttl", 30*time.Minute, "Access token TTL")
	cmd.Flags().Duration("auth-refresh-ttl", 720*time.Hour, "Refresh token TTL")
	cmd.Flags().Duration("auth-stepup-ttl", 5*time.Minute, "Step-up authentication validity window")
	cmd.Flags().String("auth-rp-name", "Foundry Platform", "WebAuthn RP display name")
	cmd.Flags().Bool("auth-require-uv", true, "Require user verification (WebAuthn)")
	cmd.Flags().Duration("auth-challenge-ttl", 5*time.Minute, "WebAuthn challenge TTL")
	cmd.Flags().String("auth-refresh-cookie-name", "__Host-refresh_token", "Name of refresh token cookie")
	cmd.Flags().Bool("auth-refresh-cookie-secure", true, "Force secure flag on refresh cookies")
	cmd.Flags().Bool("auth-rate-enabled", true, "Enable AuthKit rate limiting (requires limiter wiring)")
	cmd.Flags().Bool("auth-jwks-route", true, "Expose /.well-known/jwks.json")
	cmd.Flags().String("auth-admin-aaguids", "", "Comma-separated allowed AAGUIDs for admin users (hardware keys)")
	// Do not set a non-empty default; env/config should supply when flag omitted
	cmd.Flags().String("bootstrap-token", "", "One-time bootstrap token for creating initial admin")
	cmd.Flags().Bool("auth-rbac-seed-defaults", true, "Seed baseline RBAC roles and permissions on startup")

	// Auth: GitHub OIDC
	cmd.Flags().Bool("auth-github-enabled", false, "Enable GitHub OIDC exchange endpoints")
	cmd.Flags().String("auth-github-issuer", "https://token.actions.githubusercontent.com", "GitHub OIDC issuer")
	cmd.Flags().String("auth-github-audiences", "", "Comma-separated audiences to accept for GitHub OIDC tokens")
	cmd.Flags().Duration("auth-github-jwks-cache-ttl", 15*time.Minute, "JWKS cache TTL for GitHub OIDC verifier")
	cmd.Flags().Duration("auth-github-exchange-ttl", 15*time.Minute, "Access token TTL for GitHub OIDC exchange")

	// Database config
	cmd.Flags().String("db-host", "localhost", "Database host")
	cmd.Flags().Int("db-port", 5432, "Database port")
	cmd.Flags().String("db-user", "postgres", "Database user")
	cmd.Flags().String("db-password", "", "Database password")
	cmd.Flags().String("db-name", "releases", "Database name")
	cmd.Flags().String("db-sslmode", "disable", "Database SSL mode")

	// Logging config
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().String("log-format", "json", "Log format (json, text)")

	// Kubernetes config
	cmd.Flags().String("k8s-namespace", "default", "Kubernetes namespace to use")
	cmd.Flags().Bool("k8s-enabled", false, "Enable Kubernetes integration")

	// Email config
	cmd.Flags().Bool("email-enabled", false, "Enable outbound emails")
	cmd.Flags().String("email-provider", "none", "Email provider (ses, none)")
	cmd.Flags().String("email-sender", "", "Sender email address")
	cmd.Flags().String("email-ses-region", "", "AWS SES region")

	// Security config
	cmd.Flags().Bool("enable-naive-per-ip-ratelimit", false, "Enable in-process per-IP rate limiting")

	// Certs config
	cmd.Flags().String("certs-pca-client-ca-arn", "", "ACM-PCA ARN for client certificates")
	cmd.Flags().String("certs-pca-server-ca-arn", "", "ACM-PCA ARN for server certificates")
	cmd.Flags().String("certs-pca-client-template-arn", "", "ACM-PCA template ARN for client certs")
	cmd.Flags().String("certs-pca-server-template-arn", "", "ACM-PCA template ARN for server certs")
	cmd.Flags().String("certs-pca-signing-algo-client", "SHA256WITHECDSA", "ACM-PCA SigningAlgorithm for client certs")
	cmd.Flags().String("certs-pca-signing-algo-server", "SHA256WITHECDSA", "ACM-PCA SigningAlgorithm for server certs")
	cmd.Flags().Duration("certs-pca-timeout", 10*time.Second, "Timeout for ACM-PCA calls")
	cmd.Flags().Duration("certs-client-cert-ttl-dev", 90*time.Minute, "Default TTL for developer client certs")
	cmd.Flags().Duration("certs-client-cert-ttl-ci-max", 120*time.Minute, "Maximum TTL for CI client certs")
	cmd.Flags().Duration("certs-server-cert-ttl", 336*time.Hour, "TTL for server certificates")
	cmd.Flags().Int("certs-issuance-rate-hourly", 6, "Max certificate issuances per hour per subject/repo")
	cmd.Flags().Int("certs-session-max-active", 10, "Max concurrent build sessions per user")
	cmd.Flags().Bool("certs-require-perms-and", true, "RequireAll authorization semantics globally")
	cmd.Flags().Bool("certs-ext-authz-enabled", false, "Enable optional external authorization endpoint")
	cmd.Flags().String("certs-gh-oidc-issuer", "https://token.actions.githubusercontent.com", "GitHub OIDC issuer (certs feature)")
	cmd.Flags().String("certs-gh-oidc-audience", "forge", "Expected audience for GitHub OIDC (certs feature)")
	cmd.Flags().String("certs-gh-allowed-orgs", "", "Comma-separated allowed GitHub orgs (certs feature)")
	cmd.Flags().String("certs-gh-allowed-repos", "", "Comma-separated allowed <org>/<repo> entries (certs feature)")
	cmd.Flags().String("certs-gh-protected-refs", "", "Comma-separated protected refs (certs feature)")
	cmd.Flags().Duration("certs-gh-jwks-cache-ttl", 10*time.Minute, "JWKS cache TTL for GitHub OIDC (certs feature)")
	cmd.Flags().Duration("certs-job-token-ttl", 60*time.Minute, "Default TTL for minted CI job tokens (certs feature)")
	cmd.Flags().String("certs-ca-region", "", "AWS region for CA register")
	cmd.Flags().String("certs-ca-ddb-table", "", "DynamoDB table for CA register pointers")
	cmd.Flags().String("certs-ca-s3-bucket", "", "S3 bucket for CA register artifacts")
}

// bindRunFlags binds all flags to their corresponding Viper keys.
func bindRunFlags() {
	_ = viper.BindPFlag("server.httpport", runCmd.Flags().Lookup("http-port"))
	_ = viper.BindPFlag("server.timeout", runCmd.Flags().Lookup("server-timeout"))
	_ = viper.BindPFlag("server.publicbaseurl", runCmd.Flags().Lookup("public-base-url"))
	_ = viper.BindPFlag("server.cookiesamesite", runCmd.Flags().Lookup("cookie-samesite"))

	_ = viper.BindPFlag("auth.invitettl", runCmd.Flags().Lookup("invite-ttl"))
	_ = viper.BindPFlag("auth.accessttl", runCmd.Flags().Lookup("auth-access-ttl"))
	_ = viper.BindPFlag("auth.refreshttl", runCmd.Flags().Lookup("auth-refresh-ttl"))
	_ = viper.BindPFlag("auth.stepupttl", runCmd.Flags().Lookup("auth-stepup-ttl"))
	_ = viper.BindPFlag("auth.rpname", runCmd.Flags().Lookup("auth-rp-name"))
	_ = viper.BindPFlag("auth.requireuv", runCmd.Flags().Lookup("auth-require-uv"))
	_ = viper.BindPFlag("auth.challengettl", runCmd.Flags().Lookup("auth-challenge-ttl"))
	_ = viper.BindPFlag("auth.refreshcookiename", runCmd.Flags().Lookup("auth-refresh-cookie-name"))
	_ = viper.BindPFlag("auth.refreshcookiesecure", runCmd.Flags().Lookup("auth-refresh-cookie-secure"))
	_ = viper.BindPFlag("auth.rateenabled", runCmd.Flags().Lookup("auth-rate-enabled"))
	_ = viper.BindPFlag("auth.jwksroute", runCmd.Flags().Lookup("auth-jwks-route"))
	_ = viper.BindPFlag("auth.adminaaguids", runCmd.Flags().Lookup("auth-admin-aaguids"))
	// Intentionally avoid binding bootstrap-token to Viper to let ENV/Config take precedence

	_ = viper.BindPFlag("auth.github.enabled", runCmd.Flags().Lookup("auth-github-enabled"))
	_ = viper.BindPFlag("auth.github.issuer", runCmd.Flags().Lookup("auth-github-issuer"))
	_ = viper.BindPFlag("auth.github.audiences", runCmd.Flags().Lookup("auth-github-audiences"))
	_ = viper.BindPFlag("auth.github.jwkscachettl", runCmd.Flags().Lookup("auth-github-jwks-cache-ttl"))
	_ = viper.BindPFlag("auth.github.exchangettl", runCmd.Flags().Lookup("auth-github-exchange-ttl"))

	_ = viper.BindPFlag("database.host", runCmd.Flags().Lookup("db-host"))
	_ = viper.BindPFlag("database.dbport", runCmd.Flags().Lookup("db-port"))
	_ = viper.BindPFlag("database.user", runCmd.Flags().Lookup("db-user"))
	_ = viper.BindPFlag("database.password", runCmd.Flags().Lookup("db-password"))
	_ = viper.BindPFlag("database.name", runCmd.Flags().Lookup("db-name"))
	_ = viper.BindPFlag("database.sslmode", runCmd.Flags().Lookup("db-sslmode"))

	_ = viper.BindPFlag("logging.level", runCmd.Flags().Lookup("log-level"))
	_ = viper.BindPFlag("logging.format", runCmd.Flags().Lookup("log-format"))

	_ = viper.BindPFlag("kubernetes.namespace", runCmd.Flags().Lookup("k8s-namespace"))
	_ = viper.BindPFlag("kubernetes.enabled", runCmd.Flags().Lookup("k8s-enabled"))

	_ = viper.BindPFlag("email.enabled", runCmd.Flags().Lookup("email-enabled"))
	_ = viper.BindPFlag("email.provider", runCmd.Flags().Lookup("email-provider"))
	_ = viper.BindPFlag("email.sender", runCmd.Flags().Lookup("email-sender"))
	_ = viper.BindPFlag("email.sesregion", runCmd.Flags().Lookup("email-ses-region"))

	_ = viper.BindPFlag("security.enablenaiveperipratelimit", runCmd.Flags().Lookup("enable-naive-per-ip-ratelimit"))

	_ = viper.BindPFlag("certs.pcaclientcaarn", runCmd.Flags().Lookup("certs-pca-client-ca-arn"))
	_ = viper.BindPFlag("certs.pcaservercaarn", runCmd.Flags().Lookup("certs-pca-server-ca-arn"))
	_ = viper.BindPFlag("certs.pcaclienttemplatearn", runCmd.Flags().Lookup("certs-pca-client-template-arn"))
	_ = viper.BindPFlag("certs.pcaservertemplatearn", runCmd.Flags().Lookup("certs-pca-server-template-arn"))
	_ = viper.BindPFlag("certs.pcasigningalgoclient", runCmd.Flags().Lookup("certs-pca-signing-algo-client"))
	_ = viper.BindPFlag("certs.pcasigningalgoserver", runCmd.Flags().Lookup("certs-pca-signing-algo-server"))
	_ = viper.BindPFlag("certs.pcatimeout", runCmd.Flags().Lookup("certs-pca-timeout"))
	_ = viper.BindPFlag("certs.clientcertttldev", runCmd.Flags().Lookup("certs-client-cert-ttl-dev"))
	_ = viper.BindPFlag("certs.clientcertttlcimax", runCmd.Flags().Lookup("certs-client-cert-ttl-ci-max"))
	_ = viper.BindPFlag("certs.servercertttl", runCmd.Flags().Lookup("certs-server-cert-ttl"))
	_ = viper.BindPFlag("certs.issuanceratehourly", runCmd.Flags().Lookup("certs-issuance-rate-hourly"))
	_ = viper.BindPFlag("certs.sessionmaxactive", runCmd.Flags().Lookup("certs-session-max-active"))
	_ = viper.BindPFlag("certs.requirepermsand", runCmd.Flags().Lookup("certs-require-perms-and"))
	_ = viper.BindPFlag("certs.extauthzenabled", runCmd.Flags().Lookup("certs-ext-authz-enabled"))
	_ = viper.BindPFlag("certs.ghoidcissuer", runCmd.Flags().Lookup("certs-gh-oidc-issuer"))
	_ = viper.BindPFlag("certs.ghoidcaudience", runCmd.Flags().Lookup("certs-gh-oidc-audience"))
	_ = viper.BindPFlag("certs.ghallowedorgs", runCmd.Flags().Lookup("certs-gh-allowed-orgs"))
	_ = viper.BindPFlag("certs.ghallowedrepos", runCmd.Flags().Lookup("certs-gh-allowed-repos"))
	_ = viper.BindPFlag("certs.ghprotectedrefs", runCmd.Flags().Lookup("certs-gh-protected-refs"))
	_ = viper.BindPFlag("certs.ghjwkscachettl", runCmd.Flags().Lookup("certs-gh-jwks-cache-ttl"))
	_ = viper.BindPFlag("certs.jobtokendefaultttl", runCmd.Flags().Lookup("certs-job-token-ttl"))
	_ = viper.BindPFlag("certs.caregion", runCmd.Flags().Lookup("certs-ca-region"))
	_ = viper.BindPFlag("certs.caddbtable", runCmd.Flags().Lookup("certs-ca-ddb-table"))
	_ = viper.BindPFlag("certs.cas3bucket", runCmd.Flags().Lookup("certs-ca-s3-bucket"))
}
