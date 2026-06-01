CREATE SEQUENCE IF NOT EXISTS collect_sequence START 1;

CREATE TABLE IF NOT EXISTS inventory (
    id VARCHAR(100) NOT NULL,
    sequence INTEGER NOT NULL,
    vcenter_url VARCHAR NOT NULL,
    data JSON NOT NULL,
    collected_at TIMESTAMP DEFAULT current_timestamp,
    PRIMARY KEY (id, sequence)
);

CREATE TABLE IF NOT EXISTS virtual_machines (
    id VARCHAR(100) NOT NULL,
    sequence INTEGER NOT NULL,
    vcenter_id VARCHAR(100) NOT NULL,
    name VARCHAR NOT NULL,
    cluster VARCHAR NOT NULL,
    datacenter VARCHAR NOT NULL,
    disk_size BIGINT NOT NULL,
    memory BIGINT NOT NULL,
    vcenter_state VARCHAR NOT NULL,
    issue_count INTEGER NOT NULL DEFAULT 0,
    migratable BOOLEAN,
    migration_excluded BOOLEAN,
    template BOOLEAN,
    collected_at TIMESTAMP DEFAULT current_timestamp,
);
