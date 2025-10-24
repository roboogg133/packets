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

	L.SetGlobal("PACKETS_DATADIR", lua.LString(cfg.Config.Data_d))
	L.SetGlobal("packets_bin_dir", lua.LString(cfg.Config.Bin_d))

	L.SetGlobal("path_join", L.NewFunction(Ljoin))

	// Packets build functions

	osObject.RawSetString("remove", L.NewFunction(LSafeRemove))
	osObject.RawSetString("rename", L.NewFunction(LSafeRename))
	osObject.RawSetString("copy", L.NewFunction(LSafeCopy))
	osObject.RawSetString("symlink", L.NewFunction(LSymlink))
	osObject.RawSetString("mkdir", L.NewFunction(LMkdir))

	//ioObject.RawSetString("open", L.NewFunction(LOpen))

	return *L, nil
}
