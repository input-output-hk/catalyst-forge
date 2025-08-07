package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/input-output-hk/catalyst-forge/foundry/renderer/internal/server"
)

var version = "dev"

type CLI struct {
	Serve   ServeCmd   `cmd:"" help:"Start the gRPC renderer service." default:"1"`
	Version VersionCmd `cmd:"" help:"Print the version."`
}

type ServeCmd struct {
	Port      int    `short:"p" help:"gRPC server port" default:"8080" env:"PORT"`
	LogJSON   bool   `help:"Enable JSON logging" env:"LOG_JSON"`
	Debug     bool   `short:"d" help:"Enable debug logging" env:"DEBUG"`
	CachePath string `short:"c" help:"Path to cache directory for KCL OCI modules" default:"/tmp/renderer-cache" env:"CACHE_PATH"`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("renderer version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	return nil
}

func (c *ServeCmd) Run() error {
	// Setup structured logging
	var logger *slog.Logger
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if c.Debug {
		opts.Level = slog.LevelDebug
	}

	if c.LogJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	slog.SetDefault(logger)

	// Create server configuration
	config := server.Config{
		Port:      c.Port,
		Logger:    logger,
		CachePath: c.CachePath,
	}

	// Create and start server
	srv, err := server.NewServer(config)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		return err
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to listen for interrupt signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Server failed", "error", err)
			cancel()
		}
	}()

	logger.Info("Renderer service started", "port", c.Port)

	// Wait for interrupt signal or context cancellation
	select {
	case <-ch:
		logger.Info("Received shutdown signal")
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Start graceful shutdown in a goroutine
	done := make(chan struct{})
	go func() {
		srv.Stop()
		close(done)
	}()

	// Wait for graceful shutdown to complete or timeout
	select {
	case <-done:
		logger.Info("Server stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded, forcing stop")
		srv.ForceStop()
	}

	logger.Info("Renderer service shutdown complete")
	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("renderer"),
		kong.Description("Catalyst Forge Renderer Service - gRPC service for rendering deployment manifests"),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	if err := ctx.Run(); err != nil {
		slog.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
