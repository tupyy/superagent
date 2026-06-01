package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type InventoryRecord struct {
	ID          string
	Sequence    int64
	VCenterURL  string
	Data        json.RawMessage
	CollectedAt time.Time
}

type InventoryStore struct {
	db *sql.DB
}

func (s *InventoryStore) Save(ctx context.Context, sequence int64, id string, vcenterURL string, data json.RawMessage) error {
	query, args, err := sq.Insert("inventory").
		Columns("id", "sequence", "vcenter_url", "data", "collected_at").
		Values(id, sequence, vcenterURL, string(data), time.Now()).
		ToSql()
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *InventoryStore) List(ctx context.Context) ([]InventoryRecord, error) {
	query, args, err := sq.Select("id", "sequence", "vcenter_url", "data", "collected_at").
		From("inventory").
		OrderBy("vcenter_url").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []InventoryRecord
	for rows.Next() {
		var r InventoryRecord
		if err := rows.Scan(&r.ID, &r.Sequence, &r.VCenterURL, &r.Data, &r.CollectedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
