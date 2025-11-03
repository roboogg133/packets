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
