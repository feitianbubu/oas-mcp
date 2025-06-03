package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/feitianbubu/oas-mcp/internal/config"
	"github.com/feitianbubu/oas-mcp/internal/parser"
	"github.com/feitianbubu/oas-mcp/internal/requester"
	"github.com/feitianbubu/oas-mcp/internal/server"

	"github.com/feitianbubu/oas-mcp/internal/logger"
	"github.com/spf13/pflag"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	// Initialize all command-line flags
	showVersion := pflag.BoolP("version", "v", false, "Show version information")
	config.InitFlags()
	pflag.Parse()

	if *showVersion {
		fmt.Println(config.GetVersionInfo())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override disable_console setting if server mode is stdio
	if cfg.Server.Mode == config.ServerModeSTDIO {
		cfg.Logging.DisableConsole = true
	}

	// Initialize logger
	loggerConfig := &logger.LoggingConfig{
		Level:          cfg.Logging.Level,
		DisableConsole: cfg.Logging.DisableConsole,
		File:           cfg.Logging.File,
	}
	if err := logger.InitLogger(loggerConfig); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Application panic recovered",
				zap.Any("error", r),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	// Create app with dependencies
	app := fx.New(
		fx.NopLogger,
		parser.Module,
		server.Module,
		requester.Module,
		// Config Provider
		fx.Provide(func() *config.Config { return cfg }),
		fx.Provide(func() *config.EndpointConfig { return &cfg.EndpointConfig }),
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server) {
			appCtx, cancel := context.WithCancel(context.Background())
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						if err := srv.Start(appCtx); err != nil {
							logger.Error("Server exited with error", zap.Error(err))
							os.Exit(1)
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					cancel()
					return nil
				},
			})
		}),
	)

	// Start the application
	if err := app.Err(); err != nil {
		logger.Error("Failed to initialize application", zap.Error(err))
		os.Exit(1)
	}

	app.Run()
}
