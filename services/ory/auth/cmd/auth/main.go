package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/api"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
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
		if err := loadConfig(); err != nil {
			return err
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		r := api.SetupRouter(&cfg)
		server := api.NewServer(cfg.GetServerAddr(), r)

		go func() {
			if err := server.Start(); err != nil {
				log.Printf("server error: %v", err)
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
	// Consent flags
	runCmd.Flags().Int("consent.remember_for", 300, "Consent remember duration in seconds")
	runCmd.Flags().StringSlice("consent.scopes", []string{"openid", "offline"}, "Default scopes to grant")
	runCmd.Flags().StringSlice("consent.audience", []string{}, "Default access token audiences to grant")

	// Bind
	_ = viper.BindPFlag("server.addr", runCmd.Flags().Lookup("server.addr"))
	_ = viper.BindPFlag("hydra.admin_url", runCmd.Flags().Lookup("hydra.admin_url"))
	_ = viper.BindPFlag("kratos.public_url", runCmd.Flags().Lookup("kratos.public_url"))
	_ = viper.BindPFlag("consent.remember_for", runCmd.Flags().Lookup("consent.remember_for"))
	_ = viper.BindPFlag("consent.scopes", runCmd.Flags().Lookup("consent.scopes"))
	_ = viper.BindPFlag("consent.audience", runCmd.Flags().Lookup("consent.audience"))

	// Env
	viper.SetEnvPrefix("AUTH")
	viper.AutomaticEnv()

	rootCmd.AddCommand(runCmd, versionCmd)
}

func initViper() {
	viper.SetConfigName("auth-orchestrator")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()
}

func loadConfig() error {
	return viper.Unmarshal(&cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
