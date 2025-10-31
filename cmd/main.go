package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

func main() {
	f, err := os.ReadFile("test/bat/Packet.lua")
	if err != nil {
		log.Fatal(err)
	}

	pk, err := packet.ReadPacket(f, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(
		pk.Name,
		pk.Description,
	)

	if pk.Plataforms == nil {
		fmt.Print("NIGGER NIGGER NIGGER NIGGER NIGGER NIGGER")
		return
	}
	plataforms := *pk.Plataforms

	for _, v := range *plataforms[packet.OperationalSystem(runtime.GOOS)].Sources {
		fmt.Printf("%s %s\n", v.Method, v.Url)
		if v.Method == "GET" {
			fmt.Println(*v.Specs.(packet.GETSpecs).SHA256)
		}
	}
}
