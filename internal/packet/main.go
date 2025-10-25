package packet

type PacketLua struct {
	Name         string
	Id           string
	Version      string
	Description  string
	Dependencies map[string]string
	Author       string
	Architetures []string
	Os           []string

	PkgType   string
	GitUrl    string
	GitBranch string

	BuildDependencies string
}
