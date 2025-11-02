package main

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type PacketsConfiguration struct {
	BinDir string `toml:"BinDir"`
}

func GetConfiguration() error {
	configFile := filepath.Join(ConfigurationDir, "config.toml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config PacketsConfiguration
	err = toml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	Config = &config
	return nil
}
