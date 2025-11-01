package packet

func (pkg PacketLua) IsValid() bool {

	var a, b int

	for _, v := range *pkg.Plataforms {
		a += len(*v.Sources)
		b += len(v.Architetures)
	}

	a += len(*pkg.GlobalSources)

	if a < 1 || len(*pkg.Plataforms) > b {
		return false
	}

	switch {
	case pkg.Serial == -133:
		return false
	case pkg.Description == "" || pkg.Maintaner == "" || pkg.Name == "" || pkg.Version == "":
		return false
	}
	return true
}
