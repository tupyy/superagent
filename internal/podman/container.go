package podman

type ContainerConfig struct {
	name       string
	image      string
	cmd        []string
	ports      map[int]int
	envVars    map[string]string
	volumes    map[string]string
	bindMounts map[string]string
}

func NewContainerConfig(name, image string) *ContainerConfig {
	return &ContainerConfig{
		name:       name,
		image:      image,
		ports:      make(map[int]int),
		envVars:    make(map[string]string),
		volumes:    make(map[string]string),
		bindMounts: make(map[string]string),
	}
}

func (c *ContainerConfig) WithPort(hostPort, containerPort int) *ContainerConfig {
	c.ports[hostPort] = containerPort
	return c
}

func (c *ContainerConfig) WithEnvVar(key, value string) *ContainerConfig {
	c.envVars[key] = value
	return c
}

func (c *ContainerConfig) WithEnvVars(envVars map[string]string) *ContainerConfig {
	for k, v := range envVars {
		c.envVars[k] = v
	}
	return c
}

func (c *ContainerConfig) WithVolume(volumeName, containerPath string) *ContainerConfig {
	c.volumes[volumeName] = containerPath
	return c
}

func (c *ContainerConfig) WithBindMount(hostPath, containerPath string) *ContainerConfig {
	c.bindMounts[hostPath] = containerPath
	return c
}

func (c *ContainerConfig) WithCmd(cmd ...string) *ContainerConfig {
	c.cmd = cmd
	return c
}
