package internal

import (
	"archive/tar"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
	"github.com/pelletier/go-toml/v2"
)

// const

const DefaultLinux_d = "/etc/packets"
const DefaultCache_d = "/var/cache/packets"
const DefaultHttpPort = 9123
const DefaultData_d = "/opt/packets"

// errors
var ErrResponseNot200OK = errors.New("the request is not 200, download failed")
var ErrCantFindManifestTOML = errors.New("can't find manifest.toml when trying to read the packagefile")

// toml files

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

type ConfigTOML struct {
	Config struct {
		HttpPort int    `toml:"httpPort"`
		Cache_d  string `toml:"cache_d"`
		Data_d   string `toml:"data_d"`
		Bin_d    string `toml:"bin_d"`
	} `toml:"Config"`
}

func DownloadPackageHTTP(url string) (*[]byte, error) {

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrResponseNot200OK
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &fileBytes, nil
}

// DefaultConfigTOML generate a default toml and create the directorys
func DefaultConfigTOML() (*ConfigTOML, error) {

	var config ConfigTOML

	_, err := os.Stat(DefaultCache_d)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(DefaultCache_d, 0666)
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = os.Stat(DefaultCache_d)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(DefaultData_d, 0644)
			if err != nil {
				return nil, err
			}
		}
	}

	config.Config.Cache_d = DefaultCache_d
	config.Config.Data_d = DefaultData_d
	config.Config.HttpPort = DefaultHttpPort

	return &config, nil
}

func ReadManifest(file *os.File) (*Manifest, error) {
	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if filepath.Base(header.Name) == "manifest.toml" {
			decoder := toml.NewDecoder(tarReader)

			var manifest Manifest

			if err := decoder.Decode(&manifest); err != nil {
				log.Fatal(err)
			}

			return &manifest, nil
		}

	}
	return nil, ErrCantFindManifestTOML
}

func GetConfigTOML() (*ConfigTOML, error) {
	f, err := os.Open(filepath.Join(DefaultLinux_d, "config.toml"))
	if err != nil {
		return nil, err
	}

	decoder := toml.NewDecoder(f)

	var config ConfigTOML
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func BroadcastAddr(ip net.IP, mask net.IPMask) net.IP {
	b := make(net.IP, len(ip))
	for i := range ip {
		b[i] = ip[i] | ^mask[i]
	}
	return b
}
