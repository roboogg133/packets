package packet

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/klauspost/compress/zstd"

	lua_utils "github.com/roboogg133/packets/internal/lua"
	lua "github.com/yuin/gopher-lua"
)

type OperationalSystem string

type PacketLua struct {
	Name        string
	Version     string
	Maintainer  string
	Description string
	Serial      int

	Plataforms         *map[OperationalSystem]Plataform
	GlobalSources      *[]Source
	GlobalDependencies *PkgDependencies

	Flags []lua_utils.Flag

	Build     *lua.LFunction
	Install   *lua.LFunction
	PreRemove *lua.LFunction
	LuaState  *lua.LState
}

type Source struct {
	Method string
	Url    string
	Specs  any
}

type VersionConstraint string

type PkgDependencies struct {
	RuntimeDependencies *map[string]*VersionConstraint
	BuildDependencies   *map[string]*VersionConstraint
	Conflicts           *map[string]*VersionConstraint
}

type Plataform struct {
	Name         string
	Architetures []string
	Sources      *[]Source
	Dependencies *PkgDependencies
}

type GitSpecs struct {
	Branch string
	Tag    *string
}

type POSTSpecs struct {
	SHA256  *string
	Body    *string
	Headers *map[string]string
}

type GETSpecs struct {
	SHA256  *string
	Headers *map[string]string
}

var ErrCantFindPacketDotLua = errors.New("can't find Packet.lua in .tar.zst file")
var ErrFileDontReturnTable = errors.New("invalid Packet.lua format: the file do not return a table")
var ErrCannotFindPackageTable = errors.New("invalid Packet.lua format: can't find package table")
var ErrInstallFunctionDoesNotExist = errors.New("can not find install()")
var ErrSha256Sum = errors.New("false checksum")

// ReadPacket read a Packet.lua and alredy set global vars
func ReadPacket(f []byte, cfg *Config) (PacketLua, error) {
	cfg = checkConfig(cfg)

	L := lua.NewState()

	L.SetGlobal("error", L.NewFunction(lua_utils.LError))

	osObject := L.GetGlobal("os").(*lua.LTable)
	ioObject := L.GetGlobal("io").(*lua.LTable)

	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("io", lua.LNil)

	L.SetGlobal("BIN_DIR", lua.LString(cfg.BinDir))
	L.SetGlobal("CURRENT_ARCH", lua.LString(runtime.GOARCH))
	L.SetGlobal("CURRENT_ARCH_NORMALIZED", lua.LString(normalizeArch(runtime.GOARCH)))
	L.SetGlobal("CURRENT_PLATAFORM", lua.LString(runtime.GOOS))

	var newFlags lua_utils.Flags
	L.SetGlobal("setflags", L.NewFunction(newFlags.LSetFlag))
	L.SetGlobal("pathjoin", L.NewFunction(lua_utils.Ljoin))

	if err := L.DoString(string(f)); err != nil {
		return PacketLua{}, err
	}

	L.SetGlobal("os", osObject)
	L.SetGlobal("io", ioObject)

	tableLua := L.Get(-1)

	if tableLua.Type() != lua.LTTable {
		return PacketLua{}, ErrFileDontReturnTable
	}

	table := tableLua.(*lua.LTable)

	pkgTableLua := table.RawGetString("package")
	if pkgTableLua.Type() != lua.LTTable {
		return PacketLua{}, ErrCannotFindPackageTable
	}
	pkgTable := pkgTableLua.(*lua.LTable)

	packetLua := &PacketLua{
		Name:        getStringFromTable(pkgTable, "name"),
		Version:     getStringFromTable(pkgTable, "version"),
		Maintainer:  getStringFromTable(pkgTable, "maintainer"),
		Description: getStringFromTable(pkgTable, "description"),
		Serial:      getIntFromTable(pkgTable, "serial"),

		Plataforms: getPlataformsFromTable(pkgTable, "plataforms"),

		GlobalDependencies: getDependenciesFromTable(pkgTable, "build_dependencies"),
		GlobalSources:      getSourcesFromTable(pkgTable, "sources"),

		Build:     getFunctionFromTable(table, "build"),
		Install:   getFunctionFromTable(table, "install"),
		PreRemove: getFunctionFromTable(table, "pre_remove"),
	}

	packetLua.Flags = append(packetLua.Flags, newFlags.Flags...)

	packetLua.LuaState = L
	if packetLua.Install == nil {
		return PacketLua{}, ErrInstallFunctionDoesNotExist
	}

	return *packetLua, nil
}

func ReadPacketFromZSTDF(file io.Reader, cfg *Config) (PacketLua, error) {
	cfg = checkConfig(cfg)

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

			return ReadPacket(packageLuaBlob, cfg)
		}

	}
	return PacketLua{}, ErrCantFindPacketDotLua
}

// GetSource returns file []byte if method is "GET" or "POST", if is "git" returns *git.CloneOptions{}
func GetSource(url, method string, info any, tryAttempts int) (any, error) {

	switch method {
	case "GET":
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		specs := info.(GETSpecs)

		if specs.Headers != nil {
			for k, v := range *specs.Headers {
				req.Header.Set(k, v)
			}
		}

		client := http.Client{Timeout: 5 * time.Minute}

		var resp *http.Response
		for range tryAttempts {
			resp, err = client.Do(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				break
			}
			resp.Body.Close()
		}

		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if !verifySHA256(*specs.SHA256, data) {
			return nil, ErrSha256Sum
		}

		return data, nil
	case "POST":
		specs := info.(POSTSpecs)
		var body *bytes.Reader

		if specs.Body != nil {
			body = bytes.NewReader([]byte(*specs.Body))
		} else {
			body = nil
		}
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			return nil, err
		}

		if specs.Headers != nil {
			for k, v := range *specs.Headers {
				req.Header.Set(k, v)
			}
		}

		client := http.Client{Timeout: 5 * time.Minute}

		var resp *http.Response
		for range tryAttempts {
			resp, err = client.Do(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				break
			}
			resp.Body.Close()
		}

		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if !verifySHA256(*specs.SHA256, data) {
			return nil, ErrSha256Sum
		}

		return data, nil

	case "git":
		specs := info.(GitSpecs)

		if specs.Tag == nil {
			return &git.CloneOptions{
				URL:           url,
				SingleBranch:  true,
				ReferenceName: plumbing.NewBranchReferenceName(specs.Branch),
				Depth:         1,
			}, nil
		} else {
			return &git.CloneOptions{
				URL:           url,
				SingleBranch:  true,
				ReferenceName: plumbing.NewTagReferenceName(*specs.Tag),
				Depth:         1,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid method")
}

func verifySHA256(checksum string, src []byte) bool {

	check := sha256.Sum256(src)

	return hex.EncodeToString(check[:]) == checksum
}

func (pkg *PacketLua) ExecuteBuild(cfg *Config) {
	L := pkg.LuaState

	L.SetGlobal("error", L.NewFunction(lua_utils.LError))

	osObject := L.GetGlobal("os").(*lua.LTable)
	osObject.RawSetString("chdir", L.NewFunction(lua_utils.LCD))
	osObject.RawSetString("setenv", L.NewFunction(lua_utils.LSetEnv))
	osObject.RawSetString("copy", L.NewFunction(lua_utils.LCopy))
	osObject.RawSetString("mkdir", L.NewFunction(lua_utils.LMkdir))
	osObject.RawSetString("remove", L.NewFunction(lua_utils.LRemove))
	osObject.RawSetString("rename", L.NewFunction(lua_utils.LRename))
	osObject.RawSetString("symlink", L.NewFunction(lua_utils.LSymlink))
	osObject.RawSetString("chmod", L.NewFunction(lua_utils.LChmod))

	L.SetGlobal("BIN_DIR", lua.LString(cfg.BinDir))
	L.SetGlobal("CURRENT_ARCH", lua.LString(runtime.GOARCH))
	L.SetGlobal("CURRENT_ARCH_NORMALIZED", lua.LString(normalizeArch(runtime.GOARCH)))
	L.SetGlobal("CURRENT_PLATAFORM", lua.LString(runtime.GOOS))

	L.SetGlobal("SOURCESDIR", lua.LString(cfg.SourcesDir))
	L.SetGlobal("PACKETDIR", lua.LString(cfg.PacketDir))

	var newFlags lua_utils.Flags
	L.SetGlobal("setflags", L.NewFunction(newFlags.LSetFlag))
	L.SetGlobal("pathjoin", L.NewFunction(lua_utils.Ljoin))

	os.Chdir(cfg.RootDir)

	os.Setenv("PATH", os.Getenv("PATH")+":"+cfg.BinDir)

	L.Push(pkg.Build)
	L.Call(0, 0)

	pkg.Flags = append(pkg.Flags, newFlags.Flags...)
}

func (pkg *PacketLua) ExecuteInstall(cfg *Config) {
	L := pkg.LuaState
	defer L.Close()

	L.SetGlobal("error", L.NewFunction(lua_utils.LError))

	osObject := L.GetGlobal("os").(*lua.LTable)
	osObject.RawSetString("chdir", L.NewFunction(lua_utils.LCD))
	osObject.RawSetString("setenv", L.NewFunction(lua_utils.LSetEnv))
	osObject.RawSetString("copy", L.NewFunction(lua_utils.LCopy))
	osObject.RawSetString("mkdir", L.NewFunction(lua_utils.LMkdir))
	osObject.RawSetString("remove", L.NewFunction(lua_utils.LRemove))
	osObject.RawSetString("rename", L.NewFunction(lua_utils.LRename))
	osObject.RawSetString("symlink", L.NewFunction(lua_utils.LSymlink))
	osObject.RawSetString("chmod", L.NewFunction(lua_utils.LChmod))

	L.SetGlobal("BIN_DIR", lua.LString(cfg.BinDir))
	L.SetGlobal("CURRENT_ARCH", lua.LString(runtime.GOARCH))
	L.SetGlobal("CURRENT_ARCH_NORMALIZED", lua.LString(normalizeArch(runtime.GOARCH)))
	L.SetGlobal("CURRENT_PLATAFORM", lua.LString(runtime.GOOS))

	L.SetGlobal("SOURCESDIR", lua.LString(cfg.SourcesDir))
	L.SetGlobal("PACKETDIR", lua.LString(cfg.PacketDir))

	var newFlags lua_utils.Flags
	L.SetGlobal("setflags", L.NewFunction(newFlags.LSetFlag))
	L.SetGlobal("pathjoin", L.NewFunction(lua_utils.Ljoin))

	os.Chdir(cfg.RootDir)

	os.Setenv("PATH", os.Getenv("PATH")+":"+cfg.BinDir)

	L.Push(pkg.Install)
	L.Call(0, 0)

	pkg.Flags = append(pkg.Flags, newFlags.Flags...)

}
