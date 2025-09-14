package internal

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
)

// const

const DefaultLinux_d = "/etc/packets"
const DefaultCache_d = "/var/cache/packets"
const DefaultHttpPort = 9123
const DefaultData_d = "/opt/packets"

// errors
var ErrResponseNot200OK = errors.New("the request is not 200, download failed")

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
