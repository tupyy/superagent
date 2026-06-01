package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/tupyy/superagent/internal/config"
	"github.com/tupyy/superagent/internal/store"
)

func NewDiffCommand(cfg *config.Configuration) *cobra.Command {
	var sequences string
	var vcenterID string
	var output string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare VMs between two collection sequences",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			seqA, seqB, err := parseSequences(sequences)
			if err != nil {
				return err
			}

			if output != "table" && output != "json" {
				return fmt.Errorf("invalid output format: %s (must be table or json)", output)
			}

			db, err := store.NewDB(cfg.Store.DBPath)
			if err != nil {
				return err
			}
			s := store.New(db)
			defer s.Close()

			ctx := context.Background()
			available, err := s.VirtualMachines().SequencesByVCenter(ctx, vcenterID)
			if err != nil {
				return fmt.Errorf("listing sequences: %w", err)
			}
			seqSet := make(map[int64]struct{}, len(available))
			for _, seq := range available {
				seqSet[seq] = struct{}{}
			}
			for _, seq := range []int64{seqA, seqB} {
				if _, ok := seqSet[seq]; !ok {
					return fmt.Errorf("sequence %d not found for vcenter %s (available: %v)", seq, vcenterID, available)
				}
			}

			missing, err := s.VirtualMachines().Diff(ctx, seqA, seqB, vcenterID)
			if err != nil {
				return fmt.Errorf("computing diff: %w", err)
			}

			if output == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(toJSONRecords(missing))
			}

			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"vCenter ID", "VM Name", "VM ID", "Cluster", "Datacenter"})
			for _, m := range missing {
				t.AppendRow(table.Row{m.VCenterID, m.Name, m.ID, m.Cluster, m.Datacenter})
			}
			t.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&sequences, "sequences", "", "Comma-separated sequence pair (e.g. 1,2)")
	cmd.MarkFlagRequired("sequences")
	cmd.Flags().StringVar(&vcenterID, "vcenter-id", "", "vCenter ID to compare")
	cmd.MarkFlagRequired("vcenter-id")
	cmd.Flags().StringVar(&output, "output", "table", "Output format: table or json")
	cmd.Flags().StringVar(&cfg.Store.DBPath, "db", cfg.Store.DBPath, "Path to DuckDB database file")

	return cmd
}

func parseSequences(s string) (int64, int64, error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("--sequences must be a comma-separated pair (e.g. 1,2)")
	}
	a, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid sequence value %q: %w", parts[0], err)
	}
	b, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid sequence value %q: %w", parts[1], err)
	}
	return a, b, nil
}

type diffRecord struct {
	VCenterID  string `json:"vcenter_id"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Cluster    string `json:"cluster"`
	Datacenter string `json:"datacenter"`
}

func toJSONRecords(records []store.VirtualMachineRecord) []diffRecord {
	out := make([]diffRecord, len(records))
	for i, r := range records {
		out[i] = diffRecord{
			VCenterID:  r.VCenterID,
			ID:         r.ID,
			Name:       r.Name,
			Cluster:    r.Cluster,
			Datacenter: r.Datacenter,
		}
	}
	return out
}
