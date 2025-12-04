CREATE TABLE installed_packages(
    name TEXT NOT NULL UNIQUE,
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    serial INTEGER NOT NULL,
    maintainer TEXT NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    upload_time INTEGER NOT NULL,
    installed_time INTEGER NOT NULL,

    location TEXT NOT NULL,

    image BLOB,

    UNIQUE(name, version),
    UNIQUE(name, serial)
);

CREATE TABLE package_files(
    package_id TEXT PRIMARY KEY,
    filepath TEXT NOT NULL,
    is_dir INTEGER NOT NULL DEFAULT 0,

    UNIQUE(package_id, filepath)
);

CREATE TABLE dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);


CREATE TABLE build_dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);


CREATE TABLE conflicts(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);

CREATE TABLE package_flags(
    package_id TEXT NOT NULL,
    flag TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL
);

CREATE TABLE build_packages(
    name TEXT NOT NULL,
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    serial INTEGER NOT NULL,
    maintainer TEXT NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    upload_time INTEGER NOT NULL,
    installed_time INTEGER NOT NULL,

    location TEXT NOT NULL,

    filepath TEXT NOT NULL,

    UNIQUE(name, signature),
    UNIQUE(name, version),
    UNIQUE(name, serial)
);
