package store

import (
	"context"
	"database/sql"
)

type Store struct {
	db              *sql.DB
	inventory       *InventoryStore
	virtualMachines *VirtualMachineStore
}

func New(db *sql.DB) *Store {
	return &Store{
		db:              db,
		inventory:       &InventoryStore{db: db},
		virtualMachines: &VirtualMachineStore{db: db},
	}
}

func (s *Store) Inventory() *InventoryStore {
	return s.inventory
}

func (s *Store) VirtualMachines() *VirtualMachineStore {
	return s.virtualMachines
}

func (s *Store) NextSequenceID(ctx context.Context) (int64, error) {
	var seq int64
	err := s.db.QueryRowContext(ctx, "SELECT nextval('collect_sequence')").Scan(&seq)
	return seq, err
}

func (s *Store) Close() error {
	return s.db.Close()
}
