# Superagent

Orchestrates multiple [assisted-migration-agent](https://github.com/kubev2v/assisted-migration-agent) containers via Podman to collect vSphere inventory and virtual machine data from one or more vCenter servers. Results are stored in a local DuckDB database.

## Build

```bash
go build -tags "exclude_graphdriver_btrfs containers_image_openpgp" -o bin/superagent .
```

## Run

### Basic Usage

```bash
bin/superagent run \
  --image localhost/assisted-migration-agent:latest \
  --vcenter https://user:password@vcenter-host/sdk
```

### Multiple vCenters

Repeat the `--vcenter` flag for each vCenter server:

```bash
bin/superagent run \
  --image localhost/assisted-migration-agent:latest \
  --vcenter https://user1:pass1@vcenter-1.example.com/sdk \
  --vcenter https://user2:pass2@vcenter-2.example.com/sdk
```

Credentials with special characters must be percent-encoded (e.g. `@` becomes `%40`, `!` becomes `%21`).

### Randomize vCenter IDs

By default, the vCenter ID stored in the database comes from the inventory data. Use `--randomize-vcenter-id` to generate a random UUID instead:

```bash
bin/superagent run \
  --image localhost/assisted-migration-agent:latest \
  --vcenter https://user:password@vcenter-host/sdk \
  --randomize-vcenter-id
```

## Command Line Flags

### Agent

| Flag | Default | Description |
|------|---------|-------------|
| `--vcenter` | *required* | vCenter URL with credentials (repeatable) |
| `--image` | *required* | Agent container image |
| `--randomize-vcenter-id` | `false` | Use a random UUID as vCenter ID |

### Podman

| Flag | Default | Description |
|------|---------|-------------|
| `--podman-socket` | `unix:///run/user/<uid>/podman/podman.sock` | Podman socket path |

### Store

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | `superagent.duckdb` | Path to DuckDB database file |

### Logging

| Flag | Default | Description |
|------|---------|-------------|
| `--log-level` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `--log-format` | `console` | `console` \| `json` |

## Environment Variables

All flags can be set via environment variables with the `SUPERAGENT_` prefix. Dashes become underscores:

```bash
export SUPERAGENT_IMAGE=localhost/assisted-migration-agent:latest
export SUPERAGENT_PODMAN_SOCKET=unix:///run/user/1000/podman/podman.sock
export SUPERAGENT_DB=superagent.duckdb
export SUPERAGENT_LOG_LEVEL=debug
```

## How It Works

1. For each `--vcenter`, a containerized agent is started via Podman with port mapping starting at 18000.
2. Superagent waits for all agents to become ready (up to 2 minutes).
3. Each agent is instructed to collect inventory from its vCenter.
4. Superagent polls agents until collection completes (up to 10 minutes).
5. Inventory data is fetched from each successful agent and stored in DuckDB.
6. Virtual machine data is fetched (paginated) and stored in DuckDB.
7. All agent containers are stopped and removed.

## Database Schema

Results are stored in two tables. Each run gets an incrementing sequence number.

### inventory

| Column | Type | Description |
|--------|------|-------------|
| `id` | `VARCHAR` | vCenter ID (primary key with sequence) |
| `sequence` | `INTEGER` | Run sequence number |
| `vcenter_url` | `VARCHAR` | vCenter server URL |
| `data` | `JSON` | Raw inventory data |
| `collected_at` | `TIMESTAMP` | Collection timestamp |

### virtual_machines

| Column | Type | Description |
|--------|------|-------------|
| `id` | `VARCHAR` | VM identifier (primary key with sequence) |
| `sequence` | `INTEGER` | Run sequence number |
| `vcenter_id` | `VARCHAR` | vCenter ID |
| `name` | `VARCHAR` | VM name |
| `cluster` | `VARCHAR` | Cluster name |
| `datacenter` | `VARCHAR` | Datacenter name |
| `disk_size` | `BIGINT` | Disk size in bytes |
| `memory` | `BIGINT` | Memory in bytes |
| `vcenter_state` | `VARCHAR` | VM power state |
| `issue_count` | `INTEGER` | Number of migration issues |
| `migratable` | `BOOLEAN` | Whether the VM is migratable |
| `migration_excluded` | `BOOLEAN` | Whether the VM is excluded from migration |
| `template` | `BOOLEAN` | Whether the VM is a template |
| `collected_at` | `TIMESTAMP` | Collection timestamp |

## Prerequisites

- Podman (rootless, with socket enabled)
- The assisted-migration-agent container image
