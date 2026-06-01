package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	agentv1 "github.com/kubev2v/assisted-migration-agent/api/v1"
	"github.com/tupyy/superagent/internal/models"
	"github.com/tupyy/superagent/internal/podman"
	"go.uber.org/zap"
)

const agentContainerPort = 8000

type AgentService struct {
	runner   *podman.PodmanRunner
	image    string
	name     string
	hostPort int
	id       string
	client   *http.Client
}

func NewAgentService(runner *podman.PodmanRunner, image string, hostPort int) *AgentService {
	return &AgentService{
		runner:   runner,
		image:    image,
		name:     containerName(),
		hostPort: hostPort,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (s *AgentService) Name() string {
	return s.name
}

func (s *AgentService) Start() (models.AgentInfo, error) {
	if s.id != "" {
		return models.AgentInfo{ContainerID: s.id, HostPort: s.hostPort}, nil
	}

	cfg := podman.NewContainerConfig(s.name, s.image).
		WithPort(s.hostPort, agentContainerPort).
		WithEnvVar("AGENT_AGENT_ID", uuid.NewString()).
		WithEnvVar("AGENT_SOURCE_ID", uuid.NewString()).
		WithEnvVar("AGENT_MODE", "disconnected").
		WithEnvVar("AGENT_DATA_FOLDER", "/var/lib/agent").
		WithEnvVar("AGENT_SERVER_MODE", "prod").
		WithEnvVar("AGENT_SERVER_STATICS_FOLDER", "/app/static").
		WithEnvVar("AGENT_OPA_POLICIES_FOLDER", "/app/policies")

	id, err := s.runner.StartContainer(cfg)
	if err != nil {
		return models.AgentInfo{}, fmt.Errorf("starting container %s: %w", s.name, err)
	}

	s.id = id
	zap.S().Infow("started agent", "container", s.name, "id", id, "port", s.hostPort)
	return models.AgentInfo{ContainerID: id, HostPort: s.hostPort}, nil
}

func (s *AgentService) Stop() error {
	if s.id == "" {
		return nil
	}
	_ = s.runner.StopContainer(s.id)
	_ = s.runner.RemoveContainer(s.id)
	zap.S().Infow("stopped agent", "container", s.name)
	s.id = ""
	return nil
}

type AgentStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (s *AgentService) GetStatus(ctx context.Context) (AgentStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint("/api/v1/collector"), nil)
	if err != nil {
		return AgentStatus{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return AgentStatus{}, err
	}
	defer resp.Body.Close()

	var status AgentStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return AgentStatus{}, fmt.Errorf("decoding status: %w", err)
	}
	return status, nil
}

func (s *AgentService) Collect(ctx context.Context, creds models.Credentials) error {
	body, err := json.Marshal(map[string]string{
		"url":      creds.URL,
		"username": creds.Username,
		"password": creds.Password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint("/api/v1/collector"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("collect returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *AgentService) Inventory(ctx context.Context) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint("/api/v1/inventory"), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading inventory: %w", err)
	}
	return json.RawMessage(data), nil
}

func (s *AgentService) ListVirtualMachines(ctx context.Context) ([]agentv1.VirtualMachine, error) {
	var all []agentv1.VirtualMachine
	page := 1

	for {
		url := fmt.Sprintf("%s?pageSize=100&page=%d", s.endpoint("/api/v1/vms"), page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading vms response: %w", err)
		}

		var listResp agentv1.VirtualMachineListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("decoding vms response: %w", err)
		}

		all = append(all, listResp.Vms...)

		if page >= listResp.PageCount {
			break
		}
		page++
	}

	return all, nil
}

func (s *AgentService) endpoint(path string) string {
	return fmt.Sprintf("https://localhost:%d%s", s.hostPort, path)
}

func containerName() string {
	return fmt.Sprintf("superagent-%s", uuid.New().String()[:8])
}
