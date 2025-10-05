package consts

import "time"

const (
	DefaultLinux_d  = "/etc/packets"
	DefaultCache_d  = "/var/cache/packets"
	DefaultHttpPort = 9123
	DefaultBin_d    = "/usr/local/bin"
	DefaultData_d   = "/opt/packets"
	LANDeadline     = 2 * time.Second
	IndexDB         = "/etc/packets/index.db"
	InstalledDB     = "/etc/packets/installed.db"
	DefaultSyncUrl  = "https://servidordomal.fun/index.db"
)

const InstalledDatabaseSchema = `CREATE TABLE IF NOT EXISTS packages (
    query_name      TEXT NOT NULL UNIQUE PRIMARY KEY,
    id              TEXT NOT NULL UNIQUE, 
    version         TEXT NOT NULL, 
    dependencies    TEXT NOT NULL DEFAULT '', 
    description     TEXT NOT NULL,
    package_d       TEXT NOT NULL,
    filename        TEXT NOT NULL,
    os              TEXT NOT NULL,
    arch            TEXT NOT NULL,
    in_cache        INTEGER NOT NULL DEFAULT 1,
    serial          INTEGER NOT NULL,

    UNIQUE(query_name, version),
    UNIQUE(query_name, serial)
);

CREATE TABLE package_dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);

CREATE INDEX index_dependency_name ON package_dependencies(dependency_name);
`
