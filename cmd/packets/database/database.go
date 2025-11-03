package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

const (
	CreateInstructions = `CREATE TABLE installed_packges(
    name TEXT NOT NULL,
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    serial INTEGER NOT NULL,
    maintainer TEXT NOT NULL,
    verified INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    upload_time TEXT NOT NULL,
    installed_time TEXT NOT NULL,

    image BLOB,

    UNIQUE(name, signature),
    UNIQUE(name, version),
    UNIQUE(name, serial)
)

CREATE TABLE package_files(
    package_id TEXT PRIMARY KEY,
    filepath TEXT NOT NULL,
    is_dir INTEGER NOT NULL DEFAULT 0,

    UNIQUE(package_id, filepath)
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

CREATE TABLE package_flags(
    package_id TEXT NOT NULL,
    flag TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
)
`
)

type DatabaseOptions struct {
	// Add any additional options here
}

func MarkAsInstalled(pkg packet.PacketLua, db *sql.DB, image *[]byte) error {

	if image != nil {
		_, err := db.Exec("INSERT INTO installed_packages (name, id, version, installed_time, image) VALUES (?, ?, ?, ?, ?, ?, ?)", pkg.Name, pkg.Name+"@"+pkg.Version, pkg.Version, time.Now().UnixMilli(), image)
		if err != nil {
			return err
		}
	} else {
		_, err := db.Exec("INSERT INTO installed_packages (name, id, version, installed_time, image) VALUES (?, ?, ?, ?, ?, ?, ?)", pkg.Name, pkg.Name+"@"+pkg.Version, pkg.Version, time.Now().UnixMilli(), []byte{1})
		if err != nil {
			return err
		}
	}
	return nil
}

func MarkAsUninstalled(id string, db *sql.DB) error {
	_, err := db.Exec("DELETE FROM installed_packages WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func PrepareDataBase(db *sql.DB) { _, _ = db.Exec(CreateInstructions) }
