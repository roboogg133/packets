package packet

import (
	"math/rand"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$%!@%&*()-=+[]{}:;.,1234567890"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type PackageID struct {
	ID string
}

func (id PackageID) Name() string {
	return strings.SplitAfter(id.ID, "@")[0]
}

func (id PackageID) Version() string {
	return strings.SplitAfter(id.ID, "@")[1]
}

func NewId(id string) PackageID {
	var ID PackageID
	ID.ID = id
	return ID
}

func (pkg PacketLua) IsValid() bool {

	var a, b int

	for _, v := range *pkg.Plataforms {
		for _, src := range *v.Sources {
			a++
			if src.Method == "git" {
				if src.Specs.(GitSpecs).Branch == "" && src.Specs.(GitSpecs).Tag == nil {
					return false
				}
			}
		}
		b += len(v.Architetures)
	}

	a += len(*pkg.GlobalSources)

	if a < 1 || len(*pkg.Plataforms) > b {
		return false
	}

	switch {
	case pkg.Serial == -133:
		return false
	case pkg.Description == "" || pkg.Maintainer == "" || pkg.Name == "" || pkg.Version == "":
		return false
	}
	return true
}
