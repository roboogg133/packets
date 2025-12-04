CREATE TABLE packages(
    name TEXT NOT NULL,
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    serial INTEGER NOT NULL,
    maintainer TEXT NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    upload_time INTEGER NOT NULL,

    location TEXT NOT NULL,
    available_compiled INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (location, id)
    UNIQUE(name, version, location),
    UNIQUE(name, serial, location)
);

CREATE TABLE dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,
    location TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name, location)
);

CREATE TABLE build_dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,
    location TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name, location)
);

CREATE TABLE conflicts(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,
    location TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name, location)
);
