package utils_lua

import (
	"packets/configs"

	lua "github.com/yuin/gopher-lua"
)

func GetSandBox() (lua.LState, error) {

	cfg, err := configs.GetConfigTOML()
	if err != nil {
		return *lua.NewState(), err
	}
	L := lua.NewState()
	osObject := L.GetGlobal("os").(*lua.LTable)
	L.SetGlobal("SAFE_MODE", lua.LTrue)

	L.SetGlobal("BIN_DIR", lua.LString(cfg.Config.Bin_d))

	L.SetGlobal("path_join", L.NewFunction(Ljoin))

	osObject.RawSetString("remove", L.NewFunction(LSafeRemove))
	osObject.RawSetString("rename", L.NewFunction(LSafeRename))
	osObject.RawSetString("copy", L.NewFunction(LSafeCopy))
	osObject.RawSetString("symlink", L.NewFunction(LSymlink))
	osObject.RawSetString("mkdir", L.NewFunction(LMkdir))

	//ioObject.RawSetString("open", L.NewFunction(LOpen))

	return *L, nil
}
