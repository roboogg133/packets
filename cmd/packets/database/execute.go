package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

const (
	CreateInstructions = `CREATE TABLE IF NOT EXISTS installed_packages(
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

CREATE TABLE IF NOT EXISTS package_files(
    package_id TEXT PRIMARY KEY,
    filepath TEXT NOT NULL,
    is_dir INTEGER NOT NULL DEFAULT 0,

    UNIQUE(package_id, filepath)
);

CREATE TABLE IF NOT EXISTS dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);


CREATE TABLE IF NOT EXISTS build_dependencies(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);


CREATE TABLE IF NOT EXISTS conflicts(
    package_id TEXT NOT NULL,
    dependency_name TEXT NOT NULL,
    version_constraint TEXT NOT NULL,

    PRIMARY KEY (package_id, dependency_name)
);

CREATE TABLE IF NOT EXISTS package_flags(
    package_id TEXT NOT NULL,
    flag TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS build_packages(
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

    UNIQUE(name, version),
    UNIQUE(name, serial)
);
`
)

type DatabaseOptions struct {
	// Add any additional options here
}

func MarkAsInstalled(pkg packet.PacketLua, files []packet.InstallInstruction, flags []packet.Flag, db *sql.DB, image []byte, upload_time int64) error {

	switch {
	case upload_time == 0:
		upload_time = time.Now().UnixMilli()
	case image == nil:
		image = []byte{1}
	}

	_, err := db.Exec("INSERT INTO installed_packages (name, id, version, installed_time, image, serial, maintainer, description, upload_time) VALUES (?, ?, ?, ?, ?, ?,?,?,?)", pkg.Name, pkg.Name+"@"+pkg.Version, pkg.Version, time.Now().Unix(), image, pkg.Serial, pkg.Maintainer, pkg.Description, time.Now().Unix())
	if err != nil {
		db.Exec("DELETE FROM installed_packages WHERE id = ?", pkg.Name+"@"+pkg.Version)
		return err
	}

	for _, v := range files {
		_, err = db.Exec("INSERT INTO package_files (package_id, path, is_dir) VALUES (?, ?, ?)", pkg.Name+"@"+pkg.Version, v.Destination, v.IsDir)
		if err != nil {
			db.Exec("DELETE FROM installed_packages WHERE id = ?", pkg.Name+"@"+pkg.Version)
			db.Exec("DELETE FROM package_files WHERE package_id = ?", pkg.Name+"@"+pkg.Version)
			return err
		}
	}

	for _, v := range flags {
		_, err = db.Exec("INSERT INTO package_flags (package_id, flag, name, path) VALUES (?, ?, ?, ?)", pkg.Name+"@"+pkg.Version, v.FlagType, v.Name, v.Path)
		if err != nil {
			db.Exec("DELETE FROM installed_packages WHERE id = ?", pkg.Name+"@"+pkg.Version)
			db.Exec("DELETE FROM package_files WHERE package_id = ?", pkg.Name+"@"+pkg.Version)
			db.Exec("DELETE FROM package_flags WHERE package_id = ?", pkg.Name+"@"+pkg.Version)

			return err
		}
	}

	return nil
}

func MarkAsUninstalled(id packet.PackageID, db *sql.DB) error {
	if _, err := db.Exec("DELETE FROM installed_packages WHERE id = ?", string(id)); err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM package_files WHERE package_id = ?", string(id)); err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM package_flags WHERE package_id = ?", string(id)); err != nil {
		return err
	}
	return nil
}

func PrepareDataBase(db *sql.DB) {
	_, err := db.Exec(CreateInstructions)
	if err != nil {
		fmt.Println("Error preparing database:", err)
	}
}
