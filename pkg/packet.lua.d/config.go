package packet

type Config struct {
	BinDir *string
}

const defaultBinDir = "/usr/bin"

func checkConfig(cfg *Config) *Config {
	if cfg == nil {
		bin := defaultBinDir
		return &Config{
			BinDir: &bin,
		}
	}
	if *cfg.BinDir == "" || cfg.BinDir == nil {
		bin := defaultBinDir
		return &Config{
			BinDir: &bin,
		}
	} else {
		return cfg
	}

}
func checkConfigSrc(cfg *GetSourceConfig) *GetSourceConfig {
	if cfg == nil {
		return nil
	}

	switch {
	case *cfg.PacketDir == "" || cfg.PacketDir == nil:
		s := randStringBytes(12)
		cfg.PacketDir = &s
	}

	return cfg

}
