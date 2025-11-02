CREATE TABLE packges(
    name TEXT NOT NULL,
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    serial INTEGER NOT NULL,
    maintainer TEXT NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    upload_time TEXT NOT NULL,


    UNIQUE(name, signature),
    UNIQUE(name, version),
    UNIQUE(name, serial)
)

CREATE TABLE dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
)


CREATE TABLE build_dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
)


CREATE TABLE conflicts(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
)
