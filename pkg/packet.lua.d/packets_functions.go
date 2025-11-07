package packet

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func (f *flags) LSetFlag(L *lua.LState) int {
	flagtype := L.CheckString(1)
	name := L.CheckString(2)
	flagPath := L.CheckString(3)

	*f = append(*f, Flag{
		Name:     name,
		Path:     flagPath,
		FlagType: flagtype,
	})

	return 0
}

func (all *allInstallInstructions) LInstall(L *lua.LState) int {
	src := L.CheckString(1)
	dest := L.CheckString(2)

	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		L.Error(lua.LString(err.Error()), 2)
		return 0
	}

	stats, err := os.Stat(src)
	if err != nil {
		L.Error(lua.LString(err.Error()), 2)
		return 0
	}

	instruction := InstallInstruction{
		Source:      src,
		Destination: dest,
		IsDir:       stats.IsDir(),
		FileMode:    stats.Mode(),
	}

	*all = append(*all, instruction)

	if stats.IsDir() {
		filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			newinstruction := InstallInstruction{}

			if path == src {
				return nil
			}

			newinstruction.Source = filepath.Join(path)
			destinationFixed, _ := strings.CutPrefix(path, src)

			newinstruction.Destination = filepath.Join(dest, destinationFixed)
			newinstruction.IsDir = d.IsDir()
			newinstruction.FileMode = d.Type()

			*all = append(*all, newinstruction)
			return nil
		})
	}

	return 0
}
