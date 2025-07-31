package internal

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ulikunitz/xz"
)

type ConfigTOML struct {
	Config struct {
		HttpPort           int    `toml:"httpPort"`
		CacheDir           string `toml:"cacheDir"`
		AutoDeleteCacheDir bool   `toml:"dayToDeleteCacheDir"`
		DaysToDelete       int    `toml:"daysToDelete"`
		DataDir            string `toml:"dataDir"`
		BinDir             string `toml:"binDir"`
		LastDataDir        string `toml:"lastDataDir"`
	} `toml:"Config"`
}

type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Author       string   `json:"author"`
	Family       string   `json:"family"`
	Serial       uint     `json:"serial"`
}

func PacketsPackageDir() string {

	out, _ := exec.Command("uname", "-s").Output()

	if uname := strings.TrimSpace(string(out)); uname == "OpenTTY" {
		return "/mnt/....."
	} else {
		return "/etc/packets"
	}
}

func ManifestReadXZ(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(xzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(hdr.Name, "/manifest.json") || hdr.Name == "manifest.json" {

			var manifest Manifest
			decoder := json.NewDecoder(tr)
			if err := decoder.Decode(&manifest); err != nil {
				return nil, err
			}
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("can't find manifest.json")
}

func DefaultConfigTOML() *ConfigTOML {

	var cfg ConfigTOML
	out, _ := exec.Command("uname", "-s").Output()

	if uname := strings.TrimSpace(string(out)); uname == "OpenTTY" {
		cfg.Config.HttpPort = 9123
		cfg.Config.AutoDeleteCacheDir = false
		cfg.Config.CacheDir = "/mnt/... " // TODO
		cfg.Config.DataDir = "/mnt/... "  // TODO
		cfg.Config.DaysToDelete = -1
		cfg.Config.BinDir = "/home/...."     // TODO
		cfg.Config.LastDataDir = "/mnt/... " // TODO
		return &cfg
	} else {

		cfg.Config.HttpPort = 9123
		cfg.Config.AutoDeleteCacheDir = false
		cfg.Config.CacheDir = "/var/cache/packets"
		cfg.Config.DataDir = "/opt/packets"
		cfg.Config.DaysToDelete = -1
		cfg.Config.BinDir = "/usr/bin"
		cfg.Config.LastDataDir = "/opt/packets"

		return &cfg
	}

}
