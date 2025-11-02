package lua

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

func LRemove(L *lua.LState) int {
	filename := L.CheckString(1)

	err := os.RemoveAll(filename)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LRename(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if err := os.Rename(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}
func LCopy(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	_ = os.MkdirAll(filepath.Dir(newname), 0755)
	if err := copyDir(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2

}

func LSymlink(L *lua.LState) int {
	fileName := L.CheckString(1)
	destination := L.CheckString(2)

	_ = os.RemoveAll(destination)
	if err := os.Symlink(fileName, destination); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func Ljoin(L *lua.LState) int {

	n := L.GetTop()
	parts := make([]string, 0, n)

	for i := 1; i <= n; i++ {
		val := L.Get(i)
		parts = append(parts, val.String())
	}

	result := filepath.Join(parts...)
	L.Push(lua.LString(result))
	return 1
}

func LMkdir(L *lua.LState) int {
	path := L.CheckString(1)
	perm := L.CheckInt(2)

	modeStr := strconv.Itoa(perm)
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		fmt.Println("Error parsing mode:", err)
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if err := os.MkdirAll(path, os.FileMode(modeUint)); err != nil {
		fmt.Println("Error creating directory:", err)
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LError(L *lua.LState) int {
	n := L.GetTop()
	parts := make([]any, 0, n)

	for i := 1; i <= n; i++ {
		val := L.Get(i)
		parts = append(parts, val.String())
	}

	llogger().Panic(parts...)
	return 0
}

func LSetEnv(L *lua.LState) int {
	env := L.CheckString(1)
	value := L.CheckString(2)
	os.Setenv(env, value)
	return 0
}

func LCD(L *lua.LState) int {
	dir := L.CheckString(1)

	if err := os.Chdir(dir); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LChmod(L *lua.LState) int {
	f := L.CheckString(1)
	mode := L.CheckInt(2)

	modeStr := strconv.Itoa(mode)
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		fmt.Println("Error parsing mode:", err)
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if err := os.Chmod(f, os.FileMode(modeUint)); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

type Flag struct {
	Name     string
	Path     string
	FlagType string
}

type Flags struct {
	Flags []Flag
}

func (f *Flags) LSetFlag(L *lua.LState) int {
	flagtype := L.CheckString(1)
	name := L.CheckString(2)
	flagPath := L.CheckString(3)

	f.Flags = append(f.Flags, Flag{
		Name:     name,
		Path:     flagPath,
		FlagType: flagtype,
	})

	return 0
}

func llogger() *log.Logger { return log.New(os.Stderr, "script error: ", 0) }
