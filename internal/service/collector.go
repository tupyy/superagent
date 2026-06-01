package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kubev2v/assisted-migration-agent/pkg/work"
	v1alpha1 "github.com/kubev2v/migration-planner/api/v1alpha1"
	"github.com/tupyy/superagent/internal/models"
	"github.com/tupyy/superagent/internal/podman"
	"github.com/tupyy/superagent/internal/store"
	"go.uber.org/zap"
)

const (
	baseHostPort           = 18000
	agentReadyTimeout      = 2 * time.Minute
	agentCollectTimeout    = 10 * time.Minute
	agentPollInterval      = 2 * time.Second
	agentReadyPollInterval = 500 * time.Millisecond
)

type CollectorService struct {
	runner             *podman.PodmanRunner
	image              string
	credentials        []models.Credentials
	inventorySvc       *InventoryService
	virtualMachineSvc  *VirtualMachineService
	store              *store.Store
	randomizeVCenterID bool
	pool               *work.Pool2[CollectorStatus, CollectorResult]
}

func NewCollectorService(runner *podman.PodmanRunner, image string, creds []models.Credentials, inventorySvc *InventoryService, virtualMachineSvc *VirtualMachineService, s *store.Store, randomizeVCenterID bool) *CollectorService {
	return &CollectorService{
		runner:             runner,
		image:              image,
		credentials:        creds,
		inventorySvc:       inventorySvc,
		virtualMachineSvc:  virtualMachineSvc,
		store:              s,
		randomizeVCenterID: randomizeVCenterID,
	}
}

func (c *CollectorService) Start() error {
	builder := c.buildPipeline()
	builders := map[string]work.WorkBuilder2[CollectorStatus, CollectorResult]{
		"collector": builder,
	}
	c.pool = work.NewPool2(builders)
	return c.pool.Start()
}

func (c *CollectorService) Stop() error {
	if c.pool != nil {
		return c.pool.Stop()
	}
	return nil
}

func (c *CollectorService) IsRunning() bool {
	if c.pool == nil {
		return false
	}
	return c.pool.IsRunning()
}

func (c *CollectorService) State() CollectorStatus {
	if c.pool == nil {
		return CollectorStatus{State: CollectorStateDone}
	}
	st, err := c.pool.State("collector")
	if err != nil {
		return CollectorStatus{State: CollectorStateError, Error: err}
	}
	if st.Err != nil {
		return CollectorStatus{State: CollectorStateError, Error: st.Err}
	}
	return st.State
}

func (c *CollectorService) buildPipeline() *collectorBuilder {
	var units []work.WorkUnit[CollectorStatus, CollectorResult]

	for i, creds := range c.credentials {
		creds := creds
		port := baseHostPort + i
		units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
			Status: func() CollectorStatus {
				return CollectorStatus{State: CollectorState(fmt.Sprintf("starting agent %d", i+1))}
			},
			Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
				if result.Agents == nil {
					result.Agents = make(map[string]*collectorAgent)
				}
				svc := NewAgentService(c.runner, c.image, port)
				if _, err := svc.Start(); err != nil {
					return result, fmt.Errorf("starting agent for %s: %w", creds.URL, err)
				}
				result.Agents[svc.name] = &collectorAgent{
					Service: svc,
					Creds:   creds,
				}
				return result, nil
			},
		})
	}

	units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
		Status: func() CollectorStatus {
			return CollectorStatus{State: CollectorStateWaiting}
		},
		Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
			ctx, cancel := context.WithTimeout(ctx, agentReadyTimeout)
			defer cancel()

			timer := time.NewTicker(2 * time.Second)
			defer timer.Stop()

			for {
				allReady := true
				for _, agent := range result.Agents {
					_, err := agent.Service.GetStatus(ctx)
					if err != nil {
						allReady = false
					}
				}
				if allReady {
					zap.S().Info("all agents ready")
					break
				}

				select {
				case <-ctx.Done():
					return result, errors.New("timeout expired. One of the agent is not ready")
				case <-timer.C:
				}
			}

			return result, nil
		},
	})

	units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
		Status: func() CollectorStatus {
			return CollectorStatus{State: CollectorStateCollecting}
		},
		Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
			for _, agent := range result.Agents {
				if err := agent.Service.Collect(ctx, agent.Creds); err != nil {
					return result, fmt.Errorf("starting collection on %s: %w", agent.Creds.URL, err)
				}
				zap.S().Infow("collection started", "vcenter", agent.Creds.URL, "container", agent.Service.Name())
			}
			return result, nil
		},
	})

	units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
		Status: func() CollectorStatus {
			return CollectorStatus{State: CollectorStatePolling}
		},
		Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
			ctx, cancel := context.WithTimeout(ctx, agentCollectTimeout)
			defer cancel()

			pending := make(map[string]bool)
			for key := range result.Agents {
				pending[key] = true
			}

			timer := time.NewTicker(agentPollInterval)
			defer timer.Stop()

			countSuccess := len(result.Agents)
			for {
				for key := range pending {
					agent := result.Agents[key]
					agentStatus, err := agent.Service.GetStatus(ctx)
					if err != nil {
						zap.S().Debugw("polling agent unreachable", "container", agent.Service.Name(), "error", err)
						continue
					}
					switch agentStatus.Status {
					case "collected":
						agent.Success = true
						delete(pending, key)
						zap.S().Infow("collection completed", "vcenter", agent.Creds.URL, "container", agent.Service.Name())
					case "error":
						agent.Success = false
						delete(pending, key)
						zap.S().Errorw("collection failed", "vcenter", agent.Creds.URL, "container", agent.Service.Name(), "agent_error", agentStatus.Error)
						countSuccess--
					default:
						zap.S().Debugw("polling agent", "vcenter", agent.Creds.URL, "container", agent.Service.Name(), "status", agentStatus.Status, "pending", len(pending))
					}
				}

				if len(pending) == 0 {
					break
				}

				select {
				case <-ctx.Done():
					for key := range pending {
						result.Agents[key].Success = false
					}
					return result, errors.New("timeout expired. Some agents did not finish collecting")
				case <-timer.C:
				}
			}

			if countSuccess == 0 {
				return result, errors.New("all agents failed to collect data")
			}

			return result, nil
		},
	})

	units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
		Status: func() CollectorStatus {
			return CollectorStatus{State: CollectorStateHarvesting}
		},
		Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
			seq, err := c.store.NextSequenceID(ctx)
			if err != nil {
				return result, fmt.Errorf("getting sequence ID: %w", err)
			}
			result.Sequence = seq

			for vcURL, agent := range result.Agents {
				if !agent.Success {
					continue
				}
				data, err := agent.Service.Inventory(ctx)
				if err != nil {
					zap.S().Warnw("failed to fetch inventory", "vcenter", vcURL, "container", agent.Service.Name(), "error", err)
					continue
				}

				var inv v1alpha1.Inventory
				if err := json.Unmarshal(data, &inv); err != nil {
					zap.S().Warnw("failed to unmarshal inventory", "vcenter", vcURL, "error", err)
					continue
				}

				vcenterID := inv.VcenterId
				if c.randomizeVCenterID {
					vcenterID = uuid.New().String()
				}
				agent.VCenterID = vcenterID

				if err := c.inventorySvc.Save(ctx, seq, vcenterID, agent.Creds.URL, data); err != nil {
					zap.S().Warnw("failed to save inventory", "vcenter", vcURL, "error", err)
					continue
				}
				zap.S().Infow("inventory harvested", "vcenter", vcURL, "id", vcenterID, "sequence", seq)
			}
			return result, nil
		},
	})

	units = append(units, work.WorkUnit[CollectorStatus, CollectorResult]{
		Status: func() CollectorStatus {
			return CollectorStatus{State: CollectorStateFetchingVMs}
		},
		Work: func(ctx context.Context, result CollectorResult) (CollectorResult, error) {
			for vcURL, agent := range result.Agents {
				if !agent.Success {
					continue
				}
				vms, err := agent.Service.ListVirtualMachines(ctx)
				if err != nil {
					zap.S().Warnw("failed to list virtual machines", "vcenter", vcURL, "container", agent.Service.Name(), "error", err)
					continue
				}
				if err := c.virtualMachineSvc.SaveBatch(ctx, result.Sequence, agent.VCenterID, vms); err != nil {
					zap.S().Warnw("failed to save virtual machines", "vcenter", vcURL, "container", agent.Service.Name(), "error", err)
					continue
				}
				zap.S().Infow("virtual machines saved", "vcenter", vcURL, "container", agent.Service.Name(), "count", len(vms), "sequence", result.Sequence)
			}
			return result, nil
		},
	})

	return &collectorBuilder{
		inner: work.NewSliceWorkBuilder(units),
	}
}

type collectorBuilder struct {
	inner *work.SliceWorkBuilder[CollectorStatus, CollectorResult]
}

func (b *collectorBuilder) Next() (work.WorkUnit[CollectorStatus, CollectorResult], bool) {
	return b.inner.Next()
}

func (b *collectorBuilder) Finalize(ctx context.Context, result CollectorResult) error {
	for _, agent := range result.Agents {
		agent.Service.Stop()
	}
	return nil
}
