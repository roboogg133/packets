package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

type Dependencies struct {
	PackageId  packet.PackageID
	Name       string `json:"name"`
	Constraint string `json:"con"`
}

type PackageJsonInfo struct {
	Name        string         `json:"name"`
	Id          string         `json:"id"`
	Version     string         `json:"version"`
	Serial      int            `json:"serial"`
	Maintainer  string         `json:"maintainer"`
	Verified    bool           `json:"verified"`
	Description string         `json:"desc"`
	UploadTime  int64          `json:"time"`
	RuntimeDeps []Dependencies `json:"depn"`
	BuildDeps   []Dependencies `json:"build"`
	Conflicts   []Dependencies `json:"conflict"`

	AvailableCompiled bool `json:"compiled"`
}

func FetchPackagesToDB(url string, db *sql.DB) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return err
	}

	var data []PackageJsonInfo

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if !strings.HasPrefix(r.(string), "UNIQUE constraint failed") {
				panic(r)
			}
		}
	}()

	urlNormalized := strings.Replace(url, "https://", "", 1)
	urlNormalized = strings.Replace(url, "http://", "", 1)

	for _, info := range data {
		if _, err := db.Exec("INSERT OR REPLACE INTO packages (name, version, serial, maintainer, verified, description, upload_time, available_compiled, location) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", info.Name, info.Version, info.Serial, info.Maintainer, info.Verified, info.Description, info.UploadTime, info.AvailableCompiled, urlNormalized); err != nil {
			return err
		}

		for _, dep := range info.RuntimeDeps {
			if _, err := db.Exec("INSERT OR REPLACE INTO dependencies (package_id, name, constraint, location) VALUES (?, ?, ?, ?)", info.Id, dep.Name, dep.Constraint, urlNormalized); err != nil {
				panic(err)
			}
		}

		for _, dep := range info.BuildDeps {
			if _, err := db.Exec("INSERT INTO dependencies (package_id, name, constraint, location) VALUES (?, ?, ?, ?)", info.Id, dep.Name, dep.Constraint, urlNormalized); err != nil {
				panic(err)
			}
		}

		for _, dep := range info.Conflicts {
			if _, err := db.Exec("INSERT INTO dependencies (package_id, name, constraint, location) VALUES (?, ?, ?, ?)", info.Id, dep.Name, dep.Constraint, urlNormalized); err != nil {
				panic(err)
			}
		}
	}

	return nil
}
