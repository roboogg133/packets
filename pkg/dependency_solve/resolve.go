package dependencysolve

import "github.com/roboogg133/packets/pkg/packet.lua.d"

type InstallInstruction struct {
	Build    []string
	Install  []string
	Conflict []string
}

func ResolveDependencies(dependencies packet.PkgDependencies) InstallInstruction {
	// Implementation goes here
	return InstallInstruction{}
}
