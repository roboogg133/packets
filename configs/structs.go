package configs

/*
type Manifest struct {
	Package struct {
		Name         string            `toml:"name"`
		Id           string            `toml:"id"`
		Version      string            `toml:"version"`
		Description  string            `toml:"description"`
		Dependencies map[string]string `toml:"dependencies"`
		Author       string            `toml:"author"`
		Architeture  string            `toml:"architeture"`
		Os           string            `toml:"os"`
		PacakgeType  string            `toml:"type"`

		GitUrl string `toml:"giturl,omitempty"`
		Branch string `toml:"gitbranch,omitempty"`
	} `toml:"Package"`
	Build struct {
		BuildDependencies map[string]string `toml:"dependencies"`
	}
	Hooks struct {
		Fetch   string `toml:"fetch,omitempty"`
		Install string `toml:"install"`
		Remove  string `toml:"remove"`
		Build   string `toml:"build"`
	} `toml:"Hooks"`
}
*/

type ConfigTOML struct {
	Config struct {
		HttpPort      int    `toml:"httpPort"`
		Cache_d       string `toml:"cache_d"`
		Data_d        string `toml:"data_d"`
		Bin_d         string `toml:"bin_d"`
		StorePackages bool   `toml:"store_packages"`
	} `toml:"Config"`
}
