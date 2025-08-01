package internal

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/klauspost/compress/zstd"
	lua "github.com/yuin/gopher-lua"
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

			decoder.Decode(&manifest)

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

func IsSafe(str string) bool {
	s, err := filepath.EvalSymlinks(filepath.Clean(str))
	if err != nil {
		return false
	}
	var cfg ConfigTOML
	toml.DecodeFile(filepath.Join(PacketsPackageDir(), "config.toml"), &cfg)

	if strings.HasPrefix(s, cfg.Config.DataDir) || strings.HasPrefix(s, cfg.Config.BinDir) {
		return true

	} else if strings.Contains(s, ".ssh") {
		return false

	} else if strings.HasPrefix(s, "/etc") {
		return false

	} else if strings.HasPrefix(s, "/usr") || strings.HasPrefix(s, "/bin") {

		return strings.HasPrefix(s, "/usr/share")

	} else if strings.HasPrefix(s, "/var/mail") {
		return false

	} else if strings.HasPrefix(s, "/proc") {
		return false

	} else if strings.HasPrefix(s, "/sys") {
		return false

	} else if strings.HasPrefix(s, "/var/run") || strings.HasPrefix(s, "/run") {
		return false

	} else if strings.HasPrefix(s, "/tmp") {
		return false

	} else if strings.HasPrefix(s, "/dev") {
		return false

	} else if strings.HasPrefix(s, "/boot") {
		return false

	} else if strings.HasPrefix(s, "/home") {
		if strings.Contains(s, "/Pictures") || strings.Contains(s, "/Videos") || strings.Contains(s, "/Documents") || strings.Contains(s, "/Downloads") {
			return false
		}

	} else if strings.HasPrefix(s, "/lib") || strings.HasPrefix(s, "/lib64") || strings.HasPrefix(s, "/var/lib64") || strings.HasPrefix(s, "/lib") {
		return false

	} else if strings.HasPrefix(s, "/sbin") {
		return false

	} else if strings.HasPrefix(s, "/srv") {
		return false

	} else if strings.HasPrefix(s, "/mnt") {
		return false

	} else if strings.HasPrefix(s, "/media") {
		return false
	} else if strings.HasPrefix(s, "/snap") {
		return false
	}

	return true
}

func SafeRemove(L *lua.LState) int {
	filename := L.CheckString(1)
	if !IsSafe(filename) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}
	err := os.Remove(filename)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] remove failed\n" + err.Error()))
		return 1
	}
	L.Push(lua.LTrue)
	return 1
}

func SafeRename(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if !IsSafe(oldname) || !IsSafe(newname) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}

	if err := os.Rename(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] rename failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}
func SafeCopy(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if !IsSafe(oldname) || !IsSafe(newname) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}

	src, err := os.Open(oldname)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2

	}
	defer src.Close()

	status, err := src.Stat()
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2
	}

	err = os.MkdirAll(filepath.Dir(newname), 0755)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2
	}

	dst, err := os.Create(newname)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2
	}

	defer dst.Close()
	if err := dst.Chmod(status.Mode()); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] copy failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}
