package database

import (
	"database/sql"
)

type SDBPkg struct {
	Name           string `db:"name"`
	Id             string `db:"id"`
	Version        string `db:"version"`
	Serial         int    `db:"serial"`
	Maintainer     string `db:"maintainer"`
	Verified       bool   `db:"verified"`
	Description    string `db:"description"`
	UploadTimeUnix int64  `db:"upload_time"`

	Compiled bool `db:"available_compiled"`

	Location string `db:"location"`
}

func RetrievePackageInformation(nameOrId, favoriteLocation string, db *sql.DB) (SDBPkg, error) {

	rows, err := db.Query("SELECT name, id, version, serial, maintainer, verified, description, upload_time, available_compiled, location FROM packages WHERE id = ? OR name = ?", nameOrId, nameOrId)
	if err != nil {
		return SDBPkg{}, err
	}
	defer rows.Close()

	var obj SDBPkg

	for rows.Next() {
		if err := rows.Scan(
			&obj.Name,
			&obj.Id,
			&obj.Version,
			&obj.Serial,
			&obj.Maintainer,
			&obj.Verified,
			&obj.Description,
			&obj.UploadTimeUnix,
			&obj.Compiled,
			&obj.Location,
		); err != nil {
			return SDBPkg{}, err
		}

		if obj.Location == favoriteLocation {
			break
		}
	}

	return obj, nil
}
