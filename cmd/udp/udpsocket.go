package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func CheckDownloaded(filename string) bool {

	_, err := os.Stat(fmt.Sprintf("/var/cache/packets/%s", filename))
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}

}

func main() {
	pid := os.Getpid()
	if err := os.WriteFile("/opt/packets/packets/udp.pid", []byte(fmt.Sprint(pid)), 0644); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}

	addr := net.UDPAddr{IP: net.IPv4zero, Port: 1333}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	buf := make([]byte, 1500)

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("error creating udp socket", err)
		}
		msg := string(buf[:n])
		if !strings.HasPrefix(msg, "Q:") {
			continue
		}
		filename := strings.TrimPrefix(msg, "Q:")
		if CheckDownloaded(filename) {
			reply := fmt.Sprintf("H:%s:%d", filename, 9123)
			conn.WriteToUDP([]byte(reply), remote)
		}
	}
}
