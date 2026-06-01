package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/jzelinskie/cobrautil/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/go-extras/cobraflags"
	"github.com/tupyy/superagent/internal/config"
	"github.com/tupyy/superagent/internal/podman"
	"github.com/tupyy/superagent/internal/service"
	"github.com/tupyy/superagent/internal/store"
)

func NewRunCommand(cfg *config.Configuration) *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run agents and aggregate inventory",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return cfg.Validate()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			zap.S().Infow("using configuration",
				"vcenters", cfg.Agent.VCenters,
				"image", cfg.Agent.Image,
				"db", cfg.Store.DBPath,
				"podman-socket", cfg.Podman.Socket,
			)

			db, err := store.NewDB(cfg.Store.DBPath)
			if err != nil {
				return err
			}
			s := store.New(db)
			defer s.Close()

			runner, err := podman.NewPodmanRunner(cfg.Podman.Socket)
			if err != nil {
				return err
			}

			inventorySvc := service.NewInventoryService(s)
			virtualMachineSvc := service.NewVirtualMachineService(s)
			collector := service.NewCollectorService(runner, cfg.Agent.Image, cfg.Credentials, inventorySvc, virtualMachineSvc, s, cfg.Agent.RandomizeVCenterID)

			if err := collector.Start(); err != nil {
				return fmt.Errorf("starting collector: %w", err)
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-sigCh:
					zap.S().Info("received shutdown signal")
					return collector.Stop()
				case <-ticker.C:
					if !collector.IsRunning() {
						st := collector.State()
						if st.Error != nil {
							zap.S().Errorw("collector failed", "state", st.State, "error", st.Error)
						} else {
							zap.S().Infow("collector finished", "state", st.State)
						}
						return collector.Stop()
					}
					st := collector.State()
					zap.S().Infow("collector status", "state", st.State)
				}
			}
		},
	}

	registerFlags(runCmd, cfg)
	cobraflags.CobraOnInitialize("SUPERAGENT", runCmd)

	return runCmd
}

func registerFlags(cmd *cobra.Command, cfg *config.Configuration) {
	nfs := cobrautil.NewNamedFlagSets(cmd)

	agentFlagSet := nfs.FlagSet(color.New(color.FgBlue, color.Bold).Sprint("Agent"))
	registerAgentFlags(agentFlagSet, cfg)

	podmanFlagSet := nfs.FlagSet(color.New(color.FgBlue, color.Bold).Sprint("Podman"))
	registerPodmanFlags(podmanFlagSet, cfg)

	storeFlagSet := nfs.FlagSet(color.New(color.FgBlue, color.Bold).Sprint("Store"))
	registerStoreFlags(storeFlagSet, cfg)

	nfs.AddFlagSets(cmd)
}

func registerAgentFlags(flagSet *pflag.FlagSet, cfg *config.Configuration) {
	flagSet.StringArrayVar(&cfg.Agent.VCenters, "vcenter", cfg.Agent.VCenters, "vCenter URL (repeatable, e.g. https://user:pass@host/sdk)")
	flagSet.StringVar(&cfg.Agent.Image, "image", cfg.Agent.Image, "Agent container image")
	flagSet.BoolVar(&cfg.Agent.RandomizeVCenterID, "randomize-vcenter-id", false, "Use a random UUID as vCenter ID instead of the one from inventory")
}

func registerPodmanFlags(flagSet *pflag.FlagSet, cfg *config.Configuration) {
	flagSet.StringVar(&cfg.Podman.Socket, "podman-socket", cfg.Podman.Socket, "Podman socket path")
}

func registerStoreFlags(flagSet *pflag.FlagSet, cfg *config.Configuration) {
	flagSet.StringVar(&cfg.Store.DBPath, "db", cfg.Store.DBPath, "Path to DuckDB database file")
}
