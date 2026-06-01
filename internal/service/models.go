package service

import "github.com/tupyy/superagent/internal/models"

type CollectorState string

const (
	CollectorStateStarting    CollectorState = "starting"
	CollectorStateWaiting     CollectorState = "waiting"
	CollectorStateCollecting  CollectorState = "collecting"
	CollectorStatePolling     CollectorState = "polling"
	CollectorStateHarvesting  CollectorState = "harvesting"
	CollectorStateFetchingVMs CollectorState = "fetching_vms"
	CollectorStateDone        CollectorState = "done"
	CollectorStateError       CollectorState = "error"
)

type CollectorStatus struct {
	State CollectorState
	Error error
}

type collectorAgent struct {
	Service   *AgentService
	Creds     models.Credentials
	VCenterID string
	Success   bool
}

type CollectorResult struct {
	Agents   map[string]*collectorAgent
	Sequence int64
}
