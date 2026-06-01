package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tupyy/superagent/cmd"
	"github.com/tupyy/superagent/internal/config"
	"github.com/tupyy/superagent/pkg/logger"
)

func main() {
	cfg := &config.Configuration{
		Podman: config.Podman{
			Socket: config.DefaultPodmanSocket(),
		},
		Store: config.Store{
			DBPath: "superagent.duckdb",
		},
		LogFormat: "console",
		LogLevel:  "info",
	}

	rootCmd := &cobra.Command{
		Use:          "superagent",
		Short:        "Orchestrate migration agents and aggregate inventory",
		SilenceUsage: true,
	}

	registerLoggingFlags(rootCmd, cfg)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := validateLogConfig(cfg); err != nil {
			return err
		}
		l := logger.Init(cfg.LogFormat, cfg.LogLevel)
		zap.ReplaceGlobals(l)
		return nil
	}

	rootCmd.AddCommand(cmd.NewRunCommand(cfg))
	rootCmd.AddCommand(cmd.NewAggregateCommand(cfg))
	rootCmd.AddCommand(cmd.NewListCommand(cfg))
	rootCmd.AddCommand(cmd.NewDiffCommand(cfg))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func validateLogConfig(cfg *config.Configuration) error {
	switch cfg.LogFormat {
	case "console", "json":
	default:
		return fmt.Errorf("invalid log-format: %s", cfg.LogFormat)
	}
	if _, err := zapcore.ParseLevel(cfg.LogLevel); err != nil {
		return fmt.Errorf("invalid log level %s", cfg.LogLevel)
	}
	return nil
}

func registerLoggingFlags(cmd *cobra.Command, cfg *config.Configuration) {
	cmd.PersistentFlags().StringVar(&cfg.LogFormat, "log-format", cfg.LogFormat, "Log format: console or json")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
}
