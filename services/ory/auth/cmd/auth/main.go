package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/api"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version = "dev"
	cfg     config.Config
)

var rootCmd = &cobra.Command{Use: "auth-orchestrator", Short: "Auth Orchestrator service"}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Auth Orchestrator",
	RunE: func(cmd *cobra.Command, args []string) error {
		applyFlagOverrides(cmd)
		if err := loadConfig(); err != nil {
			return err
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Initialize logger early, after config resolved
		logging.Setup(logging.Options{
			Level:     cfg.Log.Level,
			Format:    cfg.Log.Format,
			AddSource: cfg.Log.AddSource,
		})
		// Startup config dump (resolved values)
		logging.L().Info("config resolved",
			slog.String("server.addr", cfg.GetServerAddr()),
			slog.String("hydra.admin_url", cfg.Hydra.AdminURL),
			slog.String("kratos.public_url", cfg.Kratos.PublicURL),
			slog.Int("consent.remember_for", cfg.Consent.RememberForSeconds),
			slog.String("log.level", cfg.Log.Level),
			slog.String("log.format", cfg.Log.Format),
			slog.Bool("log.add_source", cfg.Log.AddSource),
		)

		r := api.SetupRouter(&cfg)
		server := api.NewServer(cfg.GetServerAddr(), r)

		go func() {
			if err := server.Start(); err != nil {
				logging.L().Error("server error", slog.String("error", err.Error()))
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	},
}

var versionCmd = &cobra.Command{Use: "version", Run: func(cmd *cobra.Command, args []string) { fmt.Println(version) }}

func init() {
	cobra.OnInitialize(initViper)

	// Flags
	runCmd.Flags().String("server.addr", ":8080", "HTTP listen address")
	runCmd.Flags().String("hydra.admin_url", "http://hydra-admin:4445", "Hydra Admin URL")
	runCmd.Flags().String("kratos.public_url", "http://kratos-public:4433", "Kratos Public URL")
	// Logging flags
	runCmd.Flags().String("log.level", "info", "Log level: debug, info, warn, error")
	runCmd.Flags().String("log.format", "text", "Log format: text or json")
	runCmd.Flags().Bool("log.add_source", false, "Include source position in logs")
	// Consent flags
	runCmd.Flags().Int("consent.remember_for", 300, "Consent remember duration in seconds")
	runCmd.Flags().StringSlice("consent.scopes", []string{"openid", "offline"}, "Default scopes to grant")
	runCmd.Flags().StringSlice("consent.audience", []string{}, "Default access token audiences to grant")

	// Env
	viper.SetEnvPrefix("AUTH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(runCmd, versionCmd)
}

func initViper() {
	viper.SetConfigName("auth-orchestrator")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()
	// Defaults
	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("hydra.admin_url", "http://hydra-admin:4445")
	viper.SetDefault("kratos.public_url", "http://kratos-public:4433")
	viper.SetDefault("consent.remember_for", 300)
	viper.SetDefault("consent.scopes", []string{"openid", "offline"})
	viper.SetDefault("consent.audience", []string{})
	viper.SetDefault("tls.ca_file", "")
	viper.SetDefault("tls.ca_pem", "")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.add_source", false)
	// Mapping defaults
	viper.SetDefault("mapping.path", "/etc/auth/mapping.yaml")
	viper.SetDefault("mapping.on_error", "deny")
	viper.SetDefault("mapping.merge_strategy", "deep")
	viper.SetDefault("mapping.reload", true)
}

func applyFlagOverrides(cmd *cobra.Command) {
	fs := cmd.Flags()
	if fs.Changed("server.addr") {
		if v, err := fs.GetString("server.addr"); err == nil {
			viper.Set("server.addr", v)
		}
	}
	if fs.Changed("hydra.admin_url") {
		if v, err := fs.GetString("hydra.admin_url"); err == nil {
			viper.Set("hydra.admin_url", v)
		}
	}
	if fs.Changed("kratos.public_url") {
		if v, err := fs.GetString("kratos.public_url"); err == nil {
			viper.Set("kratos.public_url", v)
		}
	}
	if fs.Changed("log.level") {
		if v, err := fs.GetString("log.level"); err == nil {
			viper.Set("log.level", v)
		}
	}
	if fs.Changed("log.format") {
		if v, err := fs.GetString("log.format"); err == nil {
			viper.Set("log.format", v)
		}
	}
	if fs.Changed("log.add_source") {
		if v, err := fs.GetBool("log.add_source"); err == nil {
			viper.Set("log.add_source", v)
		}
	}
	if fs.Changed("consent.remember_for") {
		if v, err := fs.GetInt("consent.remember_for"); err == nil {
			viper.Set("consent.remember_for", v)
		}
	}
	if fs.Changed("consent.scopes") {
		if v, err := fs.GetStringSlice("consent.scopes"); err == nil {
			viper.Set("consent.scopes", v)
		}
	}
	if fs.Changed("consent.audience") {
		if v, err := fs.GetStringSlice("consent.audience"); err == nil {
			viper.Set("consent.audience", v)
		}
	}
}

func loadConfig() error {
	return viper.Unmarshal(&cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
