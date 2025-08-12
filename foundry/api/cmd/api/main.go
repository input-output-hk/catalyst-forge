package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	metrics "github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s/mocks"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/input-output-hk/catalyst-forge/foundry/api/docs"
)

var (
	version = "dev"
	cfgFile string
	cfg     config.Config
)

var mockK8sClient = mocks.ClientMock{
	CreateDeploymentFunc: func(ctx context.Context, deployment *models.ReleaseDeployment) error {
		return nil
	},
}

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "foundry-api",
	Short: "Catalyst Foundry API Server",
	Long:  `API for managing releases and deployments in the Catalyst Foundry system.`,
}

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the API server",
	Long:  `Start the Catalyst Foundry API server with the configured settings.`,
	RunE:  runServer,
}

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("foundry api version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	},
}

// seedCmd represents the seed command.
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed default data (admin user/role)",
	Long:  `Create an admin user and optionally an admin role with all permissions.`,
	RunE:  runSeed,
}

// mockCmd represents the mock command.
var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Generate mock releases, deployments and events",
	Long:  `Populate the database with mock data for testing purposes.`,
	RunE:  runMock,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/foundry/foundry-api.toml)")

	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(mockCmd)
	// Auth command will be added separately with its own structure

	// Run command flags - Server config
	runCmd.Flags().Int("http-port", 8080, "HTTP port to listen on")
	runCmd.Flags().Duration("server-timeout", 30*time.Second, "Server timeout")
	runCmd.Flags().String("public-base-url", "", "Public base URL for generating links")
	runCmd.Flags().String("cookie-samesite", "Strict", "Cookie SameSite policy (Strict|Lax|None)")

	// Auth config
	runCmd.Flags().String("auth-private-key", "", "Path to private key for JWT authentication")
	runCmd.Flags().String("auth-public-key", "", "Path to public key for JWT authentication")
	runCmd.Flags().Duration("invite-ttl", 72*time.Hour, "Default invite TTL")
	runCmd.Flags().Duration("auth-access-ttl", 30*time.Minute, "Access token TTL")
	runCmd.Flags().Duration("auth-refresh-ttl", 720*time.Hour, "Refresh token TTL for new token families")
	runCmd.Flags().Duration("auth-refresh-skew", 30*time.Second, "Clock skew tolerance for device proofs")
	runCmd.Flags().String("auth-refresh-cookie-name", "cforge_rt", "Name of refresh token cookie")
	runCmd.Flags().String("auth-refresh-cookie-domain", "", "Domain for refresh token cookies")
	runCmd.Flags().Bool("auth-refresh-cookie-secure", true, "Force secure flag on refresh cookies")
	runCmd.Flags().String("auth-allowed-web-origins", "", "Comma-separated list of allowed web origins for CORS")
	runCmd.Flags().String("refresh-hash-secret", "", "Secret for HMAC refresh token validation")
	runCmd.Flags().Int("auth-rate-limit-burst", 10, "Rate limit burst for auth endpoints per user")
	runCmd.Flags().Duration("auth-rate-limit-window", time.Minute, "Rate limit window for auth endpoints")
	runCmd.Flags().String("bootstrap-token", "", "One-time bootstrap token for creating initial admin invite")

	// Database config
	runCmd.Flags().String("db-host", "localhost", "Database host")
	runCmd.Flags().Int("db-port", 5432, "Database port")
	runCmd.Flags().String("db-user", "postgres", "Database user")
	runCmd.Flags().String("db-password", "", "Database password")
	runCmd.Flags().String("db-name", "releases", "Database name")
	runCmd.Flags().String("db-sslmode", "disable", "Database SSL mode")

	// Logging config
	runCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	runCmd.Flags().String("log-format", "json", "Log format (json, text)")

	// Kubernetes config
	runCmd.Flags().String("k8s-namespace", "default", "Kubernetes namespace to use")
	runCmd.Flags().Bool("k8s-enabled", false, "Enable Kubernetes integration")

	// Email config
	runCmd.Flags().Bool("email-enabled", false, "Enable outbound emails")
	runCmd.Flags().String("email-provider", "none", "Email provider (ses, none)")
	runCmd.Flags().String("email-sender", "", "Sender email address")
	runCmd.Flags().String("email-ses-region", "", "AWS SES region")

	// Security config
	runCmd.Flags().Bool("enable-naive-per-ip-ratelimit", false, "Enable in-process per-IP rate limiting")

	// Certs config
	runCmd.Flags().String("certs-pca-client-ca-arn", "", "ACM-PCA ARN for client certificates")
	runCmd.Flags().String("certs-pca-server-ca-arn", "", "ACM-PCA ARN for server certificates")
	runCmd.Flags().String("certs-pca-client-template-arn", "", "ACM-PCA template ARN for client certs")
	runCmd.Flags().String("certs-pca-server-template-arn", "", "ACM-PCA template ARN for server certs")
	runCmd.Flags().String("certs-pca-signing-algo-client", "SHA256WITHECDSA", "ACM-PCA SigningAlgorithm for client certs")
	runCmd.Flags().String("certs-pca-signing-algo-server", "SHA256WITHECDSA", "ACM-PCA SigningAlgorithm for server certs")
	runCmd.Flags().Duration("certs-pca-timeout", 10*time.Second, "Timeout for ACM-PCA calls")
	runCmd.Flags().Duration("certs-client-cert-ttl-dev", 90*time.Minute, "Default TTL for developer client certs")
	runCmd.Flags().Duration("certs-client-cert-ttl-ci-max", 120*time.Minute, "Maximum TTL for CI client certs")
	runCmd.Flags().Duration("certs-server-cert-ttl", 336*time.Hour, "TTL for server certificates")
	runCmd.Flags().Int("certs-issuance-rate-hourly", 6, "Max certificate issuances per hour per subject/repo")
	runCmd.Flags().Int("certs-session-max-active", 10, "Max concurrent build sessions per user")
	runCmd.Flags().Bool("certs-require-perms-and", true, "RequireAll authorization semantics globally")
	runCmd.Flags().Bool("certs-ext-authz-enabled", false, "Enable optional external authorization endpoint")
	runCmd.Flags().String("certs-gh-oidc-issuer", "https://token.actions.githubusercontent.com", "GitHub OIDC issuer")
	runCmd.Flags().String("certs-gh-oidc-audience", "forge", "Expected audience for GitHub OIDC")
	runCmd.Flags().String("certs-gh-allowed-orgs", "", "Comma-separated allowed GitHub orgs")
	runCmd.Flags().String("certs-gh-allowed-repos", "", "Comma-separated allowed <org>/<repo> entries")
	runCmd.Flags().String("certs-gh-protected-refs", "", "Comma-separated protected refs")
	runCmd.Flags().Duration("certs-gh-jwks-cache-ttl", 10*time.Minute, "JWKS cache TTL for GitHub OIDC")
	runCmd.Flags().Duration("certs-job-token-ttl", 60*time.Minute, "Default TTL for minted CI job tokens")
	runCmd.Flags().String("certs-ca-region", "", "AWS region for CA register")
	runCmd.Flags().String("certs-ca-ddb-table", "", "DynamoDB table for CA register pointers")
	runCmd.Flags().String("certs-ca-s3-bucket", "", "S3 bucket for CA register artifacts")

	// Bind flags to viper
	bindRunFlags()

	// Seed command flags
	seedCmd.Flags().String("email", "admin@foundry.dev", "Admin email to seed")
	seedCmd.Flags().Bool("with-role", true, "Create admin role with all permissions and assign to user")

	// Mock command flags
	mockCmd.Flags().String("projects", "alpha,beta,gamma", "Comma-separated project names")
	mockCmd.Flags().String("branches", "main,develop,release/1.x", "Comma-separated branch names")
	mockCmd.Flags().Int("releases", 100, "Number of releases per project")
	mockCmd.Flags().Int("deployments", 3, "Number of deployments per release")
	mockCmd.Flags().Int("events", 6, "Number of events per deployment")
}

func bindRunFlags() {
    // Bind all run command flags to viper
    _ = viper.BindPFlag("server.httpport", runCmd.Flags().Lookup("http-port"))
    _ = viper.BindPFlag("server.timeout", runCmd.Flags().Lookup("server-timeout"))
    _ = viper.BindPFlag("server.publicbaseurl", runCmd.Flags().Lookup("public-base-url"))
    _ = viper.BindPFlag("server.cookiesamesite", runCmd.Flags().Lookup("cookie-samesite"))

    _ = viper.BindPFlag("auth.privatekey", runCmd.Flags().Lookup("auth-private-key"))
    _ = viper.BindPFlag("auth.publickey", runCmd.Flags().Lookup("auth-public-key"))
    _ = viper.BindPFlag("auth.invitettl", runCmd.Flags().Lookup("invite-ttl"))
    _ = viper.BindPFlag("auth.accessttl", runCmd.Flags().Lookup("auth-access-ttl"))
    _ = viper.BindPFlag("auth.refreshttl", runCmd.Flags().Lookup("auth-refresh-ttl"))
    _ = viper.BindPFlag("auth.refreshskew", runCmd.Flags().Lookup("auth-refresh-skew"))
    _ = viper.BindPFlag("auth.refreshcookiename", runCmd.Flags().Lookup("auth-refresh-cookie-name"))
    _ = viper.BindPFlag("auth.refreshcookiedomain", runCmd.Flags().Lookup("auth-refresh-cookie-domain"))
    _ = viper.BindPFlag("auth.refreshcookiesecure", runCmd.Flags().Lookup("auth-refresh-cookie-secure"))
    _ = viper.BindPFlag("auth.allowedweborigins", runCmd.Flags().Lookup("auth-allowed-web-origins"))
    _ = viper.BindPFlag("auth.refreshhashsecret", runCmd.Flags().Lookup("refresh-hash-secret"))
    _ = viper.BindPFlag("auth.authratelimitburst", runCmd.Flags().Lookup("auth-rate-limit-burst"))
    _ = viper.BindPFlag("auth.authratelimitwindow", runCmd.Flags().Lookup("auth-rate-limit-window"))
    _ = viper.BindPFlag("auth.bootstraptoken", runCmd.Flags().Lookup("bootstrap-token"))

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

	// Bind certs flags
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

func initConfig() {
	// Set config file paths
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in standard locations
		viper.SetConfigName("foundry-api")
		viper.SetConfigType("toml")
		viper.AddConfigPath("/etc/foundry")
		viper.AddConfigPath("/etc")
		viper.AddConfigPath("$HOME/.config/foundry")
		viper.AddConfigPath(".")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind specific environment variables that match Kong's naming
    _ = viper.BindEnv("server.httpport", "HTTP_PORT")
    _ = viper.BindEnv("server.timeout", "SERVER_TIMEOUT")
    _ = viper.BindEnv("server.publicbaseurl", "PUBLIC_BASE_URL")
    _ = viper.BindEnv("server.cookiesamesite", "COOKIE_SAMESITE")

    _ = viper.BindEnv("auth.privatekey", "AUTH_PRIVATE_KEY")
    _ = viper.BindEnv("auth.publickey", "AUTH_PUBLIC_KEY")
    _ = viper.BindEnv("auth.invitettl", "INVITE_TTL")
    _ = viper.BindEnv("auth.accessttl", "AUTH_ACCESS_TTL")
    _ = viper.BindEnv("auth.refreshttl", "AUTH_REFRESH_TTL")
    _ = viper.BindEnv("auth.refreshskew", "AUTH_REFRESH_SKEW")
    _ = viper.BindEnv("auth.refreshcookiename", "AUTH_REFRESH_COOKIE_NAME")
    _ = viper.BindEnv("auth.refreshcookiedomain", "AUTH_REFRESH_COOKIE_DOMAIN")
    _ = viper.BindEnv("auth.refreshcookiesecure", "AUTH_REFRESH_COOKIE_SECURE")
    _ = viper.BindEnv("auth.allowedweborigins", "AUTH_ALLOWED_WEB_ORIGINS")
    _ = viper.BindEnv("auth.refreshhashsecret", "REFRESH_HASH_SECRET")
    _ = viper.BindEnv("auth.authratelimitburst", "AUTH_RATE_LIMIT_BURST")
    _ = viper.BindEnv("auth.authratelimitwindow", "AUTH_RATE_LIMIT_WINDOW")
    _ = viper.BindEnv("auth.bootstraptoken", "BOOTSTRAP_TOKEN")

    _ = viper.BindEnv("database.host", "DB_HOST")
    _ = viper.BindEnv("database.dbport", "DB_PORT")
    _ = viper.BindEnv("database.user", "DB_USER")
    _ = viper.BindEnv("database.password", "DB_PASSWORD")
    _ = viper.BindEnv("database.name", "DB_NAME")
    _ = viper.BindEnv("database.sslmode", "DB_SSLMODE")

    _ = viper.BindEnv("logging.level", "LOG_LEVEL")
    _ = viper.BindEnv("logging.format", "LOG_FORMAT")

    _ = viper.BindEnv("kubernetes.namespace", "K8S_NAMESPACE")
    _ = viper.BindEnv("kubernetes.enabled", "K8S_ENABLED")

    _ = viper.BindEnv("email.enabled", "EMAIL_ENABLED")
    _ = viper.BindEnv("email.provider", "EMAIL_PROVIDER")
    _ = viper.BindEnv("email.sender", "EMAIL_SENDER")
    _ = viper.BindEnv("email.sesregion", "SES_REGION")

    _ = viper.BindEnv("security.enablenaiveperipratelimit", "ENABLE_PER_IP_RATELIMIT")

	// Bind cert environment variables
    _ = viper.BindEnv("certs.pcaclientcaarn", "PCA_CLIENT_CA_ARN")
    _ = viper.BindEnv("certs.pcaservercaarn", "PCA_SERVER_CA_ARN")
    _ = viper.BindEnv("certs.pcaclienttemplatearn", "PCA_CLIENT_TEMPLATE_ARN")
    _ = viper.BindEnv("certs.pcaservertemplatearn", "PCA_SERVER_TEMPLATE_ARN")
    _ = viper.BindEnv("certs.pcasigningalgoclient", "PCA_SIGNING_ALGO_CLIENT")
    _ = viper.BindEnv("certs.pcasigningalgoserver", "PCA_SIGNING_ALGO_SERVER")
    _ = viper.BindEnv("certs.pcatimeout", "PCA_TIMEOUT")
    _ = viper.BindEnv("certs.clientcertttldev", "CLIENT_CERT_TTL_DEV")
    _ = viper.BindEnv("certs.clientcertttlcimax", "CLIENT_CERT_TTL_CI_MAX")
    _ = viper.BindEnv("certs.servercertttl", "SERVER_CERT_TTL")
    _ = viper.BindEnv("certs.issuanceratehourly", "ISSUANCE_RATE_HOURLY")
    _ = viper.BindEnv("certs.sessionmaxactive", "SESSION_MAX_ACTIVE")
    _ = viper.BindEnv("certs.requirepermsand", "REQUIRE_PERMS_AND")
    _ = viper.BindEnv("certs.extauthzenabled", "EXT_AUTHZ_ENABLED")
    _ = viper.BindEnv("certs.ghoidcissuer", "GITHUB_OIDC_ISS")
    _ = viper.BindEnv("certs.ghoidcaudience", "GITHUB_OIDC_AUD")
    _ = viper.BindEnv("certs.ghallowedorgs", "GITHUB_ALLOWED_ORGS")
    _ = viper.BindEnv("certs.ghallowedrepos", "GITHUB_ALLOWED_REPOS")
    _ = viper.BindEnv("certs.ghprotectedrefs", "GITHUB_PROTECTED_REFS")
    _ = viper.BindEnv("certs.ghjwkscachettl", "GITHUB_JWKS_CACHE_TTL")
    _ = viper.BindEnv("certs.jobtokendefaultttl", "JOB_TOKEN_TTL")
    _ = viper.BindEnv("certs.caregion", "CAREGION")
    _ = viper.BindEnv("certs.caddbtable", "CA_DDB_TABLE")
    _ = viper.BindEnv("certs.cas3bucket", "CA_S3_BUCKET")

	// Read in config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func loadConfig() error {
	// Load configuration from viper into our config struct
	cfg.Server.HttpPort = viper.GetInt("server.httpport")
	cfg.Server.Timeout = viper.GetDuration("server.timeout")
	cfg.Server.PublicBaseURL = viper.GetString("server.publicbaseurl")
	cfg.Server.CookieSameSite = viper.GetString("server.cookiesamesite")

	cfg.Auth.PrivateKey = viper.GetString("auth.privatekey")
	cfg.Auth.PublicKey = viper.GetString("auth.publickey")
	cfg.Auth.InviteTTL = viper.GetDuration("auth.invitettl")
	cfg.Auth.AccessTTL = viper.GetDuration("auth.accessttl")
	cfg.Auth.RefreshTTL = viper.GetDuration("auth.refreshttl")
	cfg.Auth.RefreshSkew = viper.GetDuration("auth.refreshskew")
	cfg.Auth.RefreshCookieName = viper.GetString("auth.refreshcookiename")
	cfg.Auth.RefreshCookieDomain = viper.GetString("auth.refreshcookiedomain")
	cfg.Auth.RefreshCookieSecure = viper.GetBool("auth.refreshcookiesecure")
	cfg.Auth.AllowedWebOrigins = viper.GetString("auth.allowedweborigins")
	cfg.Auth.RefreshHashSecret = viper.GetString("auth.refreshhashsecret")
	cfg.Auth.AuthRateLimitBurst = viper.GetInt("auth.authratelimitburst")
	cfg.Auth.AuthRateLimitWindow = viper.GetDuration("auth.authratelimitwindow")
	cfg.Auth.BootstrapToken = viper.GetString("auth.bootstraptoken")

	cfg.Database.Host = viper.GetString("database.host")
	cfg.Database.DbPort = viper.GetInt("database.dbport")
	cfg.Database.User = viper.GetString("database.user")
	cfg.Database.Password = viper.GetString("database.password")
	cfg.Database.Name = viper.GetString("database.name")
	cfg.Database.SSLMode = viper.GetString("database.sslmode")

	cfg.Logging.Level = viper.GetString("logging.level")
	cfg.Logging.Format = viper.GetString("logging.format")

	cfg.Kubernetes.Namespace = viper.GetString("kubernetes.namespace")
	cfg.Kubernetes.Enabled = viper.GetBool("kubernetes.enabled")

	cfg.Email.Enabled = viper.GetBool("email.enabled")
	cfg.Email.Provider = viper.GetString("email.provider")
	cfg.Email.Sender = viper.GetString("email.sender")
	cfg.Email.SESRegion = viper.GetString("email.sesregion")

	cfg.Security.EnableNaivePerIPRateLimit = viper.GetBool("security.enablenaiveperipratelimit")

	// Load certs config
	cfg.Certs.PCAClientCAArn = viper.GetString("certs.pcaclientcaarn")
	cfg.Certs.PCAServerCAArn = viper.GetString("certs.pcaservercaarn")
	cfg.Certs.PCAClientTemplateArn = viper.GetString("certs.pcaclienttemplatearn")
	cfg.Certs.PCAServerTemplateArn = viper.GetString("certs.pcaservertemplatearn")
	cfg.Certs.PCASigningAlgoClient = viper.GetString("certs.pcasigningalgoclient")
	cfg.Certs.PCASigningAlgoServer = viper.GetString("certs.pcasigningalgoserver")
	cfg.Certs.PCATimeout = viper.GetDuration("certs.pcatimeout")
	cfg.Certs.ClientCertTTLDev = viper.GetDuration("certs.clientcertttldev")
	cfg.Certs.ClientCertTTLCIMax = viper.GetDuration("certs.clientcertttlcimax")
	cfg.Certs.ServerCertTTL = viper.GetDuration("certs.servercertttl")
	cfg.Certs.IssuanceRateHourly = viper.GetInt("certs.issuanceratehourly")
	cfg.Certs.SessionMaxActive = viper.GetInt("certs.sessionmaxactive")
	cfg.Certs.RequirePermsAnd = viper.GetBool("certs.requirepermsand")
	cfg.Certs.ExtAuthzEnabled = viper.GetBool("certs.extauthzenabled")
	cfg.Certs.GhOIDCIssuer = viper.GetString("certs.ghoidcissuer")
	cfg.Certs.GhOIDCAudience = viper.GetString("certs.ghoidcaudience")
	cfg.Certs.GhAllowedOrgs = viper.GetString("certs.ghallowedorgs")
	cfg.Certs.GhAllowedRepos = viper.GetString("certs.ghallowedrepos")
	cfg.Certs.GhProtectedRefs = viper.GetString("certs.ghprotectedrefs")
	cfg.Certs.GhJWKSCacheTTL = viper.GetDuration("certs.ghjwkscachettl")
	cfg.Certs.JobTokenDefaultTTL = viper.GetDuration("certs.jobtokendefaultttl")
	cfg.Certs.CARegion = viper.GetString("certs.caregion")
	cfg.Certs.CADDBTable = viper.GetString("certs.caddbtable")
	cfg.Certs.CAS3Bucket = viper.GetString("certs.cas3bucket")

	return cfg.Validate()
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	if err := loadConfig(); err != nil {
		return err
	}

	// Initialize logger
	logger, err := cfg.GetLogger()
	if err != nil {
		return err
	}

	// Connect to the database
	db, err := openDB(cfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return err
	}

	// Run migrations
	logger.Info("Running database migrations")
	err = runMigrations(db)
	if err != nil {
		logger.Error("Failed to run migrations", "error", err)
		return err
	}

	// Context reserved for future init steps
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = ctx
	cancel()

	// Initialize Kubernetes client if enabled
	var k8sClient k8s.Client
	if cfg.Kubernetes.Enabled {
		logger.Info("Initializing Kubernetes client", "namespace", cfg.Kubernetes.Namespace)
		k8sClient, err = initK8sClient(cfg.Kubernetes, logger)
		if err != nil {
			logger.Error("Failed to initialize Kubernetes client", "error", err)
			return err
		}
	} else {
		k8sClient = &mockK8sClient
		logger.Info("Kubernetes integration is disabled")
	}

	// Initialize repositories
	releaseRepo := repository.NewReleaseRepository(db)
	deploymentRepo := repository.NewDeploymentRepository(db)
	counterRepo := repository.NewIDCounterRepository(db)
	aliasRepo := repository.NewAliasRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ghaAuthRepo := repository.NewGithubAuthRepository(db)

	// Initialize user repositories
	userRepo := userrepo.NewUserRepository(db)
	roleRepo := userrepo.NewRoleRepository(db)
	userRoleRepo := userrepo.NewUserRoleRepository(db)
	userKeyRepo := userrepo.NewUserKeyRepository(db)

	// Initialize services
	releaseService := service.NewReleaseService(releaseRepo, aliasRepo, counterRepo, deploymentRepo)
	deploymentService := service.NewDeploymentService(deploymentRepo, releaseRepo, eventRepo, k8sClient, db, logger)
	ghaAuthService := service.NewGithubAuthService(ghaAuthRepo, logger)

	// Initialize user services
	userService := userservice.NewUserService(userRepo, logger)
	roleService := userservice.NewRoleService(roleRepo, logger)
	userRoleService := userservice.NewUserRoleService(userRoleRepo, logger)
	userKeyService := userservice.NewUserKeyService(userKeyRepo, logger)

	// Initialize middleware
	jwtManagerImpl, err := initJWTManager(cfg.Auth, logger)
	if err != nil {
		logger.Error("Failed to initialize JWT manager", "error", err)
		return err
	}
	jwtManager := jwtManagerImpl
	revokedRepo := userrepo.NewRevokedJTIRepository(db)
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, logger, userService, revokedRepo)

	// Initialize GitHub Actions OIDC client
	ghaOIDCClient, err := ghauth.NewDefaultGithubActionsOIDCClient(context.Background(), "/tmp/gha-jwks-cache")
	if err != nil {
		logger.Error("Failed to initialize GHA OIDC client", "error", err)
		return err
	}

	// Start the GHA OIDC cache
	if err := ghaOIDCClient.StartCache(); err != nil {
		logger.Error("Failed to start GHA OIDC cache", "error", err)
		return err
	}
	defer ghaOIDCClient.StopCache()

	// Setup router
	// Optionally construct SES email service
	var emailService emailsvc.Service
	emailService, _ = initEmailService(cfg.Email, cfg.Server.PublicBaseURL)
	// Initialize Prometheus metrics
	metrics.InitDefault()

	// Initialize PCA if configured
	pcaCli, _ := initPCAClient(cfg.Certs)
	router := api.SetupRouter(
		releaseService,
		deploymentService,
		userService,
		roleService,
		userRoleService,
		userKeyService,
		authMiddleware,
		db,
		logger,
		jwtManager,
		ghaOIDCClient,
		ghaAuthService,
		emailService,
		cfg.Certs.SessionMaxActive,
		cfg.Security.EnableNaivePerIPRateLimit,
		pcaCli,
		&cfg,
	)
	// Inject defaults into request context
	injectDefaultContext(router, cfg, emailService)
	// Attach PCA client to certificate handler if available
	if pcaCli != nil {
		router.Use(func(c *gin.Context) { c.Set("pca_client_present", true); c.Next() })
	}
	// Expose cert TTL clamps
	router.Use(func(c *gin.Context) {
		c.Set("certs_client_cert_ttl_dev", cfg.Certs.ClientCertTTLDev)
		c.Set("certs_client_cert_ttl_ci_max", cfg.Certs.ClientCertTTLCIMax)
		c.Set("certs_server_cert_ttl", cfg.Certs.ServerCertTTL)
		c.Next()
	})

	// Initialize server
	server := api.NewServer(cfg.GetServerAddr(), router, logger)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Failed to start server", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	logger.Info("API server started", "addr", cfg.GetServerAddr())

	// Wait for shutdown signal
	<-quit
	logger.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
	return nil
}

func runSeed(cmd *cobra.Command, args []string) error {
	seedCmd := &SeedCmd{
		Email:    cmd.Flag("email").Value.String(),
		WithRole: cmd.Flag("with-role").Value.String() == "true",
	}
	return seedCmd.Run()
}

func runMock(cmd *cobra.Command, args []string) error {
	releases, _ := cmd.Flags().GetInt("releases")
	deployments, _ := cmd.Flags().GetInt("deployments")
	events, _ := cmd.Flags().GetInt("events")

	mockCmd := &MockDataCmd{
		Projects:    cmd.Flag("projects").Value.String(),
		Branches:    cmd.Flag("branches").Value.String(),
		Releases:    releases,
		Deployments: deployments,
		Events:      events,
	}
	return mockCmd.Run()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
