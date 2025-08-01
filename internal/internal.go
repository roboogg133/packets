package internal

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/klauspost/compress/zstd"
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
	Info struct {
		Name         string   `toml:"name"`
		Version      string   `toml:"version"`
		Description  string   `toml:"description"`
		Dependencies []string `toml:"dependencies"`
		Author       string   `toml:"author"`
		Family       string   `toml:"family"`
		Serial       uint     `toml:"serial"`
	} `toml:"Info"`
	Hooks struct {
		Install string `toml:"install"`
		Remove  string `toml:"remove"`
	} `toml:"Hooks"`
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

	zr, err := zstd.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	tarReader := tar.NewReader(zr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == "/manifest.toml" || header.Name == "manifest.toml" {
			decoder := toml.NewDecoder(tarReader)

			var manifest Manifest

			decoder.Decode(manifest)

			return &manifest, nil
		}

	}
	return nil, fmt.Errorf("can't find manifest.toml")
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
