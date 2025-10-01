package configs

type Manifest struct {
	Info struct {
		Name         string   `toml:"name"`
		Id           string   `toml:"id"`
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
		HttpPort      int    `toml:"httpPort"`
		Cache_d       string `toml:"cache_d"`
		Data_d        string `toml:"data_d"`
		Bin_d         string `toml:"bin_d"`
		StorePackages bool   `toml:"store_packages"`
	} `toml:"Config"`
}
