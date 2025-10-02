package utils_lua

import (
	"fmt"
	"log"
	"os"
	"packets/configs"
	"packets/internal/consts"
	"packets/internal/utils"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	lua "github.com/yuin/gopher-lua"
)

func IsSafe(str string) bool {
	s, err := filepath.EvalSymlinks(filepath.Clean(str))
	if err != nil {
		s = filepath.Clean(str)
	}

	var cfg configs.ConfigTOML

	f, err := os.Open(filepath.Join(consts.DefaultLinux_d, "config.toml"))
	if err != nil {
		log.Println("error here opening config.toml")
		return false
	}

	defer f.Close()

	decoder := toml.NewDecoder(f)

	if err := decoder.Decode(&cfg); err != nil {
		log.Println("error decoding")
		return false
	}

	if strings.HasPrefix(s, cfg.Config.Data_d) || strings.HasPrefix(s, cfg.Config.Bin_d) {
		return true

	} else if strings.Contains(s, ".ssh") {
		return false

	} else if strings.HasPrefix(s, "/etc") {
		return false

	} else if strings.HasPrefix(s, "/usr") || strings.HasPrefix(s, "/bin") {
		fmt.Println(s, "está dentro de usr")
		return strings.HasPrefix(s, "/usr/share")

	} else if strings.HasPrefix(s, "/var/mail") {
		return false

	} else if strings.HasPrefix(s, "/proc") {
		return false

	} else if strings.HasPrefix(s, "/sys") {
		return false

	} else if strings.HasPrefix(s, "/var/run") || strings.HasPrefix(s, "/run") {
		return false

	} else if strings.HasPrefix(s, "/tmp") {
		return false

	} else if strings.HasPrefix(s, "/dev") {
		return false

	} else if strings.HasPrefix(s, "/boot") {
		return false

	} else if strings.HasPrefix(s, "/home") {
		if strings.Contains(s, "/Pictures") || strings.Contains(s, "/Videos") || strings.Contains(s, "/Documents") || strings.Contains(s, "/Downloads") {
			return false
		}

	} else if strings.HasPrefix(s, "/lib") || strings.HasPrefix(s, "/lib64") || strings.HasPrefix(s, "/var/lib64") || strings.HasPrefix(s, "/lib") {
		return false

	} else if strings.HasPrefix(s, "/sbin") {
		return false

	} else if strings.HasPrefix(s, "/srv") {
		return false

	} else if strings.HasPrefix(s, "/mnt") {
		return false

	} else if strings.HasPrefix(s, "/media") {
		return false
	} else if strings.HasPrefix(s, "/snap") {
		return false
	}

	return true
}

// lua functions

func LSafeRemove(L *lua.LState) int {
	filename := L.CheckString(1)
	fmt.Printf("   remove %s\n", filename)

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

func LSafeRename(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	fmt.Printf("   move %s -> %s\n", oldname, newname)

	if err := os.Rename(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}
func LSafeCopy(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	fmt.Printf("   copy %s -> %s\n", oldname, newname)

	if err := utils.CopyDir(oldname, newname); err != nil {
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

	fmt.Printf("   symlink %s -> %s\n", fileName, destination)

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

/*
func LOpen(L *lua.LState) int {
	path := L.CheckString(1)
	mode := L.OptString(2, "r")

	file, err := os.OpenFile(path, modeFlags(mode), 0644)
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
*/

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
	fmt.Printf("   mkdir %s \n", path)

	/*
		if !IsSafe(path) {
			L.Push(lua.LFalse)
			L.Push(lua.LString("unsafe filepath"))
			return 2
		}
	*/

	if err := os.MkdirAll(path, os.FileMode(perm)); err != nil {
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

	Llogger().Panic(parts...)
	return 0
}

func Llogger() *log.Logger { return log.New(os.Stderr, "   script error: ", 0) }
