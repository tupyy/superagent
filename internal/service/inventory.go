package service

import (
	"context"
	"encoding/json"

	"github.com/tupyy/superagent/internal/store"
	"go.uber.org/zap"
)

type InventoryService struct {
	store *store.Store
}

func NewInventoryService(s *store.Store) *InventoryService {
	return &InventoryService{store: s}
}

func (s *InventoryService) Save(ctx context.Context, sequence int64, id string, vcenterURL string, data json.RawMessage) error {
	zap.S().Infow("saving inventory", "id", id, "sequence", sequence, "vcenter", vcenterURL)
	return s.store.Inventory().Save(ctx, sequence, id, vcenterURL, data)
}

func (s *InventoryService) List(ctx context.Context) ([]store.InventoryRecord, error) {
	return s.store.Inventory().List(ctx)
}
