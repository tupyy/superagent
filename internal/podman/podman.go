package podman

import (
	"context"
	"fmt"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/specgen"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	nettypes "go.podman.io/common/libnetwork/types"
)

type PodmanRunner struct {
	conn context.Context
}

func NewPodmanRunner(socket string) (*PodmanRunner, error) {
	conn, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to podman: %w", err)
	}
	return &PodmanRunner{conn: conn}, nil
}

func (p *PodmanRunner) StartContainer(cfg *ContainerConfig) (string, error) {
	s := specgen.NewSpecGenerator(cfg.image, false)
	s.Name = cfg.name
	s.Command = cfg.cmd
	s.Env = cfg.envVars

	if len(cfg.ports) > 0 {
		s.PortMappings = make([]nettypes.PortMapping, 0, len(cfg.ports))
		for hostPort, containerPort := range cfg.ports {
			s.PortMappings = append(s.PortMappings, nettypes.PortMapping{
				HostPort:      uint16(hostPort),
				ContainerPort: uint16(containerPort),
				Protocol:      "tcp",
			})
		}
	}

	if len(cfg.volumes) > 0 {
		s.Volumes = make([]*specgen.NamedVolume, 0, len(cfg.volumes))
		for volumeName, containerPath := range cfg.volumes {
			s.Volumes = append(s.Volumes, &specgen.NamedVolume{
				Name: volumeName,
				Dest: containerPath,
			})
		}
	}

	if len(cfg.bindMounts) > 0 {
		s.Mounts = make([]specs.Mount, 0, len(cfg.bindMounts))
		for hostPath, containerPath := range cfg.bindMounts {
			s.Mounts = append(s.Mounts, specs.Mount{
				Type:        "bind",
				Source:      hostPath,
				Destination: containerPath,
				Options:     []string{"rbind", "ro"},
			})
		}
	}

	createResponse, err := containers.CreateWithSpec(p.conn, s, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := containers.Start(p.conn, createResponse.ID, nil); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return createResponse.ID, nil
}

func (p *PodmanRunner) StopContainer(id string) error {
	if err := containers.Stop(p.conn, id, nil); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

func (p *PodmanRunner) RemoveContainer(id string) error {
	_, err := containers.Remove(p.conn, id, nil)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

func (p *PodmanRunner) IsRunning(id string) (bool, error) {
	data, err := containers.Inspect(p.conn, id, nil)
	if err != nil {
		return false, fmt.Errorf("failed to inspect container: %w", err)
	}
	return data.State.Running, nil
}

func (p *PodmanRunner) WaitForRunning(id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := p.IsRunning(id)
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("container %s did not start within %v", id, timeout)
}

func (p *PodmanRunner) Logs(id string) (string, error) {
	var stdout, stderr []string
	stdoutChan := make(chan string)
	stderrChan := make(chan string)

	go func() {
		for line := range stdoutChan {
			stdout = append(stdout, line)
		}
	}()
	go func() {
		for line := range stderrChan {
			stderr = append(stderr, line)
		}
	}()

	opts := &containers.LogOptions{}
	if err := containers.Logs(p.conn, id, opts, stdoutChan, stderrChan); err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	return fmt.Sprintf("stdout: %v\nstderr: %v", stdout, stderr), nil
}
