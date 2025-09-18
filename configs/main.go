package configs

import (
	"os"
	"packets/internal/consts"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// DefaultConfigTOML returns configTOML struct with all default values and create all directorys
func DefaultConfigTOML() (*ConfigTOML, error) {

	var config ConfigTOML

	_, err := os.Stat(consts.DefaultCache_d)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(consts.DefaultCache_d, 0666)
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = os.Stat(consts.DefaultCache_d)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(consts.DefaultData_d, 0644)
			if err != nil {
				return nil, err
			}
		}
	}

	config.Config.Cache_d = consts.DefaultCache_d
	config.Config.Data_d = consts.DefaultData_d
	config.Config.HttpPort = consts.DefaultHttpPort

	return &config, nil
}

// GetConfigTOML return settings values
func GetConfigTOML() (*ConfigTOML, error) {
	f, err := os.Open(filepath.Join(consts.DefaultLinux_d, "config.toml"))
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
