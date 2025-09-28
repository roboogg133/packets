package utils_lua

import (
	"packets/configs"

	lua "github.com/yuin/gopher-lua"
)

func GetSandBox(sandboxdir string) (lua.LState, error) {

	SandboxDir = sandboxdir

	cfg, err := configs.GetConfigTOML()
	if err != nil {
		return *lua.NewState(), err
	}
	L := lua.NewState()
	osObject := L.GetGlobal("os").(*lua.LTable)
	ioObject := L.GetGlobal("io").(*lua.LTable)

	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("SAFE_MODE", lua.LTrue)

	L.SetGlobal("PACKETS_DATADIR", lua.LString(cfg.Config.Data_d))
	L.SetGlobal("packets_bin_dir", lua.LString(cfg.Config.Bin_d))

	L.SetGlobal("path_join", L.NewFunction(Ljoin))

	// Packets build functions
	build := L.NewTable()

	L.SetField(build, "requirements", L.NewFunction(LCompileRequirements))
	L.SetField(build, "compile", L.NewFunction(LCompile))

	L.SetGlobal("build", build)

	osObject.RawSetString("execute", lua.LNil)
	osObject.RawSetString("exit", lua.LNil)
	osObject.RawSetString("getenv", lua.LNil)

	osObject.RawSetString("remove", L.NewFunction(LSafeRemove))
	osObject.RawSetString("rename", L.NewFunction(LSafeRename))
	osObject.RawSetString("copy", L.NewFunction(LSafeCopy))
	osObject.RawSetString("symlink", L.NewFunction(LSymlink))
	osObject.RawSetString("mkdir", L.NewFunction(LMkdir))

	ioObject.RawSetString("input", lua.LNil)
	ioObject.RawSetString("output", lua.LNil)
	ioObject.RawSetString("popen", lua.LNil)
	ioObject.RawSetString("tmpfile", lua.LNil)
	ioObject.RawSetString("stdout", lua.LNil)
	ioObject.RawSetString("stderr", lua.LNil)
	ioObject.RawSetString("stdin", lua.LNil)
	ioObject.RawSetString("lines", lua.LNil)
	ioObject.RawSetString("open", L.NewFunction(LOpen))

	return *L, nil
}
