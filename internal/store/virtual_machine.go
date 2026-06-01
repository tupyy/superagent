package store

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	agentv1 "github.com/kubev2v/assisted-migration-agent/api/v1"
)

type VirtualMachineRecord struct {
	ID                string
	Sequence          int64
	VCenterID         string
	Name              string
	Cluster           string
	Datacenter        string
	DiskSize          int64
	Memory            int64
	VCenterState      string
	IssueCount        int
	Migratable        *bool
	MigrationExcluded *bool
	Template          *bool
	CollectedAt       time.Time
}

type VirtualMachineStore struct {
	db *sql.DB
}

func (s *VirtualMachineStore) SaveBatch(ctx context.Context, sequence int64, vcenterID string, vms []agentv1.VirtualMachine) error {
	if len(vms) == 0 {
		return nil
	}

	insert := sq.Insert("virtual_machines").
		Columns("id", "sequence", "vcenter_id", "name", "cluster", "datacenter",
			"disk_size", "memory", "vcenter_state", "issue_count",
			"migratable", "migration_excluded", "template", "collected_at")

	now := time.Now()
	for _, vm := range vms {
		insert = insert.Values(
			vm.Id, sequence, vcenterID, vm.Name, vm.Cluster, vm.Datacenter,
			vm.DiskSize, vm.Memory, vm.VCenterState, vm.IssueCount,
			vm.Migratable, vm.MigrationExcluded, vm.Template, now,
		)
	}

	query, args, err := insert.ToSql()
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *VirtualMachineStore) List(ctx context.Context) ([]VirtualMachineRecord, error) {
	query, args, err := sq.Select("id", "sequence", "vcenter_id", "name", "cluster", "datacenter",
		"disk_size", "memory", "vcenter_state", "issue_count",
		"migratable", "migration_excluded", "template", "collected_at").
		From("virtual_machines").
		OrderBy("name").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []VirtualMachineRecord
	for rows.Next() {
		var r VirtualMachineRecord
		if err := rows.Scan(&r.ID, &r.Sequence, &r.VCenterID, &r.Name, &r.Cluster, &r.Datacenter,
			&r.DiskSize, &r.Memory, &r.VCenterState, &r.IssueCount,
			&r.Migratable, &r.MigrationExcluded, &r.Template, &r.CollectedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
