package packet

type Config struct {
	BinDir string
}

const defaultBinDir = "/usr/bin"

func checkConfig(cfg *Config) *Config {

	if cfg == nil {
		return &Config{
			BinDir: defaultBinDir,
		}
	}
	if cfg.BinDir == "" {
		return &Config{
			BinDir: defaultBinDir,
		}
	} else {
		return cfg
	}

}
