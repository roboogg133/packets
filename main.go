package main

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func main() {
	L := lua.NewState()

	if err := L.DoFile("lua.lua"); err != nil {
		fmt.Println(err)
	}
}
