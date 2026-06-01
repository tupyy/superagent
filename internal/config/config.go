package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/tupyy/superagent/internal/models"
)

type Configuration struct {
	Agent  Agent  `debugmap:"visible"`
	Podman Podman `debugmap:"visible"`
	Store  Store  `debugmap:"visible"`

	Credentials []models.Credentials

	LogFormat string `debugmap:"visible"`
	LogLevel  string `debugmap:"visible"`
}

type Agent struct {
	VCenters           []string `debugmap:"visible"`
	Image              string   `debugmap:"visible"`
	RandomizeVCenterID bool     `debugmap:"visible"`
}

type Podman struct {
	Socket string `debugmap:"visible"`
}

type Store struct {
	DBPath string `debugmap:"visible" default:"superagent.duckdb"`
}

func DefaultPodmanSocket() string {
	return fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Getuid())
}

func parseVCenterURL(raw string) (models.Credentials, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return models.Credentials{}, fmt.Errorf("parsing vcenter URL %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return models.Credentials{}, fmt.Errorf("vcenter URL %q must have scheme and host", raw)
	}
	password, _ := u.User.Password()
	return models.Credentials{
		URL:      fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path),
		Username: u.User.Username(),
		Password: password,
	}, nil
}

func (c *Configuration) Validate() error {
	if len(c.Agent.VCenters) == 0 {
		return fmt.Errorf("at least one --vcenter is required")
	}
	if c.Agent.Image == "" {
		return fmt.Errorf("--image is required")
	}

	c.Credentials = make([]models.Credentials, 0, len(c.Agent.VCenters))
	for _, raw := range c.Agent.VCenters {
		creds, err := parseVCenterURL(raw)
		if err != nil {
			return err
		}
		c.Credentials = append(c.Credentials, creds)
	}

	return nil
}
