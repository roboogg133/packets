package build

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	utils_lua "packets/internal/utils/lua"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"
)

func (container Container) lRemove(L *lua.LState) int {
	filename := L.CheckString(1)

	err := container.FS.RemoveAll(filename)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func (container Container) lRename(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if err := container.FS.Rename(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}

func (container Container) lCopy(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if err := container.copyContainer(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func modeFlags(mode string) int {
	switch mode {
	case "r", "rb":
		return os.O_RDONLY
	case "w", "wb":
		return os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	case "a", "ab":
		return os.O_CREATE | os.O_WRONLY | os.O_APPEND
	case "r+", "r+b", "rb+", "br+":
		return os.O_RDWR
	case "w+", "w+b", "wb+", "bw+":
		return os.O_CREATE | os.O_RDWR | os.O_TRUNC
	case "a+", "a+b", "ab+", "ba+":
		return os.O_CREATE | os.O_RDWR | os.O_APPEND
	default:
		return os.O_RDONLY
	}
}

func (container Container) lOpen(L *lua.LState) int {
	path := L.CheckString(1)
	mode := L.OptString(2, "r")

	file, err := container.FS.OpenFile(path, modeFlags(mode), 0o644)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = file
	L.SetMetatable(ud, L.GetTypeMetatable("file"))
	L.Push(ud)
	L.Push(lua.LNil)
	return 2
}

func (container Container) lMkdir(L *lua.LState) int {
	path := L.CheckString(1)
	perm := L.CheckInt(2)

	if err := container.FS.MkdirAll(path, os.FileMode(perm)); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func (container Container) lexecute(L *lua.LState) int {
	cmdString := L.CheckString(1)

	cmdSlice := strings.Fields(cmdString)

	files, err := afero.ReadDir(container.FS, BinDir)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("exit"))
		L.Push(lua.LNumber(127))
		return 3
	}

	for _, file := range files {
		if !file.IsDir() && file.Name() == cmdSlice[0] {
			err := exec.Command(cmdSlice[0], cmdSlice[1:]...).Run()
			if err != nil {
				if errr := err.(*exec.ExitError); errr != nil {
					L.Push(lua.LFalse)

					if err.(*exec.ExitError).Exited() {
						L.Push(lua.LString("exit"))
					} else {
						L.Push(lua.LString("signal"))
					}

					L.Push(lua.LNumber(err.(*exec.ExitError).ExitCode()))
					return 3
				}
			}
			L.Push(lua.LTrue)
			L.Push(lua.LString("exit"))
			L.Push(lua.LNumber(0))
		}
	}

	L.Push(lua.LFalse)
	L.Push(lua.LString("exit"))
	L.Push(lua.LNumber(127))
	return 3
}

func (container Container) lpopen(L *lua.LState) int {
	cmdString := L.CheckString(1)
	mode := L.CheckString(2)

	cmdSlice := strings.Fields(cmdString)

	files, err := afero.ReadDir(container.FS, BinDir)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("can't find executable"))
		return 2
	}

	for _, file := range files {
		if !file.IsDir() && file.Name() == cmdSlice[0] {
			cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
			output, _ := cmd.CombinedOutput()

			switch mode {
			case "r":
				ud := L.NewUserData()
				ud.Value = string(output)
				L.SetMetatable(ud, L.GetTypeMetatable("file"))
				L.Push(ud)
				L.Push(lua.LNil)
			case "w":
				ud := L.NewUserData()
				ud.Value = string(output)
				os.Stdout.Write(output)
				L.SetMetatable(ud, L.GetTypeMetatable("file"))
				L.Push(ud)
				L.Push(lua.LNil)
			default:
				L.Push(lua.LNil)
				L.Push(lua.LString(fmt.Sprintf("%s: Invalid argument", cmdString)))
			}

		}
	}

	L.Push(lua.LNil)
	L.Push(lua.LString("can't find any executable"))
	return 2
}

func (container Container) GetLuaState() error {
	L := lua.NewState()
	osObject := L.GetGlobal("os").(*lua.LTable)

	L.SetGlobal("path_join", L.NewFunction(utils_lua.Ljoin))

	// Packets build functions

	osObject.RawSetString("remove", L.NewFunction(container.lRemove))
	osObject.RawSetString("rename", L.NewFunction(container.lRename))
	osObject.RawSetString("copy", L.NewFunction(container.lCopy))

	osObject.RawSetString("mkdir", L.NewFunction(container.lMkdir))
	return nil
}
