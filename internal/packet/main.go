package packet

import (
	"archive/tar"
	"fmt"
	"io"
	"packets/configs"
	errors_packets "packets/internal/errors"
	"path/filepath"
	"runtime"

	"github.com/klauspost/compress/zstd"
	lua "github.com/yuin/gopher-lua"
)

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

	BuildDependencies map[string]string

	Prepare *lua.LFunction
	Build   *lua.LFunction
	Install *lua.LFunction
	Remove  *lua.LFunction
}

// ReadPacket read a Packet.lua and alredy set global vars
func ReadPacket(f []byte) (PacketLua, error) {
	cfg, err := configs.GetConfigTOML()
	if err != nil {
		return PacketLua{}, err
	}
	L := lua.NewState()
	defer L.Close()

	osObject := L.GetGlobal("os").(*lua.LTable)
	ioObject := L.GetGlobal("io").(*lua.LTable)

	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)

	if err := L.DoString(string(f)); err != nil {
		return PacketLua{}, err
	}
	L.SetGlobal("os", osObject)
	L.SetGlobal("io", ioObject)

	L.SetGlobal("BIN_DIR", lua.LString(cfg.Config.Bin_d))
	L.SetGlobal("ARCH", lua.LString(runtime.GOARCH))
	L.SetGlobal("OS", lua.LString(runtime.GOOS))

	tableLua := L.Get(-1)

	if tableLua.Type() != lua.LTTable {
		return PacketLua{}, fmt.Errorf("invalid Packet.lua format: the file do not return a table")
	}

	table := tableLua.(*lua.LTable)

	pkgTableLua := table.RawGetString("package")
	if pkgTableLua.Type() != lua.LTTable {
		return PacketLua{}, fmt.Errorf("invalid Packet.lua format: can't find package table")
	}
	pkgTable := pkgTableLua.(*lua.LTable)

	packetLua := &PacketLua{
		Name:        getStringFromTable(pkgTable, "name"),
		Id:          getStringFromTable(pkgTable, "id"),
		Version:     getStringFromTable(pkgTable, "version"),
		Author:      getStringFromTable(pkgTable, "author"),
		Description: getStringFromTable(pkgTable, "description"),
		PkgType:     getStringFromTable(pkgTable, "type"),

		Dependencies:      getDependenciesFromTable(L, pkgTable, "dependencies"),
		BuildDependencies: getDependenciesFromTable(L, pkgTable, "build_dependencies"),

		GitUrl:    getStringFromTable(pkgTable, "git_url"),
		GitBranch: getStringFromTable(pkgTable, "git_branch"),

		Prepare: getFunctionFromTable(pkgTable, "prepare"),
		Build:   getFunctionFromTable(pkgTable, "build"),
		Install: getFunctionFromTable(pkgTable, "install"),
		Remove:  getFunctionFromTable(pkgTable, "remove"),
	}

	if packetLua.Install == nil || packetLua.Remove == nil {
		return PacketLua{}, fmt.Errorf("install or remove function is not valid")
	}

	return *packetLua, nil
}

func (packetLua PacketLua) ExecuteRemove(L *lua.LState) error {
	L.Push(packetLua.Remove)
	return L.PCall(0, 0, nil)
}

func ReadPacketFromFile(file io.Reader) (PacketLua, error) {

	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return PacketLua{}, err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return PacketLua{}, err
		}

		if filepath.Base(header.Name) == "Packet.lua" {

			packageLuaBlob, err := io.ReadAll(tarReader)
			if err != nil {
				return PacketLua{}, err
			}

			return ReadPacket(packageLuaBlob)
		}

	}
	return PacketLua{}, errors_packets.ErrCantFindPacketDotLua
}
