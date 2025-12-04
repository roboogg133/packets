package database

import (
	"database/sql"
	"strings"

	"github.com/roboogg133/packets/pkg/install"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

// this function will get from a package name an id based on installed packages
func GetPackageId(name string, db *sql.DB) (packet.PackageID, error) {
	var id packet.PackageID

	if strings.Contains(name, "@") {
		id = packet.PackageID(strings.SplitAfter(name, "@")[0])
		return id, nil
	}

	var s string
	err := db.QueryRow("SELECT id FROM installed_packages WHERE name = ?", name).Scan(&s)
	return packet.NewId(s), err
}

func SearchIfIsInstalled(name string, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM installed_packages WHERE name = ? OR id = ?)", name, name).Scan(&exists)
	return exists, err
}

func CheckVersionInBuild(name string, db *sql.DB) (packet.PackageID, int, string, error) {
	var id, location string
	var serial int

	if err := db.QueryRow("SELECT id, serial, location FROM build_packages WHERE name = ?", name).Scan(&id, &serial, &location); err != nil {
		return packet.PackageID(""), 0, "", err
	}

	return packet.NewId(id), serial, location, nil
}

func CheckVersionInstalled(name string, db *sql.DB) (packet.PackageID, int, string, error) {
	var id, location string
	var serial int

	if err := db.QueryRow("SELECT id, serial, location FROM installed_packages WHERE name = ?", name).Scan(&id, &serial, &location); err != nil {
		return packet.PackageID(""), 0, "", err
	}

	return packet.NewId(id), serial, location, nil
}

func GetAllFromFlag(packageID packet.PackageID, flagType string, db *sql.DB) ([]packet.Flag, error) {
	var flags []packet.Flag

	rows, err := db.Query("SELECT name, path FROM package_flags WHERE package_id = ? AND flag = ?", string(packageID), flagType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var flag packet.Flag
		if err := rows.Scan(&flag.Name, &flag.Path); err != nil {
			return nil, err
		}
		flag.FlagType = flagType
		flags = append(flags, flag)
	}

	return flags, nil
}

func GetPackageFiles(packageID packet.PackageID, db *sql.DB) ([]install.BasicFileStatus, error) {
	var files []install.BasicFileStatus
	rows, err := db.Query("SELECT path, is_dir FROM package_files WHERE package_id = ?", string(packageID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var file install.BasicFileStatus
		if err := rows.Scan(&file.Filepath, &file.IsDir); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

/*
 *  name TEXT NOT NULL UNIQUE,
 id TEXT PRIMARY KEY,
 version TEXT NOT NULL,
 serial INTEGER NOT NULL,
 maintainer TEXT NOT NULL,
 verified INTEGER NOT NULL DEFAULT 0,
 description TEXT NOT NULL,
 upload_time INTEGER NOT NULL,
 installed_time INTEGER NOT NULL,

 location TEXT NOT NULL
 *
*/

type DBPkg struct {
	Name              string `db:"name"`
	Id                string `db:"id"`
	Version           string `db:"version"`
	Serial            int    `db:"serial"`
	Maintainer        string `db:"maintainer"`
	Verified          bool   `db:"verified"`
	Description       string `db:"description"`
	UploadTimeUnix    int64  `db:"upload_time"`
	InstalledTimeUnix int64  `db:"installed_time"`

	Location string `db:"location"`
}

func ListAllInstalledPackages(db *sql.DB) ([]DBPkg, error) {
	rows, err := db.Query("SELECT name, id, version, serial, maintainer, verified, description, upload_time, installed_time FROM installed_packages")
	if err != nil {
		return nil, err
	}

	var list []DBPkg

	for rows.Next() {
		var obj DBPkg
		if err := rows.Scan(
			&obj.Name,
			&obj.Id,
			&obj.Version,
			&obj.Serial,
			&obj.Maintainer,
			&obj.Verified,
			&obj.Description,
			&obj.UploadTimeUnix,
			&obj.InstalledTimeUnix,
		); err != nil {
			return nil, err
		}
		list = append(list, obj)
	}

	return list, nil
}
