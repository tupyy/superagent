package service

import (
	"context"

	agentv1 "github.com/kubev2v/assisted-migration-agent/api/v1"
	"github.com/tupyy/superagent/internal/store"
	"go.uber.org/zap"
)

type VirtualMachineService struct {
	store *store.Store
}

func NewVirtualMachineService(s *store.Store) *VirtualMachineService {
	return &VirtualMachineService{store: s}
}

func (s *VirtualMachineService) SaveBatch(ctx context.Context, sequence int64, vcenterID string, vms []agentv1.VirtualMachine) error {
	zap.S().Infow("saving virtual machines", "vcenter_id", vcenterID, "sequence", sequence, "count", len(vms))
	return s.store.VirtualMachines().SaveBatch(ctx, sequence, vcenterID, vms)
}
