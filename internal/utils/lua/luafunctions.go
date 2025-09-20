package utils_lua

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"packets/configs"
	"packets/internal/consts"
	"packets/internal/utils"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	lua "github.com/yuin/gopher-lua"
)

var SandboxDir string

var AllowedCmds = map[string]string{
	"go":          "go",          // "Go code compiler"
	"gcc":         "gcc",         // "C"
	"g++":         "g++",         // "C++"
	"rustc":       "rustc",       // "Rust"
	"javac":       "javac",       // "Java"
	"luac":        "luac",        // "Lua"
	"pyinstaller": "pyinstaller", // "Python"
	"kotlinc":     "kotlinc",     // "Kotlin"
	"mcs":         "mcs",         // "C# compiler"
	"swiftc":      "swiftc",      // "Swift compiler"
	"tsc":         "tsc",         // "TypeScript compiler"
	"rubyc":       "rubyc",       // "Ruby compiler"
}

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
	if !IsSafe(filename) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}
	err := os.RemoveAll(filename)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] remove failed\n" + err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LSafeRename(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if !IsSafe(oldname) || !IsSafe(newname) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}

	if err := os.Rename(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] rename failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}
func LSafeCopy(L *lua.LState) int {
	oldname := L.CheckString(1)
	newname := L.CheckString(2)

	if !IsSafe(oldname) || !IsSafe(newname) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}

	if err := utils.CopyDir(oldname, newname); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] error while copy"))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2

}

func LSymlink(L *lua.LState) int {
	fileName := L.CheckString(1)
	destination := L.CheckString(2)

	if !IsSafe(fileName) || !IsSafe(destination) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}

	_ = os.RemoveAll(destination)
	if err := os.Symlink(fileName, destination); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] symlink failed\n" + err.Error()))
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

func LOpen(L *lua.LState) int {
	path := L.CheckString(1)
	mode := L.OptString(2, "r")

	if !IsSafe(path) {
		L.Push(lua.LNil)
		L.Push(lua.LString("[packets] unsafe filepath"))
		return 2
	}
	file, err := os.OpenFile(path, modeFlags(mode), 0644)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString("[packets] open failed\n" + err.Error()))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = file
	L.SetMetatable(ud, L.GetTypeMetatable("file"))
	L.Push(ud)
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

	if !IsSafe(path) {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsafe filepath\n"))
		return 2
	}

	if err := os.MkdirAll(path, os.FileMode(perm)); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] mkdir failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LCompile(L *lua.LState) int {
	lang := L.CheckString(1)
	args := []string{}
	for i := 2; i <= L.GetTop(); i++ {

		if strings.Contains(L.CheckString(i), "/") {

			tryintoacess, err := filepath.Abs(filepath.Clean(filepath.Join(SandboxDir, L.CheckString(i))))
			if err != nil {
				L.Push(lua.LFalse)
				L.Push(lua.LString("[packets] invalid filepath\n" + err.Error()))
				return 2
			}

			fmt.Printf("sandboxdir: (%s) acessto: (%s)\n", SandboxDir, tryintoacess)
			rel, err := filepath.Rel(SandboxDir, tryintoacess)
			if err != nil || strings.HasPrefix(rel, "..") {
				L.Push(lua.LFalse)
				L.Push(lua.LString("[packets] unsafe filepath"))
				return 2
			}
		}

		args = append(args, L.CheckString(i))
	}

	bin, suc := AllowedCmds[lang]
	if !suc {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] unsupported language"))
		return 2
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = SandboxDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] compile failed\n" + err.Error() + "\n" + string(out)))
		return 2
	}
	if err := cmd.Run(); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] compile failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LString(string(out)))
	return 2
}

func LCompileRequirements(L *lua.LState) int {

	cmdLang := L.CheckString(1)

	if strings.Contains(L.CheckString(2), "/") {

		tryintoacess, err := filepath.Abs(filepath.Clean(L.CheckString(2)))
		if err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString("[packets] invalid filepath\n" + err.Error()))
			return 2
		}
		if !strings.HasPrefix(tryintoacess, SandboxDir) {
			L.Push(lua.LFalse)
			L.Push(lua.LString("[packets] unsafe filepath"))
			return 2
		}
	}

	var err error

	switch cmdLang {
	case "python":
		cmd := exec.Command("pip", "install", "--target", filepath.Join(SandboxDir, "tmp/build"), "-r", L.CheckString(2))
		cmd.Dir = filepath.Join(SandboxDir, "data")
		err = cmd.Run()
	case "java":
		cmd := exec.Command("mvn", "dependency:copy-dependencies", "-DoutputDirectory="+filepath.Join(SandboxDir, "tmp/build"))
		cmd.Dir = L.CheckString(2)
		err = cmd.Run()
	case "ruby":
		cmd := exec.Command("bundle", "install", "--path", filepath.Join(SandboxDir, "tmp/build"))
		cmd.Dir = L.CheckString(2)
		err = cmd.Run()
	}

	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString("[packets] requirements install failed\n" + err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}
