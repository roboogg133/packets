package packet

import "path/filepath"

type Config struct {
	BinDir     string
	PacketDir  string
	SourcesDir string
	RootDir    string
}

const defaultBinDir = "/usr/bin"

func checkConfig(cfg *Config) *Config {
	if cfg == nil {
		bin := defaultBinDir
		return &Config{
			BinDir: bin,
		}
	}

	switch {
	case cfg.BinDir == "":
		return &Config{
			BinDir: defaultBinDir,
		}
	case cfg.PacketDir == "":

		cfg.PacketDir = filepath.Join("/tmp", randStringBytes(12))
	}

	return cfg
}
