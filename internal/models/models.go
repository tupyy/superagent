package models

type Credentials struct {
	URL      string
	Username string
	Password string
}

type AgentInfo struct {
	ContainerID string
	HostPort    int
}

type VirtualMachine struct {
	ID                    string
	Name                  string
	PowerState            string
	Cluster               string
	Datacenter            string
	Memory                int32 // MB
	DiskSize              int64 // MB (stored as MiB in DB, treated as MB)
	IssueCount            int
	IsMigratable          bool
	IsTemplate            bool
	Groups                []string
	UtilizationCpuP95     *float64 // CPU utilization at p95 (%); nil when no utilization data
	UtilizationMemP95     *float64 // Memory utilization at p95 (%); nil when no utilization data
	UtilizationDisk       *float64 // Disk utilization (%); nil when no utilization data
	UtilizationConfidence *float64 // Data confidence (%); nil when no utilization data
	MigrationExcluded     bool
	Labels                []string
}
