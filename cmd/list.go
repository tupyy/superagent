package cmd

import (
	"context"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/tupyy/superagent/internal/config"
	"github.com/tupyy/superagent/internal/store"
)

func NewListCommand(cfg *config.Configuration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List collected inventories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := store.NewDB(cfg.Store.DBPath)
			if err != nil {
				return err
			}
			s := store.New(db)
			defer s.Close()

			records, err := s.Inventory().List(context.Background())
			if err != nil {
				return err
			}

			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"Sequence", "vCenter URL", "vCenter ID", "Date"})

			for _, r := range records {
				t.AppendRow(table.Row{
					r.Sequence,
					r.VCenterURL,
					r.ID,
					r.CollectedAt.Format("2006-01-02 15:04:05"),
				})
			}

			t.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.Store.DBPath, "db", cfg.Store.DBPath, "Path to DuckDB database file")

	return cmd
}
