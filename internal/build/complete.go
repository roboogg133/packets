package build

import (
	"packets/internal/packet"
	utils_lua "packets/internal/utils/lua"

	lua "github.com/yuin/gopher-lua"
)

func (container Container) ExecutePrepare(packetLua packet.PacketLua, L *lua.LState) error {

	gitTable := L.NewTable()

	gitTable.RawSetString("clone", L.NewFunction(utils_lua.LGitClone))
	gitTable.RawSetString("checkout", L.NewFunction(utils_lua.LGitCheckout))
	gitTable.RawSetString("pull", L.NewFunction(utils_lua.LGitPUll))

	containerTable := L.NewTable()

	containerTable.RawSetString("dir", L.NewFunction(container.lDir))

	L.SetGlobal("git", gitTable)
	L.Push(packetLua.Prepare)
	L.Push(containerTable)

	return L.PCall(1, 0, nil)

}

func (container Container) ExecuteBuild(packetLua packet.PacketLua, L *lua.LState) error {

	osObject := L.GetGlobal("os").(*lua.LTable)
	ioObject := L.GetGlobal("io").(*lua.LTable)

	OnlyContainerOS := L.NewTable()

	OnlyContainerOS.RawSetString("copy", L.NewFunction(container.lCopy))
	OnlyContainerOS.RawSetString("mkdir", L.NewFunction(container.lMkdir))
	OnlyContainerOS.RawSetString("rename", L.NewFunction(container.lRename))
	OnlyContainerOS.RawSetString("remove", L.NewFunction(container.lRemove))
	OnlyContainerOS.RawSetString("execute", L.NewFunction(container.lexecute))
	OnlyContainerOS.RawSetString("open", L.NewFunction(container.lOpen))

	OnlyContainerIO := L.NewTable()

	OnlyContainerIO.RawSetString("popen", L.NewFunction(container.lpopen))

	L.SetGlobal("io", OnlyContainerIO)
	L.SetGlobal("os", OnlyContainerOS)

	L.Push(packetLua.Build)
	err := L.PCall(0, 0, nil)
	if err != nil {
		return err
	}

	L.SetGlobal("os", osObject)
	L.SetGlobal("io", ioObject)
	return nil
}

func (container Container) ExecuteInstall(packetLua packet.PacketLua, L *lua.LState) error {

	containerTable := L.NewTable()

	containerTable.RawSetString("dir", L.NewFunction(container.lDir))

	L.Push(packetLua.Install)
	L.Push(containerTable)

	return L.PCall(1, 0, nil)
}
