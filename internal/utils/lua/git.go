package utils_lua

import (
	"os"

	"github.com/go-git/go-git"
	lua "github.com/yuin/gopher-lua"
)

func LGitClone(L *lua.LState) int {
	uri := L.CheckString(1)
	output := L.CheckString(2)

	_, err := git.PlainClone(output, false, &git.CloneOptions{
		URL:      uri,
		Progress: os.Stdout,
	})
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func LGitCheckout(L)
