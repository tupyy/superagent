package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	v1alpha1 "github.com/kubev2v/migration-planner/api/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/tupyy/superagent/internal/aggregate"
	"github.com/tupyy/superagent/internal/config"
	"github.com/tupyy/superagent/internal/store"
)

func NewAggregateCommand(cfg *config.Configuration) *cobra.Command {
	var sequence int64

	cmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Aggregate inventories from a collection run into a single inventory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := store.NewDB(cfg.Store.DBPath)
			if err != nil {
				return err
			}
			s := store.New(db)
			defer s.Close()

			records, err := s.Inventory().ListBySequence(context.Background(), sequence)
			if err != nil {
				return fmt.Errorf("loading inventories for sequence %d: %w", sequence, err)
			}
			if len(records) == 0 {
				return fmt.Errorf("no inventories found for sequence %d", sequence)
			}

			var inventories []v1alpha1.Inventory
			for _, r := range records {
				var inv v1alpha1.Inventory
				if err := json.Unmarshal(r.Data, &inv); err != nil {
					return fmt.Errorf("unmarshaling inventory %s: %w", r.ID, err)
				}
				inventories = append(inventories, inv)
			}

			merged := aggregate.MergeInventories(inventories)

			output := v1alpha1.UpdateInventory{
				AgentId:   uuid.New(),
				Inventory: merged,
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		},
	}

	cmd.Flags().Int64Var(&sequence, "sequence", 0, "Sequence ID of the collection run to aggregate")
	cmd.MarkFlagRequired("sequence")

	cmd.Flags().StringVar(&cfg.Store.DBPath, "db", cfg.Store.DBPath, "Path to DuckDB database file")

	return cmd
}
