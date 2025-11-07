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
		id.ID = strings.SplitAfter(name, "@")[0]
		return id, nil
	}

	err := db.QueryRow("SELECT id FROM installed_packages WHERE name = ?", name).Scan(&id.ID)
	return id, err
}

func SearchIfIsInstalled(name string, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM installed_packages WHERE name = ?)", name).Scan(&exists)
	return exists, err
}

func GetAllFromFlag(packageID packet.PackageID, flagType string, db *sql.DB) ([]packet.Flag, error) {
	var flags []packet.Flag

	rows, err := db.Query("SELECT name, path FROM package_flags WHERE package_id = ? AND flag = ?", packageID.ID, flagType)
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
	rows, err := db.Query("SELECT path, is_dir FROM package_files WHERE package_id = ?", packageID.ID)
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
