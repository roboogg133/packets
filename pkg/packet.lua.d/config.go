package packet

type Config struct {
	BinDir     string
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
	}

	return cfg
}
