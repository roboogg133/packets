package lockfile

import (
	"fmt"
	"strings"
)

func NewLockfile(packetsVersion, targetOS, targetArch string, packetSerial int, flagsGiven []string) string {

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("PACKETS VERSION %s = SERIAL %d\nTarget Plataform: %s\nTarget Architecture: %s\n", packetsVersion, packetSerial, targetOS, targetArch))

	builder.WriteString("FLAGS: [ ")
	for _, flag := range flagsGiven {
		builder.WriteString(flag)
		builder.WriteRune(' ')
	}
	builder.WriteRune(']')
	builder.WriteString("\n")
	builder.WriteString("\n")

	return builder.String()
}
