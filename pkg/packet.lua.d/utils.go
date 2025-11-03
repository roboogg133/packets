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
